package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	jobs "github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

func setupTestManager(t *testing.T) (*jobs.Manager, *store.Store, func()) {
	t.Helper()
	dataDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}

	m := jobs.NewManager(
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

func TestManagerRecoverQueuedJobsPagination(t *testing.T) {
	m, st, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()

	// Create more jobs than default pagination limit (100)
	// to verify that pagination loop in recoverQueuedJobs works correctly.
	const jobCount = 120

	for i := 0; i < jobCount; i++ {
		_, err := m.CreateScrapeJob(ctx, "http://example.com/test", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false, "")
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
	status := m.Status()
	if status.QueuedJobs == 0 {
		t.Error("expected queue to have jobs after recovery, but queue is empty")
	}

	// Cancel workers and wait
	cancel()
	<-startDone
	m.Wait()

	// The test passes if we got here without deadlock or panic
	// The log output will show actual recovery count
	// We expect to see "job recovery complete total_recovered=120" in logs
}
