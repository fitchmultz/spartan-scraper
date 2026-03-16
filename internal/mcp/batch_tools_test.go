// Package mcp provides tests for MCP batch lifecycle tools.
//
// Purpose:
// - Verify MCP exposes batch submission, listing, detail, and cancellation tools with stable envelopes.
//
// Responsibilities:
// - Assert tool metadata stays aligned with documented batch contracts.
// - Assert batch create/list/status/cancel calls return canonical API envelopes.
//
// Scope:
// - MCP batch tool behavior only; batch execution internals are covered elsewhere.
//
// Usage:
// - Run with `go test ./internal/mcp`.
//
// Invariants/Assumptions:
// - Batch MCP tools mirror REST batch response shaping.
// - Included job pages remain sanitized.
package mcp

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/api"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestBatchToolsInToolsList(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	tools := srv.toolsList()
	toolMap := make(map[string]tool)
	for _, entry := range tools {
		toolMap[entry.Name] = entry
	}

	expected := map[string]string{
		"batch_scrape":   "Create a batch of scrape jobs using the same request contract as POST /v1/jobs/batch/scrape",
		"batch_crawl":    "Create a batch of crawl jobs using the same request contract as POST /v1/jobs/batch/crawl",
		"batch_research": "Create a batch of research jobs using the same request contract as POST /v1/jobs/batch/research",
		"batch_list":     "List batch summaries with pagination metadata",
		"batch_status":   "Get a batch envelope by id with optional included jobs",
		"batch_cancel":   "Cancel a batch and return the updated batch envelope",
	}

	for name, description := range expected {
		entry, ok := toolMap[name]
		if !ok {
			t.Fatalf("%s tool not found in toolsList", name)
		}
		if entry.Description != description {
			t.Fatalf("unexpected %s description: %q", name, entry.Description)
		}
	}
}

func TestHandleBatchLifecycleTools(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	ctx := context.Background()

	createBase := map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]any{
			"name": "batch_scrape",
			"arguments": map[string]any{
				"jobs": []map[string]any{{"url": "http://example.com/1"}, {"url": "http://example.com/2"}},
			},
		}),
	}
	result, err := srv.handleToolCall(ctx, createBase)
	if err != nil {
		t.Fatalf("batch_scrape failed: %v", err)
	}
	createResp, ok := result.(api.BatchResponse)
	if !ok {
		t.Fatalf("expected api.BatchResponse, got %T", result)
	}
	if createResp.Batch.JobCount != 2 || len(createResp.Jobs) != 2 {
		t.Fatalf("expected 2 created jobs, got count=%d jobs=%d", createResp.Batch.JobCount, len(createResp.Jobs))
	}

	listBase := map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]any{
			"name": "batch_list",
			"arguments": map[string]any{
				"limit":  10,
				"offset": 0,
			},
		}),
	}
	result, err = srv.handleToolCall(ctx, listBase)
	if err != nil {
		t.Fatalf("batch_list failed: %v", err)
	}
	listResp, ok := result.(api.BatchListResponse)
	if !ok {
		t.Fatalf("expected api.BatchListResponse, got %T", result)
	}
	if listResp.Total != 1 || listResp.Limit != 10 || listResp.Offset != 0 {
		t.Fatalf("unexpected batch list pagination: %+v", listResp)
	}
	if len(listResp.Batches) != 1 || listResp.Batches[0].ID != createResp.Batch.ID {
		t.Fatalf("expected listed batch %s, got %+v", createResp.Batch.ID, listResp.Batches)
	}
	if listResp.Batches[0].Stats.Queued+listResp.Batches[0].Stats.Running != 2 {
		t.Fatalf("expected active-job stats in batch list, got %+v", listResp.Batches[0].Stats)
	}

	statusBase := map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]any{
			"name": "batch_status",
			"arguments": map[string]any{
				"id":          createResp.Batch.ID,
				"includeJobs": true,
				"limit":       2,
				"offset":      0,
			},
		}),
	}
	result, err = srv.handleToolCall(ctx, statusBase)
	if err != nil {
		t.Fatalf("batch_status failed: %v", err)
	}
	statusResp, ok := result.(api.BatchResponse)
	if !ok {
		t.Fatalf("expected api.BatchResponse, got %T", result)
	}
	if statusResp.Batch.ID != createResp.Batch.ID {
		t.Fatalf("expected batch id %s, got %s", createResp.Batch.ID, statusResp.Batch.ID)
	}
	if statusResp.Batch.JobCount != 2 || len(statusResp.Jobs) != 2 {
		t.Fatalf("expected 2 jobs in batch response, got count=%d jobs=%d", statusResp.Batch.JobCount, len(statusResp.Jobs))
	}
	for _, job := range statusResp.Jobs {
		if job.ResultPath != "" {
			t.Fatalf("expected sanitized batch job without result path, got %q", job.ResultPath)
		}
	}

	cancelBase := map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]any{
			"name": "batch_cancel",
			"arguments": map[string]any{
				"id": createResp.Batch.ID,
			},
		}),
	}
	result, err = srv.handleToolCall(ctx, cancelBase)
	if err != nil {
		t.Fatalf("batch_cancel failed: %v", err)
	}
	cancelResp, ok := result.(api.BatchResponse)
	if !ok {
		t.Fatalf("expected api.BatchResponse, got %T", result)
	}
	if cancelResp.Batch.ID != createResp.Batch.ID {
		t.Fatalf("expected batch id %s, got %s", createResp.Batch.ID, cancelResp.Batch.ID)
	}
	if cancelResp.Batch.Stats.Canceled == 0 {
		t.Fatalf("expected canceled jobs in batch stats, got %+v", cancelResp.Batch.Stats)
	}

	for _, created := range createResp.Jobs {
		stored, err := srv.store.Get(ctx, created.ID)
		if err != nil {
			t.Fatalf("failed to load canceled job %s: %v", created.ID, err)
		}
		if stored.Status != model.StatusCanceled {
			t.Fatalf("expected job %s to be canceled, got %s", stored.ID, stored.Status)
		}
	}
}

func TestHandleBatchResearchToolCreatesSingleJob(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	ctx := context.Background()
	result, err := srv.handleToolCall(ctx, map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]any{
			"name": "batch_research",
			"arguments": map[string]any{
				"query": "pricing model",
				"jobs":  []map[string]any{{"url": "http://example.com/1"}, {"url": "http://example.com/2"}},
			},
		}),
	})
	if err != nil {
		t.Fatalf("batch_research failed: %v", err)
	}
	resp, ok := result.(api.BatchResponse)
	if !ok {
		t.Fatalf("expected api.BatchResponse, got %T", result)
	}
	if resp.Batch.Kind != string(model.KindResearch) {
		t.Fatalf("expected research batch kind, got %s", resp.Batch.Kind)
	}
	if resp.Batch.JobCount != 1 || len(resp.Jobs) != 1 {
		t.Fatalf("expected one persisted research job, got count=%d jobs=%d", resp.Batch.JobCount, len(resp.Jobs))
	}
}
