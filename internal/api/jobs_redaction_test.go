// Package api provides integration tests for sensitive data redaction in job responses.
// Tests cover redaction of cookies, tokens, and passwords in API responses.
// Does NOT test auth resolution logic (auth package handles that).
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestHandleJobs_RedactsSensitiveData(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Create jobs with sensitive data (using valid UUIDs)
	job1 := model.Job{
		ID:          "550e8400-e29b-41d4-a716-446655440001",
		Kind:        model.KindScrape,
		Status:      model.StatusSucceeded,
		SpecVersion: -1,
		ResultPath:  "/Users/admin/.data/results/job-1.jsonl",
		Spec: map[string]interface{}{
			"url":      "https://example.com",
			"password": "secret123",
			"apiKey":   "abc-def",
		},
	}
	job2 := model.Job{
		ID:          "550e8400-e29b-41d4-a716-446655440002",
		Kind:        model.KindCrawl,
		Status:      model.StatusRunning,
		SpecVersion: -1,
		ResultPath:  "/home/user/results/job-2.jsonl",
		Spec: map[string]interface{}{
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

	// Check that local filesystem fields are not present in any job
	for _, job := range response.Jobs {
		if _, ok := job["resultPath"]; ok {
			t.Errorf("resultPath should not be present in job %v", job["id"])
		}
		if _, ok := job["screenshotPath"]; ok {
			t.Errorf("screenshotPath should not be present in job %v", job["id"])
		}
	}

	// Find job1 and verify secrets are redacted
	var job1Response map[string]interface{}
	for _, job := range response.Jobs {
		if job["id"] == "550e8400-e29b-41d4-a716-446655440001" {
			job1Response = job
			break
		}
	}

	if job1Response == nil {
		t.Fatal("job1 not found in response")
	}

	params := job1Response["spec"].(map[string]interface{})
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

	// Create job with sensitive data (using valid UUID)
	job := model.Job{
		ID:          "550e8400-e29b-41d4-a716-446655440001",
		Kind:        model.KindScrape,
		Status:      model.StatusFailed,
		SpecVersion: -1,
		ResultPath:  "/Users/admin/.data/results/job-1.jsonl",
		Spec: map[string]interface{}{
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

	req := httptest.NewRequest(http.MethodGet, "/v1/jobs/550e8400-e29b-41d4-a716-446655440001", nil)
	w := httptest.NewRecorder()

	server.handleJob(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var envelope map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	jobResponse, ok := envelope["job"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected job envelope, got %#v", envelope)
	}

	// Verify local filesystem fields are not present
	if _, ok := jobResponse["resultPath"]; ok {
		t.Error("resultPath should not be present in response")
	}
	if _, ok := jobResponse["screenshotPath"]; ok {
		t.Error("screenshotPath should not be present in response")
	}

	// Verify auth params are redacted
	params := jobResponse["spec"].(map[string]interface{})
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

	// Create job with sensitive data (using valid UUID)
	job := model.Job{
		ID:          "550e8400-e29b-41d4-a716-446655440001",
		Kind:        model.KindScrape,
		Status:      model.StatusSucceeded,
		SpecVersion: -1,
		ResultPath:  "/secret/path.jsonl",
		Spec: map[string]interface{}{
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

	// Use a valid UUID format that doesn't exist
	req := httptest.NewRequest(http.MethodGet, "/v1/jobs/550e8400-e29b-41d4-a716-446655440999", nil)
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

	// Create job with ResultPath (using valid UUID)
	job := model.Job{
		ID:          "550e8400-e29b-41d4-a716-446655440001",
		Kind:        model.KindScrape,
		Status:      model.StatusSucceeded,
		SpecVersion: -1,
		ResultPath:  "/very/secret/path.jsonl",
		Spec:        map[string]interface{}{"url": "https://example.com"},
	}

	if err := server.store.Create(ctx, job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/jobs/550e8400-e29b-41d4-a716-446655440001", nil)
	w := httptest.NewRecorder()

	server.handleJob(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()

	// Verify local filesystem fields are not in the JSON at all (due to omitempty)
	if strings.Contains(body, "resultPath") {
		t.Errorf("JSON response should not contain resultPath field, got: %s", body)
	}
	if strings.Contains(body, "screenshotPath") {
		t.Errorf("JSON response should not contain screenshotPath field, got: %s", body)
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

	// Create job with headers (using valid UUID)
	job := model.Job{
		ID:          "550e8400-e29b-41d4-a716-446655440001",
		Kind:        model.KindScrape,
		Status:      model.StatusRunning,
		SpecVersion: -1,
		Spec: map[string]interface{}{
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

	req := httptest.NewRequest(http.MethodGet, "/v1/jobs/550e8400-e29b-41d4-a716-446655440001", nil)
	w := httptest.NewRecorder()

	server.handleJob(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var envelope map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	response, ok := envelope["job"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected job envelope, got %#v", envelope)
	}

	params := response["spec"].(map[string]interface{})
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
