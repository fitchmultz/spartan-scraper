package store

import (
	"context"
	"testing"
	"time"

	"spartan-scraper/internal/model"
)

func TestStoreJobs(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	job := model.Job{
		ID:        "j1",
		Kind:      model.KindScrape,
		Status:    model.StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Params:    map[string]interface{}{"url": "http://example.com"},
	}

	if err := s.Create(ctx, job); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := s.Get(ctx, "j1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ID != job.ID || got.Status != job.Status {
		t.Errorf("Get returned unexpected job: %+v", got)
	}

	if err := s.UpdateStatus(ctx, "j1", model.StatusRunning, "error message"); err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	got, _ = s.Get(ctx, "j1")
	if got.Status != model.StatusRunning || got.Error != "error message" {
		t.Errorf("UpdateStatus did not work as expected: %+v", got)
	}

	jobs, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(jobs))
	}
}

func TestStoreCrawlState(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	state := model.CrawlState{
		URL:         "http://example.com",
		ETag:        "tag",
		LastScraped: time.Now(),
	}

	if err := s.UpsertCrawlState(ctx, state); err != nil {
		t.Fatalf("UpsertCrawlState failed: %v", err)
	}

	got, err := s.GetCrawlState(ctx, "http://example.com")
	if err != nil {
		t.Fatalf("GetCrawlState failed: %v", err)
	}
	if got.URL != state.URL || got.ETag != state.ETag {
		t.Errorf("GetCrawlState returned unexpected state: %+v", got)
	}

	// Update
	state.ETag = "new-tag"
	if err := s.UpsertCrawlState(ctx, state); err != nil {
		t.Fatalf("UpsertCrawlState (update) failed: %v", err)
	}

	got, _ = s.GetCrawlState(ctx, "http://example.com")
	if got.ETag != "new-tag" {
		t.Errorf("expected etag new-tag, got %s", got.ETag)
	}
}
