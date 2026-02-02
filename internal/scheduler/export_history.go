// Package scheduler provides export history tracking for automated exports.
//
// This file is responsible for:
// - Recording export execution history (pending, success, failed)
// - Tracking retry attempts and errors
// - Preventing duplicate exports via history lookup
// - Providing audit trail for export operations
//
// This file does NOT handle:
// - Export schedule persistence (export_storage.go handles that)
// - Export execution (export_trigger.go handles that)
// - Export validation (export_validation.go handles that)
//
// Invariants:
// - File path is derived from dataDir: <dataDir>/export_history.json
// - ExportRecord IDs are UUIDs generated on creation
// - Status transitions: pending -> success|failed
// - RetryCount increments on each retry attempt
package scheduler

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/fsutil"
	"github.com/google/uuid"
)

// ExportRecord tracks a single export execution.
type ExportRecord struct {
	ID           string     `json:"id"`
	ScheduleID   string     `json:"schedule_id"`
	JobID        string     `json:"job_id"`
	Status       string     `json:"status"` // pending, success, failed
	Destination  string     `json:"destination"`
	ExportedAt   time.Time  `json:"exported_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
	RetryCount   int        `json:"retry_count"`
	ExportSize   int64      `json:"export_size,omitempty"`
	RecordCount  int        `json:"record_count,omitempty"`
}

// ExportHistoryStore manages export history persistence.
type ExportHistoryStore struct {
	dataDir string
	mu      sync.RWMutex
}

// exportHistoryStore is the JSON persistence format.
type exportHistoryStore struct {
	Records []ExportRecord `json:"records"`
}

// NewExportHistoryStore creates a new export history store.
func NewExportHistoryStore(dataDir string) *ExportHistoryStore {
	return &ExportHistoryStore{dataDir: dataDir}
}

// CreateRecord creates a new export history record.
func (s *ExportHistoryStore) CreateRecord(scheduleID, jobID, destination string) (*ExportRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record := &ExportRecord{
		ID:          uuid.NewString(),
		ScheduleID:  scheduleID,
		JobID:       jobID,
		Status:      "pending",
		Destination: destination,
		ExportedAt:  time.Now(),
		RetryCount:  0,
	}

	history, err := s.loadUnsafe()
	if err != nil {
		return nil, err
	}

	history.Records = append(history.Records, *record)
	if err := s.saveUnsafe(history); err != nil {
		return nil, err
	}

	return record, nil
}

// UpdateRecord updates an existing export history record.
func (s *ExportHistoryStore) UpdateRecord(record ExportRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	history, err := s.loadUnsafe()
	if err != nil {
		return err
	}

	found := false
	for i := range history.Records {
		if history.Records[i].ID == record.ID {
			history.Records[i] = record
			found = true
			break
		}
	}

	if !found {
		return errors.New("export record not found")
	}

	return s.saveUnsafe(history)
}

// MarkSuccess marks an export record as successful.
func (s *ExportHistoryStore) MarkSuccess(recordID string, exportSize int64, recordCount int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	history, err := s.loadUnsafe()
	if err != nil {
		return err
	}

	now := time.Now()
	for i := range history.Records {
		if history.Records[i].ID == recordID {
			history.Records[i].Status = "success"
			history.Records[i].CompletedAt = &now
			history.Records[i].ExportSize = exportSize
			history.Records[i].RecordCount = recordCount
			return s.saveUnsafe(history)
		}
	}

	return errors.New("export record not found")
}

// MarkFailed marks an export record as failed.
func (s *ExportHistoryStore) MarkFailed(recordID string, errorMessage string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	history, err := s.loadUnsafe()
	if err != nil {
		return err
	}

	now := time.Now()
	for i := range history.Records {
		if history.Records[i].ID == recordID {
			history.Records[i].Status = "failed"
			history.Records[i].CompletedAt = &now
			history.Records[i].ErrorMessage = errorMessage
			return s.saveUnsafe(history)
		}
	}

	return errors.New("export record not found")
}

// IncrementRetry increments the retry count for a record.
func (s *ExportHistoryStore) IncrementRetry(recordID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	history, err := s.loadUnsafe()
	if err != nil {
		return err
	}

	for i := range history.Records {
		if history.Records[i].ID == recordID {
			history.Records[i].RetryCount++
			return s.saveUnsafe(history)
		}
	}

	return errors.New("export record not found")
}

// HasExported checks if a job has already been exported by a schedule.
func (s *ExportHistoryStore) HasExported(scheduleID, jobID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history, err := s.loadUnsafe()
	if err != nil {
		return false
	}

	for _, record := range history.Records {
		if record.ScheduleID == scheduleID && record.JobID == jobID {
			// Consider it exported if status is success or pending (in progress)
			if record.Status == "success" || record.Status == "pending" {
				return true
			}
		}
	}

	return false
}

// GetBySchedule returns export history for a specific schedule.
func (s *ExportHistoryStore) GetBySchedule(scheduleID string, limit, offset int) ([]ExportRecord, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history, err := s.loadUnsafe()
	if err != nil {
		return nil, 0, err
	}

	var records []ExportRecord
	for _, record := range history.Records {
		if record.ScheduleID == scheduleID {
			records = append(records, record)
		}
	}

	total := len(records)

	// Apply offset and limit
	if offset >= total {
		return []ExportRecord{}, total, nil
	}
	end := offset + limit
	if end > total || limit <= 0 {
		end = total
	}

	return records[offset:end], total, nil
}

// GetByID retrieves a single export record by ID.
func (s *ExportHistoryStore) GetByID(recordID string) (*ExportRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history, err := s.loadUnsafe()
	if err != nil {
		return nil, err
	}

	for i := range history.Records {
		if history.Records[i].ID == recordID {
			return &history.Records[i], nil
		}
	}

	return nil, errors.New("export record not found")
}

// GetPending returns export records with pending status.
func (s *ExportHistoryStore) GetPending() ([]ExportRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history, err := s.loadUnsafe()
	if err != nil {
		return nil, err
	}

	var pending []ExportRecord
	for _, record := range history.Records {
		if record.Status == "pending" {
			pending = append(pending, record)
		}
	}

	return pending, nil
}

// CleanupOldRecords removes records older than the specified retention period.
func (s *ExportHistoryStore) CleanupOldRecords(retention time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	history, err := s.loadUnsafe()
	if err != nil {
		return err
	}

	cutoff := time.Now().Add(-retention)
	var filtered []ExportRecord
	for _, record := range history.Records {
		if record.ExportedAt.After(cutoff) {
			filtered = append(filtered, record)
		}
	}

	history.Records = filtered
	return s.saveUnsafe(history)
}

// loadUnsafe loads history without locking (caller must hold lock).
func (s *ExportHistoryStore) loadUnsafe() (*exportHistoryStore, error) {
	path := s.historyPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &exportHistoryStore{Records: []ExportRecord{}}, nil
		}
		return nil, err
	}
	var history exportHistoryStore
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, err
	}
	return &history, nil
}

// saveUnsafe saves history without locking (caller must hold lock).
func (s *ExportHistoryStore) saveUnsafe(history *exportHistoryStore) error {
	if err := fsutil.EnsureDataDir(s.dataDir); err != nil {
		return err
	}
	path := s.historyPath()
	payload, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o600)
}

// historyPath returns the file path for export history.
func (s *ExportHistoryStore) historyPath() string {
	base := s.dataDir
	if base == "" {
		base = ".data"
	}
	return filepath.Join(base, "export_history.json")
}
