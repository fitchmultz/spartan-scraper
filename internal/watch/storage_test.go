// Package watch provides content change monitoring functionality.
//
// This file contains tests for watch storage.
package watch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileStorage(t *testing.T) {
	// Create temporary directory for tests
	tmpDir := t.TempDir()
	storage := NewFileStorage(tmpDir)

	t.Run("add and get", func(t *testing.T) {
		watch := &Watch{
			URL:             "https://example.com",
			Selector:        "#content",
			IntervalSeconds: 3600,
			Enabled:         true,
		}

		added, err := storage.Add(watch)
		if err != nil {
			t.Fatalf("Add() error = %v", err)
		}

		if added.ID == "" {
			t.Error("Expected ID to be generated")
		}

		if added.IntervalSeconds != 3600 {
			t.Errorf("IntervalSeconds = %d, want 3600", added.IntervalSeconds)
		}

		// Retrieve the watch
		retrieved, err := storage.Get(added.ID)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		if retrieved.URL != watch.URL {
			t.Errorf("URL = %v, want %v", retrieved.URL, watch.URL)
		}
	})

	t.Run("add with defaults", func(t *testing.T) {
		watch := &Watch{
			URL: "https://example.org",
		}

		added, err := storage.Add(watch)
		if err != nil {
			t.Fatalf("Add() error = %v", err)
		}

		if added.IntervalSeconds != 3600 {
			t.Errorf("Default IntervalSeconds = %d, want 3600", added.IntervalSeconds)
		}

		if added.DiffFormat != "unified" {
			t.Errorf("Default DiffFormat = %v, want unified", added.DiffFormat)
		}

		if added.CreatedAt.IsZero() {
			t.Error("Expected CreatedAt to be set")
		}
	})

	t.Run("add invalid watch", func(t *testing.T) {
		watch := &Watch{
			URL:             "",
			IntervalSeconds: 3600,
		}

		_, err := storage.Add(watch)
		if err == nil {
			t.Error("Expected error for invalid watch")
		}
	})

	t.Run("list", func(t *testing.T) {
		// Clear storage first
		_ = storage.SaveAll([]Watch{})

		// Add some watches
		watches := []*Watch{
			{URL: "https://a.com", IntervalSeconds: 3600, Enabled: true},
			{URL: "https://b.com", IntervalSeconds: 1800, Enabled: true},
			{URL: "https://c.com", IntervalSeconds: 7200, Enabled: false},
		}

		for _, w := range watches {
			_, err := storage.Add(w)
			if err != nil {
				t.Fatalf("Add() error = %v", err)
			}
		}

		list, err := storage.List()
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if len(list) != 3 {
			t.Errorf("List() returned %d watches, want 3", len(list))
		}
	})

	t.Run("list enabled", func(t *testing.T) {
		// Clear storage first
		_ = storage.SaveAll([]Watch{})

		// Add watches with different states
		watches := []*Watch{
			{URL: "https://enabled1.com", IntervalSeconds: 3600, Enabled: true},
			{URL: "https://disabled.com", IntervalSeconds: 3600, Enabled: false},
			{URL: "https://enabled2.com", IntervalSeconds: 3600, Enabled: true},
		}

		for _, w := range watches {
			_, err := storage.Add(w)
			if err != nil {
				t.Fatalf("Add() error = %v", err)
			}
		}

		enabled, err := storage.ListEnabled()
		if err != nil {
			t.Fatalf("ListEnabled() error = %v", err)
		}

		if len(enabled) != 2 {
			t.Errorf("ListEnabled() returned %d watches, want 2", len(enabled))
		}

		// Check that all returned watches are enabled
		for _, w := range enabled {
			if !w.Enabled {
				t.Error("ListEnabled() returned a disabled watch")
			}
		}
	})

	t.Run("update", func(t *testing.T) {
		watch := &Watch{
			URL:             "https://update-test.com",
			IntervalSeconds: 3600,
			Enabled:         true,
		}

		added, err := storage.Add(watch)
		if err != nil {
			t.Fatalf("Add() error = %v", err)
		}

		// Update the watch
		added.IntervalSeconds = 1800
		added.Enabled = false

		err = storage.Update(added)
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}

		// Retrieve and verify
		retrieved, err := storage.Get(added.ID)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		if retrieved.IntervalSeconds != 1800 {
			t.Errorf("IntervalSeconds = %d, want 1800", retrieved.IntervalSeconds)
		}

		if retrieved.Enabled != false {
			t.Error("Enabled should be false after update")
		}
	})

	t.Run("update not found", func(t *testing.T) {
		watch := &Watch{
			ID:              "non-existent-id",
			URL:             "https://example.com",
			IntervalSeconds: 3600,
		}

		err := storage.Update(watch)
		if err == nil {
			t.Error("Expected error for non-existent watch")
		}
	})

	t.Run("delete", func(t *testing.T) {
		watch := &Watch{
			URL:             "https://delete-test.com",
			IntervalSeconds: 3600,
		}

		added, err := storage.Add(watch)
		if err != nil {
			t.Fatalf("Add() error = %v", err)
		}

		// Delete the watch
		err = storage.Delete(added.ID)
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		// Verify it's gone
		_, err = storage.Get(added.ID)
		if err == nil {
			t.Error("Expected error after deleting watch")
		}
	})

	t.Run("get not found", func(t *testing.T) {
		_, err := storage.Get("non-existent-id")
		if err == nil {
			t.Error("Expected error for non-existent watch")
		}
	})

	t.Run("empty storage", func(t *testing.T) {
		// Use a new storage with non-existent directory
		emptyStorage := NewFileStorage(filepath.Join(tmpDir, "non-existent"))

		list, err := emptyStorage.List()
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if len(list) != 0 {
			t.Errorf("List() returned %d watches, want 0", len(list))
		}
	})
}

func TestFileStoragePersistence(t *testing.T) {
	tmpDir := t.TempDir()

	// Create storage and add watch
	storage1 := NewFileStorage(tmpDir)
	watch := &Watch{
		URL:             "https://persistent.com",
		Selector:        "#content",
		IntervalSeconds: 3600,
		Enabled:         true,
		CreatedAt:       time.Now(),
	}

	added, err := storage1.Add(watch)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// Create new storage instance pointing to same directory
	storage2 := NewFileStorage(tmpDir)

	// Retrieve watch from new instance
	retrieved, err := storage2.Get(added.ID)
	if err != nil {
		t.Fatalf("Get() from new instance error = %v", err)
	}

	if retrieved.URL != watch.URL {
		t.Errorf("URL = %v, want %v", retrieved.URL, watch.URL)
	}

	if retrieved.Selector != watch.Selector {
		t.Errorf("Selector = %v, want %v", retrieved.Selector, watch.Selector)
	}
}

func TestWatchesPath(t *testing.T) {
	tests := []struct {
		dataDir  string
		expected string
	}{
		{
			dataDir:  "/custom/data",
			expected: filepath.Join("/custom/data", "watches.json"),
		},
		{
			dataDir:  "",
			expected: filepath.Join(".data", "watches.json"),
		},
	}

	for _, tt := range tests {
		storage := NewFileStorage(tt.dataDir)
		path := storage.watchesPath()
		if path != tt.expected {
			t.Errorf("watchesPath() = %v, want %v", path, tt.expected)
		}
	}
}

func TestNotFoundError(t *testing.T) {
	err := &NotFoundError{ID: "test-id"}
	expected := "watch not found: test-id"
	if err.Error() != expected {
		t.Errorf("Error() = %v, want %v", err.Error(), expected)
	}
}

func TestFileStorageFilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFileStorage(tmpDir)

	watch := &Watch{
		URL:             "https://example.com",
		IntervalSeconds: 3600,
	}

	_, err := storage.Add(watch)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// Check file permissions
	path := storage.watchesPath()
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
