// Package store provides tests for job listing and counting operations.
// Tests cover ListOpts, ListByStatus, CountJobs, and pagination option defaults.
// Does NOT test job CRUD, crawl states, or database migrations.
package store

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestStoreListOptsPagination(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	for i := 0; i < 25; i++ {
		job := model.Job{
			ID:        fmt.Sprintf("j%02d", i),
			Kind:      model.KindScrape,
			Status:    model.StatusQueued,
			CreatedAt: time.Now().Add(time.Duration(i) * time.Second),
			UpdatedAt: time.Now(),
			Spec:      map[string]interface{}{"idx": i},
		}
		if err := s.Create(ctx, job); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	opts := ListOptions{Limit: 10, Offset: 0}
	jobs, err := s.ListOpts(ctx, opts)
	if err != nil {
		t.Fatalf("ListOpts failed: %v", err)
	}
	if len(jobs) != 10 {
		t.Errorf("expected 10 jobs with limit=10, got %d", len(jobs))
	}

	opts = ListOptions{Limit: 10, Offset: 10}
	jobs, err = s.ListOpts(ctx, opts)
	if err != nil {
		t.Fatalf("ListOpts failed: %v", err)
	}
	if len(jobs) != 10 {
		t.Errorf("expected 10 jobs with offset=10, got %d", len(jobs))
	}

	opts = ListOptions{Limit: 5, Offset: 0}
	jobs, err = s.ListOpts(ctx, opts)
	if err != nil {
		t.Fatalf("ListOpts failed: %v", err)
	}
	if jobs[0].ID != "j24" {
		t.Errorf("expected first job to be j24, got %s", jobs[0].ID)
	}

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

func TestStoreListByStatus(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	queuedJob := model.Job{
		ID:        "j1",
		Kind:      model.KindScrape,
		Status:    model.StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Spec:      map[string]interface{}{"url": "http://example.com/1"},
	}

	runningJob := model.Job{
		ID:        "j2",
		Kind:      model.KindScrape,
		Status:    model.StatusRunning,
		CreatedAt: time.Now().Add(-1 * time.Second),
		UpdatedAt: time.Now(),
		Spec:      map[string]interface{}{"url": "http://example.com/2"},
	}

	succeededJob := model.Job{
		ID:        "j3",
		Kind:      model.KindCrawl,
		Status:    model.StatusSucceeded,
		CreatedAt: time.Now().Add(-2 * time.Second),
		UpdatedAt: time.Now(),
		Spec:      map[string]interface{}{"url": "http://example.com/3"},
	}

	if err := s.Create(ctx, queuedJob); err != nil {
		t.Fatalf("failed to create queued job: %v", err)
	}
	if err := s.Create(ctx, runningJob); err != nil {
		t.Fatalf("failed to create running job: %v", err)
	}
	if err := s.Create(ctx, succeededJob); err != nil {
		t.Fatalf("failed to create succeeded job: %v", err)
	}

	queued, err := s.ListByStatus(ctx, model.StatusQueued, ListByStatusOptions{})
	if err != nil {
		t.Fatalf("ListByStatus failed: %v", err)
	}

	if len(queued) != 1 {
		t.Errorf("expected 1 queued job, got %d", len(queued))
	}
	if len(queued) > 0 && queued[0].ID != "j1" {
		t.Errorf("expected job j1, got %s", queued[0].ID)
	}

	running, err := s.ListByStatus(ctx, model.StatusRunning, ListByStatusOptions{})
	if err != nil {
		t.Fatalf("ListByStatus failed: %v", err)
	}

	if len(running) != 1 {
		t.Errorf("expected 1 running job, got %d", len(running))
	}
	if len(running) > 0 && running[0].ID != "j2" {
		t.Errorf("expected job j2, got %s", running[0].ID)
	}

	succeeded, err := s.ListByStatus(ctx, model.StatusSucceeded, ListByStatusOptions{})
	if err != nil {
		t.Fatalf("ListByStatus failed: %v", err)
	}

	if len(succeeded) != 1 {
		t.Errorf("expected 1 succeeded job, got %d", len(succeeded))
	}
	if len(succeeded) > 0 && succeeded[0].ID != "j3" {
		t.Errorf("expected job j3, got %s", succeeded[0].ID)
	}

	failed, err := s.ListByStatus(ctx, model.StatusFailed, ListByStatusOptions{})
	if err != nil {
		t.Fatalf("ListByStatus failed: %v", err)
	}

	if len(failed) != 0 {
		t.Errorf("expected 0 failed jobs, got %d", len(failed))
	}
}

func TestListByStatusOptionsDefaults(t *testing.T) {
	tests := []struct {
		name       string
		input      ListByStatusOptions
		wantLimit  int
		wantOffset int
	}{
		{"zero values use defaults", ListByStatusOptions{}, 100, 0},
		{"negative limit uses default", ListByStatusOptions{Limit: -1}, 100, 0},
		{"negative offset uses zero", ListByStatusOptions{Offset: -5}, 100, 0},
		{"max limit capped", ListByStatusOptions{Limit: 2000}, 1000, 0},
		{"valid values preserved", ListByStatusOptions{Limit: 50, Offset: 10}, 50, 10},
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

func TestStoreCountJobs(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	_ = s.Create(ctx, model.Job{ID: "j1", Kind: model.KindScrape, Status: model.StatusQueued, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	_ = s.Create(ctx, model.Job{ID: "j2", Kind: model.KindScrape, Status: model.StatusRunning, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	_ = s.Create(ctx, model.Job{ID: "j3", Kind: model.KindScrape, Status: model.StatusSucceeded, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	_ = s.Create(ctx, model.Job{ID: "j4", Kind: model.KindScrape, Status: model.StatusQueued, CreatedAt: time.Now(), UpdatedAt: time.Now()})

	count, err := s.CountJobs(ctx, "")
	if err != nil {
		t.Fatalf("CountJobs failed: %v", err)
	}
	if count != 4 {
		t.Errorf("expected 4 jobs total, got %d", count)
	}

	count, err = s.CountJobs(ctx, model.StatusQueued)
	if err != nil {
		t.Fatalf("CountJobs(queued) failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 queued jobs, got %d", count)
	}

	count, err = s.CountJobs(ctx, model.StatusFailed)
	if err != nil {
		t.Fatalf("CountJobs(failed) failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 failed jobs, got %d", count)
	}
}
