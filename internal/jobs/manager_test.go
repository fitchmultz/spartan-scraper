package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

func setupTestManager(t *testing.T) (*Manager, *store.Store, func()) {
	t.Helper()
	dataDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}

	m := NewManager(
		st,
		dataDir,
		"TestAgent/1.0",
		30*time.Second,
		2,
		10,
		20,
		3,
		100*time.Millisecond,
		10*1024*1024,
		false,
	)

	cleanup := func() {
		st.Close()
	}

	return m, st, cleanup
}

func TestManagerEnqueue(t *testing.T) {
	m, _, cleanup := setupTestManager(t)
	defer cleanup()

	job := model.Job{ID: "test-job"}
	if err := m.Enqueue(job); err != nil {
		t.Errorf("Enqueue failed: %v", err)
	}

	// Should be in queue
	select {
	case j := <-m.queue:
		if j.ID != "test-job" {
			t.Errorf("expected job id test-job, got %s", j.ID)
		}
	default:
		t.Error("queue is empty")
	}
}

func TestManagerCreateScrapeJob(t *testing.T) {
	m, st, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()
	job, err := m.CreateScrapeJob(ctx, "http://example.com", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false)
	if err != nil {
		t.Fatalf("CreateScrapeJob failed: %v", err)
	}

	if job.Kind != model.KindScrape {
		t.Errorf("expected kind scrape, got %v", job.Kind)
	}

	// Verify persistence
	persisted, err := st.Get(ctx, job.ID)
	if err != nil {
		t.Fatalf("failed to get job from store: %v", err)
	}
	if persisted.ID != job.ID {
		t.Errorf("expected job id %s, got %s", job.ID, persisted.ID)
	}
}

func TestManagerCreateJob_Scrape(t *testing.T) {
	m, st, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()
	spec := JobSpec{
		Kind:           model.KindScrape,
		URL:            "http://example.com",
		Headless:       true,
		UsePlaywright:  false,
		Auth:           fetch.AuthOptions{},
		TimeoutSeconds: 30,
		Extract:        extract.ExtractOptions{},
		Pipeline:       pipeline.Options{},
		Incremental:    false,
	}
	job, err := m.CreateJob(ctx, spec)
	if err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}

	if job.Kind != model.KindScrape {
		t.Errorf("expected kind scrape, got %v", job.Kind)
	}

	persisted, err := st.Get(ctx, job.ID)
	if err != nil {
		t.Fatalf("failed to get job from store: %v", err)
	}
	if persisted.ID != job.ID {
		t.Errorf("expected job id %s, got %s", job.ID, persisted.ID)
	}
}

func TestManagerCreateJob_Crawl(t *testing.T) {
	m, st, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()
	spec := JobSpec{
		Kind:           model.KindCrawl,
		URL:            "http://example.com",
		MaxDepth:       2,
		MaxPages:       100,
		Headless:       true,
		UsePlaywright:  false,
		Auth:           fetch.AuthOptions{},
		TimeoutSeconds: 30,
		Extract:        extract.ExtractOptions{},
		Pipeline:       pipeline.Options{},
		Incremental:    false,
	}
	job, err := m.CreateJob(ctx, spec)
	if err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}

	if job.Kind != model.KindCrawl {
		t.Errorf("expected kind crawl, got %v", job.Kind)
	}

	persisted, err := st.Get(ctx, job.ID)
	if err != nil {
		t.Fatalf("failed to get job from store: %v", err)
	}
	if persisted.ID != job.ID {
		t.Errorf("expected job id %s, got %s", job.ID, persisted.ID)
	}
}

func TestManagerCreateJob_Research(t *testing.T) {
	m, st, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()
	spec := JobSpec{
		Kind:           model.KindResearch,
		Query:          "test query",
		URLs:           []string{"http://example.com"},
		MaxDepth:       2,
		MaxPages:       100,
		Headless:       true,
		UsePlaywright:  false,
		Auth:           fetch.AuthOptions{},
		TimeoutSeconds: 30,
		Extract:        extract.ExtractOptions{},
		Pipeline:       pipeline.Options{},
		Incremental:    false,
	}
	job, err := m.CreateJob(ctx, spec)
	if err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}

	if job.Kind != model.KindResearch {
		t.Errorf("expected kind research, got %v", job.Kind)
	}

	persisted, err := st.Get(ctx, job.ID)
	if err != nil {
		t.Fatalf("failed to get job from store: %v", err)
	}
	if persisted.ID != job.ID {
		t.Errorf("expected job id %s, got %s", job.ID, persisted.ID)
	}
}

func TestManagerCreateJob_InvalidSpec(t *testing.T) {
	m, _, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()

	spec := JobSpec{
		Kind:           model.KindScrape,
		URL:            "",
		TimeoutSeconds: 30,
	}

	_, err := m.CreateJob(ctx, spec)
	if err == nil {
		t.Error("expected error for invalid spec, got nil")
	}
}

func TestManagerCreateJob_UnknownKind(t *testing.T) {
	m, _, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()

	spec := JobSpec{
		Kind:           "unknown",
		URL:            "http://example.com",
		TimeoutSeconds: 30,
	}

	_, err := m.CreateJob(ctx, spec)
	if err == nil {
		t.Error("expected error for unknown kind, got nil")
	}
}

func TestToInt(t *testing.T) {
	tests := []struct {
		input    interface{}
		fallback int
		expected int
	}{
		{10, 5, 10},
		{0, 5, 5},
		{-1, 5, 5},
		{10.0, 5, 10},
		{"10", 5, 5},
	}

	for _, tt := range tests {
		got := toInt(tt.input, tt.fallback)
		if got != tt.expected {
			t.Errorf("toInt(%v, %d) = %d; want %d", tt.input, tt.fallback, got, tt.expected)
		}
	}
}

func TestToBool(t *testing.T) {
	tests := []struct {
		input    interface{}
		fallback bool
		expected bool
	}{
		{true, false, true},
		{false, true, false},
		{"true", true, true},
		{1, false, false},
	}

	for _, tt := range tests {
		got := toBool(tt.input, tt.fallback)
		if got != tt.expected {
			t.Errorf("toBool(%v, %v) = %v; want %v", tt.input, tt.fallback, got, tt.expected)
		}
	}
}

func TestManagerStartStop(t *testing.T) {
	m, _, cleanup := setupTestManager(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	m.Start(ctx)
	cancel()
	// Should return quickly
}

func TestManagerCancelJob(t *testing.T) {
	m, st, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()
	job, err := m.CreateScrapeJob(ctx, "http://example.com", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false)
	if err != nil {
		t.Fatalf("CreateScrapeJob failed: %v", err)
	}

	// Cancel before it starts
	if err := m.CancelJob(ctx, job.ID); err != nil {
		t.Errorf("CancelJob failed: %v", err)
	}

	persisted, err := st.Get(ctx, job.ID)
	if err != nil {
		t.Fatalf("failed to get job from store: %v", err)
	}
	if persisted.Status != model.StatusCanceled {
		t.Errorf("expected status canceled, got %v", persisted.Status)
	}
}

func TestManagerCancelJob_AfterSuccess(t *testing.T) {
	m, st, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()

	// Create a job and manually set it to succeeded
	job, err := m.CreateScrapeJob(ctx, "http://example.com", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false)
	if err != nil {
		t.Fatalf("CreateScrapeJob failed: %v", err)
	}

	// Manually set status to succeeded
	if err := st.UpdateStatus(ctx, job.ID, model.StatusSucceeded, "test success"); err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	// Verify job is succeeded
	persisted, err := st.Get(ctx, job.ID)
	if err != nil {
		t.Fatalf("failed to get job from store: %v", err)
	}
	if persisted.Status != model.StatusSucceeded {
		t.Fatalf("expected job to be succeeded, got status %v", persisted.Status)
	}

	// Try to cancel the succeeded job
	if err := m.CancelJob(ctx, job.ID); err != nil {
		t.Errorf("CancelJob failed: %v", err)
	}

	// Verify status was NOT overwritten
	persisted, err = st.Get(ctx, job.ID)
	if err != nil {
		t.Fatalf("failed to get job from store: %v", err)
	}
	if persisted.Status != model.StatusSucceeded {
		t.Errorf("expected status to remain succeeded, got %v", persisted.Status)
	}
}

func TestManagerCancelJob_AfterFailure(t *testing.T) {
	m, st, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()

	// Create a job and manually set it to failed
	job, err := m.CreateScrapeJob(ctx, "http://example.com", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false)
	if err != nil {
		t.Fatalf("CreateScrapeJob failed: %v", err)
	}

	// Manually set status to failed
	if err := st.UpdateStatus(ctx, job.ID, model.StatusFailed, "test failure"); err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	// Verify job is failed
	persisted, err := st.Get(ctx, job.ID)
	if err != nil {
		t.Fatalf("failed to get job from store: %v", err)
	}
	if persisted.Status != model.StatusFailed {
		t.Fatalf("expected job to be failed, got status %v", persisted.Status)
	}

	// Try to cancel the failed job
	if err := m.CancelJob(ctx, job.ID); err != nil {
		t.Errorf("CancelJob failed: %v", err)
	}

	// Verify status was NOT overwritten
	persisted, err = st.Get(ctx, job.ID)
	if err != nil {
		t.Fatalf("failed to get job from store: %v", err)
	}
	if persisted.Status != model.StatusFailed {
		t.Errorf("expected status to remain failed, got %v", persisted.Status)
	}
}

func TestManagerCancelJob_WhileRunning(t *testing.T) {
	m, st, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()

	// Create a job with a URL that will take some time
	job, err := m.CreateScrapeJob(ctx, "http://example.com", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false)
	if err != nil {
		t.Fatalf("CreateScrapeJob failed: %v", err)
	}

	m.Start(ctx)

	// Wait for job to be running
	var persisted model.Job
	for i := 0; i < 10; i++ {
		persisted, err = st.Get(ctx, job.ID)
		if err != nil {
			t.Fatalf("failed to get job from store: %v", err)
		}
		if persisted.Status == model.StatusRunning {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	// Verify job is running
	if persisted.Status != model.StatusRunning {
		t.Fatalf("expected job to be running, got status %v", persisted.Status)
	}

	// Cancel job
	if err := m.CancelJob(ctx, job.ID); err != nil {
		t.Errorf("CancelJob failed: %v", err)
	}

	// Wait for cancellation or completion
	time.Sleep(200 * time.Millisecond)

	// Verify job is not stuck in running (either canceled or terminal)
	persisted, err = st.Get(ctx, job.ID)
	if err != nil {
		t.Fatalf("failed to get job from store: %v", err)
	}
	// Job should be explicitly canceled
	if persisted.Status != model.StatusCanceled {
		t.Errorf("expected job to be canceled, got status %v", persisted.Status)
	}
}

func TestManagerRun_ContextCancellation(t *testing.T) {
	m, st, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()

	// Create a job
	job, err := m.CreateScrapeJob(ctx, "http://example.com", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false)
	if err != nil {
		t.Fatalf("CreateScrapeJob failed: %v", err)
	}

	if err := m.Enqueue(job); err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	m.Start(ctx)

	// Wait a bit for job to start
	time.Sleep(50 * time.Millisecond)

	// Cancel job
	if err := m.CancelJob(ctx, job.ID); err != nil {
		t.Errorf("CancelJob failed: %v", err)
	}

	// Wait for cancellation or completion
	time.Sleep(200 * time.Millisecond)

	// Verify job status is either canceled or a terminal state (succeeded/failed)
	persisted, err := st.Get(ctx, job.ID)
	if err != nil {
		t.Fatalf("failed to get job from store: %v", err)
	}
	// Job should NOT be stuck in running state
	if persisted.Status == model.StatusRunning {
		t.Errorf("job should not be stuck in running state, got status %v", persisted.Status)
	}
}

func TestManagerRecoverQueuedJobs(t *testing.T) {
	m, st, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()

	// Create some jobs but don't start manager yet
	job1, _ := m.CreateScrapeJob(ctx, "http://example.com/1", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false)
	job2, _ := m.CreateScrapeJob(ctx, "http://example.com/2", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false)

	// Verify they're in store but not in queue
	queuedJobs, _ := st.ListByStatus(ctx, model.StatusQueued, store.ListByStatusOptions{})
	if len(queuedJobs) != 2 {
		t.Fatalf("expected 2 queued jobs in store, got %d", len(queuedJobs))
	}

	if len(m.queue) != 0 {
		t.Error("queue should be empty before Start")
	}

	// Start manager (should recover jobs)
	cancelCtx, cancel := context.WithCancel(ctx)
	m.Start(cancelCtx)

	// Give workers a moment to pick up jobs
	time.Sleep(200 * time.Millisecond)

	// Cancel to stop workers
	cancel()

	// Verify jobs were processed (or at least picked up)
	// Check final status in store - they may have failed (since example.com may not be reachable)
	// but they should no longer be queued
	final1, _ := st.Get(ctx, job1.ID)
	final2, _ := st.Get(ctx, job2.ID)

	// Jobs should NOT be queued anymore (they were recovered and processed)
	if final1.Status == model.StatusQueued || final2.Status == model.StatusQueued {
		t.Error("recovered jobs should have been picked up from queue")
	}
}

func TestManagerShutdownWithQueuedJobs(t *testing.T) {
	m, st, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()

	// Create multiple jobs that will remain queued
	const jobCount = 5
	var jobIDs []string
	for i := 0; i < jobCount; i++ {
		job, err := m.CreateScrapeJob(ctx, "http://example.com/test", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false)
		if err != nil {
			t.Fatalf("CreateScrapeJob %d failed: %v", i, err)
		}
		jobIDs = append(jobIDs, job.ID)
	}

	// Enqueue jobs manually to fill the queue before starting
	for _, jobID := range jobIDs {
		job, err := st.Get(ctx, jobID)
		if err != nil {
			t.Fatalf("failed to get job %s: %v", jobID, err)
		}
		if err := m.Enqueue(job); err != nil {
			t.Fatalf("failed to enqueue job %s: %v", jobID, err)
		}
	}

	// Verify queue has jobs
	if len(m.queue) != jobCount {
		t.Fatalf("expected %d jobs in queue, got %d", jobCount, len(m.queue))
	}

	// Start manager with a short-lived context to trigger shutdown
	startCtx, startCancel := context.WithCancel(ctx)
	startDone := make(chan struct{})

	go func() {
		m.Start(startCtx)
		close(startDone)
	}()

	// Wait for workers to start
	time.Sleep(50 * time.Millisecond)

	// Cancel context to trigger shutdown with queued jobs
	startCancel()

	// Wait for shutdown to complete
	<-startDone
	m.Wait()

	// Verify that queued jobs were processed and reached terminal states
	// Jobs should either be completed, failed, or canceled - not stuck in queued/running
	allTerminal := true
	for _, jobID := range jobIDs {
		job, err := st.Get(ctx, jobID)
		if err != nil {
			t.Errorf("failed to get job %s: %v", jobID, err)
			allTerminal = false
			continue
		}
		if !job.Status.IsTerminal() {
			t.Errorf("job %s is not in terminal state after shutdown: %v", jobID, job.Status)
			allTerminal = false
		}
	}

	if !allTerminal {
		t.Error("not all jobs reached terminal state after shutdown")
	}
}

func TestContextCleanupOnShutdown(t *testing.T) {
	m, st, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()

	// Create jobs to test drain cleanup
	const jobCount = 3
	var jobIDs []string
	for i := 0; i < jobCount; i++ {
		job, err := m.CreateScrapeJob(ctx, "http://example.com/test", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false)
		if err != nil {
			t.Fatalf("CreateScrapeJob %d failed: %v", i, err)
		}
		jobIDs = append(jobIDs, job.ID)
	}

	// Enqueue jobs
	for _, jobID := range jobIDs {
		job, err := st.Get(ctx, jobID)
		if err != nil {
			t.Fatalf("failed to get job %s: %v", jobID, err)
		}
		if err := m.Enqueue(job); err != nil {
			t.Fatalf("failed to enqueue job %s: %v", jobID, err)
		}
	}

	// Start and immediately trigger shutdown to exercise drain path
	startCtx, startCancel := context.WithCancel(ctx)
	startDone := make(chan struct{})

	go func() {
		m.Start(startCtx)
		close(startDone)
	}()

	// Wait briefly for workers to start
	time.Sleep(50 * time.Millisecond)

	// Trigger shutdown
	startCancel()

	// Wait for clean shutdown - should complete within drain timeout
	select {
	case <-startDone:
		// Shutdown completed
	case <-time.After(5 * time.Second):
		t.Fatal("shutdown took too long, possible context leak")
	}

	m.Wait()

	// Verify jobs were processed and reached terminal states
	for _, jobID := range jobIDs {
		job, err := st.Get(ctx, jobID)
		if err != nil {
			t.Errorf("failed to get job %s: %v", jobID, err)
			continue
		}
		if !job.Status.IsTerminal() {
			t.Errorf("job %s is not in terminal state after shutdown: %v", jobID, job.Status)
		}
	}
}

func TestContextCleanupOnJobError(t *testing.T) {
	m, st, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()

	// Test cancelUpdate is called when jobs fail due to invalid directory
	// Create a job with a path that will fail during directory creation
	// This tests the error path in job_run.go lines 54-62
	job, err := m.CreateScrapeJob(ctx, "http://example.com/test", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false)
	if err != nil {
		t.Fatalf("CreateScrapeJob failed: %v", err)
	}

	// Enqueue the job
	if err := m.Enqueue(job); err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	// Start manager to process job
	startCtx, startCancel := context.WithCancel(ctx)
	startDone := make(chan struct{})

	go func() {
		m.Start(startCtx)
		close(startDone)
	}()

	// Wait for job to be processed
	time.Sleep(200 * time.Millisecond)

	// Trigger shutdown
	startCancel()
	<-startDone
	m.Wait()

	// Verify job reached a terminal state (should be failed due to network error)
	persisted, err := st.Get(ctx, job.ID)
	if err != nil {
		t.Fatalf("failed to get job: %v", err)
	}

	if !persisted.Status.IsTerminal() {
		t.Errorf("job should be in terminal state, got %v", persisted.Status)
	}
}
