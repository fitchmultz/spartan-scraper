// Package feed provides RSS/Atom feed monitoring and parsing functionality.
//
// This file is responsible for:
// - Loading all feeds from JSON file storage
// - Saving all feeds to JSON file storage
// - Adding new feeds with ID generation and defaults
// - Deleting feeds by ID
// - Listing feeds sorted by next run time
//
// This file does NOT handle:
// - Concurrent access (scheduler handles this)
// - Feed validation (types.go does this)
// - Feed execution (feed.go does this)
//
// Invariants:
// - File path is derived from dataDir: <dataDir>/feeds.json
// - Empty/missing file returns empty slice, not error
// - IDs are generated via uuid.NewString() if empty
// - IntervalSeconds defaults to 3600 if <= 0
package feed

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

// Storage defines the interface for feed persistence.
type Storage interface {
	Add(feed *Feed) (*Feed, error)
	Update(feed *Feed) error
	Delete(id string) error
	Get(id string) (*Feed, error)
	List() ([]Feed, error)
	ListEnabled() ([]Feed, error)
}

// FileStorage implements Storage using JSON files.
type FileStorage struct {
	dataDir string
}

// NewFileStorage creates a new file-based storage.
func NewFileStorage(dataDir string) *FileStorage {
	return &FileStorage{dataDir: dataDir}
}

// feedsPath returns the path to the feeds file.
func (s *FileStorage) feedsPath() string {
	base := s.dataDir
	if base == "" {
		base = ".data"
	}
	return filepath.Join(base, "feeds.json")
}

type feedStore struct {
	Feeds []Feed `json:"feeds"`
}

// LoadAll loads all feeds from storage.
func (s *FileStorage) LoadAll() ([]Feed, error) {
	path := s.feedsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Feed{}, nil
		}
		return nil, err
	}
	var store feedStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	return store.Feeds, nil
}

// SaveAll saves all feeds to storage.
func (s *FileStorage) SaveAll(feeds []Feed) error {
	if err := fsutil.EnsureDataDir(s.dataDir); err != nil {
		return err
	}
	path := s.feedsPath()
	payload, err := json.MarshalIndent(feedStore{Feeds: feeds}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o600)
}

// Add adds a new feed to storage.
func (s *FileStorage) Add(feed *Feed) (*Feed, error) {
	if feed.ID == "" {
		feed.ID = uuid.NewString()
	}
	if feed.IntervalSeconds <= 0 {
		feed.IntervalSeconds = 3600
	}
	if feed.CreatedAt.IsZero() {
		feed.CreatedAt = time.Now()
	}
	if feed.FeedType == "" {
		feed.FeedType = FeedTypeAuto
	}

	if err := feed.Validate(); err != nil {
		return nil, err
	}

	items, err := s.LoadAll()
	if err != nil {
		return nil, err
	}
	items = append(items, *feed)
	if err := s.SaveAll(items); err != nil {
		return nil, err
	}
	return feed, nil
}

// Update updates an existing feed.
func (s *FileStorage) Update(feed *Feed) error {
	if err := feed.Validate(); err != nil {
		return err
	}

	items, err := s.LoadAll()
	if err != nil {
		return err
	}

	found := false
	for i, item := range items {
		if item.ID == feed.ID {
			items[i] = *feed
			found = true
			break
		}
	}

	if !found {
		return &NotFoundError{ID: feed.ID}
	}

	return s.SaveAll(items)
}

// Delete removes a feed by ID.
func (s *FileStorage) Delete(id string) error {
	items, err := s.LoadAll()
	if err != nil {
		return err
	}
	filtered := make([]Feed, 0, len(items))
	for _, item := range items {
		if item.ID != id {
			filtered = append(filtered, item)
		}
	}
	return s.SaveAll(filtered)
}

// Get retrieves a feed by ID.
func (s *FileStorage) Get(id string) (*Feed, error) {
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

// List returns all feeds sorted by created time.
func (s *FileStorage) List() ([]Feed, error) {
	items, err := s.LoadAll()
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	return items, nil
}

// ListEnabled returns all enabled feeds sorted by next run time.
func (s *FileStorage) ListEnabled() ([]Feed, error) {
	items, err := s.LoadAll()
	if err != nil {
		return nil, err
	}
	var enabled []Feed
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
