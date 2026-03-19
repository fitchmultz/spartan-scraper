// Package scheduler provides export history tracking for automated and direct exports.
//
// Purpose:
// - Persist one canonical export-outcome history shared by schedules, API, CLI, and MCP.
//
// Responsibilities:
// - Record export attempts, completion metadata, retry counts, and failure context.
// - Normalize legacy history records into the current outcome contract.
// - Provide filtered lookup helpers for schedule-scoped and job-scoped inspection.
//
// Scope:
// - History persistence only; export execution lives in export_trigger.go and direct export surfaces.
//
// Usage:
// - Used by export triggers, direct export handlers, CLI commands, and MCP tools.
//
// Invariants/Assumptions:
// - File path is derived from dataDir: <dataDir>/export_history.json.
// - ExportRecord IDs are UUIDs generated on creation.
// - Status transitions normalize to pending -> succeeded|failed.
// - Legacy persisted status values such as success are migrated on load.
package scheduler

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/exporter"
	"github.com/fitchmultz/spartan-scraper/internal/fsutil"
	"github.com/google/uuid"
)

type CreateRecordInput struct {
	ScheduleID  string
	JobID       string
	Trigger     exporter.OutcomeTrigger
	Destination string
	Request     exporter.ResultExportConfig
}

// ExportRecord tracks a single export execution.
type ExportRecord struct {
	ID          string                  `json:"id"`
	ScheduleID  string                  `json:"schedule_id,omitempty"`
	JobID       string                  `json:"job_id"`
	Trigger     exporter.OutcomeTrigger `json:"trigger,omitempty"`
	Status      exporter.OutcomeStatus  `json:"status"`
	Destination string                  `json:"destination,omitempty"`
	ExportedAt  time.Time               `json:"exported_at"`
	CompletedAt *time.Time              `json:"completed_at,omitempty"`
	RetryCount  int                     `json:"retry_count"`

	Request     exporter.ResultExportConfig `json:"request,omitempty"`
	Format      string                      `json:"format,omitempty"`
	Filename    string                      `json:"filename,omitempty"`
	ContentType string                      `json:"content_type,omitempty"`
	ExportSize  int64                       `json:"export_size,omitempty"`
	RecordCount int                         `json:"record_count,omitempty"`

	Failure      *exporter.FailureContext `json:"failure,omitempty"`
	ErrorMessage string                   `json:"error_message,omitempty"`
}

// ExportHistoryStore manages export history persistence.
type ExportHistoryStore struct {
	dataDir string
	mu      sync.RWMutex
}

type exportHistoryStore struct {
	Records []ExportRecord `json:"records"`
}

// NewExportHistoryStore creates a new export history store.
func NewExportHistoryStore(dataDir string) *ExportHistoryStore {
	return &ExportHistoryStore{dataDir: dataDir}
}

// CreateRecord creates a new export history record.
func (s *ExportHistoryStore) CreateRecord(input CreateRecordInput) (*ExportRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	request := exporter.NormalizeResultExportConfig(input.Request)
	record := &ExportRecord{
		ID:          uuid.NewString(),
		ScheduleID:  strings.TrimSpace(input.ScheduleID),
		JobID:       strings.TrimSpace(input.JobID),
		Trigger:     input.Trigger,
		Status:      exporter.OutcomePending,
		Destination: strings.TrimSpace(input.Destination),
		ExportedAt:  time.Now(),
		RetryCount:  0,
		Request:     request,
		Format:      request.Format,
		ContentType: exporter.ResultExportContentType(request.Format),
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

	record.Request = exporter.NormalizeResultExportConfig(record.Request)
	normalizeLegacyExportRecord(&record)

	history, err := s.loadUnsafe()
	if err != nil {
		return err
	}

	for i := range history.Records {
		if history.Records[i].ID == record.ID {
			history.Records[i] = record
			return s.saveUnsafe(history)
		}
	}

	return errors.New("export record not found")
}

// MarkSuccess marks an export record as successful.
func (s *ExportHistoryStore) MarkSuccess(recordID string, rendered exporter.RenderedResultExport) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	history, err := s.loadUnsafe()
	if err != nil {
		return err
	}

	now := time.Now()
	for i := range history.Records {
		if history.Records[i].ID != recordID {
			continue
		}
		history.Records[i].Status = exporter.OutcomeSucceeded
		history.Records[i].CompletedAt = &now
		history.Records[i].Failure = nil
		history.Records[i].ErrorMessage = ""
		history.Records[i].Format = rendered.Format
		history.Records[i].Filename = rendered.Filename
		history.Records[i].ContentType = rendered.ContentType
		history.Records[i].ExportSize = rendered.Size
		history.Records[i].RecordCount = rendered.RecordCount
		return s.saveUnsafe(history)
	}

	return errors.New("export record not found")
}

// MarkFailed marks an export record as failed.
func (s *ExportHistoryStore) MarkFailed(recordID string, exportErr error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	history, err := s.loadUnsafe()
	if err != nil {
		return err
	}

	now := time.Now()
	for i := range history.Records {
		if history.Records[i].ID != recordID {
			continue
		}
		history.Records[i].Status = exporter.OutcomeFailed
		history.Records[i].CompletedAt = &now
		history.Records[i].Failure = exporter.BuildFailureContext(exportErr)
		if history.Records[i].Failure != nil {
			history.Records[i].ErrorMessage = history.Records[i].Failure.Summary
		}
		return s.saveUnsafe(history)
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
			if record.Status == exporter.OutcomeSucceeded || record.Status == exporter.OutcomePending {
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

	records := make([]ExportRecord, 0)
	for _, record := range history.Records {
		if record.ScheduleID == scheduleID {
			records = append(records, record)
		}
	}

	return paginateExportRecords(records, limit, offset), len(records), nil
}

// GetByJob returns export history for a specific job.
func (s *ExportHistoryStore) GetByJob(jobID string, limit, offset int) ([]ExportRecord, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history, err := s.loadUnsafe()
	if err != nil {
		return nil, 0, err
	}

	records := make([]ExportRecord, 0)
	for _, record := range history.Records {
		if record.JobID == jobID {
			records = append(records, record)
		}
	}

	return paginateExportRecords(records, limit, offset), len(records), nil
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
			record := history.Records[i]
			return &record, nil
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

	pending := make([]ExportRecord, 0)
	for _, record := range history.Records {
		if record.Status == exporter.OutcomePending {
			pending = append(pending, record)
		}
	}
	sortExportRecords(pending)
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
	filtered := make([]ExportRecord, 0, len(history.Records))
	for _, record := range history.Records {
		if record.ExportedAt.After(cutoff) {
			filtered = append(filtered, record)
		}
	}

	history.Records = filtered
	return s.saveUnsafe(history)
}

func paginateExportRecords(records []ExportRecord, limit, offset int) []ExportRecord {
	sortExportRecords(records)
	total := len(records)
	if offset >= total {
		return []ExportRecord{}
	}
	end := offset + limit
	if limit <= 0 || end > total {
		end = total
	}
	return records[offset:end]
}

func sortExportRecords(records []ExportRecord) {
	sort.SliceStable(records, func(i, j int) bool {
		return records[i].ExportedAt.After(records[j].ExportedAt)
	})
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
	for i := range history.Records {
		normalizeLegacyExportRecord(&history.Records[i])
	}
	return &history, nil
}

func normalizeLegacyExportRecord(record *ExportRecord) {
	record.Request = exporter.NormalizeResultExportConfig(record.Request)
	if record.Status == "success" {
		record.Status = exporter.OutcomeSucceeded
	}
	if record.Trigger == "" {
		if strings.TrimSpace(record.ScheduleID) != "" {
			record.Trigger = exporter.OutcomeTriggerSchedule
		} else {
			record.Trigger = exporter.OutcomeTriggerAPI
		}
	}
	if record.Failure == nil && strings.TrimSpace(record.ErrorMessage) != "" {
		record.Failure = exporter.BuildFailureContext(errors.New(record.ErrorMessage))
	}
	if record.Format == "" {
		record.Format = record.Request.Format
	}
	if record.ContentType == "" {
		record.ContentType = exporter.ResultExportContentType(record.Format)
	}
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
