// Package store provides tests for dependency-related storage operations.
package store

import (
	"context"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestJobWithDependencies(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	// Create a job with dependencies
	job := model.Job{
		ID:               "job-with-deps",
		Kind:             model.KindScrape,
		Status:           model.StatusQueued,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		Params:           map[string]interface{}{"url": "http://example.com"},
		DependsOn:        []string{"dep-job-1", "dep-job-2"},
		DependencyStatus: model.DependencyStatusPending,
		ChainID:          "test-chain",
	}

	err = s.Create(ctx, job)
	if err != nil {
		t.Fatalf("Create job failed: %v", err)
	}

	// Retrieve and verify
	got, err := s.Get(ctx, job.ID)
	if err != nil {
		t.Fatalf("Get job failed: %v", err)
	}

	if len(got.DependsOn) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(got.DependsOn))
	}
	if got.DependencyStatus != model.DependencyStatusPending {
		t.Errorf("Expected dependency status pending, got %s", got.DependencyStatus)
	}
	if got.ChainID != "test-chain" {
		t.Errorf("Expected chain ID test-chain, got %s", got.ChainID)
	}
}

func TestUpdateDependencyStatus(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	job := model.Job{
		ID:               "job-dep-status",
		Kind:             model.KindScrape,
		Status:           model.StatusQueued,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		Params:           map[string]interface{}{},
		DependsOn:        []string{"dep1"},
		DependencyStatus: model.DependencyStatusPending,
	}

	err = s.Create(ctx, job)
	if err != nil {
		t.Fatalf("Create job failed: %v", err)
	}

	// Update dependency status
	err = s.UpdateDependencyStatus(ctx, job.ID, model.DependencyStatusReady)
	if err != nil {
		t.Fatalf("UpdateDependencyStatus failed: %v", err)
	}

	// Verify
	got, err := s.Get(ctx, job.ID)
	if err != nil {
		t.Fatalf("Get job failed: %v", err)
	}

	if got.DependencyStatus != model.DependencyStatusReady {
		t.Errorf("Expected dependency status ready, got %s", got.DependencyStatus)
	}
}

func TestGetJobsByDependencyStatus(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	// Create jobs with different dependency statuses
	jobs := []model.Job{
		{
			ID:               "job-pending-1",
			Kind:             model.KindScrape,
			Status:           model.StatusQueued,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
			Params:           map[string]interface{}{},
			DependsOn:        []string{"dep1"},
			DependencyStatus: model.DependencyStatusPending,
		},
		{
			ID:               "job-pending-2",
			Kind:             model.KindCrawl,
			Status:           model.StatusQueued,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
			Params:           map[string]interface{}{},
			DependsOn:        []string{"dep2"},
			DependencyStatus: model.DependencyStatusPending,
		},
		{
			ID:               "job-ready",
			Kind:             model.KindScrape,
			Status:           model.StatusQueued,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
			Params:           map[string]interface{}{},
			DependsOn:        []string{},
			DependencyStatus: model.DependencyStatusReady,
		},
	}

	for _, job := range jobs {
		err := s.Create(ctx, job)
		if err != nil {
			t.Fatalf("Create job failed: %v", err)
		}
	}

	// Get pending jobs
	pending, err := s.GetJobsByDependencyStatus(ctx, model.DependencyStatusPending)
	if err != nil {
		t.Fatalf("GetJobsByDependencyStatus failed: %v", err)
	}

	if len(pending) != 2 {
		t.Errorf("Expected 2 pending jobs, got %d", len(pending))
	}

	// Get ready jobs
	ready, err := s.GetJobsByDependencyStatus(ctx, model.DependencyStatusReady)
	if err != nil {
		t.Fatalf("GetJobsByDependencyStatus failed: %v", err)
	}

	if len(ready) != 1 {
		t.Errorf("Expected 1 ready job, got %d", len(ready))
	}
}

func TestGetDependentJobs(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	// Create dependency jobs first
	depJob := model.Job{
		ID:        "dep-job",
		Kind:      model.KindScrape,
		Status:    model.StatusSucceeded,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Params:    map[string]interface{}{},
	}

	err = s.Create(ctx, depJob)
	if err != nil {
		t.Fatalf("Create dep job failed: %v", err)
	}

	// Create jobs that depend on dep-job
	dependentJobs := []model.Job{
		{
			ID:               "dependent-1",
			Kind:             model.KindScrape,
			Status:           model.StatusQueued,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
			Params:           map[string]interface{}{},
			DependsOn:        []string{"dep-job"},
			DependencyStatus: model.DependencyStatusPending,
		},
		{
			ID:               "dependent-2",
			Kind:             model.KindCrawl,
			Status:           model.StatusQueued,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
			Params:           map[string]interface{}{},
			DependsOn:        []string{"dep-job", "other-dep"},
			DependencyStatus: model.DependencyStatusPending,
		},
		{
			ID:               "independent",
			Kind:             model.KindScrape,
			Status:           model.StatusQueued,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
			Params:           map[string]interface{}{},
			DependsOn:        []string{"other-dep"},
			DependencyStatus: model.DependencyStatusPending,
		},
	}

	for _, job := range dependentJobs {
		err := s.Create(ctx, job)
		if err != nil {
			t.Fatalf("Create job failed: %v", err)
		}
	}

	// Get jobs that depend on dep-job
	dependents, err := s.GetDependentJobs(ctx, "dep-job")
	if err != nil {
		t.Fatalf("GetDependentJobs failed: %v", err)
	}

	if len(dependents) != 2 {
		t.Errorf("Expected 2 dependent jobs, got %d", len(dependents))
	}

	// Verify the correct jobs are returned
	ids := make(map[string]bool)
	for _, d := range dependents {
		ids[d.ID] = true
	}
	if !ids["dependent-1"] {
		t.Error("Expected dependent-1 in results")
	}
	if !ids["dependent-2"] {
		t.Error("Expected dependent-2 in results")
	}
	if ids["independent"] {
		t.Error("Did not expect independent in results")
	}
}

func TestGetJobsByChain(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	// Create jobs in different chains
	jobs := []model.Job{
		{
			ID:        "chain1-job1",
			Kind:      model.KindScrape,
			Status:    model.StatusQueued,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Params:    map[string]interface{}{},
			ChainID:   "chain-1",
		},
		{
			ID:        "chain1-job2",
			Kind:      model.KindCrawl,
			Status:    model.StatusQueued,
			CreatedAt: time.Now().Add(time.Second),
			UpdatedAt: time.Now().Add(time.Second),
			Params:    map[string]interface{}{},
			ChainID:   "chain-1",
		},
		{
			ID:        "chain2-job1",
			Kind:      model.KindScrape,
			Status:    model.StatusQueued,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Params:    map[string]interface{}{},
			ChainID:   "chain-2",
		},
		{
			ID:        "no-chain-job",
			Kind:      model.KindScrape,
			Status:    model.StatusQueued,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Params:    map[string]interface{}{},
			ChainID:   "",
		},
	}

	for _, job := range jobs {
		err := s.Create(ctx, job)
		if err != nil {
			t.Fatalf("Create job failed: %v", err)
		}
	}

	// Get jobs by chain
	chain1Jobs, err := s.GetJobsByChain(ctx, "chain-1")
	if err != nil {
		t.Fatalf("GetJobsByChain failed: %v", err)
	}

	if len(chain1Jobs) != 2 {
		t.Errorf("Expected 2 jobs in chain-1, got %d", len(chain1Jobs))
	}

	// Verify ordering (by created_at ASC)
	if len(chain1Jobs) == 2 {
		if chain1Jobs[0].ID != "chain1-job1" {
			t.Errorf("Expected first job to be chain1-job1, got %s", chain1Jobs[0].ID)
		}
		if chain1Jobs[1].ID != "chain1-job2" {
			t.Errorf("Expected second job to be chain1-job2, got %s", chain1Jobs[1].ID)
		}
	}

	// Get jobs from chain-2
	chain2Jobs, err := s.GetJobsByChain(ctx, "chain-2")
	if err != nil {
		t.Fatalf("GetJobsByChain failed: %v", err)
	}

	if len(chain2Jobs) != 1 {
		t.Errorf("Expected 1 job in chain-2, got %d", len(chain2Jobs))
	}

	// Get jobs from non-existent chain
	emptyJobs, err := s.GetJobsByChain(ctx, "non-existent")
	if err != nil {
		t.Fatalf("GetJobsByChain failed: %v", err)
	}

	if len(emptyJobs) != 0 {
		t.Errorf("Expected 0 jobs in non-existent chain, got %d", len(emptyJobs))
	}
}

func TestJobDefaultDependencyStatus(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	// Job without dependencies should default to ready
	jobNoDeps := model.Job{
		ID:        "job-no-deps",
		Kind:      model.KindScrape,
		Status:    model.StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Params:    map[string]interface{}{},
		DependsOn: []string{},
		// DependencyStatus not set
	}

	err = s.Create(ctx, jobNoDeps)
	if err != nil {
		t.Fatalf("Create job failed: %v", err)
	}

	got, err := s.Get(ctx, jobNoDeps.ID)
	if err != nil {
		t.Fatalf("Get job failed: %v", err)
	}

	if got.DependencyStatus != model.DependencyStatusReady {
		t.Errorf("Expected default dependency status ready for job without deps, got %s", got.DependencyStatus)
	}
}
