// Package webhook provides persistent delivery tracking for webhook notifications.
//
// The store tracks delivery attempts, their status, and any errors encountered.
// This enables querying delivery history and debugging webhook issues.
//
// This file does NOT:
//   - Handle webhook dispatch (see dispatcher.go)
//   - Manage webhook configuration
//   - Provide retry logic for failed deliveries
package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

// DeliveryStatus represents the state of a webhook delivery.
type DeliveryStatus string

const (
	// DeliveryStatusPending indicates the delivery is in progress.
	DeliveryStatusPending DeliveryStatus = "pending"
	// DeliveryStatusDelivered indicates the delivery was successful.
	DeliveryStatusDelivered DeliveryStatus = "delivered"
	// DeliveryStatusFailed indicates the delivery failed after all retries.
	DeliveryStatusFailed DeliveryStatus = "failed"
)

// DeliveryRecord tracks a single webhook delivery attempt.
type DeliveryRecord struct {
	ID           string         `json:"id"`
	EventID      string         `json:"eventId"`
	EventType    EventType      `json:"eventType"`
	JobID        string         `json:"jobId"`
	URL          string         `json:"url"`
	Status       DeliveryStatus `json:"status"`
	Attempts     int            `json:"attempts"`
	LastError    string         `json:"lastError,omitempty"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	DeliveredAt  *time.Time     `json:"deliveredAt,omitempty"`
	ResponseCode int            `json:"responseCode,omitempty"`
}

// Store provides persistent storage for webhook delivery records.
type Store struct {
	dataDir string
	mu      sync.RWMutex
	records map[string]*DeliveryRecord
}

// NewStore creates a new webhook delivery store.
func NewStore(dataDir string) *Store {
	return &Store{
		dataDir: dataDir,
		records: make(map[string]*DeliveryRecord),
	}
}

// storePath returns the path to the delivery records file.
func (s *Store) storePath() string {
	return filepath.Join(s.dataDir, "webhook_deliveries.json")
}

// Load reads existing delivery records from disk.
func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.storePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// No existing file, start with empty store
			return nil
		}
		return apperrors.Wrap(apperrors.KindInternal, "failed to read webhook deliveries file", err)
	}

	var records []*DeliveryRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to unmarshal webhook deliveries", err)
	}

	s.records = make(map[string]*DeliveryRecord, len(records))
	for _, r := range records {
		s.records[r.ID] = r
	}

	return nil
}

// Save persists delivery records to disk.
func (s *Store) Save() error {
	s.mu.RLock()
	records := make([]*DeliveryRecord, 0, len(s.records))
	for _, r := range s.records {
		records = append(records, r)
	}
	s.mu.RUnlock()

	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to marshal webhook deliveries", err)
	}

	path := s.storePath()
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to create data directory", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to write webhook deliveries file", err)
	}

	return nil
}

// CreateRecord creates a new delivery record.
func (s *Store) CreateRecord(ctx context.Context, record *DeliveryRecord) error {
	if record.ID == "" {
		return apperrors.Validation("delivery record ID is required")
	}
	if record.EventID == "" {
		return apperrors.Validation("delivery record EventID is required")
	}
	if record.URL == "" {
		return apperrors.Validation("delivery record URL is required")
	}

	s.mu.Lock()
	s.records[record.ID] = record
	s.mu.Unlock()

	return s.Save()
}

// UpdateRecord updates an existing delivery record.
func (s *Store) UpdateRecord(ctx context.Context, record *DeliveryRecord) error {
	if record.ID == "" {
		return apperrors.Validation("delivery record ID is required")
	}

	s.mu.Lock()
	if _, exists := s.records[record.ID]; !exists {
		s.mu.Unlock()
		return apperrors.NotFound(fmt.Sprintf("delivery record not found: %s", record.ID))
	}
	record.UpdatedAt = time.Now()
	s.records[record.ID] = record
	s.mu.Unlock()

	return s.Save()
}

// GetRecord retrieves a delivery record by ID.
func (s *Store) GetRecord(ctx context.Context, id string) (*DeliveryRecord, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	record, exists := s.records[id]
	if !exists {
		return nil, false, nil
	}

	// Return a copy to prevent external modification
	copy := *record
	return &copy, true, nil
}

// ListRecords lists delivery records with optional filtering.
// If jobID is non-empty, only records for that job are returned.
// Results are sorted by CreatedAt descending (newest first).
// Offset can be used for pagination (skips the first N results).
func (s *Store) ListRecords(ctx context.Context, jobID string, limit, offset int) ([]*DeliveryRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	var results []*DeliveryRecord
	for _, r := range s.records {
		if jobID != "" && r.JobID != jobID {
			continue
		}
		// Make a copy
		copy := *r
		results = append(results, &copy)
	}

	// Sort by CreatedAt descending
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].CreatedAt.After(results[i].CreatedAt) {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Apply offset
	if offset >= len(results) {
		return []*DeliveryRecord{}, nil
	}
	results = results[offset:]

	// Apply limit
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// CountRecords returns the total number of delivery records, optionally filtered by jobID.
func (s *Store) CountRecords(ctx context.Context, jobID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if jobID == "" {
		return len(s.records), nil
	}

	count := 0
	for _, r := range s.records {
		if r.JobID == jobID {
			count++
		}
	}
	return count, nil
}

// DeleteRecord removes a delivery record by ID.
func (s *Store) DeleteRecord(ctx context.Context, id string) error {
	s.mu.Lock()
	delete(s.records, id)
	s.mu.Unlock()

	return s.Save()
}

// Count returns the total number of delivery records.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.records)
}
