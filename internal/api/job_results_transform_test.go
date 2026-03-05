// Package api provides integration tests for job results transform export functionality.
// Tests cover applying JMESPath and JSONata transformations during export.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/fsutil"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestHandleJobResultsWithTransform_JMESPath(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	jobID := "test-job-jmespath"

	// Create job directly in store to avoid async processing
	job := model.Job{
		ID:        jobID,
		Kind:      model.KindScrape,
		Status:    model.StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Params:    map[string]any{"url": "https://example.com"},
	}
	if err := srv.store.Create(ctx, job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	resultDir := filepath.Join(srv.store.DataDir(), "jobs", jobID)
	if err := fsutil.MkdirAllSecure(resultDir); err != nil {
		t.Fatalf("failed to create result directory: %v", err)
	}

	resultPath := filepath.Join(resultDir, "results.jsonl")
	resultContent := `{"url":"https://example.com","status":200,"title":"Test Page","content":"Hello World"}`
	if err := os.WriteFile(resultPath, []byte(resultContent), 0o644); err != nil {
		t.Fatalf("failed to write result file: %v", err)
	}

	if err := srv.store.UpdateResultPath(ctx, jobID, resultPath); err != nil {
		t.Fatalf("failed to update job result_path: %v", err)
	}
	if err := srv.store.UpdateStatus(ctx, jobID, model.StatusSucceeded, ""); err != nil {
		t.Fatalf("failed to update job status: %v", err)
	}

	// Test JMESPath transformation
	req := httptest.NewRequest("GET", fmt.Sprintf("/v1/jobs/%s/results?format=json&transform_expression=%s&transform_language=jmespath",
		jobID, "%7Btitle%3A%20title%2C%20url%3A%20url%7D"), nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected 200, got %v: %s", status, rr.Body.String())
	}

	var results []map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &results); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}

	// Verify transformation was applied (only title and url fields)
	if _, ok := results[0]["title"]; !ok {
		t.Error("expected result to have 'title' field")
	}
	if _, ok := results[0]["url"]; !ok {
		t.Error("expected result to have 'url' field")
	}
	if _, ok := results[0]["content"]; ok {
		t.Error("result should not have 'content' field after transformation")
	}
	if _, ok := results[0]["status"]; ok {
		t.Error("result should not have 'status' field after transformation")
	}
}

func TestHandleJobResultsWithTransform_JSONata(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	jobID := "test-job-jsonata"

	// Create job directly in store to avoid async processing
	job := model.Job{
		ID:        jobID,
		Kind:      model.KindScrape,
		Status:    model.StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Params:    map[string]any{"url": "https://example.com"},
	}
	if err := srv.store.Create(ctx, job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	resultDir := filepath.Join(srv.store.DataDir(), "jobs", jobID)
	if err := fsutil.MkdirAllSecure(resultDir); err != nil {
		t.Fatalf("failed to create result directory: %v", err)
	}

	resultPath := filepath.Join(resultDir, "results.jsonl")
	resultContent := `{"name":"Product A","price":100,"quantity":2}`
	if err := os.WriteFile(resultPath, []byte(resultContent), 0o644); err != nil {
		t.Fatalf("failed to write result file: %v", err)
	}

	if err := srv.store.UpdateResultPath(ctx, jobID, resultPath); err != nil {
		t.Fatalf("failed to update job result_path: %v", err)
	}
	if err := srv.store.UpdateStatus(ctx, jobID, model.StatusSucceeded, ""); err != nil {
		t.Fatalf("failed to update job status: %v", err)
	}

	// Test JSONata transformation with calculation
	req := httptest.NewRequest("GET", fmt.Sprintf("/v1/jobs/%s/results?format=json&transform_expression=%s&transform_language=jsonata",
		jobID, "%7B%22item%22%3A%20name%2C%20%22total%22%3A%20price%20*%20quantity%7D"), nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected 200, got %v: %s", status, rr.Body.String())
	}

	var results []map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &results); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}

	// Verify transformation was applied
	if results[0]["item"] != "Product A" {
		t.Errorf("expected item to be 'Product A', got %v", results[0]["item"])
	}
	if results[0]["total"] != float64(200) {
		t.Errorf("expected total to be 200, got %v", results[0]["total"])
	}
}

func TestHandleJobResultsWithTransform_InvalidLanguage(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	jobID := "test-job-invalid-lang"

	// Create job directly in store to avoid async processing
	job := model.Job{
		ID:        jobID,
		Kind:      model.KindScrape,
		Status:    model.StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Params:    map[string]any{"url": "https://example.com"},
	}
	if err := srv.store.Create(ctx, job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	resultDir := filepath.Join(srv.store.DataDir(), "jobs", jobID)
	if err := fsutil.MkdirAllSecure(resultDir); err != nil {
		t.Fatalf("failed to create result directory: %v", err)
	}

	resultPath := filepath.Join(resultDir, "results.jsonl")
	if err := os.WriteFile(resultPath, []byte(`{"test":"data"}`), 0o644); err != nil {
		t.Fatalf("failed to write result file: %v", err)
	}

	if err := srv.store.UpdateResultPath(ctx, jobID, resultPath); err != nil {
		t.Fatalf("failed to update job result_path: %v", err)
	}
	if err := srv.store.UpdateStatus(ctx, jobID, model.StatusSucceeded, ""); err != nil {
		t.Fatalf("failed to update job status: %v", err)
	}

	// Test with invalid transform language
	req := httptest.NewRequest("GET", fmt.Sprintf("/v1/jobs/%s/results?format=json&transform_expression=%s&transform_language=invalid",
		jobID, "%7Btitle%3A%20title%7D"), nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid language, got %v", status)
	}
}

func TestHandleJobResultsWithTransform_InvalidExpression(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	jobID := "test-job-invalid-expr"

	// Create job directly in store to avoid async processing
	job := model.Job{
		ID:        jobID,
		Kind:      model.KindScrape,
		Status:    model.StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Params:    map[string]any{"url": "https://example.com"},
	}
	if err := srv.store.Create(ctx, job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	resultDir := filepath.Join(srv.store.DataDir(), "jobs", jobID)
	if err := fsutil.MkdirAllSecure(resultDir); err != nil {
		t.Fatalf("failed to create result directory: %v", err)
	}

	resultPath := filepath.Join(resultDir, "results.jsonl")
	if err := os.WriteFile(resultPath, []byte(`{"test":"data"}`), 0o644); err != nil {
		t.Fatalf("failed to write result file: %v", err)
	}

	if err := srv.store.UpdateResultPath(ctx, jobID, resultPath); err != nil {
		t.Fatalf("failed to update job result_path: %v", err)
	}
	if err := srv.store.UpdateStatus(ctx, jobID, model.StatusSucceeded, ""); err != nil {
		t.Fatalf("failed to update job status: %v", err)
	}

	// Test with invalid JMESPath expression
	req := httptest.NewRequest("GET", fmt.Sprintf("/v1/jobs/%s/results?format=json&transform_expression=%s&transform_language=jmespath",
		jobID, "%7Binvalid"), nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid expression, got %v", status)
	}
}

func TestHandleJobResultsWithTransform_CSVFormat(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	jobID := "test-job-csv-transform"

	// Create job directly in store to avoid async processing
	job := model.Job{
		ID:        jobID,
		Kind:      model.KindScrape,
		Status:    model.StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Params:    map[string]any{"url": "https://example.com"},
	}
	if err := srv.store.Create(ctx, job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	resultDir := filepath.Join(srv.store.DataDir(), "jobs", jobID)
	if err := fsutil.MkdirAllSecure(resultDir); err != nil {
		t.Fatalf("failed to create result directory: %v", err)
	}

	resultPath := filepath.Join(resultDir, "results.jsonl")
	resultContent := `{"url":"https://example.com","status":200,"title":"Test Page"}`
	if err := os.WriteFile(resultPath, []byte(resultContent), 0o644); err != nil {
		t.Fatalf("failed to write result file: %v", err)
	}

	if err := srv.store.UpdateResultPath(ctx, jobID, resultPath); err != nil {
		t.Fatalf("failed to update job result_path: %v", err)
	}
	if err := srv.store.UpdateStatus(ctx, jobID, model.StatusSucceeded, ""); err != nil {
		t.Fatalf("failed to update job status: %v", err)
	}

	// Test CSV export with transformation
	req := httptest.NewRequest("GET", fmt.Sprintf("/v1/jobs/%s/results?format=csv&transform_expression=%s&transform_language=jmespath",
		jobID, "%7Btitle%3A%20title%7D"), nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected 200, got %v: %s", status, rr.Body.String())
	}

	ct := rr.Header().Get("Content-Type")
	if ct != "text/csv; charset=utf-8" {
		t.Errorf("expected Content-Type text/csv; charset=utf-8, got %v", ct)
	}

	bodyStr := rr.Body.String()
	if !strings.Contains(bodyStr, "title") {
		t.Error("CSV should contain 'title' header")
	}
	if strings.Contains(bodyStr, "status") {
		t.Error("CSV should not contain 'status' column after transformation")
	}
}

func TestLoadAllJobResults(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	jobID := "test-job-load-all"

	// Create job directly in store to avoid async processing
	job := model.Job{
		ID:        jobID,
		Kind:      model.KindScrape,
		Status:    model.StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Params:    map[string]any{"url": "https://example.com"},
	}
	if err := srv.store.Create(ctx, job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	resultDir := filepath.Join(srv.store.DataDir(), "jobs", jobID)
	if err := fsutil.MkdirAllSecure(resultDir); err != nil {
		t.Fatalf("failed to create result directory: %v", err)
	}

	resultPath := filepath.Join(resultDir, "results.jsonl")
	resultContent := `{"id":1,"name":"First"}
{"id":2,"name":"Second"}
{"id":3,"name":"Third"}`
	if err := os.WriteFile(resultPath, []byte(resultContent), 0o644); err != nil {
		t.Fatalf("failed to write result file: %v", err)
	}

	if err := srv.store.UpdateResultPath(ctx, jobID, resultPath); err != nil {
		t.Fatalf("failed to update job result_path: %v", err)
	}
	if err := srv.store.UpdateStatus(ctx, jobID, model.StatusSucceeded, ""); err != nil {
		t.Fatalf("failed to update job status: %v", err)
	}

	// Create a minimal job struct for testing
	testJob := model.Job{
		ID:         jobID,
		ResultPath: resultPath,
	}

	results, err := srv.loadAllJobResults(testJob)
	if err != nil {
		t.Fatalf("failed to load all job results: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	// Verify each result
	for i, r := range results {
		result, ok := r.(map[string]interface{})
		if !ok {
			t.Fatalf("result %d is not an object", i)
		}
		expectedID := float64(i + 1)
		if result["id"] != expectedID {
			t.Errorf("expected result %d to have id %v, got %v", i, expectedID, result["id"])
		}
	}
}

func TestLoadAllJobResults_EmptyPath(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	defer cleanup()

	testJob := model.Job{
		ID:         "test-job",
		ResultPath: "",
	}

	results, err := srv.loadAllJobResults(testJob)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results for empty path, got %d", len(results))
	}
}

func TestLoadAllJobResults_MissingFile(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	jobID := "test-job-missing-file"

	// Create job directly in store to avoid async processing
	job := model.Job{
		ID:        jobID,
		Kind:      model.KindScrape,
		Status:    model.StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Params:    map[string]any{"url": "https://example.com"},
	}
	if err := srv.store.Create(ctx, job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	// Create the result directory but not the file
	resultDir := filepath.Join(srv.store.DataDir(), "jobs", jobID)
	if err := fsutil.MkdirAllSecure(resultDir); err != nil {
		t.Fatalf("failed to create result directory: %v", err)
	}

	// Use a path within the allowed directory that doesn't exist
	resultPath := filepath.Join(resultDir, "results.jsonl")

	testJob := model.Job{
		ID:         jobID,
		ResultPath: resultPath,
	}

	_, err := srv.loadAllJobResults(testJob)
	if err == nil {
		t.Error("expected error for missing file")
	}
}
