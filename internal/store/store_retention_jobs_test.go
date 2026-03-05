// Package store provides tests for job retention operations.
// Tests cover listing and counting jobs by age, status, and kind,
// batch deletion without artifacts, and storage statistics.
// Does NOT test artifact deletion or crawl state operations.
package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestListJobsOlderThan(t *testing.T) {
	dataDir := t.TempDir()
	st, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer st.Close()

	ctx := context.Background()
	now := time.Now()

	// Create jobs with different ages
	oldJob := model.Job{
		ID:        "old-job",
		Kind:      model.KindScrape,
		Status:    model.StatusSucceeded,
		CreatedAt: now.AddDate(0, 0, -60),
		UpdatedAt: now.AddDate(0, 0, -60),
		Params:    map[string]interface{}{"url": "http://example.com"},
	}
	newJob := model.Job{
		ID:        "new-job",
		Kind:      model.KindScrape,
		Status:    model.StatusSucceeded,
		CreatedAt: now.AddDate(0, 0, -5),
		UpdatedAt: now.AddDate(0, 0, -5),
		Params:    map[string]interface{}{"url": "http://example.com"},
	}

	if err := st.Create(ctx, oldJob); err != nil {
		t.Fatalf("Create oldJob failed: %v", err)
	}
	if err := st.Create(ctx, newJob); err != nil {
		t.Fatalf("Create newJob failed: %v", err)
	}

	// List jobs older than 30 days
	cutoff := now.AddDate(0, 0, -30)
	jobs, err := st.ListJobsOlderThan(ctx, cutoff, ListOptions{})
	if err != nil {
		t.Fatalf("ListJobsOlderThan failed: %v", err)
	}

	if len(jobs) != 1 {
		t.Errorf("expected 1 old job, got %d", len(jobs))
	}
	if len(jobs) > 0 && jobs[0].ID != "old-job" {
		t.Errorf("expected old-job, got %s", jobs[0].ID)
	}
}

func TestListJobsByStatusAndAge(t *testing.T) {
	dataDir := t.TempDir()
	st, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer st.Close()

	ctx := context.Background()
	now := time.Now()

	// Create jobs with different statuses and ages
	oldFailedJob := model.Job{
		ID:        "old-failed",
		Kind:      model.KindScrape,
		Status:    model.StatusFailed,
		CreatedAt: now.AddDate(0, 0, -60),
		UpdatedAt: now.AddDate(0, 0, -60),
		Params:    map[string]interface{}{"url": "http://example.com"},
	}
	oldSucceededJob := model.Job{
		ID:        "old-succeeded",
		Kind:      model.KindScrape,
		Status:    model.StatusSucceeded,
		CreatedAt: now.AddDate(0, 0, -60),
		UpdatedAt: now.AddDate(0, 0, -60),
		Params:    map[string]interface{}{"url": "http://example.com"},
	}

	if err := st.Create(ctx, oldFailedJob); err != nil {
		t.Fatalf("Create oldFailedJob failed: %v", err)
	}
	if err := st.Create(ctx, oldSucceededJob); err != nil {
		t.Fatalf("Create oldSucceededJob failed: %v", err)
	}

	// List old failed jobs
	cutoff := now.AddDate(0, 0, -30)
	jobs, err := st.ListJobsByStatusAndAge(ctx, model.StatusFailed, cutoff, ListOptions{})
	if err != nil {
		t.Fatalf("ListJobsByStatusAndAge failed: %v", err)
	}

	if len(jobs) != 1 {
		t.Errorf("expected 1 failed job, got %d", len(jobs))
	}
	if len(jobs) > 0 && jobs[0].ID != "old-failed" {
		t.Errorf("expected old-failed, got %s", jobs[0].ID)
	}
}

func TestCountJobsOlderThan(t *testing.T) {
	dataDir := t.TempDir()
	st, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer st.Close()

	ctx := context.Background()
	now := time.Now()

	// Create jobs
	for i := 0; i < 3; i++ {
		job := model.Job{
			ID:        "old-job-" + string(rune('a'+i)),
			Kind:      model.KindScrape,
			Status:    model.StatusSucceeded,
			CreatedAt: now.AddDate(0, 0, -60),
			UpdatedAt: now.AddDate(0, 0, -60),
			Params:    map[string]interface{}{"url": "http://example.com"},
		}
		if err := st.Create(ctx, job); err != nil {
			t.Fatalf("Create job failed: %v", err)
		}
	}

	// Create a new job
	newJob := model.Job{
		ID:        "new-job",
		Kind:      model.KindScrape,
		Status:    model.StatusSucceeded,
		CreatedAt: now.AddDate(0, 0, -5),
		UpdatedAt: now.AddDate(0, 0, -5),
		Params:    map[string]interface{}{"url": "http://example.com"},
	}
	if err := st.Create(ctx, newJob); err != nil {
		t.Fatalf("Create newJob failed: %v", err)
	}

	// Count jobs older than 30 days
	cutoff := now.AddDate(0, 0, -30)
	count, err := st.CountJobsOlderThan(ctx, cutoff)
	if err != nil {
		t.Fatalf("CountJobsOlderThan failed: %v", err)
	}

	if count != 3 {
		t.Errorf("expected 3 old jobs, got %d", count)
	}
}

func TestDeleteJobsBatch(t *testing.T) {
	dataDir := t.TempDir()
	st, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer st.Close()

	ctx := context.Background()
	now := time.Now()

	// Create jobs
	for i := 0; i < 5; i++ {
		job := model.Job{
			ID:        "job-" + string(rune('a'+i)),
			Kind:      model.KindScrape,
			Status:    model.StatusSucceeded,
			CreatedAt: now,
			UpdatedAt: now,
			Params:    map[string]interface{}{"url": "http://example.com"},
		}
		if err := st.Create(ctx, job); err != nil {
			t.Fatalf("Create job failed: %v", err)
		}
	}

	// Delete 3 jobs
	ids := []string{"job-a", "job-b", "job-c"}
	deleted, err := st.DeleteJobsBatch(ctx, ids)
	if err != nil {
		t.Fatalf("DeleteJobsBatch failed: %v", err)
	}

	if deleted != 3 {
		t.Errorf("expected 3 deleted, got %d", deleted)
	}

	// Verify remaining jobs
	remaining, err := st.CountJobs(ctx, "")
	if err != nil {
		t.Fatalf("CountJobs failed: %v", err)
	}
	if remaining != 2 {
		t.Errorf("expected 2 remaining jobs, got %d", remaining)
	}
}

func TestListJobsByKind(t *testing.T) {
	dataDir := t.TempDir()
	st, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer st.Close()

	ctx := context.Background()
	now := time.Now()

	// Create jobs of different kinds
	scrapeJob := model.Job{
		ID:        "scrape-job",
		Kind:      model.KindScrape,
		Status:    model.StatusSucceeded,
		CreatedAt: now,
		UpdatedAt: now,
		Params:    map[string]interface{}{"url": "http://example.com"},
	}
	crawlJob := model.Job{
		ID:        "crawl-job",
		Kind:      model.KindCrawl,
		Status:    model.StatusSucceeded,
		CreatedAt: now,
		UpdatedAt: now,
		Params:    map[string]interface{}{"url": "http://example.com"},
	}

	if err := st.Create(ctx, scrapeJob); err != nil {
		t.Fatalf("Create scrapeJob failed: %v", err)
	}
	if err := st.Create(ctx, crawlJob); err != nil {
		t.Fatalf("Create crawlJob failed: %v", err)
	}

	// List scrape jobs
	jobs, err := st.ListJobsByKind(ctx, model.KindScrape, ListOptions{})
	if err != nil {
		t.Fatalf("ListJobsByKind failed: %v", err)
	}

	if len(jobs) != 1 {
		t.Errorf("expected 1 scrape job, got %d", len(jobs))
	}
	if len(jobs) > 0 && jobs[0].ID != "scrape-job" {
		t.Errorf("expected scrape-job, got %s", jobs[0].ID)
	}
}

func TestCountJobsByKind(t *testing.T) {
	dataDir := t.TempDir()
	st, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer st.Close()

	ctx := context.Background()
	now := time.Now()

	// Create jobs of different kinds
	for i := 0; i < 3; i++ {
		job := model.Job{
			ID:        "scrape-job-" + string(rune('a'+i)),
			Kind:      model.KindScrape,
			Status:    model.StatusSucceeded,
			CreatedAt: now,
			UpdatedAt: now,
			Params:    map[string]interface{}{"url": "http://example.com"},
		}
		if err := st.Create(ctx, job); err != nil {
			t.Fatalf("Create job failed: %v", err)
		}
	}

	crawlJob := model.Job{
		ID:        "crawl-job",
		Kind:      model.KindCrawl,
		Status:    model.StatusSucceeded,
		CreatedAt: now,
		UpdatedAt: now,
		Params:    map[string]interface{}{"url": "http://example.com"},
	}
	if err := st.Create(ctx, crawlJob); err != nil {
		t.Fatalf("Create crawlJob failed: %v", err)
	}

	// Count scrape jobs
	count, err := st.CountJobsByKind(ctx, model.KindScrape)
	if err != nil {
		t.Fatalf("CountJobsByKind failed: %v", err)
	}

	if count != 3 {
		t.Errorf("expected 3 scrape jobs, got %d", count)
	}
}

func TestGetStorageStats(t *testing.T) {
	dataDir := t.TempDir()
	st, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer st.Close()

	ctx := context.Background()
	now := time.Now()

	// Create jobs
	for i := 0; i < 5; i++ {
		job := model.Job{
			ID:        "job-" + string(rune('a'+i)),
			Kind:      model.KindScrape,
			Status:    model.StatusSucceeded,
			CreatedAt: now.AddDate(0, 0, -i*10),
			UpdatedAt: now.AddDate(0, 0, -i*10),
			Params:    map[string]interface{}{"url": "http://example.com"},
		}
		if err := st.Create(ctx, job); err != nil {
			t.Fatalf("Create job failed: %v", err)
		}
	}

	// Create a failed job
	failedJob := model.Job{
		ID:        "failed-job",
		Kind:      model.KindScrape,
		Status:    model.StatusFailed,
		CreatedAt: now,
		UpdatedAt: now,
		Params:    map[string]interface{}{"url": "http://example.com"},
	}
	if err := st.Create(ctx, failedJob); err != nil {
		t.Fatalf("Create failedJob failed: %v", err)
	}

	stats, err := st.GetStorageStats(ctx)
	if err != nil {
		t.Fatalf("GetStorageStats failed: %v", err)
	}

	if stats.TotalJobs != 6 {
		t.Errorf("expected 6 total jobs, got %d", stats.TotalJobs)
	}

	if stats.JobsByStatus[model.StatusSucceeded] != 5 {
		t.Errorf("expected 5 succeeded jobs, got %d", stats.JobsByStatus[model.StatusSucceeded])
	}

	if stats.JobsByStatus[model.StatusFailed] != 1 {
		t.Errorf("expected 1 failed job, got %d", stats.JobsByStatus[model.StatusFailed])
	}
}

func TestGetStorageStatsMissingJobsDir(t *testing.T) {
	dataDir := t.TempDir()
	st, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer st.Close()

	ctx := context.Background()
	now := time.Now()

	// Create a job WITHOUT creating the jobs directory
	job := model.Job{
		ID:        "test-job",
		Kind:      model.KindScrape,
		Status:    model.StatusSucceeded,
		CreatedAt: now,
		UpdatedAt: now,
		Params:    map[string]interface{}{"url": "http://example.com"},
	}
	if err := st.Create(ctx, job); err != nil {
		t.Fatalf("Create job failed: %v", err)
	}

	// Ensure jobs directory does NOT exist
	jobsDir := filepath.Join(dataDir, "jobs")
	if _, err := os.Stat(jobsDir); !os.IsNotExist(err) {
		t.Fatalf("jobs directory should not exist for this test")
	}

	// GetStorageStats should succeed and return 0 TotalStorageMB
	stats, err := st.GetStorageStats(ctx)
	if err != nil {
		t.Fatalf("GetStorageStats failed when jobs directory missing: %v", err)
	}

	if stats.TotalStorageMB != 0 {
		t.Errorf("expected 0 TotalStorageMB when jobs directory missing, got %d", stats.TotalStorageMB)
	}

	// Verify other stats are still populated correctly
	if stats.TotalJobs != 1 {
		t.Errorf("expected 1 total job, got %d", stats.TotalJobs)
	}
}
