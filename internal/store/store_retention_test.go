// Package store provides tests for retention-related database operations.
//
// Tests cover:
// - Listing jobs by age and status
// - Counting jobs by various criteria
// - Batch deletion of jobs with artifacts
// - Storage size calculations
// - Crawl state cleanup by age
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

func TestDeleteJobsWithArtifactsBatch(t *testing.T) {
	dataDir := t.TempDir()
	st, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer st.Close()

	ctx := context.Background()
	now := time.Now()

	// Create job with artifacts
	job := model.Job{
		ID:        "job-with-artifacts",
		Kind:      model.KindScrape,
		Status:    model.StatusSucceeded,
		CreatedAt: now,
		UpdatedAt: now,
		Params:    map[string]interface{}{"url": "http://example.com"},
	}
	if err := st.Create(ctx, job); err != nil {
		t.Fatalf("Create job failed: %v", err)
	}

	// Create artifact directory and file
	jobDir := filepath.Join(dataDir, "jobs", job.ID)
	if err := os.MkdirAll(jobDir, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	artifactPath := filepath.Join(jobDir, "result.json")
	if err := os.WriteFile(artifactPath, []byte(`{"data": "test"}`), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Delete job with artifacts
	ids := []string{job.ID}
	deleted, spaceReclaimed, err := st.DeleteJobsWithArtifactsBatch(ctx, ids)
	if err != nil {
		t.Fatalf("DeleteJobsWithArtifactsBatch failed: %v", err)
	}

	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}
	if spaceReclaimed < 1 {
		t.Errorf("expected at least 1 MB reclaimed, got %d", spaceReclaimed)
	}

	// Verify artifacts are deleted
	if _, err := os.Stat(jobDir); !os.IsNotExist(err) {
		t.Error("expected artifact directory to be deleted")
	}
}

func TestGetJobStorageSize(t *testing.T) {
	dataDir := t.TempDir()
	st, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer st.Close()

	ctx := context.Background()
	jobID := "test-job"

	// Create artifact directory and files
	jobDir := filepath.Join(dataDir, "jobs", jobID)
	if err := os.MkdirAll(jobDir, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	content := []byte("test content for size calculation")
	if err := os.WriteFile(filepath.Join(jobDir, "file1.txt"), content, 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(jobDir, "file2.txt"), content, 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	size, err := st.GetJobStorageSize(ctx, jobID)
	if err != nil {
		t.Fatalf("GetJobStorageSize failed: %v", err)
	}

	expectedSize := int64(len(content) * 2)
	if size != expectedSize {
		t.Errorf("expected size %d, got %d", expectedSize, size)
	}
}

func TestGetJobStorageSizePathTraversal(t *testing.T) {
	dataDir := t.TempDir()
	st, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer st.Close()

	ctx := context.Background()

	// Try path traversal
	_, err = st.GetJobStorageSize(ctx, "../../../etc/passwd")
	if err == nil {
		t.Error("expected error for path traversal attempt")
	}
}

func TestDeleteCrawlStatesOlderThan(t *testing.T) {
	dataDir := t.TempDir()
	st, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer st.Close()

	ctx := context.Background()
	now := time.Now()

	// Create crawl states with different ages
	oldState := model.CrawlState{
		URL:         "http://old.example.com",
		LastScraped: now.AddDate(0, 0, -100),
		ContentHash: "abc123",
	}
	newState := model.CrawlState{
		URL:         "http://new.example.com",
		LastScraped: now.AddDate(0, 0, -5),
		ContentHash: "def456",
	}

	if err := st.UpsertCrawlState(ctx, oldState); err != nil {
		t.Fatalf("UpsertCrawlState old failed: %v", err)
	}
	if err := st.UpsertCrawlState(ctx, newState); err != nil {
		t.Fatalf("UpsertCrawlState new failed: %v", err)
	}

	// Delete crawl states older than 60 days
	cutoff := now.AddDate(0, 0, -60)
	deleted, err := st.DeleteCrawlStatesOlderThan(ctx, cutoff)
	if err != nil {
		t.Fatalf("DeleteCrawlStatesOlderThan failed: %v", err)
	}

	if deleted != 1 {
		t.Errorf("expected 1 deleted crawl state, got %d", deleted)
	}

	// Verify remaining crawl state
	states, err := st.ListCrawlStates(ctx, ListCrawlStatesOptions{})
	if err != nil {
		t.Fatalf("ListCrawlStates failed: %v", err)
	}
	if len(states) != 1 {
		t.Errorf("expected 1 remaining crawl state, got %d", len(states))
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
