// Package mcp provides tests for the job_export MCP tool.
// Tests cover tool schema validation and result export in multiple formats (jsonl, json, md, csv).
// Does NOT test actual scraping or crawling execution.
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

func TestJobExportToolInToolsList(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	tools := srv.toolsList()
	toolMap := make(map[string]tool)
	for _, t := range tools {
		toolMap[t.Name] = t
	}
	jobExportTool, ok := toolMap["job_export"]
	if !ok {
		t.Fatal("job_export tool not found in toolsList")
	}
	if jobExportTool.Description != "Export job results in specified text format (jsonl, json, md, csv) with optional transform controls" {
		t.Errorf("unexpected description: %s", jobExportTool.Description)
	}
	schema := jobExportTool.InputSchema
	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("properties not found in schema")
	}
	if _, ok := props["id"]; !ok {
		t.Error("expected 'id' in properties")
	}
	if _, ok := props["format"]; !ok {
		t.Error("expected 'format' in properties")
	}
	if _, ok := props["transformExpression"]; !ok {
		t.Error("expected 'transformExpression' in properties")
	}
	if _, ok := props["transformLanguage"]; !ok {
		t.Error("expected 'transformLanguage' in properties")
	}
	required := schema["required"].([]string)
	if len(required) != 1 || required[0] != "id" {
		t.Error("expected 'id' to be required and 'format' to be optional")
	}
}

func TestHandleJobExport(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	ctx := context.Background()

	t.Run("export job as jsonl", func(t *testing.T) {
		job, err := srv.manager.CreateScrapeJob(ctx, "http://example.com", "GET", nil, "", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false, "", "", nil, "")
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
				"name":      "job_export",
				"arguments": map[string]interface{}{"id": job.ID, "format": "jsonl"},
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
		resultStr = resultStr[:len(resultStr)-1]
		if resultStr != resultContent {
			t.Errorf("expected result '%s', got '%s'", resultContent, resultStr)
		}
	})

	t.Run("export job with default format", func(t *testing.T) {
		job, err := srv.manager.CreateScrapeJob(ctx, "http://example.com", "GET", nil, "", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false, "", "", nil, "")
		if err != nil {
			t.Fatalf("CreateScrapeJob failed: %v", err)
		}

		resultFile := job.ResultPath
		resultDir := filepath.Join(tmpDir, "jobs", job.ID)
		if err := fsutil.MkdirAllSecure(resultDir); err != nil {
			t.Fatalf("failed to create job directory: %v", err)
		}
		resultContent := `{"url":"http://example.com","status":200}`
		if err := os.WriteFile(resultFile, []byte(resultContent), 0o644); err != nil {
			t.Fatalf("failed to write result file: %v", err)
		}

		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name":      "job_export",
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
		resultStr = resultStr[:len(resultStr)-1]
		if resultStr != resultContent {
			t.Errorf("expected result '%s', got '%s'", resultContent, resultStr)
		}
	})

	t.Run("export job with transform", func(t *testing.T) {
		job, err := srv.manager.CreateScrapeJob(ctx, "http://example.com", "GET", nil, "", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false, "", "", nil, "")
		if err != nil {
			t.Fatalf("CreateScrapeJob failed: %v", err)
		}

		resultFile := job.ResultPath
		resultDir := filepath.Join(tmpDir, "jobs", job.ID)
		if err := fsutil.MkdirAllSecure(resultDir); err != nil {
			t.Fatalf("failed to create job directory: %v", err)
		}
		resultContent := `{"url":"http://example.com","status":200,"title":"Test"}`
		if err := os.WriteFile(resultFile, []byte(resultContent), 0o644); err != nil {
			t.Fatalf("failed to write result file: %v", err)
		}

		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name": "job_export",
				"arguments": map[string]interface{}{
					"id":                  job.ID,
					"format":              "json",
					"transformExpression": "{title: title}",
					"transformLanguage":   "jmespath",
				},
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
		if strings.Contains(resultStr, "status") || !strings.Contains(resultStr, "title") {
			t.Fatalf("unexpected transformed export: %s", resultStr)
		}
	})

	t.Run("export job with invalid format", func(t *testing.T) {
		job, err := srv.manager.CreateScrapeJob(ctx, "http://example.com", "GET", nil, "", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false, "", "", nil, "")
		if err != nil {
			t.Fatalf("CreateScrapeJob failed: %v", err)
		}

		resultFile := job.ResultPath
		resultDir := filepath.Join(tmpDir, "jobs", job.ID)
		if err := fsutil.MkdirAllSecure(resultDir); err != nil {
			t.Fatalf("failed to create job directory: %v", err)
		}
		resultContent := `{"url":"http://example.com","status":200}`
		if err := os.WriteFile(resultFile, []byte(resultContent), 0o644); err != nil {
			t.Fatalf("failed to write result file: %v", err)
		}

		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name":      "job_export",
				"arguments": map[string]interface{}{"id": job.ID, "format": "txt"},
			}),
		}

		_, err = srv.handleToolCall(ctx, base)
		if err == nil {
			t.Error("expected error for invalid format")
		}
		if !apperrors.IsKind(err, apperrors.KindValidation) {
			t.Errorf("expected KindValidation, got %v", err)
		}
	})

	t.Run("export job without results", func(t *testing.T) {
		job, err := srv.manager.CreateScrapeJob(ctx, "http://example.com", "GET", nil, "", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false, "", "", nil, "")
		if err != nil {
			t.Fatalf("CreateScrapeJob failed: %v", err)
		}

		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name":      "job_export",
				"arguments": map[string]interface{}{"id": job.ID, "format": "jsonl"},
			}),
		}

		_, err = srv.handleToolCall(ctx, base)
		if err == nil {
			t.Error("expected error for job without results")
		}
	})

	t.Run("export non-existent job", func(t *testing.T) {
		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name":      "job_export",
				"arguments": map[string]interface{}{"id": "non-existent-id", "format": "jsonl"},
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
}
