// Package feed provides RSS/Atom feed monitoring and parsing functionality.
//
// This file is responsible for:
// - Tracking seen feed items for deduplication
// - Persisting seen items to JSON file storage
// - Cleaning up old seen items
//
// Invariants:
// - File path is derived from dataDir: <dataDir>/feed_seen_items.json
// - Items are keyed by feed ID, then by item GUID
// - Empty/missing file returns empty map, not error
package feed

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/fsutil"
)

// SeenItemStorage defines the interface for seen item persistence.
type SeenItemStorage interface {
	IsSeen(feedID, guid string) bool
	MarkSeen(feedID string, item SeenItem) error
	GetSeen(feedID string) ([]SeenItem, error)
	Cleanup(feedID string, before time.Time) error
	CleanupAll(before time.Time) error
}

// FileSeenStorage implements SeenItemStorage using JSON files.
type FileSeenStorage struct {
	dataDir string
}

// NewFileSeenStorage creates a new file-based seen item storage.
func NewFileSeenStorage(dataDir string) *FileSeenStorage {
	return &FileSeenStorage{dataDir: dataDir}
}

// seenItemsPath returns the path to the seen items file.
func (s *FileSeenStorage) seenItemsPath() string {
	base := s.dataDir
	if base == "" {
		base = ".data"
	}
	return filepath.Join(base, "feed_seen_items.json")
}

type seenItemStore struct {
	Items map[string]map[string]SeenItem `json:"items"` // feedID -> guid -> SeenItem
}

// LoadAll loads all seen items from storage.
func (s *FileSeenStorage) LoadAll() (map[string]map[string]SeenItem, error) {
	path := s.seenItemsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return make(map[string]map[string]SeenItem), nil
		}
		return nil, err
	}
	var store seenItemStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	if store.Items == nil {
		store.Items = make(map[string]map[string]SeenItem)
	}
	return store.Items, nil
}

// SaveAll saves all seen items to storage.
func (s *FileSeenStorage) SaveAll(items map[string]map[string]SeenItem) error {
	if err := fsutil.EnsureDataDir(s.dataDir); err != nil {
		return err
	}
	path := s.seenItemsPath()
	payload, err := json.MarshalIndent(seenItemStore{Items: items}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o600)
}

// IsSeen returns true if an item has been seen before.
func (s *FileSeenStorage) IsSeen(feedID, guid string) bool {
	items, err := s.LoadAll()
	if err != nil {
		return false
	}
	feedItems, ok := items[feedID]
	if !ok {
		return false
	}
	_, seen := feedItems[guid]
	return seen
}

// MarkSeen marks an item as seen.
func (s *FileSeenStorage) MarkSeen(feedID string, item SeenItem) error {
	items, err := s.LoadAll()
	if err != nil {
		return err
	}
	if items[feedID] == nil {
		items[feedID] = make(map[string]SeenItem)
	}
	items[feedID][item.GUID] = item
	return s.SaveAll(items)
}

// GetSeen returns all seen items for a feed.
func (s *FileSeenStorage) GetSeen(feedID string) ([]SeenItem, error) {
	items, err := s.LoadAll()
	if err != nil {
		return nil, err
	}
	feedItems, ok := items[feedID]
	if !ok {
		return []SeenItem{}, nil
	}
	result := make([]SeenItem, 0, len(feedItems))
	for _, item := range feedItems {
		result = append(result, item)
	}
	return result, nil
}

// Cleanup removes seen items older than the specified time for a specific feed.
func (s *FileSeenStorage) Cleanup(feedID string, before time.Time) error {
	items, err := s.LoadAll()
	if err != nil {
		return err
	}
	feedItems, ok := items[feedID]
	if !ok {
		return nil
	}
	filtered := make(map[string]SeenItem)
	for guid, item := range feedItems {
		if !item.SeenAt.Before(before) {
			filtered[guid] = item
		}
	}
	items[feedID] = filtered
	return s.SaveAll(items)
}

// CleanupAll removes seen items older than the specified time across all feeds.
func (s *FileSeenStorage) CleanupAll(before time.Time) error {
	items, err := s.LoadAll()
	if err != nil {
		return err
	}
	for feedID, feedItems := range items {
		filtered := make(map[string]SeenItem)
		for guid, item := range feedItems {
			if !item.SeenAt.Before(before) {
				filtered[guid] = item
			}
		}
		items[feedID] = filtered
	}
	return s.SaveAll(items)
}
