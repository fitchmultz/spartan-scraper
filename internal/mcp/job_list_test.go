// Package mcp provides tests for the job_list and job_failure_list MCP tools.
//
// Purpose:
// - Verify run-history listing parity for MCP job inspection tools.
//
// Responsibilities:
// - Assert tool schemas expose pagination and filtering correctly.
// - Assert job list responses return canonical paginated envelopes.
// - Assert failed-job inspection returns only failed jobs with derived failure context.
//
// Scope:
// - MCP tool metadata and handler behavior only; job execution internals are not under test here.
//
// Usage:
// - Run with `go test ./internal/mcp/...`.
//
// Invariants/Assumptions:
// - MCP job list tools mirror the REST job-list contracts.
// - Failed-job inspection is a filtered view over persisted job state.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/api"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

func TestJobListToolInToolsList(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	tools := srv.toolsList()
	toolMap := make(map[string]tool)
	for _, candidate := range tools {
		toolMap[candidate.Name] = candidate
	}
	jobListTool, ok := toolMap["job_list"]
	if !ok {
		t.Fatal("job_list tool not found in toolsList")
	}
	if jobListTool.Description != "List recent job run envelopes with pagination metadata and optional status filtering" {
		t.Errorf("unexpected description: %q", jobListTool.Description)
	}
	schema := jobListTool.InputSchema
	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("properties not found in schema")
	}
	for _, key := range []string{"limit", "offset", "status"} {
		if _, ok := props[key]; !ok {
			t.Errorf("expected %q in properties", key)
		}
	}
	required := schema["required"]
	if len(required.([]string)) != 0 {
		t.Error("expected no required fields")
	}
}

func TestJobFailureListToolInToolsList(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	tools := srv.toolsList()
	toolMap := make(map[string]tool)
	for _, candidate := range tools {
		toolMap[candidate.Name] = candidate
	}
	jobFailureTool, ok := toolMap["job_failure_list"]
	if !ok {
		t.Fatal("job_failure_list tool not found in toolsList")
	}
	if jobFailureTool.Description != "List recent failed job runs with derived failure context" {
		t.Errorf("unexpected description: %q", jobFailureTool.Description)
	}
}

func TestHandleJobList(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	ctx := context.Background()

	t.Run("list all jobs", func(t *testing.T) {
		_, err := srv.manager.CreateScrapeJob(ctx, "http://example.com/1", "GET", nil, "", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false, "", "", nil, "")
		if err != nil {
			t.Fatalf("CreateScrapeJob failed: %v", err)
		}
		_, err = srv.manager.CreateScrapeJob(ctx, "http://example.com/2", "GET", nil, "", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false, "", "", nil, "")
		if err != nil {
			t.Fatalf("CreateScrapeJob failed: %v", err)
		}

		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name":      "job_list",
				"arguments": map[string]interface{}{},
			}),
		}

		result, err := srv.handleToolCall(ctx, base)
		if err != nil {
			t.Fatalf("handleToolCall failed: %v", err)
		}

		response, ok := result.(api.JobListResponse)
		if !ok {
			t.Fatalf("result is not an api.JobListResponse: %T", result)
		}
		if len(response.Jobs) != 2 {
			t.Errorf("expected 2 jobs, got %d", len(response.Jobs))
		}
		if response.Total != 2 {
			t.Errorf("expected total 2, got %d", response.Total)
		}
		if response.Jobs[0].Run.TotalMs < 0 {
			t.Errorf("expected non-negative total runtime, got %d", response.Jobs[0].Run.TotalMs)
		}
	})

	t.Run("list with limit and offset", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			_, err := srv.manager.CreateScrapeJob(ctx, fmt.Sprintf("http://example.com/%d", i), "GET", nil, "", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false, "", "", nil, "")
			if err != nil {
				t.Fatalf("CreateScrapeJob failed: %v", err)
			}
		}

		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name":      "job_list",
				"arguments": map[string]interface{}{"limit": 2, "offset": 2},
			}),
		}

		result, err := srv.handleToolCall(ctx, base)
		if err != nil {
			t.Fatalf("handleToolCall failed: %v", err)
		}

		response, ok := result.(api.JobListResponse)
		if !ok {
			t.Fatalf("result is not an api.JobListResponse: %T", result)
		}
		if len(response.Jobs) != 2 {
			t.Errorf("expected 2 jobs (offset 2, limit 2), got %d", len(response.Jobs))
		}
		if response.Total != 7 {
			t.Errorf("expected total 7, got %d", response.Total)
		}
		if response.Limit != 2 || response.Offset != 2 {
			t.Errorf("expected limit/offset 2/2, got %d/%d", response.Limit, response.Offset)
		}
	})

	t.Run("filter by status", func(t *testing.T) {
		failedJob := model.Job{
			ID:          "failed-job",
			Kind:        model.KindScrape,
			Status:      model.StatusFailed,
			SpecVersion: -1,
			CreatedAt:   time.Now().Add(-2 * time.Minute),
			UpdatedAt:   time.Now().Add(-time.Minute),
			FinishedAt:  ptrTime(time.Now().Add(-time.Minute)),
			Error:       "request timeout",
			Spec:        map[string]interface{}{"url": "https://failed.example.com"},
		}
		if err := srv.store.Create(ctx, failedJob); err != nil {
			t.Fatalf("failed to create failed job: %v", err)
		}

		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name":      "job_list",
				"arguments": map[string]interface{}{"status": "failed"},
			}),
		}

		result, err := srv.handleToolCall(ctx, base)
		if err != nil {
			t.Fatalf("handleToolCall failed: %v", err)
		}

		response := result.(api.JobListResponse)
		for _, job := range response.Jobs {
			if job.Status != model.StatusFailed {
				t.Fatalf("expected only failed jobs, got %s", job.Status)
			}
		}
	})
}

func TestHandleJobFailureList(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	ctx := context.Background()
	failedJob := model.Job{
		ID:          "failed-job",
		Kind:        model.KindScrape,
		Status:      model.StatusFailed,
		SpecVersion: -1,
		CreatedAt:   time.Now().Add(-2 * time.Minute),
		UpdatedAt:   time.Now().Add(-time.Minute),
		FinishedAt:  ptrTime(time.Now().Add(-time.Minute)),
		Error:       "browser timeout while waiting for selector",
		Spec:        map[string]interface{}{"url": "https://failed.example.com"},
	}
	if err := srv.store.Create(ctx, failedJob); err != nil {
		t.Fatalf("failed to create failed job: %v", err)
	}
	queuedJob := model.Job{
		ID:          "queued-job",
		Kind:        model.KindScrape,
		Status:      model.StatusQueued,
		SpecVersion: -1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Spec:        map[string]interface{}{"url": "https://queued.example.com"},
	}
	if err := srv.store.Create(ctx, queuedJob); err != nil {
		t.Fatalf("failed to create queued job: %v", err)
	}

	base := map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name":      "job_failure_list",
			"arguments": map[string]interface{}{"limit": 10, "offset": 0},
		}),
	}

	result, err := srv.handleToolCall(ctx, base)
	if err != nil {
		t.Fatalf("handleToolCall failed: %v", err)
	}

	response, ok := result.(api.JobListResponse)
	if !ok {
		t.Fatalf("result is not an api.JobListResponse: %T", result)
	}
	if len(response.Jobs) != 1 {
		t.Fatalf("expected 1 failed job, got %d", len(response.Jobs))
	}
	if response.Jobs[0].ID != failedJob.ID {
		t.Fatalf("expected failed job %q, got %q", failedJob.ID, response.Jobs[0].ID)
	}
	if response.Jobs[0].Run.Failure == nil {
		t.Fatal("expected failure context on failed job")
	}
	if response.Jobs[0].Run.Failure.Category != "timeout" {
		t.Fatalf("expected timeout classification, got %q", response.Jobs[0].Run.Failure.Category)
	}
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
