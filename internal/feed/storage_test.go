// Package feed provides RSS/Atom feed monitoring and parsing functionality.
//
// This file contains tests for the feed storage layer.
package feed

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileStorage_Add(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFileStorage(tmpDir)

	feed := &Feed{
		URL:             "https://example.com/feed.xml",
		FeedType:        FeedTypeRSS,
		IntervalSeconds: 3600,
		Enabled:         true,
		AutoScrape:      true,
	}

	result, err := storage.Add(feed)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if result.ID == "" {
		t.Error("Add() did not generate ID")
	}

	if result.IntervalSeconds != 3600 {
		t.Errorf("Add() IntervalSeconds = %d, want 3600", result.IntervalSeconds)
	}

	if result.FeedType != FeedTypeRSS {
		t.Errorf("Add() FeedType = %q, want %q", result.FeedType, FeedTypeRSS)
	}

	// Verify it was saved
	feeds, err := storage.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(feeds) != 1 {
		t.Errorf("List() len = %d, want 1", len(feeds))
	}
}

func TestFileStorage_Add_Defaults(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFileStorage(tmpDir)

	feed := &Feed{
		URL:      "https://example.com/feed.xml",
		FeedType: "",
		// IntervalSeconds defaults to 3600
		// CreatedAt defaults to now
	}

	result, err := storage.Add(feed)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if result.IntervalSeconds != 3600 {
		t.Errorf("Add() IntervalSeconds default = %d, want 3600", result.IntervalSeconds)
	}

	if result.FeedType != FeedTypeAuto {
		t.Errorf("Add() FeedType default = %q, want %q", result.FeedType, FeedTypeAuto)
	}

	if result.CreatedAt.IsZero() {
		t.Error("Add() CreatedAt should not be zero")
	}
}

func TestFileStorage_Add_InvalidFeed(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFileStorage(tmpDir)

	feed := &Feed{
		URL:             "", // Invalid - empty URL
		IntervalSeconds: 3600,
	}

	_, err := storage.Add(feed)
	if err == nil {
		t.Error("Add() expected error for invalid feed, got nil")
	}
}

func TestFileStorage_Get(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFileStorage(tmpDir)

	feed := &Feed{
		ID:              "test-id-123",
		URL:             "https://example.com/feed.xml",
		FeedType:        FeedTypeRSS,
		IntervalSeconds: 3600,
		CreatedAt:       time.Now(),
	}

	if _, err := storage.Add(feed); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	result, err := storage.Get("test-id-123")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if result.ID != "test-id-123" {
		t.Errorf("Get() ID = %q, want %q", result.ID, "test-id-123")
	}

	if result.URL != "https://example.com/feed.xml" {
		t.Errorf("Get() URL = %q, want %q", result.URL, "https://example.com/feed.xml")
	}
}

func TestFileStorage_Get_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFileStorage(tmpDir)

	_, err := storage.Get("non-existent-id")
	if err == nil {
		t.Error("Get() expected error for non-existent feed, got nil")
	}

	if !IsNotFoundError(err) {
		t.Error("Get() error should be NotFoundError")
	}
}

func TestFileStorage_Update(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFileStorage(tmpDir)

	feed := &Feed{
		ID:              "test-id-123",
		URL:             "https://example.com/feed.xml",
		FeedType:        FeedTypeRSS,
		IntervalSeconds: 3600,
		CreatedAt:       time.Now(),
	}

	if _, err := storage.Add(feed); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// Update the feed
	feed.URL = "https://example.com/updated.xml"
	feed.IntervalSeconds = 7200

	if err := storage.Update(feed); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify the update
	result, err := storage.Get("test-id-123")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if result.URL != "https://example.com/updated.xml" {
		t.Errorf("Get() URL = %q, want %q", result.URL, "https://example.com/updated.xml")
	}

	if result.IntervalSeconds != 7200 {
		t.Errorf("Get() IntervalSeconds = %d, want 7200", result.IntervalSeconds)
	}
}

func TestFileStorage_Update_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFileStorage(tmpDir)

	feed := &Feed{
		ID:              "non-existent-id",
		URL:             "https://example.com/feed.xml",
		IntervalSeconds: 3600,
	}

	err := storage.Update(feed)
	if err == nil {
		t.Error("Update() expected error for non-existent feed, got nil")
	}

	if !IsNotFoundError(err) {
		t.Error("Update() error should be NotFoundError")
	}
}

func TestFileStorage_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFileStorage(tmpDir)

	feed1 := &Feed{
		ID:        "feed-1",
		URL:       "https://example.com/feed1.xml",
		CreatedAt: time.Now(),
	}
	feed2 := &Feed{
		ID:        "feed-2",
		URL:       "https://example.com/feed2.xml",
		CreatedAt: time.Now(),
	}

	if _, err := storage.Add(feed1); err != nil {
		t.Fatalf("Add(feed1) error = %v", err)
	}
	if _, err := storage.Add(feed2); err != nil {
		t.Fatalf("Add(feed2) error = %v", err)
	}

	if err := storage.Delete("feed-1"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	feeds, err := storage.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(feeds) != 1 {
		t.Errorf("List() len = %d, want 1", len(feeds))
	}

	if feeds[0].ID != "feed-2" {
		t.Errorf("List()[0].ID = %q, want %q", feeds[0].ID, "feed-2")
	}
}

func TestFileStorage_Delete_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFileStorage(tmpDir)

	err := storage.Delete("missing-feed")
	if err == nil {
		t.Fatal("Delete() expected error for non-existent feed, got nil")
	}
	if !IsNotFoundError(err) {
		t.Fatalf("Delete() error = %T, want NotFoundError", err)
	}
}

func TestFileStorage_List(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFileStorage(tmpDir)

	// Add feeds in order
	now := time.Now()
	feed1 := &Feed{
		ID:        "feed-1",
		URL:       "https://example.com/feed1.xml",
		CreatedAt: now.Add(-2 * time.Hour),
	}
	feed2 := &Feed{
		ID:        "feed-2",
		URL:       "https://example.com/feed2.xml",
		CreatedAt: now.Add(-1 * time.Hour),
	}
	feed3 := &Feed{
		ID:        "feed-3",
		URL:       "https://example.com/feed3.xml",
		CreatedAt: now,
	}

	if _, err := storage.Add(feed1); err != nil {
		t.Fatalf("Add(feed1) error = %v", err)
	}
	if _, err := storage.Add(feed2); err != nil {
		t.Fatalf("Add(feed2) error = %v", err)
	}
	if _, err := storage.Add(feed3); err != nil {
		t.Fatalf("Add(feed3) error = %v", err)
	}

	feeds, err := storage.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(feeds) != 3 {
		t.Fatalf("List() len = %d, want 3", len(feeds))
	}

	// Should be sorted by CreatedAt descending (newest first)
	if feeds[0].ID != "feed-3" {
		t.Errorf("List()[0].ID = %q, want %q", feeds[0].ID, "feed-3")
	}
	if feeds[1].ID != "feed-2" {
		t.Errorf("List()[1].ID = %q, want %q", feeds[1].ID, "feed-2")
	}
	if feeds[2].ID != "feed-1" {
		t.Errorf("List()[2].ID = %q, want %q", feeds[2].ID, "feed-1")
	}
}

func TestFileStorage_ListEnabled(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFileStorage(tmpDir)

	now := time.Now()

	// Add enabled feed that is due
	feed1 := &Feed{
		ID:              "feed-1",
		URL:             "https://example.com/feed1.xml",
		Enabled:         true,
		IntervalSeconds: 3600,
		LastCheckedAt:   now.Add(-2 * time.Hour), // Due
		CreatedAt:       now,
	}

	// Add disabled feed
	feed2 := &Feed{
		ID:        "feed-2",
		URL:       "https://example.com/feed2.xml",
		Enabled:   false,
		CreatedAt: now,
	}

	// Add enabled feed that is not due
	feed3 := &Feed{
		ID:              "feed-3",
		URL:             "https://example.com/feed3.xml",
		Enabled:         true,
		IntervalSeconds: 3600,
		LastCheckedAt:   now.Add(-30 * time.Minute), // Not due
		CreatedAt:       now,
	}

	if _, err := storage.Add(feed1); err != nil {
		t.Fatalf("Add(feed1) error = %v", err)
	}
	if _, err := storage.Add(feed2); err != nil {
		t.Fatalf("Add(feed2) error = %v", err)
	}
	if _, err := storage.Add(feed3); err != nil {
		t.Fatalf("Add(feed3) error = %v", err)
	}

	enabled, err := storage.ListEnabled()
	if err != nil {
		t.Fatalf("ListEnabled() error = %v", err)
	}

	if len(enabled) != 2 {
		t.Errorf("ListEnabled() len = %d, want 2", len(enabled))
	}

	// Should be sorted by next run time (due feeds first)
	if enabled[0].ID != "feed-1" {
		t.Errorf("ListEnabled()[0].ID = %q, want %q", enabled[0].ID, "feed-1")
	}
}

func TestFileStorage_LoadAll_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFileStorage(tmpDir)

	// Don't create any file - LoadAll should return empty slice
	feeds, err := storage.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}

	if len(feeds) != 0 {
		t.Errorf("LoadAll() len = %d, want 0", len(feeds))
	}
}

func TestFileStorage_feedsPath(t *testing.T) {
	tests := []struct {
		name    string
		dataDir string
		want    string
	}{
		{
			name:    "with data dir",
			dataDir: "/tmp/test",
			want:    filepath.Join("/tmp/test", "feeds.json"),
		},
		{
			name:    "empty data dir uses default",
			dataDir: "",
			want:    filepath.Join(".data", "feeds.json"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewFileStorage(tt.dataDir)
			got := storage.feedsPath()
			if got != tt.want {
				t.Errorf("feedsPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFileStorage_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFileStorage(tmpDir)

	feed := &Feed{
		ID:              "persist-test",
		URL:             "https://example.com/feed.xml",
		FeedType:        FeedTypeAtom,
		IntervalSeconds: 1800,
		Enabled:         true,
		AutoScrape:      false,
		CreatedAt:       time.Now().UTC().Truncate(time.Second),
	}

	if _, err := storage.Add(feed); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// Create a new storage instance pointing to the same directory
	storage2 := NewFileStorage(tmpDir)

	result, err := storage2.Get("persist-test")
	if err != nil {
		t.Fatalf("Get() from new storage error = %v", err)
	}

	if result.ID != feed.ID {
		t.Errorf("ID = %q, want %q", result.ID, feed.ID)
	}
	if result.URL != feed.URL {
		t.Errorf("URL = %q, want %q", result.URL, feed.URL)
	}
	if result.FeedType != feed.FeedType {
		t.Errorf("FeedType = %q, want %q", result.FeedType, feed.FeedType)
	}
	if result.IntervalSeconds != feed.IntervalSeconds {
		t.Errorf("IntervalSeconds = %d, want %d", result.IntervalSeconds, feed.IntervalSeconds)
	}
	if result.Enabled != feed.Enabled {
		t.Errorf("Enabled = %v, want %v", result.Enabled, feed.Enabled)
	}
	if result.AutoScrape != feed.AutoScrape {
		t.Errorf("AutoScrape = %v, want %v", result.AutoScrape, feed.AutoScrape)
	}
}

func TestFileStorage_FilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFileStorage(tmpDir)

	feed := &Feed{
		URL:             "https://example.com/feed.xml",
		IntervalSeconds: 3600,
	}

	if _, err := storage.Add(feed); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// Check file permissions
	path := storage.feedsPath()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	mode := info.Mode().Perm()
	expectedMode := os.FileMode(0o600)
	if mode != expectedMode {
		t.Errorf("File permissions = %o, want %o", mode, expectedMode)
	}
}
