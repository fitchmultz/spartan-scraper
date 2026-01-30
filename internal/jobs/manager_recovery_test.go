// Package jobs provides tests for manager recovery of queued jobs.
// Tests cover recovery of persisted queued jobs on manager startup.
// Does NOT test recovery of running jobs or partial execution state.
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

func TestManagerRecoverQueuedJobs(t *testing.T) {
	m, st, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()

	job1, _ := m.CreateScrapeJob(ctx, "http://example.com/1", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false, "")
	job2, _ := m.CreateScrapeJob(ctx, "http://example.com/2", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false, "")

	queuedJobs, _ := st.ListByStatus(ctx, model.StatusQueued, store.ListByStatusOptions{})
	if len(queuedJobs) != 2 {
		t.Fatalf("expected 2 queued jobs in store, got %d", len(queuedJobs))
	}

	if len(m.queue) != 0 {
		t.Error("queue should be empty before Start")
	}

	cancelCtx, cancel := context.WithCancel(ctx)
	m.Start(cancelCtx)

	time.Sleep(200 * time.Millisecond)

	cancel()

	final1, _ := st.Get(ctx, job1.ID)
	final2, _ := st.Get(ctx, job2.ID)

	if final1.Status == model.StatusQueued || final2.Status == model.StatusQueued {
		t.Error("recovered jobs should have been picked up from queue")
	}
}
