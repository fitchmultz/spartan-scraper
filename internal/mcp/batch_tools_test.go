// Package mcp provides tests for batch_status and batch_cancel MCP tools.
//
// Purpose:
// - Verify MCP exposes stable batch envelopes for status inspection and cancellation.
//
// Responsibilities:
// - Assert tool metadata stays aligned with the documented batch contract.
// - Assert batch status and cancel calls return canonical api.BatchResponse envelopes.
//
// Scope:
// - MCP tool behavior only; batch execution internals are covered elsewhere.
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
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
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

	batchStatusTool, ok := toolMap["batch_status"]
	if !ok {
		t.Fatal("batch_status tool not found in toolsList")
	}
	if batchStatusTool.Description != "Get a batch envelope by id with optional included jobs" {
		t.Fatalf("unexpected batch_status description: %q", batchStatusTool.Description)
	}

	batchCancelTool, ok := toolMap["batch_cancel"]
	if !ok {
		t.Fatal("batch_cancel tool not found in toolsList")
	}
	if batchCancelTool.Description != "Cancel a batch and return the updated batch envelope" {
		t.Fatalf("unexpected batch_cancel description: %q", batchCancelTool.Description)
	}
}

func TestHandleBatchStatusAndCancel(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	ctx := context.Background()
	jobsCreated, err := srv.manager.CreateBatchJobs(ctx, model.KindScrape, []jobs.JobSpec{
		{Kind: model.KindScrape, URL: "http://example.com/1", Method: "GET", TimeoutSeconds: 30},
		{Kind: model.KindScrape, URL: "http://example.com/2", Method: "GET", TimeoutSeconds: 30},
	}, "batch-mcp-1")
	if err != nil {
		t.Fatalf("CreateBatchJobs failed: %v", err)
	}

	statusBase := map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]any{
			"name": "batch_status",
			"arguments": map[string]any{
				"id":          "batch-mcp-1",
				"includeJobs": true,
				"limit":       2,
				"offset":      0,
			},
		}),
	}
	result, err := srv.handleToolCall(ctx, statusBase)
	if err != nil {
		t.Fatalf("batch_status failed: %v", err)
	}
	statusResp, ok := result.(api.BatchResponse)
	if !ok {
		t.Fatalf("expected api.BatchResponse, got %T", result)
	}
	if statusResp.Batch.ID != "batch-mcp-1" {
		t.Fatalf("expected batch id batch-mcp-1, got %s", statusResp.Batch.ID)
	}
	if statusResp.Batch.JobCount != 2 || len(statusResp.Jobs) != 2 {
		t.Fatalf("expected 2 jobs in batch response, got count=%d jobs=%d", statusResp.Batch.JobCount, len(statusResp.Jobs))
	}
	if statusResp.Total != 2 || statusResp.Limit != 2 || statusResp.Offset != 0 {
		t.Fatalf("unexpected pagination metadata: %+v", statusResp)
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
				"id": "batch-mcp-1",
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
	if cancelResp.Batch.ID != "batch-mcp-1" {
		t.Fatalf("expected batch id batch-mcp-1, got %s", cancelResp.Batch.ID)
	}
	if cancelResp.Batch.Stats.Canceled == 0 {
		t.Fatalf("expected canceled jobs in batch stats, got %+v", cancelResp.Batch.Stats)
	}

	for _, created := range jobsCreated {
		stored, err := srv.store.Get(ctx, created.ID)
		if err != nil {
			t.Fatalf("failed to load canceled job %s: %v", created.ID, err)
		}
		if stored.Status != model.StatusCanceled {
			t.Fatalf("expected job %s to be canceled, got %s", stored.ID, stored.Status)
		}
	}
}
