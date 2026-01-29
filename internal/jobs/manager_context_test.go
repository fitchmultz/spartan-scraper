package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

func TestManagerRun_ContextCancellation(t *testing.T) {
	m, st, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()

	job, err := m.CreateScrapeJob(ctx, "http://example.com", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false, "")
	if err != nil {
		t.Fatalf("CreateScrapeJob failed: %v", err)
	}

	if err := m.Enqueue(job); err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	m.Start(ctx)

	time.Sleep(50 * time.Millisecond)

	if err := m.CancelJob(ctx, job.ID); err != nil {
		t.Errorf("CancelJob failed: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	persisted, err := st.Get(ctx, job.ID)
	if err != nil {
		t.Fatalf("failed to get job from store: %v", err)
	}
	if persisted.Status == model.StatusRunning {
		t.Errorf("job should not be stuck in running state, got status %v", persisted.Status)
	}
}

func TestContextCleanupOnShutdown(t *testing.T) {
	m, st, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()

	const jobCount = 3
	var jobIDs []string
	for i := 0; i < jobCount; i++ {
		job, err := m.CreateScrapeJob(ctx, "http://example.com/test", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false, "")
		if err != nil {
			t.Fatalf("CreateScrapeJob %d failed: %v", i, err)
		}
		jobIDs = append(jobIDs, job.ID)
	}

	for _, jobID := range jobIDs {
		job, err := st.Get(ctx, jobID)
		if err != nil {
			t.Fatalf("failed to get job %s: %v", jobID, err)
		}
		if err := m.Enqueue(job); err != nil {
			t.Fatalf("failed to enqueue job %s: %v", jobID, err)
		}
	}

	startCtx, startCancel := context.WithCancel(ctx)
	startDone := make(chan struct{})

	go func() {
		m.Start(startCtx)
		close(startDone)
	}()

	time.Sleep(50 * time.Millisecond)

	startCancel()

	select {
	case <-startDone:
	case <-time.After(5 * time.Second):
		t.Fatal("shutdown took too long, possible context leak")
	}

	m.Wait()

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

	job, err := m.CreateScrapeJob(ctx, "http://example.com/test", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false, "")
	if err != nil {
		t.Fatalf("CreateScrapeJob failed: %v", err)
	}

	if err := m.Enqueue(job); err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	startCtx, startCancel := context.WithCancel(ctx)
	startDone := make(chan struct{})

	go func() {
		m.Start(startCtx)
		close(startDone)
	}()

	time.Sleep(200 * time.Millisecond)

	startCancel()
	<-startDone
	m.Wait()

	persisted, err := st.Get(ctx, job.ID)
	if err != nil {
		t.Fatalf("failed to get job: %v", err)
	}

	if !persisted.Status.IsTerminal() {
		t.Errorf("job should be in terminal state, got %v", persisted.Status)
	}
}
