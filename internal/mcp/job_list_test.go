// Package mcp provides tests for the job_list MCP tool.
// Tests cover tool schema validation and job listing with pagination (limit/offset).
// Does NOT test job creation or execution behavior.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/api"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

func TestJobListToolInToolsList(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	tools := srv.toolsList()
	toolMap := make(map[string]tool)
	for _, t := range tools {
		toolMap[t.Name] = t
	}
	jobListTool, ok := toolMap["job_list"]
	if !ok {
		t.Fatal("job_list tool not found in toolsList")
	}
	if jobListTool.Description != "List job envelopes with pagination metadata" {
		t.Errorf("expected description 'List job envelopes with pagination metadata', got '%s'", jobListTool.Description)
	}
	schema := jobListTool.InputSchema
	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("properties not found in schema")
	}
	if _, ok := props["limit"]; !ok {
		t.Error("expected 'limit' in properties")
	}
	if _, ok := props["offset"]; !ok {
		t.Error("expected 'offset' in properties")
	}
	required := schema["required"]
	if len(required.([]string)) != 0 {
		t.Error("expected no required fields")
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
}
