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
