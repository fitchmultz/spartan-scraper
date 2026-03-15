// Package mcp provides tests for the job_status MCP tool.
// Tests cover tool schema validation and job status retrieval by ID.
// Does NOT test job state transitions or lifecycle management.
package mcp

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/api"
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

func TestJobStatusToolInToolsList(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	tools := srv.toolsList()
	toolMap := make(map[string]tool)
	for _, t := range tools {
		toolMap[t.Name] = t
	}
	jobStatusTool, ok := toolMap["job_status"]
	if !ok {
		t.Fatal("job_status tool not found in toolsList")
	}
	if jobStatusTool.Description != "Get a single job envelope by id" {
		t.Errorf("expected description 'Get a single job envelope by id', got '%s'", jobStatusTool.Description)
	}
	schema := jobStatusTool.InputSchema
	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("properties not found in schema")
	}
	if _, ok := props["id"]; !ok {
		t.Error("expected 'id' in properties")
	}
	required := schema["required"].([]string)
	if len(required) != 1 || required[0] != "id" {
		t.Error("expected 'id' to be required")
	}
}

func TestHandleJobStatus(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	ctx := context.Background()

	t.Run("get job status", func(t *testing.T) {
		job, err := srv.manager.CreateScrapeJob(ctx, "http://example.com", "GET", nil, "", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false, "", "", nil, "")
		if err != nil {
			t.Fatalf("CreateScrapeJob failed: %v", err)
		}

		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name":      "job_status",
				"arguments": map[string]interface{}{"id": job.ID},
			}),
		}

		result, err := srv.handleToolCall(ctx, base)
		if err != nil {
			t.Fatalf("handleToolCall failed: %v", err)
		}

		response, ok := result.(api.JobResponse)
		if !ok {
			t.Fatalf("result is not an api.JobResponse, type is %T", result)
		}
		if response.Job.ID != job.ID {
			t.Errorf("expected id '%s', got '%s'", job.ID, response.Job.ID)
		}
		if response.Job.Kind != model.KindScrape {
			t.Errorf("expected kind '%s', got '%s'", model.KindScrape, response.Job.Kind)
		}
	})

	t.Run("get status of non-existent job", func(t *testing.T) {
		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name":      "job_status",
				"arguments": map[string]interface{}{"id": "non-existent-id"},
			}),
		}

		_, err := srv.handleToolCall(ctx, base)
		if err == nil {
			t.Error("expected error for non-existent job")
		}
		if !apperrors.IsKind(err, apperrors.KindNotFound) {
			t.Errorf("expected KindNotFound, got %v", err)
		}
	})

	t.Run("get status without id", func(t *testing.T) {
		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name":      "job_status",
				"arguments": map[string]interface{}{},
			}),
		}

		_, err := srv.handleToolCall(ctx, base)
		if err == nil {
			t.Error("expected error for missing id")
		}
	})
}
