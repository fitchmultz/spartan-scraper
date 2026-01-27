package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"spartan-scraper/internal/model"
)

func TestHandleJobs_RedactsSensitiveData(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Create jobs with sensitive data
	job1 := model.Job{
		ID:         "job-1",
		Kind:       model.KindScrape,
		Status:     model.StatusSucceeded,
		ResultPath: "/Users/admin/.data/results/job-1.jsonl",
		Params: map[string]interface{}{
			"url":      "https://example.com",
			"password": "secret123",
			"apiKey":   "abc-def",
		},
	}
	job2 := model.Job{
		ID:         "job-2",
		Kind:       model.KindCrawl,
		Status:     model.StatusRunning,
		ResultPath: "/home/user/results/job-2.jsonl",
		Params: map[string]interface{}{
			"url":   "https://test.com",
			"token": "bearer-token",
		},
	}

	// Store jobs directly
	if err := server.store.Create(ctx, job1); err != nil {
		t.Fatalf("failed to create job1: %v", err)
	}
	if err := server.store.Create(ctx, job2); err != nil {
		t.Fatalf("failed to create job2: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/jobs", nil)
	w := httptest.NewRecorder()

	server.handleJobs(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response struct {
		Jobs []map[string]interface{} `json:"jobs"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response.Jobs) != 2 {
		t.Fatalf("Expected 2 jobs, got %d", len(response.Jobs))
	}

	// Check that resultPath is not present in any job
	for _, job := range response.Jobs {
		if _, ok := job["resultPath"]; ok {
			t.Errorf("resultPath should not be present in job %v", job["id"])
		}
	}

	// Find job-1 and verify secrets are redacted
	var job1Response map[string]interface{}
	for _, job := range response.Jobs {
		if job["id"] == "job-1" {
			job1Response = job
			break
		}
	}

	if job1Response == nil {
		t.Fatal("job-1 not found in response")
	}

	params := job1Response["params"].(map[string]interface{})
	if params["password"] != "[REDACTED]" {
		t.Errorf("password should be redacted, got: %v", params["password"])
	}
	if params["apiKey"] != "[REDACTED]" {
		t.Errorf("apiKey should be redacted, got: %v", params["apiKey"])
	}
	if params["url"] != "https://example.com" {
		t.Errorf("url should not be redacted, got: %v", params["url"])
	}
}

func TestHandleJob_Get_RedactsSensitiveData(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Create job with sensitive data
	job := model.Job{
		ID:         "job-1",
		Kind:       model.KindScrape,
		Status:     model.StatusFailed,
		ResultPath: "/Users/admin/.data/results/job-1.jsonl",
		Params: map[string]interface{}{
			"url": "https://example.com",
			"auth": map[string]interface{}{
				"username": "admin",
				"password": "secret123",
			},
		},
		Error: "Failed to write to /Users/admin/.data/temp/file.txt: permission denied",
	}

	if err := server.store.Create(ctx, job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/jobs/job-1", nil)
	w := httptest.NewRecorder()

	server.handleJob(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var jobResponse map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &jobResponse); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify resultPath is not present
	if _, ok := jobResponse["resultPath"]; ok {
		t.Error("resultPath should not be present in response")
	}

	// Verify auth params are redacted
	params := jobResponse["params"].(map[string]interface{})
	if params["auth"] != "[REDACTED]" {
		t.Errorf("auth should be redacted, got: %v", params["auth"])
	}

	// Verify error has paths redacted
	errorMsg := jobResponse["error"].(string)
	if strings.Contains(errorMsg, "/Users/admin/.data") {
		t.Errorf("Error should not contain filesystem path, got: %s", errorMsg)
	}
	if !strings.Contains(errorMsg, "[REDACTED]") {
		t.Errorf("Error should contain [REDACTED] placeholder, got: %s", errorMsg)
	}
}

func TestHandleJobs_ByStatus_RedactsData(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Create job with sensitive data
	job := model.Job{
		ID:         "job-1",
		Kind:       model.KindScrape,
		Status:     model.StatusSucceeded,
		ResultPath: "/secret/path.jsonl",
		Params: map[string]interface{}{
			"secret": "top-secret",
		},
	}

	if err := server.store.Create(ctx, job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/jobs?status=succeeded", nil)
	w := httptest.NewRecorder()

	server.handleJobs(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()

	// Verify no filesystem paths in response
	if strings.Contains(body, "/secret/path") {
		t.Errorf("Response should not contain filesystem path, got: %s", body)
	}

	// Verify secrets are redacted
	if strings.Contains(body, "top-secret") {
		t.Errorf("Response should not contain secret value, got: %s", body)
	}
}

func TestHandleJobs_InvalidStatus(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/v1/jobs?status=invalid", nil)
	w := httptest.NewRecorder()

	server.handleJobs(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid status, got %d", w.Code)
	}
}

func TestHandleJob_NotFound(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/v1/jobs/nonexistent", nil)
	w := httptest.NewRecorder()

	server.handleJob(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for nonexistent job, got %d", w.Code)
	}
}

func TestHandleJob_JSONOmitsResultPath(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Create job with ResultPath
	job := model.Job{
		ID:         "job-1",
		Kind:       model.KindScrape,
		Status:     model.StatusSucceeded,
		ResultPath: "/very/secret/path.jsonl",
		Params:     map[string]interface{}{"url": "https://example.com"},
	}

	if err := server.store.Create(ctx, job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/jobs/job-1", nil)
	w := httptest.NewRecorder()

	server.handleJob(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()

	// Verify resultPath is not in the JSON at all (due to omitempty)
	if strings.Contains(body, "resultPath") {
		t.Errorf("JSON response should not contain resultPath field, got: %s", body)
	}

	// Verify the path itself is not in the response
	if strings.Contains(body, "/very/secret/path") {
		t.Errorf("JSON response should not contain filesystem path, got: %s", body)
	}
}

func TestHandleJob_HeadersRedacted(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Create job with headers
	job := model.Job{
		ID:     "job-1",
		Kind:   model.KindScrape,
		Status: model.StatusRunning,
		Params: map[string]interface{}{
			"url": "https://example.com",
			"headers": map[string]interface{}{
				"Authorization": "Bearer secret-token",
				"Content-Type":  "application/json",
				"X-Custom":      "custom-value",
			},
		},
	}

	if err := server.store.Create(ctx, job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/jobs/job-1", nil)
	w := httptest.NewRecorder()

	server.handleJob(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	params := response["params"].(map[string]interface{})
	headers := params["headers"].(map[string]interface{})

	if headers["Authorization"] != "[REDACTED]" {
		t.Errorf("Authorization header should be redacted, got: %v", headers["Authorization"])
	}
	if headers["Content-Type"] != "application/json" {
		t.Errorf("Content-Type header should not be redacted, got: %v", headers["Content-Type"])
	}
	if headers["X-Custom"] != "custom-value" {
		t.Errorf("X-Custom header should not be redacted, got: %v", headers["X-Custom"])
	}
}
