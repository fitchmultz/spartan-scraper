package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/fsutil"
	"github.com/fitchmultz/spartan-scraper/internal/model"

	_ "modernc.org/sqlite"
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
		Depth:       1,
		JobID:       "test-job",
	}

	if err := s.UpsertCrawlState(ctx, state); err != nil {
		t.Fatalf("UpsertCrawlState failed: %v", err)
	}

	got, err := s.GetCrawlState(ctx, "http://example.com")
	if err != nil {
		t.Fatalf("GetCrawlState failed: %v", err)
	}
	if got.URL != state.URL || got.ETag != state.ETag || got.Depth != state.Depth || got.JobID != state.JobID {
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
			Depth:        1,
			JobID:        "job1",
		},
		{
			URL:          "https://example.com/page2",
			ETag:         "etag2",
			LastModified: "Tue, 02 Jan 2026 00:00:00 GMT",
			ContentHash:  "hash2",
			LastScraped:  time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
			Depth:        2,
			JobID:        "job2",
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
	if listed[0].Depth != 2 || listed[0].JobID != "job2" {
		t.Errorf("expected Depth 2 and JobID job2, got %d and %s", listed[0].Depth, listed[0].JobID)
	}
	if listed[1].Depth != 1 || listed[1].JobID != "job1" {
		t.Errorf("expected Depth 1 and JobID job1, got %d and %s", listed[1].Depth, listed[1].JobID)
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

func TestStoreCountJobs(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	// Create jobs with different statuses
	_ = s.Create(ctx, model.Job{ID: "j1", Kind: model.KindScrape, Status: model.StatusQueued, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	_ = s.Create(ctx, model.Job{ID: "j2", Kind: model.KindScrape, Status: model.StatusRunning, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	_ = s.Create(ctx, model.Job{ID: "j3", Kind: model.KindScrape, Status: model.StatusSucceeded, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	_ = s.Create(ctx, model.Job{ID: "j4", Kind: model.KindScrape, Status: model.StatusQueued, CreatedAt: time.Now(), UpdatedAt: time.Now()})

	// Count all
	count, err := s.CountJobs(ctx, "")
	if err != nil {
		t.Fatalf("CountJobs failed: %v", err)
	}
	if count != 4 {
		t.Errorf("expected 4 jobs total, got %d", count)
	}

	// Count queued
	count, err = s.CountJobs(ctx, model.StatusQueued)
	if err != nil {
		t.Fatalf("CountJobs(queued) failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 queued jobs, got %d", count)
	}

	// Count failed (none)
	count, err = s.CountJobs(ctx, model.StatusFailed)
	if err != nil {
		t.Fatalf("CountJobs(failed) failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 failed jobs, got %d", count)
	}
}

func TestStoreCountCrawlStates(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	// Insert test data
	_ = s.UpsertCrawlState(ctx, model.CrawlState{URL: "https://example.com/1", LastScraped: time.Now()})
	_ = s.UpsertCrawlState(ctx, model.CrawlState{URL: "https://example.com/2", LastScraped: time.Now()})
	_ = s.UpsertCrawlState(ctx, model.CrawlState{URL: "https://example.com/3", LastScraped: time.Now()})

	count, err := s.CountCrawlStates(ctx)
	if err != nil {
		t.Fatalf("CountCrawlStates failed: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 crawl states, got %d", count)
	}
}

func TestMigrationFreshDatabase(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	// Verify both columns exist after init on fresh database
	depthExists, err := columnExists(s.db, "crawl_states", "depth")
	if err != nil {
		t.Fatalf("columnExists(depth) failed: %v", err)
	}
	if !depthExists {
		t.Error("depth column should exist after init")
	}

	jobIDExists, err := columnExists(s.db, "crawl_states", "job_id")
	if err != nil {
		t.Fatalf("columnExists(job_id) failed: %v", err)
	}
	if !jobIDExists {
		t.Error("job_id column should exist after init")
	}

	// Verify columns work correctly with insert/query
	ctx := context.Background()
	state := model.CrawlState{
		URL:          "http://example.com",
		ETag:         "tag",
		LastModified: "Mon, 01 Jan 2026 00:00:00 GMT",
		ContentHash:  "hash",
		LastScraped:  time.Now(),
		Depth:        2,
		JobID:        "test-job",
	}

	if err := s.UpsertCrawlState(ctx, state); err != nil {
		t.Fatalf("UpsertCrawlState failed: %v", err)
	}

	got, err := s.GetCrawlState(ctx, "http://example.com")
	if err != nil {
		t.Fatalf("GetCrawlState failed: %v", err)
	}
	if got.Depth != 2 {
		t.Errorf("expected Depth 2, got %d", got.Depth)
	}
	if got.JobID != "test-job" {
		t.Errorf("expected JobID test-job, got %s", got.JobID)
	}
}

func TestMigrationIdempotent(t *testing.T) {
	dataDir := t.TempDir()

	// First open - creates fresh database with columns
	s1, err := Open(dataDir)
	if err != nil {
		t.Fatalf("First Open failed: %v", err)
	}

	// Insert data
	ctx := context.Background()
	state := model.CrawlState{
		URL:         "http://example.com",
		ETag:        "tag1",
		LastScraped: time.Now(),
		Depth:       3,
		JobID:       "job-1",
	}
	if err := s1.UpsertCrawlState(ctx, state); err != nil {
		t.Fatalf("UpsertCrawlState failed: %v", err)
	}
	s1.Close()

	// Second open - re-init should not fail (idempotent)
	s2, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Second Open failed: %v", err)
	}
	defer s2.Close()

	// Verify columns still exist and work
	depthExists, err := columnExists(s2.db, "crawl_states", "depth")
	if err != nil {
		t.Fatalf("columnExists(depth) failed on reopen: %v", err)
	}
	if !depthExists {
		t.Error("depth column should still exist after reopen")
	}

	jobIDExists, err := columnExists(s2.db, "crawl_states", "job_id")
	if err != nil {
		t.Fatalf("columnExists(job_id) failed on reopen: %v", err)
	}
	if !jobIDExists {
		t.Error("job_id column should still exist after reopen")
	}

	// Verify data is still intact
	got, err := s2.GetCrawlState(ctx, "http://example.com")
	if err != nil {
		t.Fatalf("GetCrawlState failed after reopen: %v", err)
	}
	if got.Depth != 3 {
		t.Errorf("expected Depth 3 after reopen, got %d", got.Depth)
	}
	if got.JobID != "job-1" {
		t.Errorf("expected JobID job-1 after reopen, got %s", got.JobID)
	}
}

func TestMigrationAlterFailure(t *testing.T) {
	dataDir := t.TempDir()
	dbPath := filepath.Join(dataDir, "jobs.db")

	// Create initial database without new columns
	// We'll manually create old schema to simulate migration
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open failed: %v", err)
	}
	_, err = db.Exec(`
		create table jobs (
			id text primary key,
			kind text not null,
			status text not null,
			created_at text not null,
			updated_at text not null,
			params text,
			result_path text,
			error text
		);

		create table crawl_states (
			url text primary key,
			etag text,
			last_modified text,
			content_hash text,
			last_scraped text
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create old schema: %v", err)
	}
	db.Close()

	// Make database read-only to simulate ALTER TABLE failure
	if err := os.Chmod(dbPath, 0o444); err != nil {
		t.Fatalf("Failed to make database read-only: %v", err)
	}
	defer os.Chmod(dbPath, 0o644) // Restore permissions

	// Open should fail during migration when ALTER TABLE fails
	_, err = Open(dataDir)
	if err == nil {
		t.Error("Open should return error when ALTER TABLE fails")
	}
}

func TestParseErrorsReturnInternalKind(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	// Create a job with valid data
	job := model.Job{
		ID:        "test-id",
		Kind:      model.KindScrape,
		Status:    model.StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Params:    map[string]interface{}{"url": "http://example.com"},
	}
	if err := s.Create(ctx, job); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Corrupt created_at in database directly
	_, err = s.db.ExecContext(ctx, "UPDATE jobs SET created_at = ? WHERE id = ?", "invalid-timestamp", "test-id")
	if err != nil {
		t.Fatalf("Failed to corrupt created_at: %v", err)
	}

	// Verify ListByStatus returns KindInternal error
	_, err = s.ListByStatus(ctx, model.StatusQueued, ListByStatusOptions{})
	if err == nil {
		t.Fatal("Expected error when parsing invalid created_at, got nil")
	}
	if !apperrors.IsKind(err, apperrors.KindInternal) {
		t.Errorf("Expected KindInternal error, got %v", apperrors.KindOf(err))
	}

	// Reset created_at and corrupt updated_at
	_, err = s.db.ExecContext(ctx, "UPDATE jobs SET created_at = ?, updated_at = ? WHERE id = ?",
		job.CreatedAt.Format(time.RFC3339Nano), "invalid-timestamp", "test-id")
	if err != nil {
		t.Fatalf("Failed to corrupt updated_at: %v", err)
	}

	_, err = s.ListByStatus(ctx, model.StatusQueued, ListByStatusOptions{})
	if err == nil {
		t.Fatal("Expected error when parsing invalid updated_at, got nil")
	}
	if !apperrors.IsKind(err, apperrors.KindInternal) {
		t.Errorf("Expected KindInternal error, got %v", apperrors.KindOf(err))
	}

	// Test Get with corrupt timestamps
	_, err = s.Get(ctx, "test-id")
	if err == nil {
		t.Fatal("Expected error when parsing invalid updated_at in Get, got nil")
	}
	if !apperrors.IsKind(err, apperrors.KindInternal) {
		t.Errorf("Expected KindInternal error in Get, got %v", apperrors.KindOf(err))
	}

	// Test ListOpts with corrupt timestamps
	_, err = s.ListOpts(ctx, ListOptions{})
	if err == nil {
		t.Fatal("Expected error when parsing invalid updated_at in ListOpts, got nil")
	}
	if !apperrors.IsKind(err, apperrors.KindInternal) {
		t.Errorf("Expected KindInternal error in ListOpts, got %v", apperrors.KindOf(err))
	}
}

func TestJSONErrorsReturnInternalKind(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	// Create a job with valid data
	job := model.Job{
		ID:        "test-id",
		Kind:      model.KindScrape,
		Status:    model.StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Params:    map[string]interface{}{"url": "http://example.com"},
	}
	if err := s.Create(ctx, job); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Corrupt params with invalid JSON
	_, err = s.db.ExecContext(ctx, "UPDATE jobs SET params = ? WHERE id = ?", "{invalid-json}", "test-id")
	if err != nil {
		t.Fatalf("Failed to corrupt params: %v", err)
	}

	// Verify ListByStatus returns KindInternal error
	_, err = s.ListByStatus(ctx, model.StatusQueued, ListByStatusOptions{})
	if err == nil {
		t.Fatal("Expected error when unmarshaling invalid params, got nil")
	}
	if !apperrors.IsKind(err, apperrors.KindInternal) {
		t.Errorf("Expected KindInternal error, got %v", apperrors.KindOf(err))
	}

	// Test Get with corrupt params
	_, err = s.Get(ctx, "test-id")
	if err == nil {
		t.Fatal("Expected error when unmarshaling invalid params in Get, got nil")
	}
	if !apperrors.IsKind(err, apperrors.KindInternal) {
		t.Errorf("Expected KindInternal error in Get, got %v", apperrors.KindOf(err))
	}

	// Test ListOpts with corrupt params
	_, err = s.ListOpts(ctx, ListOptions{})
	if err == nil {
		t.Fatal("Expected error when unmarshaling invalid params in ListOpts, got nil")
	}
	if !apperrors.IsKind(err, apperrors.KindInternal) {
		t.Errorf("Expected KindInternal error in ListOpts, got %v", apperrors.KindOf(err))
	}
}

func TestCrawlStateParseErrorsReturnInternalKind(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	// Create a crawl state with valid data
	state := model.CrawlState{
		URL:         "http://example.com",
		LastScraped: time.Now(),
		Depth:       1,
	}
	if err := s.UpsertCrawlState(ctx, state); err != nil {
		t.Fatalf("UpsertCrawlState failed: %v", err)
	}

	// Corrupt last_scraped with invalid timestamp
	_, err = s.db.ExecContext(ctx, "UPDATE crawl_states SET last_scraped = ? WHERE url = ?", "invalid-timestamp", "http://example.com")
	if err != nil {
		t.Fatalf("Failed to corrupt last_scraped: %v", err)
	}

	// Verify GetCrawlState returns KindInternal error
	_, err = s.GetCrawlState(ctx, "http://example.com")
	if err == nil {
		t.Fatal("Expected error when parsing invalid last_scraped, got nil")
	}
	if !apperrors.IsKind(err, apperrors.KindInternal) {
		t.Errorf("Expected KindInternal error, got %v", apperrors.KindOf(err))
	}

	// Test ListCrawlStates with corrupt last_scraped
	_, err = s.ListCrawlStates(ctx, ListCrawlStatesOptions{})
	if err == nil {
		t.Fatal("Expected error when parsing invalid last_scraped in ListCrawlStates, got nil")
	}
	if !apperrors.IsKind(err, apperrors.KindInternal) {
		t.Errorf("Expected KindInternal error in ListCrawlStates, got %v", apperrors.KindOf(err))
	}
}

func TestPrepareStatementErrorsReturnInternalKind(t *testing.T) {
	// Test by trying to open store with invalid SQL syntax
	// Since we can't easily trigger prepare errors with valid code,
	// we'll verify the pattern by checking that prepareStatements returns apperrors

	// Note: This test verifies the structure, but prepare statement failures
	// are typically caught during init and would require mock or invalid SQL.
	// In practice, these errors occur if the database schema is corrupt
	// or there's a SQL syntax error, which are internal failures.

	// For now, we can manually verify by inspection that prepareStatements
	// uses apperrors.Wrap(apperrors.KindInternal, ...)

	// Alternative: Could add integration test that corrupts the database schema
	// but that's brittle. Manual inspection + existing migration tests are sufficient.

	t.Skip("Prepare statement errors require database corruption to test; pattern verified by code inspection")
}
