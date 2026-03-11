// Package scheduler provides tests for export schedule storage.
package scheduler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExportStorage(t *testing.T) {
	// Create temp directory for tests
	tempDir := t.TempDir()
	storage := NewExportStorage(tempDir)

	t.Run("empty list", func(t *testing.T) {
		schedules, err := storage.List()
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(schedules) != 0 {
			t.Errorf("List() returned %d schedules, want 0", len(schedules))
		}
	})

	t.Run("add and get", func(t *testing.T) {
		schedule := ExportSchedule{
			Name:    "Test Schedule",
			Enabled: true,
			Filters: ExportFilters{
				JobKinds: []string{"crawl"},
			},
			Export: ExportConfig{
				Format:          "jsonl",
				DestinationType: "local",
				LocalPath:       "/tmp/exports/{job_id}.jsonl",
			},
		}

		created, err := storage.Add(schedule)
		if err != nil {
			t.Fatalf("Add() error = %v", err)
		}

		if created.ID == "" {
			t.Error("Add() did not generate ID")
		}

		if created.CreatedAt.IsZero() {
			t.Error("Add() did not set CreatedAt")
		}

		// Get by ID
		retrieved, err := storage.Get(created.ID)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		if retrieved.Name != schedule.Name {
			t.Errorf("Get() Name = %q, want %q", retrieved.Name, schedule.Name)
		}
	})

	t.Run("get not found", func(t *testing.T) {
		_, err := storage.Get("non-existent-id")
		if err == nil {
			t.Error("Get() should return error for non-existent ID")
		}
	})

	t.Run("update", func(t *testing.T) {
		schedule := ExportSchedule{
			Name:    "Update Test",
			Enabled: true,
			Filters: ExportFilters{
				JobKinds: []string{"scrape"},
			},
			Export: ExportConfig{
				Format:          "json",
				DestinationType: "local",
				LocalPath:       "/tmp/test.json",
			},
		}

		created, err := storage.Add(schedule)
		if err != nil {
			t.Fatalf("Add() error = %v", err)
		}

		// Update
		created.Name = "Updated Name"
		created.Enabled = false

		updated, err := storage.Update(*created)
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}

		if updated.Name != "Updated Name" {
			t.Errorf("Update() Name = %q, want %q", updated.Name, "Updated Name")
		}

		if updated.Enabled != false {
			t.Errorf("Update() Enabled = %v, want false", updated.Enabled)
		}

		// Verify by getting again
		retrieved, err := storage.Get(created.ID)
		if err != nil {
			t.Fatalf("Get() after update error = %v", err)
		}

		if retrieved.Name != "Updated Name" {
			t.Errorf("Get() after update Name = %q, want %q", retrieved.Name, "Updated Name")
		}
	})

	t.Run("update not found", func(t *testing.T) {
		schedule := ExportSchedule{
			ID:      "non-existent",
			Name:    "Test",
			Filters: ExportFilters{JobKinds: []string{"crawl"}},
			Export:  ExportConfig{Format: "jsonl", DestinationType: "local", LocalPath: "/tmp/test.jsonl"},
		}

		_, err := storage.Update(schedule)
		if err == nil {
			t.Error("Update() should return error for non-existent ID")
		}
	})

	t.Run("delete", func(t *testing.T) {
		schedule := ExportSchedule{
			Name:    "Delete Test",
			Enabled: true,
			Filters: ExportFilters{
				JobKinds: []string{"research"},
			},
			Export: ExportConfig{
				Format:          "json",
				DestinationType: "local",
				LocalPath:       "/tmp/test.json",
			},
		}

		created, err := storage.Add(schedule)
		if err != nil {
			t.Fatalf("Add() error = %v", err)
		}

		// Delete
		err = storage.Delete(created.ID)
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		// Verify deletion
		_, err = storage.Get(created.ID)
		if err == nil {
			t.Error("Get() should return error after deletion")
		}
	})

	t.Run("delete not found", func(t *testing.T) {
		err := storage.Delete("non-existent-id")
		if err == nil {
			t.Fatal("Delete() should return error for non-existent ID")
		}
		if !IsNotFoundError(err) {
			t.Fatalf("Delete() error = %T, want NotFoundError", err)
		}
	})

	t.Run("list multiple", func(t *testing.T) {
		// Clear existing
		os.RemoveAll(filepath.Join(tempDir, "export_schedules.json"))

		// Add multiple schedules
		for i := 0; i < 3; i++ {
			schedule := ExportSchedule{
				Name:    "Schedule " + string(rune('A'+i)),
				Enabled: true,
				Filters: ExportFilters{
					JobKinds: []string{"crawl"},
				},
				Export: ExportConfig{
					Format:          "jsonl",
					DestinationType: "local",
					LocalPath:       "/tmp/test.jsonl",
				},
			}
			_, err := storage.Add(schedule)
			if err != nil {
				t.Fatalf("Add() error = %v", err)
			}
		}

		schedules, err := storage.List()
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if len(schedules) != 3 {
			t.Errorf("List() returned %d schedules, want 3", len(schedules))
		}
	})
}

func TestExportStorage_AddDefaults(t *testing.T) {
	tempDir := t.TempDir()
	storage := NewExportStorage(tempDir)

	schedule := ExportSchedule{
		Name:    "Test",
		Enabled: true,
		Filters: ExportFilters{
			JobKinds: []string{"crawl"},
		},
		Export: ExportConfig{
			Format:          "jsonl",
			DestinationType: "local",
			LocalPath:       "/tmp/test.jsonl",
		},
		// Retry is zero value
	}

	created, err := storage.Add(schedule)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// Check that defaults were applied
	if created.Retry.MaxRetries != 3 {
		t.Errorf("Retry.MaxRetries = %d, want 3", created.Retry.MaxRetries)
	}

	if created.Retry.BaseDelayMs != 1000 {
		t.Errorf("Retry.BaseDelayMs = %d, want 1000", created.Retry.BaseDelayMs)
	}
}

func TestExportStorage_InvalidSchedule(t *testing.T) {
	tempDir := t.TempDir()
	storage := NewExportStorage(tempDir)

	// Try to add invalid schedule (no name)
	schedule := ExportSchedule{
		Name: "",
		Filters: ExportFilters{
			JobKinds: []string{"crawl"},
		},
		Export: ExportConfig{
			Format:          "jsonl",
			DestinationType: "local",
			LocalPath:       "/tmp/test.jsonl",
		},
	}

	_, err := storage.Add(schedule)
	if err == nil {
		t.Error("Add() should return error for invalid schedule")
	}
}

func TestExportStorage_LocalPathRoundTrip(t *testing.T) {
	tempDir := t.TempDir()
	storage := NewExportStorage(tempDir)

	schedule := ExportSchedule{
		Name:    "Local Export",
		Enabled: true,
		Filters: ExportFilters{
			JobKinds: []string{"crawl"},
		},
		Export: ExportConfig{
			Format:          "jsonl",
			DestinationType: "local",
			LocalPath:       "exports/{kind}/{job_id}.jsonl",
			PathTemplate:    "exports/{kind}/{job_id}.jsonl",
		},
	}

	created, err := storage.Add(schedule)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	retrieved, err := storage.Get(created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if retrieved.Export.LocalPath != "exports/{kind}/{job_id}.jsonl" {
		t.Fatalf("LocalPath = %q, want %q", retrieved.Export.LocalPath, "exports/{kind}/{job_id}.jsonl")
	}

	if retrieved.Export.PathTemplate != "exports/{kind}/{job_id}.jsonl" {
		t.Fatalf("PathTemplate = %q, want %q", retrieved.Export.PathTemplate, "exports/{kind}/{job_id}.jsonl")
	}
}
