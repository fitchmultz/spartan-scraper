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
	"fmt"
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
	deleted, attempted, spaceReclaimed, failedIDs, err := st.DeleteJobsWithArtifactsBatch(ctx, ids)
	if err != nil {
		t.Fatalf("DeleteJobsWithArtifactsBatch failed: %v", err)
	}

	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}
	if attempted != 1 {
		t.Errorf("expected 1 attempted, got %d", attempted)
	}
	if spaceReclaimed < 1 {
		t.Errorf("expected at least 1 MB reclaimed, got %d", spaceReclaimed)
	}
	if len(failedIDs) > 0 {
		t.Errorf("expected no failed IDs, got %v", failedIDs)
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

func TestDeleteJobsWithArtifactsBatch_PartialFailure(t *testing.T) {
	dataDir := t.TempDir()
	st, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer st.Close()

	ctx := context.Background()
	now := time.Now()

	// Create two jobs with artifacts
	job1 := model.Job{
		ID:        "job-1",
		Kind:      model.KindScrape,
		Status:    model.StatusSucceeded,
		CreatedAt: now,
		UpdatedAt: now,
		Params:    map[string]interface{}{"url": "http://example.com"},
	}
	job2 := model.Job{
		ID:        "job-2",
		Kind:      model.KindScrape,
		Status:    model.StatusSucceeded,
		CreatedAt: now,
		UpdatedAt: now,
		Params:    map[string]interface{}{"url": "http://example.com"},
	}
	if err := st.Create(ctx, job1); err != nil {
		t.Fatalf("Create job1 failed: %v", err)
	}
	if err := st.Create(ctx, job2); err != nil {
		t.Fatalf("Create job2 failed: %v", err)
	}

	// Create artifact directories with files
	job1Dir := filepath.Join(dataDir, "jobs", job1.ID)
	if err := os.MkdirAll(job1Dir, 0755); err != nil {
		t.Fatalf("MkdirAll job1 failed: %v", err)
	}

	content := []byte("test content data")
	if err := os.WriteFile(filepath.Join(job1Dir, "file.txt"), content, 0644); err != nil {
		t.Fatalf("WriteFile job1 failed: %v", err)
	}
	// Note: job2 has no artifact directory - this is a valid scenario

	// Delete both jobs
	ids := []string{job1.ID, job2.ID}
	deleted, attempted, spaceReclaimed, failedIDs, err := st.DeleteJobsWithArtifactsBatch(ctx, ids)
	if err != nil {
		t.Fatalf("DeleteJobsWithArtifactsBatch failed: %v", err)
	}

	// Both jobs should be deleted from DB
	if deleted != 2 {
		t.Errorf("expected 2 deleted, got %d", deleted)
	}
	if attempted != 2 {
		t.Errorf("expected 2 attempted, got %d", attempted)
	}

	// Only job1's space should be reclaimed (job2 had no artifacts)
	expectedMB := (int64(len(content)) + 1024*1024 - 1) / (1024 * 1024)
	if spaceReclaimed != expectedMB {
		t.Errorf("expected %d MB reclaimed, got %d", expectedMB, spaceReclaimed)
	}

	// job2 should not be in failed IDs (missing artifacts is OK, not a failure)
	if len(failedIDs) > 0 {
		t.Errorf("expected no failed IDs (job2 had no artifacts), got %v", failedIDs)
	}
}

func TestDeleteJobsWithArtifactsBatch_PathTraversalFailure(t *testing.T) {
	dataDir := t.TempDir()
	st, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer st.Close()

	ctx := context.Background()
	now := time.Now()

	// Create a valid job
	validJob := model.Job{
		ID:        "valid-job",
		Kind:      model.KindScrape,
		Status:    model.StatusSucceeded,
		CreatedAt: now,
		UpdatedAt: now,
		Params:    map[string]interface{}{"url": "http://example.com"},
	}
	// Create a job with malicious ID (path traversal)
	maliciousJob := model.Job{
		ID:        "../../../etc/passwd",
		Kind:      model.KindScrape,
		Status:    model.StatusSucceeded,
		CreatedAt: now,
		UpdatedAt: now,
		Params:    map[string]interface{}{"url": "http://example.com"},
	}
	if err := st.Create(ctx, validJob); err != nil {
		t.Fatalf("Create validJob failed: %v", err)
	}
	if err := st.Create(ctx, maliciousJob); err != nil {
		t.Fatalf("Create maliciousJob failed: %v", err)
	}

	// Create artifact for valid job
	jobDir := filepath.Join(dataDir, "jobs", validJob.ID)
	if err := os.MkdirAll(jobDir, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	content := []byte("test content")
	if err := os.WriteFile(filepath.Join(jobDir, "file.txt"), content, 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Try to delete both jobs - the malicious one will fail at artifact deletion
	ids := []string{validJob.ID, maliciousJob.ID}
	deleted, attempted, spaceReclaimed, failedIDs, err := st.DeleteJobsWithArtifactsBatch(ctx, ids)
	if err != nil {
		t.Fatalf("DeleteJobsWithArtifactsBatch failed: %v", err)
	}

	// Only valid job should be deleted from DB (artifact deletion succeeded)
	// Malicious job DB record should be preserved (artifact deletion failed)
	if deleted != 1 {
		t.Errorf("expected 1 deleted (valid job only), got %d", deleted)
	}
	if attempted != 2 {
		t.Errorf("expected 2 attempted, got %d", attempted)
	}

	// Space from valid job should be reclaimed
	expectedMB := (int64(len(content)) + 1024*1024 - 1) / (1024 * 1024)
	if spaceReclaimed != expectedMB {
		t.Errorf("expected %d MB reclaimed, got %d", expectedMB, spaceReclaimed)
	}

	// Path traversal ID should be in failed IDs
	if len(failedIDs) != 1 || failedIDs[0] != maliciousJob.ID {
		t.Errorf("expected failedIDs to contain malicious ID, got %v", failedIDs)
	}
}

func TestDeleteJobsWithArtifactsBatch_NoArtifacts(t *testing.T) {
	dataDir := t.TempDir()
	st, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer st.Close()

	ctx := context.Background()
	now := time.Now()

	// Create job without artifacts
	job := model.Job{
		ID:        "job-no-artifacts",
		Kind:      model.KindScrape,
		Status:    model.StatusSucceeded,
		CreatedAt: now,
		UpdatedAt: now,
		Params:    map[string]interface{}{"url": "http://example.com"},
	}
	if err := st.Create(ctx, job); err != nil {
		t.Fatalf("Create job failed: %v", err)
	}

	// Delete job (no artifacts exist)
	ids := []string{job.ID}
	deleted, attempted, spaceReclaimed, failedIDs, err := st.DeleteJobsWithArtifactsBatch(ctx, ids)
	if err != nil {
		t.Fatalf("DeleteJobsWithArtifactsBatch failed: %v", err)
	}

	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}
	if attempted != 1 {
		t.Errorf("expected 1 attempted, got %d", attempted)
	}
	if spaceReclaimed != 0 {
		t.Errorf("expected 0 MB reclaimed (no artifacts), got %d", spaceReclaimed)
	}
	if len(failedIDs) > 0 {
		t.Errorf("expected no failed IDs (missing artifacts is OK), got %v", failedIDs)
	}
}

// TestDeleteJobsWithArtifactsBatch_DiskFullScenario simulates artifact deletion
// failure scenario (e.g., disk full, permission denied).
// Verifies that when artifact deletion fails, the DB record is preserved.
func TestDeleteJobsWithArtifactsBatch_DiskFullScenario(t *testing.T) {
	dataDir := t.TempDir()
	st, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer st.Close()

	ctx := context.Background()
	now := time.Now()

	// Create 3 jobs with artifacts
	jobs := make([]model.Job, 3)
	for i := 0; i < 3; i++ {
		jobs[i] = model.Job{
			ID:        fmt.Sprintf("job-%d", i),
			Kind:      model.KindScrape,
			Status:    model.StatusSucceeded,
			CreatedAt: now,
			UpdatedAt: now,
			Params:    map[string]interface{}{"url": "http://example.com"},
		}
		if err := st.Create(ctx, jobs[i]); err != nil {
			t.Fatalf("Create job %d failed: %v", i, err)
		}
	}

	// Create artifact directories with files for all jobs
	for i := 0; i < 3; i++ {
		jobDir := filepath.Join(dataDir, "jobs", jobs[i].ID)
		if err := os.MkdirAll(jobDir, 0755); err != nil {
			t.Fatalf("MkdirAll job %d failed: %v", i, err)
		}
		content := []byte(fmt.Sprintf("test content for job %d", i))
		if err := os.WriteFile(filepath.Join(jobDir, "file.txt"), content, 0644); err != nil {
			t.Fatalf("WriteFile job %d failed: %v", i, err)
		}
	}

	// Create a read-only file inside job-1 that cannot be deleted
	// This simulates a "disk full" or permission scenario
	job1Dir := filepath.Join(dataDir, "jobs", jobs[1].ID)
	readOnlyFile := filepath.Join(job1Dir, "readonly.txt")
	if err := os.WriteFile(readOnlyFile, []byte("readonly content"), 0444); err != nil {
		t.Fatalf("WriteFile readonly failed: %v", err)
	}
	// Make the file read-only (no write permission)
	if err := os.Chmod(readOnlyFile, 0444); err != nil {
		t.Fatalf("Chmod readonly file failed: %v", err)
	}
	// Make the directory read-only too so the file can't be removed
	if err := os.Chmod(job1Dir, 0555); err != nil {
		t.Fatalf("Chmod job1 dir failed: %v", err)
	}

	// Restore permissions after test
	defer func() {
		os.Chmod(job1Dir, 0755)
		os.Chmod(readOnlyFile, 0644)
	}()

	// Attempt to delete all 3 jobs
	ids := []string{jobs[0].ID, jobs[1].ID, jobs[2].ID}
	deleted, attempted, spaceReclaimed, failedIDs, err := st.DeleteJobsWithArtifactsBatch(ctx, ids)
	if err != nil {
		t.Fatalf("DeleteJobsWithArtifactsBatch failed: %v", err)
	}

	// Restore permissions to verify state
	os.Chmod(job1Dir, 0755)
	os.Chmod(readOnlyFile, 0644)

	// Verify: 2 deleted from DB (job-0 and job-2), 1 preserved (job-1 with read-only artifacts)
	if deleted != 2 {
		t.Errorf("expected 2 deleted, got %d", deleted)
	}
	if attempted != 3 {
		t.Errorf("expected 3 attempted, got %d", attempted)
	}

	// Verify: failedIDs contains job-1
	if len(failedIDs) != 1 || failedIDs[0] != jobs[1].ID {
		t.Errorf("expected failedIDs to contain %s, got %v", jobs[1].ID, failedIDs)
	}

	// Verify: spaceReclaimed reflects only the 2 successful deletions (not job-1)
	// job-0 and job-2 each have ~24 bytes, total ~48 bytes = 1 MB when rounded up
	if spaceReclaimed < 1 {
		t.Errorf("expected at least 1 MB reclaimed, got %d", spaceReclaimed)
	}

	// Verify: job-0 and job-2 are deleted from DB
	for _, id := range []string{jobs[0].ID, jobs[2].ID} {
		_, err := st.Get(ctx, id)
		if err == nil {
			t.Errorf("expected job %s to be deleted from DB", id)
		}
	}

	// Verify: job-1 still exists in DB (artifact deletion failed)
	_, err = st.Get(ctx, jobs[1].ID)
	if err != nil {
		t.Errorf("expected job %s to still exist in DB (artifact deletion failed)", jobs[1].ID)
	}

	// Verify: job-0 and job-2 artifacts are deleted
	for _, id := range []string{jobs[0].ID, jobs[2].ID} {
		jobDir := filepath.Join(dataDir, "jobs", id)
		if _, err := os.Stat(jobDir); !os.IsNotExist(err) {
			t.Errorf("expected artifact directory %s to be deleted", id)
		}
	}

	// Verify: job-1 artifacts still exist (deletion failed)
	if _, err := os.Stat(job1Dir); os.IsNotExist(err) {
		t.Errorf("expected artifact directory %s to still exist (deletion failed)", jobs[1].ID)
	}
}
