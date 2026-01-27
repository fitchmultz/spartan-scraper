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
		Params:    map[string]interface{}{"url": "http://example.com"},
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

func TestStoreCrawlState(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	state := model.CrawlState{
		URL:         "http://example.com",
		ETag:        "tag",
		LastScraped: time.Now(),
	}

	if err := s.UpsertCrawlState(ctx, state); err != nil {
		t.Fatalf("UpsertCrawlState failed: %v", err)
	}

	got, err := s.GetCrawlState(ctx, "http://example.com")
	if err != nil {
		t.Fatalf("GetCrawlState failed: %v", err)
	}
	if got.URL != state.URL || got.ETag != state.ETag {
		t.Errorf("GetCrawlState returned unexpected state: %+v", got)
	}

	// Update
	state.ETag = "new-tag"
	if err := s.UpsertCrawlState(ctx, state); err != nil {
		t.Fatalf("UpsertCrawlState (update) failed: %v", err)
	}

	got, _ = s.GetCrawlState(ctx, "http://example.com")
	if got.ETag != "new-tag" {
		t.Errorf("expected etag new-tag, got %s", got.ETag)
	}
}

func TestStoreListOptsPagination(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	// Create 25 jobs
	for i := 0; i < 25; i++ {
		job := model.Job{
			ID:        fmt.Sprintf("j%02d", i),
			Kind:      model.KindScrape,
			Status:    model.StatusQueued,
			CreatedAt: time.Now().Add(time.Duration(i) * time.Second),
			UpdatedAt: time.Now(),
			Params:    map[string]interface{}{"idx": i},
		}
		if err := s.Create(ctx, job); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// Test limit
	opts := ListOptions{Limit: 10, Offset: 0}
	jobs, err := s.ListOpts(ctx, opts)
	if err != nil {
		t.Fatalf("ListOpts failed: %v", err)
	}
	if len(jobs) != 10 {
		t.Errorf("expected 10 jobs with limit=10, got %d", len(jobs))
	}

	// Test offset
	opts = ListOptions{Limit: 10, Offset: 10}
	jobs, err = s.ListOpts(ctx, opts)
	if err != nil {
		t.Fatalf("ListOpts failed: %v", err)
	}
	if len(jobs) != 10 {
		t.Errorf("expected 10 jobs with offset=10, got %d", len(jobs))
	}

	// Test ordering (should be desc by created_at)
	opts = ListOptions{Limit: 5, Offset: 0}
	jobs, err = s.ListOpts(ctx, opts)
	if err != nil {
		t.Fatalf("ListOpts failed: %v", err)
	}
	// First job should have highest created_at (j24)
	if jobs[0].ID != "j24" {
		t.Errorf("expected first job to be j24, got %s", jobs[0].ID)
	}

	// Test offset beyond available jobs
	opts = ListOptions{Limit: 10, Offset: 100}
	jobs, err = s.ListOpts(ctx, opts)
	if err != nil {
		t.Fatalf("ListOpts failed: %v", err)
	}
	if len(jobs) != 0 {
		t.Errorf("expected 0 jobs with offset=100, got %d", len(jobs))
	}
}

func TestListOptionsDefaults(t *testing.T) {
	tests := []struct {
		name       string
		input      ListOptions
		wantLimit  int
		wantOffset int
	}{
		{"zero values use defaults", ListOptions{}, 100, 0},
		{"negative limit uses default", ListOptions{Limit: -1}, 100, 0},
		{"negative offset uses zero", ListOptions{Offset: -5}, 100, 0},
		{"max limit capped", ListOptions{Limit: 2000}, 1000, 0},
		{"valid values preserved", ListOptions{Limit: 50, Offset: 10}, 50, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.input.Defaults()
			if got.Limit != tt.wantLimit || got.Offset != tt.wantOffset {
				t.Errorf("Defaults() = {%d, %d}, want {%d, %d}",
					got.Limit, got.Offset, tt.wantLimit, tt.wantOffset)
			}
		})
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

	// Create 5 jobs
	for i := 0; i < 5; i++ {
		job := model.Job{
			ID:        fmt.Sprintf("j%d", i),
			Kind:      model.KindScrape,
			Status:    model.StatusQueued,
			CreatedAt: time.Now().Add(time.Duration(i) * time.Second),
			UpdatedAt: time.Now(),
			Params:    map[string]interface{}{"idx": i},
		}
		if err := s.Create(ctx, job); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// Test that List() uses default limit of 100 (all 5 jobs should be returned)
	jobs, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(jobs) != 5 {
		t.Errorf("expected 5 jobs with List(), got %d", len(jobs))
	}
}

func TestStoreListByStatus(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	// Create jobs with different statuses
	queuedJob := model.Job{
		ID:        "j1",
		Kind:      model.KindScrape,
		Status:    model.StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Params:    map[string]interface{}{"url": "http://example.com/1"},
	}

	runningJob := model.Job{
		ID:        "j2",
		Kind:      model.KindScrape,
		Status:    model.StatusRunning,
		CreatedAt: time.Now().Add(-1 * time.Second),
		UpdatedAt: time.Now(),
		Params:    map[string]interface{}{"url": "http://example.com/2"},
	}

	succeededJob := model.Job{
		ID:        "j3",
		Kind:      model.KindCrawl,
		Status:    model.StatusSucceeded,
		CreatedAt: time.Now().Add(-2 * time.Second),
		UpdatedAt: time.Now(),
		Params:    map[string]interface{}{"url": "http://example.com/3"},
	}

	if err := s.Create(ctx, queuedJob); err != nil {
		t.Fatalf("failed to create queued job: %v", err)
	}
	if err := s.Create(ctx, runningJob); err != nil {
		t.Fatalf("failed to create running job: %v", err)
	}
	if err := s.Create(ctx, succeededJob); err != nil {
		t.Fatalf("failed to create succeeded job: %v", err)
	}

	// Query for queued jobs
	queued, err := s.ListByStatus(ctx, model.StatusQueued, ListByStatusOptions{})
	if err != nil {
		t.Fatalf("ListByStatus failed: %v", err)
	}

	if len(queued) != 1 {
		t.Errorf("expected 1 queued job, got %d", len(queued))
	}
	if len(queued) > 0 && queued[0].ID != "j1" {
		t.Errorf("expected job j1, got %s", queued[0].ID)
	}

	// Query for running jobs
	running, err := s.ListByStatus(ctx, model.StatusRunning, ListByStatusOptions{})
	if err != nil {
		t.Fatalf("ListByStatus failed: %v", err)
	}

	if len(running) != 1 {
		t.Errorf("expected 1 running job, got %d", len(running))
	}
	if len(running) > 0 && running[0].ID != "j2" {
		t.Errorf("expected job j2, got %s", running[0].ID)
	}

	// Query for succeeded jobs
	succeeded, err := s.ListByStatus(ctx, model.StatusSucceeded, ListByStatusOptions{})
	if err != nil {
		t.Fatalf("ListByStatus failed: %v", err)
	}

	if len(succeeded) != 1 {
		t.Errorf("expected 1 succeeded job, got %d", len(succeeded))
	}
	if len(succeeded) > 0 && succeeded[0].ID != "j3" {
		t.Errorf("expected job j3, got %s", succeeded[0].ID)
	}

	// Query for failed jobs (none exist)
	failed, err := s.ListByStatus(ctx, model.StatusFailed, ListByStatusOptions{})
	if err != nil {
		t.Fatalf("ListByStatus failed: %v", err)
	}

	if len(failed) != 0 {
		t.Errorf("expected 0 failed jobs, got %d", len(failed))
	}
}

func TestListByStatusOptionsDefaults(t *testing.T) {
	tests := []struct {
		name       string
		input      ListByStatusOptions
		wantLimit  int
		wantOffset int
	}{
		{"zero values use defaults", ListByStatusOptions{}, 100, 0},
		{"negative limit uses default", ListByStatusOptions{Limit: -1}, 100, 0},
		{"negative offset uses zero", ListByStatusOptions{Offset: -5}, 100, 0},
		{"max limit capped", ListByStatusOptions{Limit: 2000}, 1000, 0},
		{"valid values preserved", ListByStatusOptions{Limit: 50, Offset: 10}, 50, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.input.Defaults()
			if got.Limit != tt.wantLimit || got.Offset != tt.wantOffset {
				t.Errorf("Defaults() = {%d, %d}, want {%d, %d}",
					got.Limit, got.Offset, tt.wantLimit, tt.wantOffset)
			}
		})
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

	// Create a job
	job := model.Job{
		ID:        "j1",
		Kind:      model.KindScrape,
		Status:    model.StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Params:    map[string]interface{}{"url": "http://example.com"},
	}

	if err := s.Create(ctx, job); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify it exists
	got, err := s.Get(ctx, "j1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ID != job.ID {
		t.Errorf("expected job j1, got %s", got.ID)
	}

	// Delete job
	if err := s.Delete(ctx, "j1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it's gone
	_, err = s.Get(ctx, "j1")
	if err == nil {
		t.Error("expected error when getting deleted job, got nil")
	}

	// Delete non-existent job should not error (idempotent)
	if err := s.Delete(ctx, "j1"); err != nil {
		t.Errorf("Delete of non-existent job should succeed, got: %v", err)
	}

	// Delete with empty ID should not panic
	if err := s.Delete(ctx, ""); err != nil {
		// Empty ID just won't match any rows, so it succeeds
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

	// Create a job
	job := model.Job{
		ID:        "j1",
		Kind:      model.KindScrape,
		Status:    model.StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Params:    map[string]interface{}{"url": "http://example.com"},
	}

	if err := s.Create(ctx, job); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Create job directory and result file
	jobDir := filepath.Join(dataDir, "jobs", "j1")
	if err := fsutil.MkdirAllSecure(jobDir); err != nil {
		t.Fatalf("failed to create job directory: %v", err)
	}

	resultPath := filepath.Join(jobDir, "results.jsonl")
	resultContent := `{"test":"data"}`
	if err := os.WriteFile(resultPath, []byte(resultContent), 0o644); err != nil {
		t.Fatalf("failed to write result file: %v", err)
	}

	// Update job with result path
	job.ResultPath = resultPath
	if err := s.UpdateResultPath(ctx, "j1", resultPath); err != nil {
		t.Fatalf("failed to update result path: %v", err)
	}

	// Verify job and artifacts exist
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

	// Delete with artifacts
	if err := s.DeleteWithArtifacts(ctx, "j1"); err != nil {
		t.Fatalf("DeleteWithArtifacts failed: %v", err)
	}

	// Verify job is gone from DB
	_, err = s.Get(ctx, "j1")
	if err == nil {
		t.Error("job should be deleted from database")
	}

	// Verify result file is deleted
	if _, err := os.Stat(resultPath); !os.IsNotExist(err) {
		t.Error("result file should be deleted")
	}

	// Verify job directory is deleted
	if _, err := os.Stat(jobDir); !os.IsNotExist(err) {
		t.Error("job directory should be deleted")
	}

	// Test idempotency: deleting non-existent job should succeed
	if err := s.DeleteWithArtifacts(ctx, "j1"); err != nil {
		t.Errorf("deleting already-deleted job should succeed, got: %v", err)
	}
}

func TestListCrawlStates(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	// Insert test data
	states := []model.CrawlState{
		{
			URL:          "https://example.com/page1",
			ETag:         "etag1",
			LastModified: "Mon, 01 Jan 2026 00:00:00 GMT",
			ContentHash:  "hash1",
			LastScraped:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			URL:          "https://example.com/page2",
			ETag:         "etag2",
			LastModified: "Tue, 02 Jan 2026 00:00:00 GMT",
			ContentHash:  "hash2",
			LastScraped:  time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, state := range states {
		err := s.UpsertCrawlState(ctx, state)
		if err != nil {
			t.Fatalf("failed to insert crawl state: %v", err)
		}
	}

	// List all
	listed, err := s.ListCrawlStates(ctx, ListCrawlStatesOptions{})
	if err != nil {
		t.Fatalf("failed to list crawl states: %v", err)
	}

	if len(listed) != 2 {
		t.Errorf("expected 2 states, got %d", len(listed))
	}

	// Verify ordering (most recent first)
	if listed[0].URL != "https://example.com/page2" {
		t.Errorf("expected page2 first, got %s", listed[0].URL)
	}
}

func TestListCrawlStatesPagination(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	// Insert 3 states
	for i := 1; i <= 3; i++ {
		state := model.CrawlState{
			URL:         fmt.Sprintf("https://example.com/page%d", i),
			ETag:        fmt.Sprintf("etag%d", i),
			ContentHash: fmt.Sprintf("hash%d", i),
			LastScraped: time.Date(2026, 1, i, 0, 0, 0, 0, time.UTC),
		}
		err := s.UpsertCrawlState(ctx, state)
		if err != nil {
			t.Fatalf("failed to insert crawl state: %v", err)
		}
	}

	// Test limit
	listed, err := s.ListCrawlStates(ctx, ListCrawlStatesOptions{Limit: 2})
	if err != nil {
		t.Fatalf("failed to list crawl states: %v", err)
	}
	if len(listed) != 2 {
		t.Errorf("expected 2 states with limit, got %d", len(listed))
	}

	// Test offset
	listed, err = s.ListCrawlStates(ctx, ListCrawlStatesOptions{Offset: 1})
	if err != nil {
		t.Fatalf("failed to list crawl states: %v", err)
	}
	if len(listed) != 2 {
		t.Errorf("expected 2 states with offset 1, got %d", len(listed))
	}
}

func TestListCrawlStatesOptionsDefaults(t *testing.T) {
	tests := []struct {
		name       string
		input      ListCrawlStatesOptions
		wantLimit  int
		wantOffset int
	}{
		{"zero values use defaults", ListCrawlStatesOptions{}, 100, 0},
		{"negative limit uses default", ListCrawlStatesOptions{Limit: -1}, 100, 0},
		{"negative offset uses zero", ListCrawlStatesOptions{Offset: -5}, 100, 0},
		{"max limit capped", ListCrawlStatesOptions{Limit: 2000}, 1000, 0},
		{"valid values preserved", ListCrawlStatesOptions{Limit: 50, Offset: 10}, 50, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.input.Defaults()
			if got.Limit != tt.wantLimit || got.Offset != tt.wantOffset {
				t.Errorf("Defaults() = {%d, %d}, want {%d, %d}",
					got.Limit, got.Offset, tt.wantLimit, tt.wantOffset)
			}
		})
	}
}
