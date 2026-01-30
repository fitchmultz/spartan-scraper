// Package mcp provides tests for the job_results MCP tool.
// Tests cover tool schema validation and retrieval of job result files.
// Does NOT test result parsing or content transformation.
package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/fsutil"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

func TestJobResultsToolInToolsList(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	tools := srv.toolsList()
	toolMap := make(map[string]tool)
	for _, t := range tools {
		toolMap[t.Name] = t
	}
	jobResultsTool, ok := toolMap["job_results"]
	if !ok {
		t.Fatal("job_results tool not found in toolsList")
	}
	if jobResultsTool.Description != "Get job results by id" {
		t.Errorf("expected description 'Get job results by id', got '%s'", jobResultsTool.Description)
	}
	schema := jobResultsTool.InputSchema
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

func TestHandleJobResults(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	ctx := context.Background()

	t.Run("get job results", func(t *testing.T) {
		job, err := srv.manager.CreateScrapeJob(ctx, "http://example.com", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false, "", "", nil, "")
		if err != nil {
			t.Fatalf("CreateScrapeJob failed: %v", err)
		}

		resultFile := job.ResultPath
		resultDir := filepath.Join(tmpDir, "jobs", job.ID)
		if err := fsutil.MkdirAllSecure(resultDir); err != nil {
			t.Fatalf("failed to create job directory: %v", err)
		}
		resultContent := `{"url":"http://example.com","status":200,"title":"Test","text":"Content"}`
		if err := os.WriteFile(resultFile, []byte(resultContent), 0o644); err != nil {
			t.Fatalf("failed to write result file: %v", err)
		}

		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name":      "job_results",
				"arguments": map[string]interface{}{"id": job.ID},
			}),
		}

		result, err := srv.handleToolCall(ctx, base)
		if err != nil {
			t.Fatalf("handleToolCall failed: %v", err)
		}

		resultStr, ok := result.(string)
		if !ok {
			t.Fatal("result is not a string")
		}
		resultStr = strings.TrimSpace(resultStr)
		if resultStr != resultContent {
			t.Errorf("expected result '%s', got '%s'", resultContent, resultStr)
		}
	})

	t.Run("get results of non-existent job", func(t *testing.T) {
		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name":      "job_results",
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

	t.Run("get results without id", func(t *testing.T) {
		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name":      "job_results",
				"arguments": map[string]interface{}{},
			}),
		}

		_, err := srv.handleToolCall(ctx, base)
		if err == nil {
			t.Error("expected error for missing id")
		}
	})

	t.Run("get results of job without results", func(t *testing.T) {
		job, err := srv.manager.CreateScrapeJob(ctx, "http://example.com", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false, "", "", nil, "")
		if err != nil {
			t.Fatalf("CreateScrapeJob failed: %v", err)
		}

		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name":      "job_results",
				"arguments": map[string]interface{}{"id": job.ID},
			}),
		}

		_, err = srv.handleToolCall(ctx, base)
		if err == nil {
			t.Error("expected error for job without results")
		}
	})
}
