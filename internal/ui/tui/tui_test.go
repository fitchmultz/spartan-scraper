package tui

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
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
		fetch.DefaultCircuitBreakerConfig(),
		nil, // no adaptive rate limiting in tests
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

// Test vim-style navigation
func TestTUIVimNavigation(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer st.Close()

	// Create test jobs
	for i := 0; i < 5; i++ {
		job := model.Job{
			ID:         fmt.Sprintf("job-%d", i),
			Kind:       model.KindScrape,
			Status:     model.StatusSucceeded,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			Params:     map[string]interface{}{},
			ResultPath: "",
		}
		if err := st.Create(ctx, job); err != nil {
			t.Fatalf("failed to create job: %v", err)
		}
	}

	// Fetch jobs to populate model
	msg := fetchJobs(ctx, st, nil, 20, 0)()
	jobsMsg, ok := msg.(jobsMsg)
	if !ok {
		t.Fatalf("expected jobsMsg, got %T", msg)
	}

	// Test model with vim navigation
	m := appModel{
		ctx:       ctx,
		store:     st,
		tab:       "jobs",
		pageLimit: 20,
		cursor:    0,
		jobs:      jobsMsg.jobs,
		fullJobs:  jobsMsg.fullJobs,
		width:     100,
		height:    30,
	}

	// Simulate 'j' key (down)
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if newModel, ok := newM.(appModel); ok {
		if newModel.cursor != 1 {
			t.Errorf("expected cursor to move to 1 with 'j', got %d", newModel.cursor)
		}
	}

	// Simulate 'k' key (up)
	m.cursor = 2
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if newModel, ok := newM.(appModel); ok {
		if newModel.cursor != 1 {
			t.Errorf("expected cursor to move to 1 with 'k', got %d", newModel.cursor)
		}
	}
}

// Test help modal toggle
func TestTUIHelpModal(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer st.Close()

	m := appModel{
		ctx:       ctx,
		store:     st,
		tab:       "jobs",
		pageLimit: 20,
		showHelp:  false,
		width:     100,
		height:    30,
	}

	// Toggle help on with '?'
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if newModel, ok := newM.(appModel); ok {
		if !newModel.showHelp {
			t.Error("expected help modal to be shown after '?' key")
		}
	}

	// Toggle help off with '?'
	m.showHelp = true
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if newModel, ok := newM.(appModel); ok {
		if newModel.showHelp {
			t.Error("expected help modal to be hidden after second '?' key")
		}
	}
}

// Test status badge rendering
func TestStatusBadgeRendering(t *testing.T) {
	tests := []struct {
		status model.Status
		want   string // Should contain the status name
	}{
		{model.StatusQueued, "queued"},
		{model.StatusRunning, "running"},
		{model.StatusSucceeded, "succeeded"},
		{model.StatusFailed, "failed"},
		{model.StatusCanceled, "canceled"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			badge := RenderStatusBadge(tt.status)
			if !strings.Contains(strings.ToLower(badge), tt.want) {
				t.Errorf("RenderStatusBadge(%s) = %s, should contain %s", tt.status, badge, tt.want)
			}
		})
	}
}
