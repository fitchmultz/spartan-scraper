// Package mcp provides tests for the job_export MCP tool.
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
	if jobExportTool.Description != "Export saved job results in jsonl, json, md, csv, or xlsx with optional shape or transform controls" {
		t.Errorf("unexpected description: %s", jobExportTool.Description)
	}
	schema := jobExportTool.InputSchema
	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("properties not found in schema")
	}
	for _, key := range []string{"id", "format", "shape", "transform"} {
		if _, ok := props[key]; !ok {
			t.Errorf("expected %q in properties", key)
		}
	}
	required := schema["required"].([]string)
	if len(required) != 1 || required[0] != "id" {
		t.Error("expected only 'id' to be required")
	}
}

func TestHandleJobExport(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	ctx := context.Background()

	t.Run("export job as jsonl", func(t *testing.T) {
		jobID := writeScrapeResultForMCPTest(t, srv, ctx, tmpDir, `{"url":"http://example.com","status":200,"title":"Test","text":"Content"}`)
		result := mustCallJobExport(t, srv, ctx, map[string]interface{}{"id": jobID, "format": "jsonl"})
		if result["encoding"] != "utf8" {
			t.Fatalf("expected utf8 encoding, got %#v", result)
		}
		content := result["content"].(string)
		if !strings.Contains(content, `"title":"Test"`) {
			t.Fatalf("unexpected jsonl export: %s", content)
		}
	})

	t.Run("export job with default format", func(t *testing.T) {
		jobID := writeScrapeResultForMCPTest(t, srv, ctx, tmpDir, `{"url":"http://example.com","status":200}`)
		result := mustCallJobExport(t, srv, ctx, map[string]interface{}{"id": jobID})
		if result["format"] != "jsonl" {
			t.Fatalf("expected default format jsonl, got %#v", result["format"])
		}
	})

	t.Run("export job with transform", func(t *testing.T) {
		jobID := writeScrapeResultForMCPTest(t, srv, ctx, tmpDir, `{"url":"http://example.com","status":200,"title":"Test"}`)
		result := mustCallJobExport(t, srv, ctx, map[string]interface{}{
			"id":     jobID,
			"format": "json",
			"transform": map[string]interface{}{
				"expression": "{title: title}",
				"language":   "jmespath",
			},
		})
		content := result["content"].(string)
		if strings.Contains(content, "status") || !strings.Contains(content, "title") {
			t.Fatalf("unexpected transformed export: %s", content)
		}
	})

	t.Run("export job with shape", func(t *testing.T) {
		jobID := writeScrapeResultForMCPTest(t, srv, ctx, tmpDir, `{"url":"http://example.com","status":200,"title":"Test","normalized":{"fields":{"price":{"values":["$10"]}}}}`)
		result := mustCallJobExport(t, srv, ctx, map[string]interface{}{
			"id":     jobID,
			"format": "md",
			"shape": map[string]interface{}{
				"summaryFields":    []string{"title", "url"},
				"normalizedFields": []string{"field.price"},
			},
		})
		content := result["content"].(string)
		if !strings.Contains(content, "Test") || !strings.Contains(content, "$10") {
			t.Fatalf("unexpected shaped markdown: %s", content)
		}
	})

	t.Run("export job as xlsx base64 payload", func(t *testing.T) {
		jobID := writeScrapeResultForMCPTest(t, srv, ctx, tmpDir, `{"url":"http://example.com","status":200,"title":"Test"}`)
		result := mustCallJobExport(t, srv, ctx, map[string]interface{}{"id": jobID, "format": "xlsx"})
		if result["encoding"] != "base64" {
			t.Fatalf("expected base64 encoding, got %#v", result)
		}
		if result["contentType"] != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
			t.Fatalf("unexpected content type: %#v", result)
		}
		if strings.TrimSpace(result["content"].(string)) == "" {
			t.Fatal("expected base64 xlsx content")
		}
	})

	t.Run("export job with invalid format", func(t *testing.T) {
		jobID := writeScrapeResultForMCPTest(t, srv, ctx, tmpDir, `{"url":"http://example.com","status":200}`)
		_, err := callJobExport(t, srv, ctx, map[string]interface{}{"id": jobID, "format": "txt"})
		if err == nil {
			t.Error("expected error for invalid format")
		}
		if !apperrors.IsKind(err, apperrors.KindValidation) {
			t.Errorf("expected KindValidation, got %v", err)
		}
	})

	t.Run("export job with shape and transform", func(t *testing.T) {
		jobID := writeScrapeResultForMCPTest(t, srv, ctx, tmpDir, `{"url":"http://example.com","status":200}`)
		_, err := callJobExport(t, srv, ctx, map[string]interface{}{
			"id":     jobID,
			"format": "csv",
			"shape":  map[string]interface{}{"topLevelFields": []string{"url"}},
			"transform": map[string]interface{}{
				"expression": "{url: url}",
				"language":   "jmespath",
			},
		})
		if err == nil {
			t.Error("expected error for shape+transform")
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
		_, err = callJobExport(t, srv, ctx, map[string]interface{}{"id": job.ID, "format": "jsonl"})
		if err == nil {
			t.Error("expected error for job without results")
		}
	})

	t.Run("export non-existent job", func(t *testing.T) {
		_, err := callJobExport(t, srv, ctx, map[string]interface{}{"id": "non-existent-id", "format": "jsonl"})
		if err == nil {
			t.Error("expected error for non-existent job")
		}
		if !apperrors.IsKind(err, apperrors.KindNotFound) {
			t.Errorf("expected KindNotFound, got %v", err)
		}
	})
}

func writeScrapeResultForMCPTest(t *testing.T, srv *Server, ctx context.Context, tmpDir string, content string) string {
	t.Helper()
	job, err := srv.manager.CreateScrapeJob(ctx, "http://example.com", "GET", nil, "", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false, "", "", nil, "")
	if err != nil {
		t.Fatalf("CreateScrapeJob failed: %v", err)
	}
	resultFile := job.ResultPath
	resultDir := filepath.Join(tmpDir, "jobs", job.ID)
	if err := fsutil.MkdirAllSecure(resultDir); err != nil {
		t.Fatalf("failed to create job directory: %v", err)
	}
	if err := os.WriteFile(resultFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write result file: %v", err)
	}
	return job.ID
}

func callJobExport(t *testing.T, srv *Server, ctx context.Context, arguments map[string]interface{}) (map[string]interface{}, error) {
	t.Helper()
	base := map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name":      "job_export",
			"arguments": arguments,
		}),
	}
	result, err := srv.handleToolCall(ctx, base)
	if err != nil {
		return nil, err
	}
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("result is not an object: %#v", result)
	}
	return resultMap, nil
}

func mustCallJobExport(t *testing.T, srv *Server, ctx context.Context, arguments map[string]interface{}) map[string]interface{} {
	t.Helper()
	result, err := callJobExport(t, srv, ctx, arguments)
	if err != nil {
		t.Fatalf("handleToolCall failed: %v", err)
	}
	return result
}
