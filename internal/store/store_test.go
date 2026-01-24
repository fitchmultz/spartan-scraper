package store

import (
	"context"
	"fmt"
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

func TestStoreListOptsPagination(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	// Create 25 jobs
	for i := 0; i < 25; i++ {
		job := model.Job{
			ID:        fmt.Sprintf("j%02d", i),
			Kind:      model.KindScrape,
			Status:    model.StatusQueued,
			CreatedAt: time.Now().Add(time.Duration(i) * time.Second),
			UpdatedAt: time.Now(),
			Params:    map[string]interface{}{"idx": i},
		}
		if err := s.Create(ctx, job); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// Test limit
	opts := ListOptions{Limit: 10, Offset: 0}
	jobs, err := s.ListOpts(ctx, opts)
	if err != nil {
		t.Fatalf("ListOpts failed: %v", err)
	}
	if len(jobs) != 10 {
		t.Errorf("expected 10 jobs with limit=10, got %d", len(jobs))
	}

	// Test offset
	opts = ListOptions{Limit: 10, Offset: 10}
	jobs, err = s.ListOpts(ctx, opts)
	if err != nil {
		t.Fatalf("ListOpts failed: %v", err)
	}
	if len(jobs) != 10 {
		t.Errorf("expected 10 jobs with offset=10, got %d", len(jobs))
	}

	// Test ordering (should be desc by created_at)
	opts = ListOptions{Limit: 5, Offset: 0}
	jobs, err = s.ListOpts(ctx, opts)
	if err != nil {
		t.Fatalf("ListOpts failed: %v", err)
	}
	// First job should have highest created_at (j24)
	if jobs[0].ID != "j24" {
		t.Errorf("expected first job to be j24, got %s", jobs[0].ID)
	}

	// Test offset beyond available jobs
	opts = ListOptions{Limit: 10, Offset: 100}
	jobs, err = s.ListOpts(ctx, opts)
	if err != nil {
		t.Fatalf("ListOpts failed: %v", err)
	}
	if len(jobs) != 0 {
		t.Errorf("expected 0 jobs with offset=100, got %d", len(jobs))
	}
}

func TestListOptionsDefaults(t *testing.T) {
	tests := []struct {
		name       string
		input      ListOptions
		wantLimit  int
		wantOffset int
	}{
		{"zero values use defaults", ListOptions{}, 100, 0},
		{"negative limit uses default", ListOptions{Limit: -1}, 100, 0},
		{"negative offset uses zero", ListOptions{Offset: -5}, 100, 0},
		{"max limit capped", ListOptions{Limit: 2000}, 1000, 0},
		{"valid values preserved", ListOptions{Limit: 50, Offset: 10}, 50, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.input.Defaults()
			if got.Limit != tt.wantLimit || got.Offset != tt.wantOffset {
				t.Errorf("Defaults() = {%d, %d}, want {%d, %d}",
					got.Limit, got.Offset, tt.wantLimit, tt.wantOffset)
			}
		})
	}
}

func TestStoreListUsesDefaults(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	// Create 5 jobs
	for i := 0; i < 5; i++ {
		job := model.Job{
			ID:        fmt.Sprintf("j%d", i),
			Kind:      model.KindScrape,
			Status:    model.StatusQueued,
			CreatedAt: time.Now().Add(time.Duration(i) * time.Second),
			UpdatedAt: time.Now(),
			Params:    map[string]interface{}{"idx": i},
		}
		if err := s.Create(ctx, job); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// Test that List() uses default limit of 100 (all 5 jobs should be returned)
	jobs, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(jobs) != 5 {
		t.Errorf("expected 5 jobs with List(), got %d", len(jobs))
	}
}
