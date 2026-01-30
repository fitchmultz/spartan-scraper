// Package jobs provides tests for job creation via the manager.
// Tests cover creating scrape, crawl, and research jobs, plus validation errors.
// Does NOT test job execution or persistence edge cases.
package jobs

import (
	"context"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

func TestManagerEnqueue(t *testing.T) {
	m, _, cleanup := setupTestManager(t)
	defer cleanup()

	job := model.Job{ID: "test-job"}
	if err := m.Enqueue(job); err != nil {
		t.Errorf("Enqueue failed: %v", err)
	}

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
	job, err := m.CreateScrapeJob(ctx, "http://example.com", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false, "", "", nil, "")
	if err != nil {
		t.Fatalf("CreateScrapeJob failed: %v", err)
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
