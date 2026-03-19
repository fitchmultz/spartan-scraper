// Package mcp provides tests for export outcome MCP tools.
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
	if jobExportTool.Description != "Run an export, persist an export outcome, and return guided inspection with inline artifact content when available" {
		t.Errorf("unexpected description: %s", jobExportTool.Description)
	}
	for _, name := range []string{"job_export_history", "export_outcome_get"} {
		if _, ok := toolMap[name]; !ok {
			t.Fatalf("%s tool not found in toolsList", name)
		}
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
		result := mustCallToolObject(t, srv, ctx, "job_export", map[string]interface{}{"id": jobID, "format": "jsonl"})
		exportValue := requireExportEnvelope(t, result)
		if exportValue["status"] != "succeeded" {
			t.Fatalf("expected succeeded export, got %#v", exportValue)
		}
		artifact := requireArtifact(t, exportValue)
		if artifact["encoding"] != "utf8" {
			t.Fatalf("expected utf8 encoding, got %#v", artifact)
		}
		content := artifact["content"].(string)
		if !strings.Contains(content, `"title":"Test"`) {
			t.Fatalf("unexpected jsonl export: %s", content)
		}
	})

	t.Run("export job with default format", func(t *testing.T) {
		jobID := writeScrapeResultForMCPTest(t, srv, ctx, tmpDir, `{"url":"http://example.com","status":200}`)
		result := mustCallToolObject(t, srv, ctx, "job_export", map[string]interface{}{"id": jobID})
		exportValue := requireExportEnvelope(t, result)
		artifact := requireArtifact(t, exportValue)
		if artifact["format"] != "jsonl" {
			t.Fatalf("expected default format jsonl, got %#v", artifact["format"])
		}
	})

	t.Run("export job with transform", func(t *testing.T) {
		jobID := writeScrapeResultForMCPTest(t, srv, ctx, tmpDir, `{"url":"http://example.com","status":200,"title":"Test"}`)
		result := mustCallToolObject(t, srv, ctx, "job_export", map[string]interface{}{
			"id":     jobID,
			"format": "json",
			"transform": map[string]interface{}{
				"expression": "{title: title}",
				"language":   "jmespath",
			},
		})
		content := requireArtifact(t, requireExportEnvelope(t, result))["content"].(string)
		if strings.Contains(content, "status") || !strings.Contains(content, "title") {
			t.Fatalf("unexpected transformed export: %s", content)
		}
	})

	t.Run("export job with shape", func(t *testing.T) {
		jobID := writeScrapeResultForMCPTest(t, srv, ctx, tmpDir, `{"url":"http://example.com","status":200,"title":"Test","normalized":{"fields":{"price":{"values":["$10"]}}}}`)
		result := mustCallToolObject(t, srv, ctx, "job_export", map[string]interface{}{
			"id":     jobID,
			"format": "md",
			"shape": map[string]interface{}{
				"summaryFields":    []string{"title", "url"},
				"normalizedFields": []string{"field.price"},
			},
		})
		content := requireArtifact(t, requireExportEnvelope(t, result))["content"].(string)
		if !strings.Contains(content, "Test") || !strings.Contains(content, "$10") {
			t.Fatalf("unexpected shaped markdown: %s", content)
		}
	})

	t.Run("export job as xlsx base64 payload", func(t *testing.T) {
		jobID := writeScrapeResultForMCPTest(t, srv, ctx, tmpDir, `{"url":"http://example.com","status":200,"title":"Test"}`)
		result := mustCallToolObject(t, srv, ctx, "job_export", map[string]interface{}{"id": jobID, "format": "xlsx"})
		artifact := requireArtifact(t, requireExportEnvelope(t, result))
		if artifact["encoding"] != "base64" {
			t.Fatalf("expected base64 encoding, got %#v", artifact)
		}
		if artifact["contentType"] != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
			t.Fatalf("unexpected content type: %#v", artifact)
		}
		if strings.TrimSpace(artifact["content"].(string)) == "" {
			t.Fatal("expected base64 xlsx content")
		}
	})

	t.Run("export job history and lookup", func(t *testing.T) {
		jobID := writeScrapeResultForMCPTest(t, srv, ctx, tmpDir, `{"url":"http://example.com","status":200,"title":"History"}`)
		result := mustCallToolObject(t, srv, ctx, "job_export", map[string]interface{}{"id": jobID, "format": "json"})
		exportValue := requireExportEnvelope(t, result)
		exportID, _ := exportValue["id"].(string)
		if exportID == "" {
			t.Fatalf("expected export id, got %#v", exportValue)
		}

		history := mustCallToolObject(t, srv, ctx, "job_export_history", map[string]interface{}{"id": jobID, "limit": 10, "offset": 0})
		exports, ok := history["exports"].([]interface{})
		if !ok || len(exports) != 1 {
			t.Fatalf("expected one export in history, got %#v", history)
		}

		single := mustCallToolObject(t, srv, ctx, "export_outcome_get", map[string]interface{}{"id": exportID})
		singleExport := requireExportEnvelope(t, single)
		if singleExport["id"] != exportID {
			t.Fatalf("unexpected export lookup payload: %#v", single)
		}
		artifact, _ := singleExport["artifact"].(map[string]interface{})
		if artifact != nil {
			if content, _ := artifact["content"].(string); content != "" {
				t.Fatalf("expected history lookup without inline content, got %#v", artifact)
			}
		}
	})

	t.Run("export job with invalid format", func(t *testing.T) {
		jobID := writeScrapeResultForMCPTest(t, srv, ctx, tmpDir, `{"url":"http://example.com","status":200}`)
		_, err := callToolObject(t, srv, ctx, "job_export", map[string]interface{}{"id": jobID, "format": "txt"})
		if err == nil {
			t.Error("expected error for invalid format")
		}
		if !apperrors.IsKind(err, apperrors.KindValidation) {
			t.Errorf("expected KindValidation, got %v", err)
		}
	})

	t.Run("export job with shape and transform", func(t *testing.T) {
		jobID := writeScrapeResultForMCPTest(t, srv, ctx, tmpDir, `{"url":"http://example.com","status":200}`)
		_, err := callToolObject(t, srv, ctx, "job_export", map[string]interface{}{
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

	t.Run("export job without results returns failed outcome", func(t *testing.T) {
		job, err := srv.manager.CreateScrapeJob(ctx, "http://example.com", "GET", nil, "", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false, "", "", nil, "")
		if err != nil {
			t.Fatalf("CreateScrapeJob failed: %v", err)
		}
		result := mustCallToolObject(t, srv, ctx, "job_export", map[string]interface{}{"id": job.ID, "format": "jsonl"})
		exportValue := requireExportEnvelope(t, result)
		if exportValue["status"] != "failed" {
			t.Fatalf("expected failed export outcome, got %#v", exportValue)
		}
	})

	t.Run("export non-existent job", func(t *testing.T) {
		_, err := callToolObject(t, srv, ctx, "job_export", map[string]interface{}{"id": "non-existent-id", "format": "jsonl"})
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

func callToolObject(t *testing.T, srv *Server, ctx context.Context, name string, arguments map[string]interface{}) (map[string]interface{}, error) {
	t.Helper()
	base := map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name":      name,
			"arguments": arguments,
		}),
	}
	result, err := srv.handleToolCall(ctx, base)
	if err != nil {
		return nil, err
	}
	resultMap, ok := result.(map[string]interface{})
	if ok {
		return resultMap, nil
	}
	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal result: %v", err)
	}
	if err := json.Unmarshal(payload, &resultMap); err != nil {
		t.Fatalf("failed to decode result object: %v", err)
	}
	return resultMap, nil
}

func mustCallToolObject(t *testing.T, srv *Server, ctx context.Context, name string, arguments map[string]interface{}) map[string]interface{} {
	t.Helper()
	result, err := callToolObject(t, srv, ctx, name, arguments)
	if err != nil {
		t.Fatalf("handleToolCall failed: %v", err)
	}
	return result
}

func requireExportEnvelope(t *testing.T, result map[string]interface{}) map[string]interface{} {
	t.Helper()
	exportValue, ok := result["export"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected export envelope, got %#v", result)
	}
	return exportValue
}

func requireArtifact(t *testing.T, exportValue map[string]interface{}) map[string]interface{} {
	t.Helper()
	artifact, ok := exportValue["artifact"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected artifact payload, got %#v", exportValue)
	}
	return artifact
}
