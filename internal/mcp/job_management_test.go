// Tests for job management MCP tools.
// Tests verify job_list (list with pagination), job_cancel (cancel by id),
// job_export (export results in various formats), job_status (get job by id),
// and job_results (get job results by id) tools.
//
// Does NOT handle:
// - Actual scraping/crawling execution (only job operations)
// - Job state transitions beyond cancel
//
// Invariants:
// - job_list accepts optional limit and offset parameters
// - job_cancel requires id parameter and marks job as canceled
// - job_export requires id parameter and optional format parameter
// - job_export supports formats: jsonl, json, md, csv (defaults to jsonl)
// - job_status requires id parameter and returns full job record
// - job_results requires id parameter and returns job result content
// - Exporting a job without results should error
// - Exporting a non-existent job should error
// - Getting status/results of non-existent job should error
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"spartan-scraper/internal/extract"
	"spartan-scraper/internal/fetch"
	"spartan-scraper/internal/fsutil"
	"spartan-scraper/internal/model"
	"spartan-scraper/internal/pipeline"
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
	if jobListTool.Description != "List all jobs with pagination" {
		t.Errorf("expected description 'List all jobs with pagination', got '%s'", jobListTool.Description)
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
		_, err := srv.manager.CreateScrapeJob(ctx, "http://example.com/1", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false)
		if err != nil {
			t.Fatalf("CreateScrapeJob failed: %v", err)
		}
		_, err = srv.manager.CreateScrapeJob(ctx, "http://example.com/2", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false)
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

		resultMap, ok := result.(map[string]interface{})
		if !ok {
			t.Fatal("result is not a map")
		}
		jobs := resultMap["jobs"]
		if jobs == nil {
			t.Fatal("jobs not found in result")
		}

		jobCount := reflect.ValueOf(jobs).Len()
		if jobCount != 2 {
			t.Errorf("expected 2 jobs, got %d", jobCount)
		}
	})

	t.Run("list with limit and offset", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			_, err := srv.manager.CreateScrapeJob(ctx, fmt.Sprintf("http://example.com/%d", i), false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false)
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

		resultMap, ok := result.(map[string]interface{})
		if !ok {
			t.Fatal("result is not a map")
		}
		jobs := resultMap["jobs"]
		if jobs == nil {
			t.Fatal("jobs not found in result")
		}
		jobCount := reflect.ValueOf(jobs).Len()
		if jobCount != 2 {
			t.Errorf("expected 2 jobs (offset 2, limit 2), got %d", jobCount)
		}
	})
}

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
	if jobCancelTool.Description != "Cancel a running or queued job by id" {
		t.Errorf("expected description 'Cancel a running or queued job by id', got '%s'", jobCancelTool.Description)
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
		job, err := srv.manager.CreateScrapeJob(ctx, "http://example.com", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false)
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

		resultMap, ok := result.(map[string]interface{})
		if !ok {
			t.Fatal("result is not a map")
		}
		if resultMap["status"] != "canceled" {
			t.Errorf("expected status 'canceled', got '%v'", resultMap["status"])
		}
		if resultMap["id"] != job.ID {
			t.Errorf("expected id '%s', got '%v'", job.ID, resultMap["id"])
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
	if jobExportTool.Description != "Export job results in specified format (jsonl, json, md, csv)" {
		t.Errorf("expected description 'Export job results in specified format (jsonl, json, md, csv)', got '%s'", jobExportTool.Description)
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
		job, err := srv.manager.CreateScrapeJob(ctx, "http://example.com", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false)
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
		job, err := srv.manager.CreateScrapeJob(ctx, "http://example.com", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false)
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

	t.Run("export job with invalid format", func(t *testing.T) {
		job, err := srv.manager.CreateScrapeJob(ctx, "http://example.com", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false)
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
	})

	t.Run("export job without results", func(t *testing.T) {
		job, err := srv.manager.CreateScrapeJob(ctx, "http://example.com", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false)
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
	})
}

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
	if jobStatusTool.Description != "Get job status by id" {
		t.Errorf("expected description 'Get job status by id', got '%s'", jobStatusTool.Description)
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
		job, err := srv.manager.CreateScrapeJob(ctx, "http://example.com", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false)
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

		resultJob, ok := result.(model.Job)
		if !ok {
			t.Fatalf("result is not a model.Job, type is %T", result)
		}
		if resultJob.ID != job.ID {
			t.Errorf("expected id '%s', got '%s'", job.ID, resultJob.ID)
		}
		if resultJob.Kind != model.KindScrape {
			t.Errorf("expected kind '%s', got '%s'", model.KindScrape, resultJob.Kind)
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
		job, err := srv.manager.CreateScrapeJob(ctx, "http://example.com", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false)
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
		job, err := srv.manager.CreateScrapeJob(ctx, "http://example.com", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false)
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
