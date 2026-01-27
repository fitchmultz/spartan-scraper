package tui

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

func TestTUIPagination(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer st.Close()

	for i := 0; i < 50; i++ {
		job := model.Job{
			ID:         fmt.Sprintf("test-job-%d", i),
			Kind:       model.KindScrape,
			Status:     model.StatusSucceeded,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			Params:     map[string]interface{}{"url": fmt.Sprintf("https://example.com/%d", i)},
			ResultPath: "",
		}
		if err := st.Create(ctx, job); err != nil {
			t.Fatalf("failed to create job: %v", err)
		}
	}

	opts := store.ListOptions{Limit: 20, Offset: 0}
	jobsList, err := st.ListOpts(ctx, opts)
	if err != nil {
		t.Fatalf("failed to list jobs: %v", err)
	}
	if len(jobsList) != 20 {
		t.Errorf("expected 20 jobs on first page, got %d", len(jobsList))
	}

	opts = store.ListOptions{Limit: 20, Offset: 20}
	jobsList, err = st.ListOpts(ctx, opts)
	if err != nil {
		t.Fatalf("failed to list jobs: %v", err)
	}
	if len(jobsList) != 20 {
		t.Errorf("expected 20 jobs on second page, got %d", len(jobsList))
	}

	opts = store.ListOptions{Limit: 20, Offset: 100}
	jobsList, err = st.ListOpts(ctx, opts)
	if err != nil {
		t.Fatalf("failed to list jobs: %v", err)
	}
	if len(jobsList) != 0 {
		t.Errorf("expected 0 jobs beyond available, got %d", len(jobsList))
	}
}

func TestTUIJobSelection(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer st.Close()

	jobID := "test-selection-job"
	job := model.Job{
		ID:         jobID,
		Kind:       model.KindScrape,
		Status:     model.StatusRunning,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Params:     map[string]interface{}{"url": "https://example.com"},
		ResultPath: "",
	}
	if err := st.Create(ctx, job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	msg := fetchJobDetail(ctx, st, jobID)()
	detailMsg, ok := msg.(jobDetailMsg)
	if !ok {
		t.Fatalf("expected jobDetailMsg, got %T", msg)
	}
	if detailMsg.err != nil {
		t.Fatalf("failed to fetch job detail: %v", detailMsg.err)
	}
	if detailMsg.job.ID != jobID {
		t.Errorf("expected job ID %s, got %s", jobID, detailMsg.job.ID)
	}
	if detailMsg.job.Kind != model.KindScrape {
		t.Errorf("expected kind scrape, got %s", detailMsg.job.Kind)
	}
	if detailMsg.job.Status != model.StatusRunning {
		t.Errorf("expected status running, got %s", detailMsg.job.Status)
	}
}

func TestTUICancelJob(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer st.Close()

	manager := jobs.NewManager(
		st,
		dataDir,
		"SpartanScraper/0.1",
		30*time.Second,
		4,
		2,
		4,
		2,
		400*time.Millisecond,
		10*1024*1024,
		false,
	)

	jobID := "test-cancel-job"
	job := model.Job{
		ID:         jobID,
		Kind:       model.KindScrape,
		Status:     model.StatusQueued,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Params:     map[string]interface{}{"url": "https://example.com"},
		ResultPath: "",
	}
	if err := st.Create(ctx, job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	err = manager.CancelJob(ctx, jobID)
	if err != nil {
		t.Fatalf("CancelJob should not fail, got: %v", err)
	}

	updated, err := st.Get(ctx, jobID)
	if err != nil {
		t.Fatalf("failed to get job: %v", err)
	}
	if updated.Status != model.StatusCanceled {
		t.Errorf("expected status canceled, got %s", updated.Status)
	}
}
