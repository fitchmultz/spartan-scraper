// Package watch provides content change monitoring functionality.
//
// This file is responsible for:
// - Loading all watches from JSON file storage
// - Saving all watches to JSON file storage
// - Adding new watches with ID generation and defaults
// - Deleting watches by ID
// - Listing watches sorted by next run time
//
// This file does NOT handle:
// - Concurrent access (scheduler handles this)
// - Watch validation (types.go does this)
// - Watch execution (watch.go does this)
//
// Invariants:
// - File path is derived from dataDir: <dataDir>/watches.json
// - Empty/missing file returns empty slice, not error
// - IDs are generated via uuid.NewString() if empty
// - IntervalSeconds defaults to 3600 if <= 0
package watch

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/fsutil"
	"github.com/google/uuid"
)

// Storage defines the interface for watch persistence.
type Storage interface {
	Add(watch *Watch) (*Watch, error)
	Update(watch *Watch) error
	Delete(id string) error
	Get(id string) (*Watch, error)
	List() ([]Watch, error)
	ListEnabled() ([]Watch, error)
}

// FileStorage implements Storage using JSON files.
type FileStorage struct {
	dataDir string
}

// NewFileStorage creates a new file-based storage.
func NewFileStorage(dataDir string) *FileStorage {
	return &FileStorage{dataDir: dataDir}
}

// watchesPath returns the path to the watches file.
func (s *FileStorage) watchesPath() string {
	base := s.dataDir
	if base == "" {
		base = ".data"
	}
	return filepath.Join(base, "watches.json")
}

type watchStore struct {
	Watches []Watch `json:"watches"`
}

// LoadAll loads all watches from storage.
func (s *FileStorage) LoadAll() ([]Watch, error) {
	path := s.watchesPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Watch{}, nil
		}
		return nil, err
	}
	var store watchStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	return store.Watches, nil
}

// SaveAll saves all watches to storage.
func (s *FileStorage) SaveAll(watches []Watch) error {
	if err := fsutil.EnsureDataDir(s.dataDir); err != nil {
		return err
	}
	path := s.watchesPath()
	payload, err := json.MarshalIndent(watchStore{Watches: watches}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o600)
}

// Add adds a new watch to storage.
func (s *FileStorage) Add(watch *Watch) (*Watch, error) {
	if watch.ID == "" {
		watch.ID = uuid.NewString()
	}
	if watch.IntervalSeconds <= 0 {
		watch.IntervalSeconds = 3600
	}
	if watch.CreatedAt.IsZero() {
		watch.CreatedAt = time.Now()
	}
	if watch.DiffFormat == "" {
		watch.DiffFormat = "unified"
	}

	if err := watch.Validate(); err != nil {
		return nil, err
	}

	items, err := s.LoadAll()
	if err != nil {
		return nil, err
	}
	items = append(items, *watch)
	if err := s.SaveAll(items); err != nil {
		return nil, err
	}
	return watch, nil
}

// Update updates an existing watch.
func (s *FileStorage) Update(watch *Watch) error {
	if err := watch.Validate(); err != nil {
		return err
	}

	items, err := s.LoadAll()
	if err != nil {
		return err
	}

	found := false
	for i, item := range items {
		if item.ID == watch.ID {
			items[i] = *watch
			found = true
			break
		}
	}

	if !found {
		return &NotFoundError{ID: watch.ID}
	}

	return s.SaveAll(items)
}

// Delete removes a watch by ID.
func (s *FileStorage) Delete(id string) error {
	items, err := s.LoadAll()
	if err != nil {
		return err
	}
	filtered := make([]Watch, 0, len(items))
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
	if err := s.SaveAll(filtered); err != nil {
		return err
	}
	return NewArtifactStore(s.dataDir).RemoveAll(id)
}

// Get retrieves a watch by ID.
func (s *FileStorage) Get(id string) (*Watch, error) {
	items, err := s.LoadAll()
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item.ID == id {
			return &item, nil
		}
	}
	return nil, &NotFoundError{ID: id}
}

// List returns all watches sorted by created time.
func (s *FileStorage) List() ([]Watch, error) {
	items, err := s.LoadAll()
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	return items, nil
}

// ListEnabled returns all enabled watches sorted by next run time.
func (s *FileStorage) ListEnabled() ([]Watch, error) {
	items, err := s.LoadAll()
	if err != nil {
		return nil, err
	}
	var enabled []Watch
	for _, item := range items {
		if item.Enabled {
			enabled = append(enabled, item)
		}
	}
	sort.Slice(enabled, func(i, j int) bool {
		return enabled[i].NextRun().Before(enabled[j].NextRun())
	})
	return enabled, nil
}

// NotFoundError is returned when a watch is not found.
type NotFoundError struct {
	ID string
}

func (e *NotFoundError) Error() string {
	return "watch not found: " + e.ID
}

// IsNotFoundError reports whether err wraps a watch NotFoundError.
func IsNotFoundError(err error) bool {
	var notFoundErr *NotFoundError
	return errors.As(err, &notFoundErr)
}
