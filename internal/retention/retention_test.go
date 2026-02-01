// Package retention provides tests for the retention engine.
//
// Tests cover:
// - Policy evaluation and cleanup by age
// - Cleanup by count limits
// - Cleanup by storage limits
// - Dry-run mode
// - Status reporting
package retention

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

func setupTestStore(t *testing.T) (*store.Store, string, func()) {
	dataDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	cleanup := func() {
		st.Close()
	}
	return st, dataDir, cleanup
}

func createTestJob(id string, kind model.Kind, status model.Status, ageDays int) model.Job {
	now := time.Now()
	return model.Job{
		ID:        id,
		Kind:      kind,
		Status:    status,
		CreatedAt: now.AddDate(0, 0, -ageDays),
		UpdatedAt: now.AddDate(0, 0, -ageDays),
		Params:    map[string]interface{}{"url": "http://example.com"},
	}
}

func TestRunCleanupByAge(t *testing.T) {
	st, dataDir, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create old and new jobs
	oldJob := createTestJob("old-job", model.KindScrape, model.StatusSucceeded, 60)
	newJob := createTestJob("new-job", model.KindScrape, model.StatusSucceeded, 5)

	if err := st.Create(ctx, oldJob); err != nil {
		t.Fatalf("Create oldJob failed: %v", err)
	}
	if err := st.Create(ctx, newJob); err != nil {
		t.Fatalf("Create newJob failed: %v", err)
	}

	cfg := config.Config{
		DataDir:               dataDir,
		RetentionEnabled:      true,
		RetentionJobDays:      30,
		RetentionMaxJobs:      0, // unlimited
		RetentionMaxStorageGB: 0, // unlimited
	}

	engine := NewEngine(st, cfg)
	result, err := engine.RunCleanup(ctx, CleanupOptions{})
	if err != nil {
		t.Fatalf("RunCleanup failed: %v", err)
	}

	if result.JobsDeleted != 1 {
		t.Errorf("expected 1 job deleted, got %d", result.JobsDeleted)
	}

	// Verify old job is deleted
	_, err = st.Get(ctx, "old-job")
	if err == nil {
		t.Error("expected old job to be deleted")
	}

	// Verify new job still exists
	_, err = st.Get(ctx, "new-job")
	if err != nil {
		t.Error("expected new job to still exist")
	}
}

func TestRunCleanupByCount(t *testing.T) {
	st, dataDir, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create 5 jobs
	for i := 0; i < 5; i++ {
		job := createTestJob("job-"+string(rune('a'+i)), model.KindScrape, model.StatusSucceeded, i*10)
		if err := st.Create(ctx, job); err != nil {
			t.Fatalf("Create job failed: %v", err)
		}
	}

	cfg := config.Config{
		DataDir:               dataDir,
		RetentionEnabled:      true,
		RetentionJobDays:      0, // unlimited
		RetentionMaxJobs:      3,
		RetentionMaxStorageGB: 0, // unlimited
	}

	engine := NewEngine(st, cfg)
	result, err := engine.RunCleanup(ctx, CleanupOptions{})
	if err != nil {
		t.Fatalf("RunCleanup failed: %v", err)
	}

	if result.JobsDeleted != 2 {
		t.Errorf("expected 2 jobs deleted, got %d", result.JobsDeleted)
	}

	// Verify only 3 jobs remain
	count, err := st.CountJobs(ctx, "")
	if err != nil {
		t.Fatalf("CountJobs failed: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 remaining jobs, got %d", count)
	}
}

func TestRunCleanupDryRun(t *testing.T) {
	st, dataDir, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create old job
	oldJob := createTestJob("old-job", model.KindScrape, model.StatusSucceeded, 60)
	if err := st.Create(ctx, oldJob); err != nil {
		t.Fatalf("Create oldJob failed: %v", err)
	}

	cfg := config.Config{
		DataDir:               dataDir,
		RetentionEnabled:      true,
		RetentionJobDays:      30,
		RetentionMaxJobs:      0,
		RetentionMaxStorageGB: 0,
	}

	engine := NewEngine(st, cfg)
	result, err := engine.RunCleanup(ctx, CleanupOptions{DryRun: true})
	if err != nil {
		t.Fatalf("RunCleanup failed: %v", err)
	}

	if result.JobsDeleted != 1 {
		t.Errorf("expected 1 job would be deleted, got %d", result.JobsDeleted)
	}

	// Verify job still exists (dry run)
	_, err = st.Get(ctx, "old-job")
	if err != nil {
		t.Error("expected old job to still exist after dry run")
	}
}

func TestRunCleanupDisabled(t *testing.T) {
	st, dataDir, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create old job
	oldJob := createTestJob("old-job", model.KindScrape, model.StatusSucceeded, 60)
	if err := st.Create(ctx, oldJob); err != nil {
		t.Fatalf("Create oldJob failed: %v", err)
	}

	cfg := config.Config{
		DataDir:               dataDir,
		RetentionEnabled:      false, // disabled
		RetentionJobDays:      30,
		RetentionMaxJobs:      0,
		RetentionMaxStorageGB: 0,
	}

	engine := NewEngine(st, cfg)
	result, err := engine.RunCleanup(ctx, CleanupOptions{})
	if err != nil {
		t.Fatalf("RunCleanup failed: %v", err)
	}

	if result.JobsDeleted != 0 {
		t.Errorf("expected 0 jobs deleted when disabled, got %d", result.JobsDeleted)
	}
}

func TestRunCleanupForce(t *testing.T) {
	st, dataDir, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create old job
	oldJob := createTestJob("old-job", model.KindScrape, model.StatusSucceeded, 60)
	if err := st.Create(ctx, oldJob); err != nil {
		t.Fatalf("Create oldJob failed: %v", err)
	}

	cfg := config.Config{
		DataDir:               dataDir,
		RetentionEnabled:      false, // disabled
		RetentionJobDays:      30,
		RetentionMaxJobs:      0,
		RetentionMaxStorageGB: 0,
	}

	engine := NewEngine(st, cfg)
	result, err := engine.RunCleanup(ctx, CleanupOptions{Force: true})
	if err != nil {
		t.Fatalf("RunCleanup failed: %v", err)
	}

	if result.JobsDeleted != 1 {
		t.Errorf("expected 1 job deleted with force, got %d", result.JobsDeleted)
	}
}

func TestRunCleanupByKind(t *testing.T) {
	st, dataDir, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create jobs of different kinds
	scrapeJob := createTestJob("scrape-job", model.KindScrape, model.StatusSucceeded, 60)
	crawlJob := createTestJob("crawl-job", model.KindCrawl, model.StatusSucceeded, 60)

	if err := st.Create(ctx, scrapeJob); err != nil {
		t.Fatalf("Create scrapeJob failed: %v", err)
	}
	if err := st.Create(ctx, crawlJob); err != nil {
		t.Fatalf("Create crawlJob failed: %v", err)
	}

	cfg := config.Config{
		DataDir:               dataDir,
		RetentionEnabled:      true,
		RetentionJobDays:      30,
		RetentionMaxJobs:      0,
		RetentionMaxStorageGB: 0,
	}

	kind := model.KindScrape
	engine := NewEngine(st, cfg)
	result, err := engine.RunCleanup(ctx, CleanupOptions{Kind: &kind})
	if err != nil {
		t.Fatalf("RunCleanup failed: %v", err)
	}

	if result.JobsDeleted != 1 {
		t.Errorf("expected 1 job deleted, got %d", result.JobsDeleted)
	}

	// Verify crawl job still exists
	_, err = st.Get(ctx, "crawl-job")
	if err != nil {
		t.Error("expected crawl job to still exist")
	}
}

func TestGetStatus(t *testing.T) {
	st, dataDir, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create jobs
	for i := 0; i < 5; i++ {
		job := createTestJob("job-"+string(rune('a'+i)), model.KindScrape, model.StatusSucceeded, i*10)
		if err := st.Create(ctx, job); err != nil {
			t.Fatalf("Create job failed: %v", err)
		}
	}

	cfg := config.Config{
		DataDir:               dataDir,
		RetentionEnabled:      true,
		RetentionJobDays:      30,
		RetentionMaxJobs:      100,
		RetentionMaxStorageGB: 10,
	}

	engine := NewEngine(st, cfg)
	status, err := engine.GetStatus(ctx)
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}

	if !status.Enabled {
		t.Error("expected enabled to be true")
	}
	if status.TotalJobs != 5 {
		t.Errorf("expected 5 total jobs, got %d", status.TotalJobs)
	}
	if status.JobRetentionDays != 30 {
		t.Errorf("expected 30 retention days, got %d", status.JobRetentionDays)
	}
}

func TestEvaluatePolicies(t *testing.T) {
	st, dataDir, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create old and new jobs
	oldJob := createTestJob("old-job", model.KindScrape, model.StatusSucceeded, 60)
	newJob := createTestJob("new-job", model.KindScrape, model.StatusSucceeded, 5)

	if err := st.Create(ctx, oldJob); err != nil {
		t.Fatalf("Create oldJob failed: %v", err)
	}
	if err := st.Create(ctx, newJob); err != nil {
		t.Fatalf("Create newJob failed: %v", err)
	}

	cfg := config.Config{
		DataDir:               dataDir,
		RetentionEnabled:      true,
		RetentionJobDays:      30,
		RetentionMaxJobs:      0,
		RetentionMaxStorageGB: 0,
	}

	engine := NewEngine(st, cfg)
	toDelete, err := engine.EvaluatePolicies(ctx, nil)
	if err != nil {
		t.Fatalf("EvaluatePolicies failed: %v", err)
	}

	if len(toDelete) != 1 {
		t.Errorf("expected 1 job to delete, got %d", len(toDelete))
	}
	if len(toDelete) > 0 && toDelete[0] != "old-job" {
		t.Errorf("expected old-job, got %s", toDelete[0])
	}
}

func TestEvaluatePoliciesDisabled(t *testing.T) {
	st, dataDir, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	cfg := config.Config{
		DataDir:               dataDir,
		RetentionEnabled:      false,
		RetentionJobDays:      30,
		RetentionMaxJobs:      0,
		RetentionMaxStorageGB: 0,
	}

	engine := NewEngine(st, cfg)
	_, err := engine.EvaluatePolicies(ctx, nil)
	if err == nil {
		t.Error("expected error when retention is disabled")
	}
}

func TestFormatResult(t *testing.T) {
	result := CleanupResult{
		JobsDeleted:        5,
		CrawlStatesDeleted: 10,
		SpaceReclaimedMB:   100,
		Duration:           time.Second * 5,
	}

	formatted := FormatResult(result, false)
	expected := "Deleted 5 jobs, 10 crawl states, reclaimed 100 MB in 5s"
	if formatted != expected {
		t.Errorf("expected %q, got %q", expected, formatted)
	}

	formattedDryRun := FormatResult(result, true)
	expectedDryRun := "Would delete 5 jobs, 10 crawl states, reclaimed 100 MB in 5s"
	if formattedDryRun != expectedDryRun {
		t.Errorf("expected %q, got %q", expectedDryRun, formattedDryRun)
	}
}

func TestRunCleanupWithArtifacts(t *testing.T) {
	st, dataDir, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create job with artifacts
	oldJob := createTestJob("old-job", model.KindScrape, model.StatusSucceeded, 60)
	if err := st.Create(ctx, oldJob); err != nil {
		t.Fatalf("Create oldJob failed: %v", err)
	}

	// Create artifact directory and file
	jobDir := filepath.Join(dataDir, "jobs", oldJob.ID)
	if err := os.MkdirAll(jobDir, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	content := []byte(`{"data": "test content for artifact"}`)
	if err := os.WriteFile(filepath.Join(jobDir, "result.json"), content, 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	cfg := config.Config{
		DataDir:               dataDir,
		RetentionEnabled:      true,
		RetentionJobDays:      30,
		RetentionMaxJobs:      0,
		RetentionMaxStorageGB: 0,
	}

	engine := NewEngine(st, cfg)
	result, err := engine.RunCleanup(ctx, CleanupOptions{})
	if err != nil {
		t.Fatalf("RunCleanup failed: %v", err)
	}

	if result.JobsDeleted != 1 {
		t.Errorf("expected 1 job deleted, got %d", result.JobsDeleted)
	}
	if result.SpaceReclaimedMB < 1 {
		t.Errorf("expected at least 1 MB reclaimed, got %d", result.SpaceReclaimedMB)
	}

	// Verify artifacts are deleted
	if _, err := os.Stat(jobDir); !os.IsNotExist(err) {
		t.Error("expected artifact directory to be deleted")
	}
}

func TestRunCleanupPriorityOrder(t *testing.T) {
	st, dataDir, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create jobs with different statuses (all old)
	failedJob := createTestJob("failed-job", model.KindScrape, model.StatusFailed, 60)
	succeededJob := createTestJob("succeeded-job", model.KindScrape, model.StatusSucceeded, 60)
	canceledJob := createTestJob("canceled-job", model.KindScrape, model.StatusCanceled, 60)

	if err := st.Create(ctx, failedJob); err != nil {
		t.Fatalf("Create failedJob failed: %v", err)
	}
	if err := st.Create(ctx, succeededJob); err != nil {
		t.Fatalf("Create succeededJob failed: %v", err)
	}
	if err := st.Create(ctx, canceledJob); err != nil {
		t.Fatalf("Create canceledJob failed: %v", err)
	}

	cfg := config.Config{
		DataDir:               dataDir,
		RetentionEnabled:      true,
		RetentionJobDays:      30,
		RetentionMaxJobs:      0,
		RetentionMaxStorageGB: 0,
	}

	engine := NewEngine(st, cfg)
	result, err := engine.RunCleanup(ctx, CleanupOptions{})
	if err != nil {
		t.Fatalf("RunCleanup failed: %v", err)
	}

	if result.JobsDeleted != 3 {
		t.Errorf("expected 3 jobs deleted, got %d", result.JobsDeleted)
	}
}

// TestCleanupByKindWithMixedBatches verifies that cleanup works correctly when
// early batches contain only non-matching job kinds but later batches contain
// matching kinds. This is a regression test for the bug where len(toDelete)==0
// would cause a break instead of continue, prematurely stopping the scan.
func TestCleanupByKindWithMixedBatches(t *testing.T) {
	st, dataDir, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Create 110 crawl jobs (fills more than one 100-job batch)
	// These are created first and will appear in earlier batches
	for i := 0; i < 110; i++ {
		job := createTestJob(fmt.Sprintf("crawl-job-%d", i), model.KindCrawl, model.StatusSucceeded, 60)
		if err := st.Create(ctx, job); err != nil {
			t.Fatalf("Create crawl job failed: %v", err)
		}
	}

	// Create 10 scrape jobs (all old enough for cleanup)
	// These will appear in later batches due to creation order
	for i := 0; i < 10; i++ {
		job := createTestJob(fmt.Sprintf("scrape-job-%d", i), model.KindScrape, model.StatusSucceeded, 60)
		if err := st.Create(ctx, job); err != nil {
			t.Fatalf("Create scrape job failed: %v", err)
		}
	}

	cfg := config.Config{
		DataDir:               dataDir,
		RetentionEnabled:      true,
		RetentionJobDays:      30, // Jobs older than 30 days should be deleted
		RetentionMaxJobs:      0,  // unlimited
		RetentionMaxStorageGB: 0,  // unlimited
	}

	// Run cleanup filtering only for KindScrape
	kind := model.KindScrape
	engine := NewEngine(st, cfg)
	result, err := engine.RunCleanup(ctx, CleanupOptions{Kind: &kind})
	if err != nil {
		t.Fatalf("RunCleanup failed: %v", err)
	}

	// Verify that all 10 scrape jobs were deleted
	if result.JobsDeleted != 10 {
		t.Errorf("expected 10 scrape jobs deleted, got %d", result.JobsDeleted)
	}

	// Verify that crawl jobs still exist (different kind, should not be affected)
	for i := 0; i < 110; i++ {
		_, err := st.Get(ctx, fmt.Sprintf("crawl-job-%d", i))
		if err != nil {
			t.Errorf("expected crawl job %d to still exist", i)
		}
	}

	// Verify that scrape jobs are deleted
	for i := 0; i < 10; i++ {
		_, err := st.Get(ctx, fmt.Sprintf("scrape-job-%d", i))
		if err == nil {
			t.Errorf("expected scrape job %d to be deleted", i)
		}
	}
}
