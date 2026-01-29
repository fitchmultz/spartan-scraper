package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

func TestManagerStartStop(t *testing.T) {
	m, _, cleanup := setupTestManager(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	m.Start(ctx)
	cancel()
}

func TestManagerShutdownWithQueuedJobs(t *testing.T) {
	m, st, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()

	const jobCount = 5
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

	if len(m.queue) != jobCount {
		t.Fatalf("expected %d jobs in queue, got %d", jobCount, len(m.queue))
	}

	startCtx, startCancel := context.WithCancel(ctx)
	startDone := make(chan struct{})

	go func() {
		m.Start(startCtx)
		close(startDone)
	}()

	time.Sleep(50 * time.Millisecond)

	startCancel()

	<-startDone
	m.Wait()

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
