// Package jobs provides tests for job cancellation operations.
// Tests cover canceling queued, running, and already-terminal jobs.
// Does NOT test cancellation of non-existent jobs or concurrent cancel races.
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

func TestManagerCancelJob(t *testing.T) {
	m, st, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()
	job, err := m.CreateScrapeJob(ctx, "http://example.com", "GET", nil, "", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false, "", "", nil, "")
	if err != nil {
		t.Fatalf("CreateScrapeJob failed: %v", err)
	}

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

	job, err := m.CreateScrapeJob(ctx, "http://example.com", "GET", nil, "", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false, "", "", nil, "")
	if err != nil {
		t.Fatalf("CreateScrapeJob failed: %v", err)
	}

	if err := st.UpdateStatus(ctx, job.ID, model.StatusSucceeded, "test success"); err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	persisted, err := st.Get(ctx, job.ID)
	if err != nil {
		t.Fatalf("failed to get job from store: %v", err)
	}
	if persisted.Status != model.StatusSucceeded {
		t.Fatalf("expected job to be succeeded, got status %v", persisted.Status)
	}

	if err := m.CancelJob(ctx, job.ID); err != nil {
		t.Errorf("CancelJob failed: %v", err)
	}

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

	job, err := m.CreateScrapeJob(ctx, "http://example.com", "GET", nil, "", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false, "", "", nil, "")
	if err != nil {
		t.Fatalf("CreateScrapeJob failed: %v", err)
	}

	if err := st.UpdateStatus(ctx, job.ID, model.StatusFailed, "test failure"); err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	persisted, err := st.Get(ctx, job.ID)
	if err != nil {
		t.Fatalf("failed to get job from store: %v", err)
	}
	if persisted.Status != model.StatusFailed {
		t.Fatalf("expected job to be failed, got status %v", persisted.Status)
	}

	if err := m.CancelJob(ctx, job.ID); err != nil {
		t.Errorf("CancelJob failed: %v", err)
	}

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

	job, err := m.CreateScrapeJob(ctx, "http://example.com", "GET", nil, "", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false, "", "", nil, "")
	if err != nil {
		t.Fatalf("CreateScrapeJob failed: %v", err)
	}

	m.Start(ctx)

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

	if persisted.Status != model.StatusRunning {
		t.Fatalf("expected job to be running, got status %v", persisted.Status)
	}

	if err := m.CancelJob(ctx, job.ID); err != nil {
		t.Errorf("CancelJob failed: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	persisted, err = st.Get(ctx, job.ID)
	if err != nil {
		t.Fatalf("failed to get job from store: %v", err)
	}
	if persisted.Status != model.StatusCanceled {
		t.Errorf("expected job to be canceled, got status %v", persisted.Status)
	}
}
