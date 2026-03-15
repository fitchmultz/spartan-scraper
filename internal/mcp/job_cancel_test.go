// Package mcp provides tests for the job_cancel MCP tool.
// Tests cover tool schema validation and job cancellation behavior.
// Does NOT test actual job execution or worker pool cancellation.
package mcp

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/api"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

func TestJobCancelToolInToolsList(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	tools := srv.toolsList()
	toolMap := make(map[string]tool)
	for _, t := range tools {
		toolMap[t.Name] = t
	}
	jobCancelTool, ok := toolMap["job_cancel"]
	if !ok {
		t.Fatal("job_cancel tool not found in toolsList")
	}
	if jobCancelTool.Description != "Cancel a running or queued job and return the updated job envelope" {
		t.Errorf("expected description 'Cancel a running or queued job and return the updated job envelope', got '%s'", jobCancelTool.Description)
	}
	schema := jobCancelTool.InputSchema
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

func TestHandleJobCancel(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	ctx := context.Background()

	t.Run("cancel queued job", func(t *testing.T) {
		job, err := srv.manager.CreateScrapeJob(ctx, "http://example.com", "GET", nil, "", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false, "", "", nil, "")
		if err != nil {
			t.Fatalf("CreateScrapeJob failed: %v", err)
		}

		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name":      "job_cancel",
				"arguments": map[string]interface{}{"id": job.ID},
			}),
		}

		result, err := srv.handleToolCall(ctx, base)
		if err != nil {
			t.Fatalf("handleToolCall failed: %v", err)
		}

		response, ok := result.(api.JobResponse)
		if !ok {
			t.Fatalf("result is not an api.JobResponse: %T", result)
		}
		if response.Job.Status != "canceled" {
			t.Errorf("expected status 'canceled', got '%v'", response.Job.Status)
		}
		if response.Job.ID != job.ID {
			t.Errorf("expected id '%s', got '%v'", job.ID, response.Job.ID)
		}

		updatedJob, err := srv.store.Get(ctx, job.ID)
		if err != nil {
			t.Fatalf("Get job failed: %v", err)
		}
		if updatedJob.Status != "canceled" {
			t.Errorf("expected job status 'canceled', got '%s'", updatedJob.Status)
		}
	})

	t.Run("cancel non-existent job", func(t *testing.T) {
		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name":      "job_cancel",
				"arguments": map[string]interface{}{"id": "non-existent-id"},
			}),
		}

		_, err := srv.handleToolCall(ctx, base)
		if err == nil {
			t.Error("expected error for non-existent job")
		}
	})

	t.Run("cancel without id", func(t *testing.T) {
		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name":      "job_cancel",
				"arguments": map[string]interface{}{},
			}),
		}

		_, err := srv.handleToolCall(ctx, base)
		if err == nil {
			t.Error("expected error for missing id")
		}
	})
}
