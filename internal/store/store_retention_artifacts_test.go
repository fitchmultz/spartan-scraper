// Package store provides tests for artifact deletion operations.
// Tests cover batch deletion with artifacts, storage size calculations,
// and comprehensive edge cases including path traversal protection,
// partial failures, permission errors, and missing artifacts.
// Does NOT test simple job deletion or crawl state operations.
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
