package jobs

import (
	"context"
	"testing"
	"time"

	"spartan-scraper/internal/extract"
	"spartan-scraper/internal/fetch"
	"spartan-scraper/internal/model"
	"spartan-scraper/internal/pipeline"
	"spartan-scraper/internal/store"
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

func TestManagerRecoverQueuedJobsPagination(t *testing.T) {
	m, st, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()

	// Create more jobs than the default pagination limit (100)
	// to verify that the pagination loop in recoverQueuedJobs works correctly.
	const jobCount = 120

	for i := 0; i < jobCount; i++ {
		_, err := m.CreateScrapeJob(ctx, "http://example.com/test", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false)
		if err != nil {
			t.Fatalf("CreateScrapeJob %d failed: %v", i, err)
		}
	}

	// Verify all jobs are in store using pagination (this tests the store's pagination too)
	var queuedCount int
	opts := store.ListByStatusOptions{Limit: 100}
	for {
		queuedJobs, err := st.ListByStatus(ctx, model.StatusQueued, opts)
		if err != nil {
			t.Fatalf("ListByStatus failed: %v", err)
		}
		queuedCount += len(queuedJobs)
		if len(queuedJobs) < opts.Limit {
			break
		}
		opts.Offset += len(queuedJobs)
	}
	if queuedCount != jobCount {
		t.Fatalf("expected %d queued jobs in store, got %d", jobCount, queuedCount)
	}

	if len(m.queue) != 0 {
		t.Error("queue should be empty before Start")
	}

	// Start manager - it should recover all jobs via pagination
	// The key observable behavior: since recovery happens synchronously
	// before workers start processing, we can check the queue size immediately
	// after Start is called (but in a goroutine so we don't block forever)
	cancelCtx, cancel := context.WithCancel(ctx)
	startDone := make(chan struct{})

	go func() {
		m.Start(cancelCtx)
		close(startDone)
	}()

	// Wait for recovery to complete and jobs to be enqueued
	// Recovery is synchronous, so once workers start picking up jobs,
	// recovery is done
	time.Sleep(50 * time.Millisecond)

	// At this point, recovery should have completed
	// The queue should have jobs in it (up to capacity of 128)
	queueSize := len(m.queue)
	if queueSize == 0 {
		t.Error("expected queue to have jobs after recovery, but queue is empty")
	}

	// Cancel workers and wait
	cancel()
	<-startDone
	m.Wait()

	// The test passes if we got here without deadlock or panic
	// The log output will show the actual recovery count
	// We expect to see "job recovery complete total_recovered=120" in logs
}
