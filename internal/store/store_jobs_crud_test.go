// Package store provides tests for job CRUD operations.
// Tests cover Create, Get, UpdateStatus, Delete, DeleteWithArtifacts,
// and path traversal protection during artifact deletion.
// Does NOT test job listing, pagination, or crawl state operations.
package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/fsutil"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestStoreJobs(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	job := model.Job{
		ID:        "j1",
		Kind:      model.KindScrape,
		Status:    model.StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Spec:      map[string]interface{}{"url": "http://example.com"},
	}

	if err := s.Create(ctx, job); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := s.Get(ctx, "j1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ID != job.ID || got.Status != job.Status {
		t.Errorf("Get returned unexpected job: %+v", got)
	}

	if err := s.UpdateStatus(ctx, "j1", model.StatusRunning, "error message"); err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	got, _ = s.Get(ctx, "j1")
	if got.Status != model.StatusRunning || got.Error != "error message" {
		t.Errorf("UpdateStatus did not work as expected: %+v", got)
	}

	jobs, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(jobs))
	}
}

func TestStoreDelete(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	job := model.Job{
		ID:        "j1",
		Kind:      model.KindScrape,
		Status:    model.StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Spec:      map[string]interface{}{"url": "http://example.com"},
	}

	if err := s.Create(ctx, job); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := s.Get(ctx, "j1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ID != job.ID {
		t.Errorf("expected job j1, got %s", got.ID)
	}

	if err := s.Delete(ctx, "j1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = s.Get(ctx, "j1")
	if err == nil {
		t.Error("expected error when getting deleted job, got nil")
	}

	if err := s.Delete(ctx, "j1"); err != nil {
		t.Errorf("Delete of non-existent job should succeed, got: %v", err)
	}

	if err := s.Delete(ctx, ""); err != nil {
		t.Errorf("Delete with empty ID should succeed, got: %v", err)
	}
}

func TestStoreDeleteWithArtifacts(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	job := model.Job{
		ID:        "j1",
		Kind:      model.KindScrape,
		Status:    model.StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Spec:      map[string]interface{}{"url": "http://example.com"},
	}

	if err := s.Create(ctx, job); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	jobDir := filepath.Join(dataDir, "jobs", "j1")
	if err := fsutil.MkdirAllSecure(jobDir); err != nil {
		t.Fatalf("failed to create job directory: %v", err)
	}

	resultPath := filepath.Join(jobDir, "results.jsonl")
	resultContent := `{"test":"data"}`
	if err := os.WriteFile(resultPath, []byte(resultContent), 0o644); err != nil {
		t.Fatalf("failed to write result file: %v", err)
	}

	job.ResultPath = resultPath
	if err := s.UpdateResultPath(ctx, "j1", resultPath); err != nil {
		t.Fatalf("failed to update result path: %v", err)
	}

	_, err = s.Get(ctx, "j1")
	if err != nil {
		t.Fatalf("job should exist before delete: %v", err)
	}

	if _, err := os.Stat(resultPath); os.IsNotExist(err) {
		t.Fatalf("result file should exist before delete")
	}

	if _, err := os.Stat(jobDir); os.IsNotExist(err) {
		t.Fatalf("job directory should exist before delete")
	}

	if err := s.DeleteWithArtifacts(ctx, "j1"); err != nil {
		t.Fatalf("DeleteWithArtifacts failed: %v", err)
	}

	_, err = s.Get(ctx, "j1")
	if err == nil {
		t.Error("job should be deleted from database")
	}

	if _, err := os.Stat(resultPath); !os.IsNotExist(err) {
		t.Error("result file should be deleted")
	}

	if _, err := os.Stat(jobDir); !os.IsNotExist(err) {
		t.Error("job directory should be deleted")
	}

	// Deleting already-deleted job should return NotFound error
	if err := s.DeleteWithArtifacts(ctx, "j1"); err == nil {
		t.Error("deleting already-deleted job should return error")
	}
}

func TestStoreListUsesDefaults(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	for i := 0; i < 5; i++ {
		job := model.Job{
			ID:        fmt.Sprintf("j%d", i),
			Kind:      model.KindScrape,
			Status:    model.StatusQueued,
			CreatedAt: time.Now().Add(time.Duration(i) * time.Second),
			UpdatedAt: time.Now(),
			Spec:      map[string]interface{}{"idx": i},
		}
		if err := s.Create(ctx, job); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	jobs, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(jobs) != 5 {
		t.Errorf("expected 5 jobs with List(), got %d", len(jobs))
	}
}

func TestStoreDeleteWithArtifactsPathTraversal(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	// Create a file outside the jobs directory that should NOT be deleted
	outsideFile := filepath.Join(dataDir, "sensitive.txt")
	if err := os.WriteFile(outsideFile, []byte("secret"), 0o644); err != nil {
		t.Fatalf("failed to create outside file: %v", err)
	}

	// Attempt to delete using path traversal - should fail because job doesn't exist
	traversalID := "../../../sensitive"
	err = s.DeleteWithArtifacts(ctx, traversalID)
	if err == nil {
		t.Error("expected error for path traversal attempt, got nil")
	}

	// Verify the outside file still exists
	if _, err := os.Stat(outsideFile); os.IsNotExist(err) {
		t.Error("outside file should not have been deleted")
	}
}

func TestStoreDeleteWithArtifactsPathTraversalWithJob(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	// Create a job with a malicious ID that looks like path traversal
	// This tests the defense-in-depth path check in DeleteWithArtifacts
	maliciousID := "../../../etc/passwd"
	job := model.Job{
		ID:        maliciousID,
		Kind:      model.KindScrape,
		Status:    model.StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Spec:      map[string]interface{}{"url": "http://example.com"},
	}

	if err := s.Create(ctx, job); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Create a file outside the jobs directory that should NOT be deleted
	outsideFile := filepath.Join(dataDir, "sensitive.txt")
	if err := os.WriteFile(outsideFile, []byte("secret"), 0o644); err != nil {
		t.Fatalf("failed to create outside file: %v", err)
	}

	// Attempt to delete - the path traversal check should prevent deletion outside jobs dir
	err = s.DeleteWithArtifacts(ctx, maliciousID)
	if err == nil {
		t.Error("expected error for path traversal attempt, got nil")
	}

	// Verify the outside file still exists
	if _, err := os.Stat(outsideFile); os.IsNotExist(err) {
		t.Error("outside file should not have been deleted")
	}
}
