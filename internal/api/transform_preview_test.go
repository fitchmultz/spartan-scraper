// Package api provides integration tests for the preview-transform endpoint.
// Tests cover JMESPath and JSONata transformations, error handling for invalid
// expressions, job state validation, and result limiting.
// Does NOT test the validate endpoint or helper functions directly.
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestHandlePreviewTransform_JMESPathSuccess(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a job with results
	jobID := "test-job-001"
	job := model.Job{
		ID:         jobID,
		Kind:       model.KindScrape,
		Status:     model.StatusSucceeded,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Spec:       map[string]interface{}{"url": "https://example.com"},
		ResultPath: filepath.Join(srv.cfg.DataDir, "jobs", jobID, "results.jsonl"),
	}

	// Create results directory and file
	jobDir := filepath.Join(srv.cfg.DataDir, "jobs", jobID)
	if err := os.MkdirAll(jobDir, 0755); err != nil {
		t.Fatalf("failed to create job dir: %v", err)
	}

	// Write test results
	results := []map[string]interface{}{
		{"title": "First Article", "url": "https://example.com/1", "views": 100},
		{"title": "Second Article", "url": "https://example.com/2", "views": 200},
	}
	file, err := os.Create(job.ResultPath)
	if err != nil {
		t.Fatalf("failed to create results file: %v", err)
	}
	for _, r := range results {
		data, _ := json.Marshal(r)
		file.WriteString(string(data) + "\n")
	}
	file.Close()

	if err := srv.store.Create(context.Background(), job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	// Test JMESPath projection
	reqBody := TransformPreviewRequest{
		JobID:      jobID,
		Expression: "{title: title, url: url}",
		Language:   "jmespath",
		Limit:      10,
	}
	body, _ := json.Marshal(reqBody)

	req := newJSONRequest("POST", "/v1/jobs/"+jobID+"/preview-transform", body)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %v: %s", status, rr.Body.String())
	}

	var resp TransformPreviewResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.ResultCount != 2 {
		t.Errorf("expected 2 results, got %d", resp.ResultCount)
	}

	if resp.Error != "" {
		t.Errorf("unexpected error: %s", resp.Error)
	}

	// Verify transformed results contain only title and url
	if len(resp.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(resp.Results))
	}
	for i, r := range resp.Results {
		result, ok := r.(map[string]interface{})
		if !ok {
			t.Fatalf("result %d is not an object", i)
		}
		if _, hasTitle := result["title"]; !hasTitle {
			t.Errorf("result %d missing title", i)
		}
		if _, hasURL := result["url"]; !hasURL {
			t.Errorf("result %d missing url", i)
		}
		if _, hasViews := result["views"]; hasViews {
			t.Errorf("result %d should not have views field", i)
		}
	}
}

func TestHandlePreviewTransform_JSONataSuccess(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a job with results
	jobID := "test-job-002"
	job := model.Job{
		ID:         jobID,
		Kind:       model.KindScrape,
		Status:     model.StatusSucceeded,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Spec:       map[string]interface{}{"url": "https://example.com"},
		ResultPath: filepath.Join(srv.cfg.DataDir, "jobs", jobID, "results.jsonl"),
	}

	// Create results directory and file
	jobDir := filepath.Join(srv.cfg.DataDir, "jobs", jobID)
	if err := os.MkdirAll(jobDir, 0755); err != nil {
		t.Fatalf("failed to create job dir: %v", err)
	}

	// Write test results
	results := []map[string]interface{}{
		{"name": "Product A", "price": 100, "quantity": 2},
		{"name": "Product B", "price": 50, "quantity": 3},
	}
	file, err := os.Create(job.ResultPath)
	if err != nil {
		t.Fatalf("failed to create results file: %v", err)
	}
	for _, r := range results {
		data, _ := json.Marshal(r)
		file.WriteString(string(data) + "\n")
	}
	file.Close()

	if err := srv.store.Create(context.Background(), job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	// Test JSONata transformation with calculation
	reqBody := TransformPreviewRequest{
		JobID:      jobID,
		Expression: `{"item": name, "total": price * quantity}`,
		Language:   "jsonata",
		Limit:      10,
	}
	body, _ := json.Marshal(reqBody)

	req := newJSONRequest("POST", "/v1/jobs/"+jobID+"/preview-transform", body)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %v: %s", status, rr.Body.String())
	}

	var resp TransformPreviewResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.ResultCount != 2 {
		t.Errorf("expected 2 results, got %d", resp.ResultCount)
	}

	if resp.Error != "" {
		t.Errorf("unexpected error: %s", resp.Error)
	}
}

func TestHandlePreviewTransform_InvalidExpression(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a job with results
	jobID := "test-job-003"
	job := model.Job{
		ID:         jobID,
		Kind:       model.KindScrape,
		Status:     model.StatusSucceeded,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Spec:       map[string]interface{}{"url": "https://example.com"},
		ResultPath: filepath.Join(srv.cfg.DataDir, "jobs", jobID, "results.jsonl"),
	}

	// Create results directory and file
	jobDir := filepath.Join(srv.cfg.DataDir, "jobs", jobID)
	if err := os.MkdirAll(jobDir, 0755); err != nil {
		t.Fatalf("failed to create job dir: %v", err)
	}

	// Write test result
	file, err := os.Create(job.ResultPath)
	if err != nil {
		t.Fatalf("failed to create results file: %v", err)
	}
	data, _ := json.Marshal(map[string]interface{}{"title": "Test"})
	file.WriteString(string(data) + "\n")
	file.Close()

	if err := srv.store.Create(context.Background(), job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	// Test invalid JMESPath expression
	reqBody := TransformPreviewRequest{
		JobID:      jobID,
		Expression: "{title: ", // Invalid syntax
		Language:   "jmespath",
		Limit:      10,
	}
	body, _ := json.Marshal(reqBody)

	req := newJSONRequest("POST", "/v1/jobs/"+jobID+"/preview-transform", body)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	// Should return 200 with error in response body (not HTTP error)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %v", status)
	}

	var resp TransformPreviewResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Error == "" {
		t.Error("expected error in response for invalid expression")
	}
}

func TestHandlePreviewTransform_JobNotFound(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	reqBody := TransformPreviewRequest{
		JobID:      "non-existent-job",
		Expression: "{title: title}",
		Language:   "jmespath",
		Limit:      10,
	}
	body, _ := json.Marshal(reqBody)

	req := newJSONRequest("POST", "/v1/jobs/non-existent-job/preview-transform", body)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("expected status 404, got %v", status)
	}
}

func TestHandlePreviewTransform_JobNoResults(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a job without results
	jobID := "test-job-004"
	job := model.Job{
		ID:         jobID,
		Kind:       model.KindScrape,
		Status:     model.StatusSucceeded,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Spec:       map[string]interface{}{"url": "https://example.com"},
		ResultPath: "", // No result path
	}

	if err := srv.store.Create(context.Background(), job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	reqBody := TransformPreviewRequest{
		JobID:      jobID,
		Expression: "{title: title}",
		Language:   "jmespath",
		Limit:      10,
	}
	body, _ := json.Marshal(reqBody)

	req := newJSONRequest("POST", "/v1/jobs/"+jobID+"/preview-transform", body)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("expected status 404, got %v: %s", status, rr.Body.String())
	}
}

func TestHandlePreviewTransform_JobNotReady(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	testCases := []struct {
		name   string
		status model.Status
	}{
		{"queued", model.StatusQueued},
		{"running", model.StatusRunning},
		{"failed", model.StatusFailed},
		{"canceled", model.StatusCanceled},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jobID := "test-job-" + tc.name
			job := model.Job{
				ID:        jobID,
				Kind:      model.KindScrape,
				Status:    tc.status,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Spec:      map[string]interface{}{"url": "https://example.com"},
			}

			if err := srv.store.Create(context.Background(), job); err != nil {
				t.Fatalf("failed to create job: %v", err)
			}

			reqBody := TransformPreviewRequest{
				JobID:      jobID,
				Expression: "{title: title}",
				Language:   "jmespath",
				Limit:      10,
			}
			body, _ := json.Marshal(reqBody)

			req := newJSONRequest("POST", "/v1/jobs/"+jobID+"/preview-transform", body)
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if status := rr.Code; status != http.StatusBadRequest {
				t.Errorf("expected status 400, got %v: %s", status, rr.Body.String())
			}
		})
	}
}

func TestHandlePreviewTransform_LimitBounds(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a job with results
	jobID := "test-job-limit"
	job := model.Job{
		ID:         jobID,
		Kind:       model.KindScrape,
		Status:     model.StatusSucceeded,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Spec:       map[string]interface{}{"url": "https://example.com"},
		ResultPath: filepath.Join(srv.cfg.DataDir, "jobs", jobID, "results.jsonl"),
	}

	// Create results directory and file
	jobDir := filepath.Join(srv.cfg.DataDir, "jobs", jobID)
	if err := os.MkdirAll(jobDir, 0755); err != nil {
		t.Fatalf("failed to create job dir: %v", err)
	}

	// Write 20 test results
	file, err := os.Create(job.ResultPath)
	if err != nil {
		t.Fatalf("failed to create results file: %v", err)
	}
	for i := 0; i < 20; i++ {
		data, _ := json.Marshal(map[string]interface{}{"id": i})
		file.WriteString(string(data) + "\n")
	}
	file.Close()

	if err := srv.store.Create(context.Background(), job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	// Test with limit=0 (should default to 10)
	reqBody := TransformPreviewRequest{
		JobID:      jobID,
		Expression: "@",
		Language:   "jmespath",
		Limit:      0,
	}
	body, _ := json.Marshal(reqBody)

	req := newJSONRequest("POST", "/v1/jobs/"+jobID+"/preview-transform", body)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	var resp TransformPreviewResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.ResultCount != 10 {
		t.Errorf("expected 10 results (default), got %d", resp.ResultCount)
	}

	// Test with limit > 100 (should cap at 100)
	reqBody.Limit = 200
	body, _ = json.Marshal(reqBody)

	req = newJSONRequest("POST", "/v1/jobs/"+jobID+"/preview-transform", body)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.ResultCount != 20 {
		t.Errorf("expected 20 results (all available), got %d", resp.ResultCount)
	}
}
