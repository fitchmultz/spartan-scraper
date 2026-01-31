// Package webhook provides tests for webhook delivery record storage.
//
// Tests cover:
// - Delivery record creation with validation
// - Record retrieval and updates
// - Listing with filtering by JobID
// - Pagination (limit/offset)
// - Record counting
// - Persistence to JSON file (Load/Save)
// - Copy-on-read behavior for stored records
//
// Does NOT test:
// - Webhook dispatch logic (see dispatcher_test.go)
// - Actual HTTP delivery
// - Concurrent access patterns
//
// Assumes:
// - Records are stored as JSON in the data directory
// - Record IDs are unique and non-empty
// - Required fields: ID, EventID, URL
package webhook

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewStore(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	if store.dataDir != tmpDir {
		t.Errorf("expected dataDir=%s, got %s", tmpDir, store.dataDir)
	}
	if store.records == nil {
		t.Error("expected records map to be initialized")
	}
}

func TestStore_CreateRecord(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	ctx := context.Background()
	record := &DeliveryRecord{
		ID:        "rec-123",
		EventID:   "evt-456",
		EventType: EventJobCompleted,
		JobID:     "job-789",
		URL:       "https://example.com/webhook",
		Status:    DeliveryStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := store.CreateRecord(ctx, record)
	if err != nil {
		t.Fatalf("CreateRecord failed: %v", err)
	}

	// Verify record was stored
	retrieved, found, err := store.GetRecord(ctx, record.ID)
	if err != nil {
		t.Fatalf("GetRecord failed: %v", err)
	}
	if !found {
		t.Fatal("expected record to be found")
	}
	if retrieved.ID != record.ID {
		t.Errorf("expected ID=%s, got %s", record.ID, retrieved.ID)
	}
}

func TestStore_CreateRecord_Validation(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	tests := []struct {
		name    string
		record  *DeliveryRecord
		wantErr bool
	}{
		{
			name: "missing ID",
			record: &DeliveryRecord{
				ID:        "",
				EventID:   "evt-456",
				URL:       "https://example.com/webhook",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "missing EventID",
			record: &DeliveryRecord{
				ID:        "rec-123",
				EventID:   "",
				URL:       "https://example.com/webhook",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "missing URL",
			record: &DeliveryRecord{
				ID:        "rec-123",
				EventID:   "evt-456",
				URL:       "",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.CreateRecord(ctx, tt.record)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateRecord() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_UpdateRecord(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	// Create initial record
	record := &DeliveryRecord{
		ID:        "rec-123",
		EventID:   "evt-456",
		EventType: EventJobCompleted,
		JobID:     "job-789",
		URL:       "https://example.com/webhook",
		Status:    DeliveryStatusPending,
		Attempts:  0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := store.CreateRecord(ctx, record); err != nil {
		t.Fatalf("CreateRecord failed: %v", err)
	}

	// Update record
	record.Status = DeliveryStatusDelivered
	record.Attempts = 1
	now := time.Now()
	record.DeliveredAt = &now

	if err := store.UpdateRecord(ctx, record); err != nil {
		t.Fatalf("UpdateRecord failed: %v", err)
	}

	// Verify update
	retrieved, found, _ := store.GetRecord(ctx, record.ID)
	if !found {
		t.Fatal("expected record to be found")
	}
	if retrieved.Status != DeliveryStatusDelivered {
		t.Errorf("expected status=%s, got %s", DeliveryStatusDelivered, retrieved.Status)
	}
	if retrieved.Attempts != 1 {
		t.Errorf("expected attempts=1, got %d", retrieved.Attempts)
	}
	if retrieved.DeliveredAt == nil {
		t.Error("expected DeliveredAt to be set")
	}
}

func TestStore_UpdateRecord_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	record := &DeliveryRecord{
		ID:        "non-existent",
		EventID:   "evt-456",
		URL:       "https://example.com/webhook",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := store.UpdateRecord(ctx, record)
	if err == nil {
		t.Error("expected error for non-existent record")
	}
}

func TestStore_GetRecord_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	_, found, err := store.GetRecord(ctx, "non-existent")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if found {
		t.Error("expected record to not be found")
	}
}

func TestStore_ListRecords(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	// Create multiple records
	now := time.Now()
	records := []*DeliveryRecord{
		{
			ID:        "rec-1",
			EventID:   "evt-1",
			JobID:     "job-a",
			URL:       "https://example.com/webhook1",
			Status:    DeliveryStatusPending,
			CreatedAt: now.Add(-2 * time.Hour),
			UpdatedAt: now.Add(-2 * time.Hour),
		},
		{
			ID:        "rec-2",
			EventID:   "evt-2",
			JobID:     "job-b",
			URL:       "https://example.com/webhook2",
			Status:    DeliveryStatusDelivered,
			CreatedAt: now.Add(-1 * time.Hour),
			UpdatedAt: now.Add(-1 * time.Hour),
		},
		{
			ID:        "rec-3",
			EventID:   "evt-3",
			JobID:     "job-a",
			URL:       "https://example.com/webhook3",
			Status:    DeliveryStatusFailed,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	for _, r := range records {
		if err := store.CreateRecord(ctx, r); err != nil {
			t.Fatalf("CreateRecord failed: %v", err)
		}
	}

	// Test list all (sorted by CreatedAt descending)
	results, err := store.ListRecords(ctx, "", 10, 0)
	if err != nil {
		t.Fatalf("ListRecords failed: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
	// Should be sorted newest first
	if results[0].ID != "rec-3" {
		t.Errorf("expected first result to be rec-3 (newest), got %s", results[0].ID)
	}

	// Test filter by jobID
	results, err = store.ListRecords(ctx, "job-a", 10, 0)
	if err != nil {
		t.Fatalf("ListRecords failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results for job-a, got %d", len(results))
	}

	// Test limit
	results, err = store.ListRecords(ctx, "", 2, 0)
	if err != nil {
		t.Fatalf("ListRecords failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results with limit=2, got %d", len(results))
	}

	// Test offset
	results, err = store.ListRecords(ctx, "", 10, 1)
	if err != nil {
		t.Fatalf("ListRecords failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results with offset=1, got %d", len(results))
	}
	if results[0].ID != "rec-2" {
		t.Errorf("expected first result to be rec-2 after offset, got %s", results[0].ID)
	}
}

func TestStore_ListRecords_OffsetBeyondRange(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	// Create one record
	record := &DeliveryRecord{
		ID:        "rec-1",
		EventID:   "evt-1",
		URL:       "https://example.com/webhook",
		Status:    DeliveryStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := store.CreateRecord(ctx, record); err != nil {
		t.Fatalf("CreateRecord failed: %v", err)
	}

	// Request with offset beyond range
	results, err := store.ListRecords(ctx, "", 10, 5)
	if err != nil {
		t.Fatalf("ListRecords failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results with offset beyond range, got %d", len(results))
	}
}

func TestStore_DeleteRecord(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	record := &DeliveryRecord{
		ID:        "rec-123",
		EventID:   "evt-456",
		URL:       "https://example.com/webhook",
		Status:    DeliveryStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := store.CreateRecord(ctx, record); err != nil {
		t.Fatalf("CreateRecord failed: %v", err)
	}

	// Verify record exists
	_, found, _ := store.GetRecord(ctx, record.ID)
	if !found {
		t.Fatal("expected record to exist")
	}

	// Delete record
	if err := store.DeleteRecord(ctx, record.ID); err != nil {
		t.Fatalf("DeleteRecord failed: %v", err)
	}

	// Verify record is gone
	_, found, _ = store.GetRecord(ctx, record.ID)
	if found {
		t.Error("expected record to be deleted")
	}
}

func TestStore_CountRecords(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	// Create records for different jobs
	records := []*DeliveryRecord{
		{ID: "rec-1", EventID: "evt-1", JobID: "job-a", URL: "https://example.com/1", Status: DeliveryStatusPending, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "rec-2", EventID: "evt-2", JobID: "job-b", URL: "https://example.com/2", Status: DeliveryStatusPending, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "rec-3", EventID: "evt-3", JobID: "job-a", URL: "https://example.com/3", Status: DeliveryStatusPending, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	for _, r := range records {
		if err := store.CreateRecord(ctx, r); err != nil {
			t.Fatalf("CreateRecord failed: %v", err)
		}
	}

	// Test count all
	count, err := store.CountRecords(ctx, "")
	if err != nil {
		t.Fatalf("CountRecords failed: %v", err)
	}
	if count != 3 {
		t.Errorf("expected count=3, got %d", count)
	}

	// Test count by jobID
	count, err = store.CountRecords(ctx, "job-a")
	if err != nil {
		t.Fatalf("CountRecords failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected count=2 for job-a, got %d", count)
	}
}

func TestStore_LoadAndSave(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	// Create records
	record := &DeliveryRecord{
		ID:           "rec-123",
		EventID:      "evt-456",
		EventType:    EventJobCompleted,
		JobID:        "job-789",
		URL:          "https://example.com/webhook",
		Status:       DeliveryStatusDelivered,
		Attempts:     1,
		ResponseCode: 200,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := store.CreateRecord(ctx, record); err != nil {
		t.Fatalf("CreateRecord failed: %v", err)
	}

	// Create new store instance pointing to same directory
	store2 := NewStore(tmpDir)
	if err := store2.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify records were loaded
	retrieved, found, _ := store2.GetRecord(ctx, record.ID)
	if !found {
		t.Fatal("expected record to be found after load")
	}
	if retrieved.ID != record.ID {
		t.Errorf("expected ID=%s, got %s", record.ID, retrieved.ID)
	}
	if retrieved.Status != record.Status {
		t.Errorf("expected Status=%s, got %s", record.Status, retrieved.Status)
	}
	if retrieved.ResponseCode != record.ResponseCode {
		t.Errorf("expected ResponseCode=%d, got %d", record.ResponseCode, retrieved.ResponseCode)
	}
}

func TestStore_Load_NoExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	// Load should succeed even if file doesn't exist
	err := store.Load()
	if err != nil {
		t.Errorf("Load should succeed with no existing file: %v", err)
	}
}

func TestStore_Load_CorruptedFile(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	// Create corrupted file
	path := filepath.Join(tmpDir, "webhook_deliveries.json")
	if err := os.WriteFile(path, []byte("not valid json"), 0600); err != nil {
		t.Fatalf("failed to write corrupted file: %v", err)
	}

	err := store.Load()
	if err == nil {
		t.Error("expected error when loading corrupted file")
	}
}

func TestStore_ReturnsCopy(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	record := &DeliveryRecord{
		ID:        "rec-123",
		EventID:   "evt-456",
		URL:       "https://example.com/webhook",
		Status:    DeliveryStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := store.CreateRecord(ctx, record); err != nil {
		t.Fatalf("CreateRecord failed: %v", err)
	}

	// Get record and modify it
	retrieved, found, _ := store.GetRecord(ctx, record.ID)
	if !found {
		t.Fatal("expected record to be found")
	}
	retrieved.Status = DeliveryStatusFailed

	// Get record again - should not be modified
	retrieved2, _, _ := store.GetRecord(ctx, record.ID)
	if retrieved2.Status != DeliveryStatusPending {
		t.Error("modifying retrieved record should not affect stored record")
	}
}
