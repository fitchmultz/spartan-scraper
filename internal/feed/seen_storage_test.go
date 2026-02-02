// Package feed provides RSS/Atom feed monitoring and parsing functionality.
//
// This file contains tests for the seen item storage layer.
package feed

import (
	"testing"
	"time"
)

func TestFileSeenStorage_IsSeen(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFileSeenStorage(tmpDir)

	// Initially nothing should be seen
	if storage.IsSeen("feed-1", "guid-1") {
		t.Error("IsSeen() = true for unseen item, want false")
	}

	// Mark an item as seen
	item := SeenItem{
		GUID:   "guid-1",
		Link:   "https://example.com/item1",
		Title:  "Item 1",
		SeenAt: time.Now(),
	}
	if err := storage.MarkSeen("feed-1", item); err != nil {
		t.Fatalf("MarkSeen() error = %v", err)
	}

	// Now it should be seen
	if !storage.IsSeen("feed-1", "guid-1") {
		t.Error("IsSeen() = false for seen item, want true")
	}

	// Different GUID should not be seen
	if storage.IsSeen("feed-1", "guid-2") {
		t.Error("IsSeen() = true for different GUID, want false")
	}

	// Different feed should not be seen
	if storage.IsSeen("feed-2", "guid-1") {
		t.Error("IsSeen() = true for different feed, want false")
	}
}

func TestFileSeenStorage_MarkSeen(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFileSeenStorage(tmpDir)

	item := SeenItem{
		GUID:   "guid-1",
		Link:   "https://example.com/item1",
		Title:  "Test Item",
		SeenAt: time.Now().UTC().Truncate(time.Second),
	}

	if err := storage.MarkSeen("feed-1", item); err != nil {
		t.Fatalf("MarkSeen() error = %v", err)
	}

	// Verify by getting seen items
	items, err := storage.GetSeen("feed-1")
	if err != nil {
		t.Fatalf("GetSeen() error = %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("GetSeen() len = %d, want 1", len(items))
	}

	if items[0].GUID != "guid-1" {
		t.Errorf("GetSeen()[0].GUID = %q, want %q", items[0].GUID, "guid-1")
	}

	if items[0].Title != "Test Item" {
		t.Errorf("GetSeen()[0].Title = %q, want %q", items[0].Title, "Test Item")
	}
}

func TestFileSeenStorage_GetSeen_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFileSeenStorage(tmpDir)

	items, err := storage.GetSeen("feed-1")
	if err != nil {
		t.Fatalf("GetSeen() error = %v", err)
	}

	if len(items) != 0 {
		t.Errorf("GetSeen() len = %d, want 0", len(items))
	}
}

func TestFileSeenStorage_GetSeen_MultipleFeeds(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFileSeenStorage(tmpDir)

	// Add items to different feeds
	item1 := SeenItem{GUID: "guid-1", Title: "Feed 1 Item", SeenAt: time.Now()}
	item2 := SeenItem{GUID: "guid-2", Title: "Feed 2 Item", SeenAt: time.Now()}

	if err := storage.MarkSeen("feed-1", item1); err != nil {
		t.Fatalf("MarkSeen(feed-1) error = %v", err)
	}
	if err := storage.MarkSeen("feed-2", item2); err != nil {
		t.Fatalf("MarkSeen(feed-2) error = %v", err)
	}

	// Get items for feed-1
	items, err := storage.GetSeen("feed-1")
	if err != nil {
		t.Fatalf("GetSeen(feed-1) error = %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("GetSeen(feed-1) len = %d, want 1", len(items))
	}

	if items[0].GUID != "guid-1" {
		t.Errorf("GetSeen(feed-1)[0].GUID = %q, want %q", items[0].GUID, "guid-1")
	}
}

func TestFileSeenStorage_Cleanup(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFileSeenStorage(tmpDir)

	now := time.Now()

	// Add items with different seen times
	oldItem := SeenItem{GUID: "old", Title: "Old Item", SeenAt: now.Add(-48 * time.Hour)}
	newItem := SeenItem{GUID: "new", Title: "New Item", SeenAt: now.Add(-12 * time.Hour)}

	if err := storage.MarkSeen("feed-1", oldItem); err != nil {
		t.Fatalf("MarkSeen(old) error = %v", err)
	}
	if err := storage.MarkSeen("feed-1", newItem); err != nil {
		t.Fatalf("MarkSeen(new) error = %v", err)
	}

	// Cleanup items older than 24 hours
	cutoff := now.Add(-24 * time.Hour)
	if err := storage.Cleanup("feed-1", cutoff); err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}

	// Old item should be gone
	if storage.IsSeen("feed-1", "old") {
		t.Error("IsSeen(old) = true after cleanup, want false")
	}

	// New item should still be there
	if !storage.IsSeen("feed-1", "new") {
		t.Error("IsSeen(new) = false after cleanup, want true")
	}
}

func TestFileSeenStorage_CleanupAll(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFileSeenStorage(tmpDir)

	now := time.Now()

	// Add items to different feeds
	oldItem1 := SeenItem{GUID: "old1", Title: "Old Item 1", SeenAt: now.Add(-48 * time.Hour)}
	oldItem2 := SeenItem{GUID: "old2", Title: "Old Item 2", SeenAt: now.Add(-48 * time.Hour)}
	newItem := SeenItem{GUID: "new", Title: "New Item", SeenAt: now.Add(-12 * time.Hour)}

	if err := storage.MarkSeen("feed-1", oldItem1); err != nil {
		t.Fatalf("MarkSeen(old1) error = %v", err)
	}
	if err := storage.MarkSeen("feed-2", oldItem2); err != nil {
		t.Fatalf("MarkSeen(old2) error = %v", err)
	}
	if err := storage.MarkSeen("feed-1", newItem); err != nil {
		t.Fatalf("MarkSeen(new) error = %v", err)
	}

	// Cleanup all items older than 24 hours
	cutoff := now.Add(-24 * time.Hour)
	if err := storage.CleanupAll(cutoff); err != nil {
		t.Fatalf("CleanupAll() error = %v", err)
	}

	// Old items should be gone
	if storage.IsSeen("feed-1", "old1") {
		t.Error("IsSeen(old1) = true after cleanup, want false")
	}
	if storage.IsSeen("feed-2", "old2") {
		t.Error("IsSeen(old2) = true after cleanup, want false")
	}

	// New item should still be there
	if !storage.IsSeen("feed-1", "new") {
		t.Error("IsSeen(new) = false after cleanup, want true")
	}
}

func TestFileSeenStorage_LoadAll_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFileSeenStorage(tmpDir)

	// Don't create any file - LoadAll should return empty map
	items, err := storage.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}

	if len(items) != 0 {
		t.Errorf("LoadAll() len = %d, want 0", len(items))
	}
}

func TestFileSeenStorage_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFileSeenStorage(tmpDir)

	item := SeenItem{
		GUID:   "persist-guid",
		Link:   "https://example.com/item",
		Title:  "Persistent Item",
		SeenAt: time.Now().UTC().Truncate(time.Second),
	}

	if err := storage.MarkSeen("feed-1", item); err != nil {
		t.Fatalf("MarkSeen() error = %v", err)
	}

	// Create a new storage instance pointing to the same directory
	storage2 := NewFileSeenStorage(tmpDir)

	// Item should still be seen
	if !storage2.IsSeen("feed-1", "persist-guid") {
		t.Error("IsSeen() = false after reload, want true")
	}

	// Verify details
	items, err := storage2.GetSeen("feed-1")
	if err != nil {
		t.Fatalf("GetSeen() error = %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("GetSeen() len = %d, want 1", len(items))
	}

	if items[0].GUID != item.GUID {
		t.Errorf("GUID = %q, want %q", items[0].GUID, item.GUID)
	}
	if items[0].Title != item.Title {
		t.Errorf("Title = %q, want %q", items[0].Title, item.Title)
	}
	if items[0].Link != item.Link {
		t.Errorf("Link = %q, want %q", items[0].Link, item.Link)
	}
}

func TestFileSeenStorage_MultipleItemsSameFeed(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFileSeenStorage(tmpDir)

	now := time.Now()

	// Add multiple items to the same feed
	items := []SeenItem{
		{GUID: "guid-1", Title: "Item 1", SeenAt: now},
		{GUID: "guid-2", Title: "Item 2", SeenAt: now.Add(-time.Hour)},
		{GUID: "guid-3", Title: "Item 3", SeenAt: now.Add(-2 * time.Hour)},
	}

	for _, item := range items {
		if err := storage.MarkSeen("feed-1", item); err != nil {
			t.Fatalf("MarkSeen(%s) error = %v", item.GUID, err)
		}
	}

	// Get all items
	seenItems, err := storage.GetSeen("feed-1")
	if err != nil {
		t.Fatalf("GetSeen() error = %v", err)
	}

	if len(seenItems) != 3 {
		t.Errorf("GetSeen() len = %d, want 3", len(seenItems))
	}

	// All items should be seen
	for _, item := range items {
		if !storage.IsSeen("feed-1", item.GUID) {
			t.Errorf("IsSeen(%s) = false, want true", item.GUID)
		}
	}
}
