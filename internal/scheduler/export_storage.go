// Package scheduler provides export schedule persistence operations.
//
// This file is responsible for:
// - Loading all export schedules from JSON file storage
// - Saving all export schedules to JSON file storage
// - Adding new export schedules with ID generation and defaults
// - Getting, updating, and deleting export schedules by ID
// - Listing all export schedules
//
// This file does NOT handle:
// - Concurrent access (ExportTrigger handles this with its own mutex)
// - Schedule validation (export_validation.go handles this)
// - Export execution (export_trigger.go handles that)
//
// Invariants:
// - File path is derived from dataDir: <dataDir>/export_schedules.json
// - Empty/missing file returns empty slice, not error
// - IDs are generated via uuid.NewString() if empty
// - CreatedAt is set to now if zero
// - UpdatedAt is updated on every save
package scheduler

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/fsutil"
	"github.com/google/uuid"
)

// ExportStorage handles persistence for export schedules.
type ExportStorage struct {
	dataDir string
}

// IsNotFoundError reports whether err wraps an export-schedule NotFoundError.
func IsNotFoundError(err error) bool {
	var notFoundErr *NotFoundError
	return errors.As(err, &notFoundErr)
}

// NotFoundError is returned when an export schedule is not found.
type NotFoundError struct {
	ID string
}

func (e *NotFoundError) Error() string {
	return "export schedule not found: " + e.ID
}

// NewExportStorage creates a new export schedule storage.
func NewExportStorage(dataDir string) *ExportStorage {
	return &ExportStorage{dataDir: dataDir}
}

// LoadAll loads all export schedules from storage.
func (s *ExportStorage) LoadAll() ([]ExportSchedule, error) {
	path := s.schedulesPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []ExportSchedule{}, nil
		}
		return nil, err
	}
	var store exportScheduleStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	return store.Schedules, nil
}

// SaveAll saves all export schedules to storage.
func (s *ExportStorage) SaveAll(schedules []ExportSchedule) error {
	if err := fsutil.EnsureDataDir(s.dataDir); err != nil {
		return err
	}
	path := s.schedulesPath()
	payload, err := json.MarshalIndent(exportScheduleStore{Schedules: schedules}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o600)
}

// Get retrieves a single export schedule by ID.
func (s *ExportStorage) Get(id string) (*ExportSchedule, error) {
	schedules, err := s.LoadAll()
	if err != nil {
		return nil, err
	}
	for i := range schedules {
		if schedules[i].ID == id {
			return &schedules[i], nil
		}
	}
	return nil, &NotFoundError{ID: id}
}

// Add adds a new export schedule with generated ID and defaults.
func (s *ExportStorage) Add(schedule ExportSchedule) (*ExportSchedule, error) {
	if schedule.ID == "" {
		schedule.ID = uuid.NewString()
	}
	now := time.Now()
	if schedule.CreatedAt.IsZero() {
		schedule.CreatedAt = now
	}
	schedule.UpdatedAt = now

	// Apply default retry config if not set
	if schedule.Retry.MaxRetries == 0 && schedule.Retry.BaseDelayMs == 0 {
		schedule.Retry = DefaultRetryConfig()
	}

	if err := ValidateExportSchedule(schedule); err != nil {
		return nil, err
	}

	items, err := s.LoadAll()
	if err != nil {
		return nil, err
	}
	items = append(items, schedule)
	if err := s.SaveAll(items); err != nil {
		return nil, err
	}
	return &schedule, nil
}

// Update updates an existing export schedule.
func (s *ExportStorage) Update(schedule ExportSchedule) (*ExportSchedule, error) {
	schedule.UpdatedAt = time.Now()

	if err := ValidateExportSchedule(schedule); err != nil {
		return nil, err
	}

	items, err := s.LoadAll()
	if err != nil {
		return nil, err
	}

	found := false
	for i := range items {
		if items[i].ID == schedule.ID {
			items[i] = schedule
			found = true
			break
		}
	}

	if !found {
		return nil, &NotFoundError{ID: schedule.ID}
	}

	if err := s.SaveAll(items); err != nil {
		return nil, err
	}
	return &schedule, nil
}

// Delete removes an export schedule by ID.
func (s *ExportStorage) Delete(id string) error {
	items, err := s.LoadAll()
	if err != nil {
		return err
	}
	filtered := make([]ExportSchedule, 0, len(items))
	found := false
	for _, item := range items {
		if item.ID != id {
			filtered = append(filtered, item)
			continue
		}
		found = true
	}
	if !found {
		return &NotFoundError{ID: id}
	}
	return s.SaveAll(filtered)
}

// List returns all export schedules.
func (s *ExportStorage) List() ([]ExportSchedule, error) {
	return s.LoadAll()
}

// schedulesPath returns the file path for export schedules.
func (s *ExportStorage) schedulesPath() string {
	base := s.dataDir
	if base == "" {
		base = ".data"
	}
	return filepath.Join(base, "export_schedules.json")
}
