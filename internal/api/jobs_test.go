// Package api provides integration tests for jobs endpoints (/v1/jobs).
// Tests cover job listing, filtering by status, deletion, and cancellation.
// Does NOT test individual job creation, execution, or result retrieval handlers.
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

func TestHandleJobs(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/v1/jobs", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %v", ct)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Errorf("failed to parse JSON response: %v", err)
	}
	if _, ok := resp["jobs"]; !ok {
		t.Errorf("expected 'jobs' field in response, got: %v", resp)
	}
}

func TestHandleJobsWithStatusFilter(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	jobs := []struct {
		body   string
		status model.Status
	}{
		{`{"url": "https://example1.com"}`, model.StatusQueued},
		{`{"url": "https://example2.com"}`, model.StatusRunning},
		{`{"url": "https://example3.com"}`, model.StatusSucceeded},
		{`{"url": "https://example4.com"}`, model.StatusFailed},
	}

	jobIDs := make([]string, 0, len(jobs))
	for i, job := range jobs {
		var scrapeReq map[string]interface{}
		if err := json.Unmarshal([]byte(job.body), &scrapeReq); err != nil {
			t.Fatalf("failed to unmarshal request: %v", err)
		}

		newJob := model.Job{
			ID:        fmt.Sprintf("550e8400-e29b-41d4-a716-44665544000%d", i),
			Kind:      model.KindScrape,
			Status:    job.status,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Spec:      scrapeReq,
		}

		if err := srv.store.Create(context.Background(), newJob); err != nil {
			t.Fatalf("failed to create job: %v", err)
		}
		jobIDs = append(jobIDs, newJob.ID)
	}

	req := httptest.NewRequest("GET", "/v1/jobs?status=queued", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %v", status)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	jobsList, ok := resp["jobs"].([]interface{})
	if !ok {
		t.Fatal("expected 'jobs' array in response")
	}

	if len(jobsList) != 1 {
		t.Errorf("expected 1 queued job, got %d", len(jobsList))
	}
}

func TestHandleJobsWithInvalidStatus(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/v1/jobs?status=invalid_status", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid status, got %v", status)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	if _, ok := resp["error"]; !ok {
		t.Error("expected 'error' field in response")
	}
}

func TestHandleJobsNoStatusFilter(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	for i := 0; i < 3; i++ {
		body := fmt.Sprintf(`{"url": "https://example%d.com"}`, i)
		var scrapeReq map[string]interface{}
		if err := json.Unmarshal([]byte(body), &scrapeReq); err != nil {
			t.Fatalf("failed to unmarshal request: %v", err)
		}

		newJob := model.Job{
			ID:        fmt.Sprintf("550e8400-e29b-41d4-a716-44665544000%d", i),
			Kind:      model.KindScrape,
			Status:    model.StatusQueued,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Spec:      scrapeReq,
		}

		if err := srv.store.Create(context.Background(), newJob); err != nil {
			t.Fatalf("failed to create job: %v", err)
		}
	}

	req := httptest.NewRequest("GET", "/v1/jobs", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %v", status)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	jobsList, ok := resp["jobs"].([]interface{})
	if !ok {
		t.Fatal("expected 'jobs' array in response")
	}

	if len(jobsList) != 3 {
		t.Errorf("expected all 3 jobs without filter, got %d", len(jobsList))
	}
}

func TestHandleJobsPagination(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create 10 jobs
	for i := 0; i < 10; i++ {
		newJob := model.Job{
			ID:        fmt.Sprintf("550e8400-e29b-41d4-a716-44665544000%d", i),
			Kind:      model.KindScrape,
			Status:    model.StatusQueued,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Spec:      map[string]interface{}{"url": "https://example.com"},
		}
		if err := srv.store.Create(context.Background(), newJob); err != nil {
			t.Fatalf("failed to create job: %v", err)
		}
	}

	tests := []struct {
		name           string
		query          string
		expectedStatus int
		expectedCount  int
	}{
		{"limit 5", "?limit=5", http.StatusOK, 5},
		{"limit 2", "?limit=2", http.StatusOK, 2},
		{"offset 8", "?offset=8", http.StatusOK, 2},
		{"limit 5 offset 8", "?limit=5&offset=8", http.StatusOK, 2},
		{"invalid limit", "?limit=abc", http.StatusBadRequest, 0},
		{"negative limit", "?limit=-1", http.StatusBadRequest, 0},
		{"invalid offset", "?offset=abc", http.StatusBadRequest, 0},
		{"negative offset", "?offset=-5", http.StatusBadRequest, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/v1/jobs"+tt.query, nil)
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("expected status %d, got %v", tt.expectedStatus, status)
			}

			if tt.expectedStatus == http.StatusOK {
				// Verify X-Total-Count header
				if total := rr.Header().Get("X-Total-Count"); total != "10" {
					t.Errorf("expected X-Total-Count 10, got %s", total)
				}

				var resp map[string]interface{}
				json.Unmarshal(rr.Body.Bytes(), &resp)
				jobsList := resp["jobs"].([]interface{})
				if len(jobsList) != tt.expectedCount {
					t.Errorf("expected %d jobs, got %d", tt.expectedCount, len(jobsList))
				}
			} else {
				// For error cases, verify error response
				var resp map[string]interface{}
				json.Unmarshal(rr.Body.Bytes(), &resp)
				if _, ok := resp["error"]; !ok {
					t.Error("expected 'error' field in response for invalid pagination")
				}
			}
		})
	}
}

func TestHandleJobRouting(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "missing id",
			method:         "GET",
			path:           "/v1/jobs/",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "unexpected trailing segment",
			method:         "GET",
			path:           "/v1/jobs/550e8400-e29b-41d4-a716-446655440001/invalid",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "method not allowed on base",
			method:         "POST",
			path:           "/v1/jobs",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "method not allowed on id",
			method:         "POST",
			path:           "/v1/jobs/550e8400-e29b-41d4-a716-446655440001",
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("%s: expected status %v, got %v", tt.name, tt.expectedStatus, status)
			}
		})
	}
}

func TestHandleJobForceDelete(t *testing.T) {
	srv, cleanup := setupTestServerWithConcurrency(t, 0)
	defer cleanup()

	body := `{"url": "https://example.com"}`
	req := httptest.NewRequest("POST", "/v1/scrape", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("failed to create job: got status %v, body: %s", status, rr.Body.String())
	}

	var job map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &job); err != nil {
		t.Fatalf("failed to parse job response: %v", err)
	}

	jobID, ok := job["id"].(string)
	if !ok {
		t.Fatalf("job response missing id field")
	}

	dataDir := srv.cfg.DataDir
	jobDir := filepath.Join(dataDir, "jobs", jobID)
	if err := fsutil.MkdirAllSecure(jobDir); err != nil {
		t.Fatalf("failed to create job directory: %v", err)
	}

	resultPath := filepath.Join(jobDir, "results.jsonl")
	resultContent := `{"test":"data"}`
	if err := os.WriteFile(resultPath, []byte(resultContent), 0o644); err != nil {
		t.Fatalf("failed to write result file: %v", err)
	}

	st := srv.store
	ctx := context.Background()
	if err := st.UpdateResultPath(ctx, jobID, resultPath); err != nil {
		t.Fatalf("failed to update result path: %v", err)
	}

	req = httptest.NewRequest("DELETE", fmt.Sprintf("/v1/jobs/%s?force=true", jobID), nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("force delete failed: got status %v, body: %s", status, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse delete response: %v", err)
	}

	if status, ok := resp["status"].(string); !ok || status != "deleted" {
		t.Errorf("expected status 'deleted', got %v", resp["status"])
	}

	_, err := st.Get(ctx, jobID)
	if err == nil {
		t.Error("job should be deleted from database after force delete")
	}

	if _, err := os.Stat(resultPath); !os.IsNotExist(err) {
		t.Error("result file should be deleted after force delete")
	}

	if _, err := os.Stat(jobDir); !os.IsNotExist(err) {
		t.Error("job directory should be deleted after force delete")
	}
}

func TestHandleJobCancelNotDelete(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{"url": "https://example.com"}`
	req := httptest.NewRequest("POST", "/v1/scrape", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("failed to create job: got status %v", status)
	}

	var job map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &job); err != nil {
		t.Fatalf("failed to parse job response: %v", err)
	}

	jobID, ok := job["id"].(string)
	if !ok {
		t.Fatalf("job response missing id field")
	}

	dataDir := srv.cfg.DataDir
	jobDir := filepath.Join(dataDir, "jobs", jobID)
	if err := fsutil.MkdirAllSecure(jobDir); err != nil {
		t.Fatalf("failed to create job directory: %v", err)
	}

	resultPath := filepath.Join(jobDir, "results.jsonl")
	resultContent := `{"test":"data"}`
	if err := os.WriteFile(resultPath, []byte(resultContent), 0o644); err != nil {
		t.Fatalf("failed to write result file: %v", err)
	}

	st := srv.store
	ctx := context.Background()
	if err := st.UpdateResultPath(ctx, jobID, resultPath); err != nil {
		t.Fatalf("failed to update result path: %v", err)
	}

	req = httptest.NewRequest("DELETE", fmt.Sprintf("/v1/jobs/%s", jobID), nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("cancel failed: got status %v", status)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse delete response: %v", err)
	}

	if status, ok := resp["status"].(string); !ok || status != "canceled" {
		t.Errorf("expected status 'canceled', got %v", resp["status"])
	}

	gotJob, err := st.Get(ctx, jobID)
	if err != nil {
		t.Error("job should still exist in database after cancel")
	}
	if gotJob.Status != model.StatusCanceled {
		t.Errorf("job status should be 'canceled', got %s", gotJob.Status)
	}

	if _, err := os.Stat(resultPath); os.IsNotExist(err) {
		t.Error("result file should still exist after cancel")
	}

	if _, err := os.Stat(jobDir); os.IsNotExist(err) {
		t.Error("job directory should still exist after cancel")
	}
}

func TestHandleJobForceDeletePathTraversal(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Test various path traversal attempts (encoded to bypass router path cleaning)
	traversalAttempts := []string{
		"..%2f..%2f..%2fetc%2fpasswd",
		"..%5c..%5c..%5cwindows%5csystem32",
		".%2f..%2fetc",
	}

	for _, maliciousID := range traversalAttempts {
		t.Run(fmt.Sprintf("traversal_%s", maliciousID), func(t *testing.T) {
			req := httptest.NewRequest("DELETE", fmt.Sprintf("/v1/jobs/%s?force=true", maliciousID), nil)
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			// Should return 400 Bad Request for invalid ID format
			if status := rr.Code; status != http.StatusBadRequest {
				t.Errorf("expected status 400 for path traversal attempt %q, got %v", maliciousID, status)
			}
		})
	}
}

func TestHandleJobForceDeleteNonExistent(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Use a valid UUID format that doesn't exist
	nonExistentID := "550e8400-e29b-41d4-a716-446655440000"

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/v1/jobs/%s?force=true", nonExistentID), nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	// Should return 404 Not Found
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("expected status 404 for non-existent job, got %v", status)
	}
}

func TestHandleJobForceDeleteInvalidID(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	invalidIDs := []string{
		"not-a-uuid",
		"123",
		"test-job-with-special-chars!",
	}

	for _, invalidID := range invalidIDs {
		t.Run(fmt.Sprintf("invalid_%s", invalidID), func(t *testing.T) {
			req := httptest.NewRequest("DELETE", fmt.Sprintf("/v1/jobs/%s?force=true", invalidID), nil)
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			// Should return 400 Bad Request
			if status := rr.Code; status != http.StatusBadRequest {
				t.Errorf("expected status 400 for invalid id %q, got %v", invalidID, status)
			}
		})
	}
}

func TestHandleJobGetInvalidID(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	invalidIDs := []string{
		"not-a-uuid",
		"123",
	}

	for _, invalidID := range invalidIDs {
		t.Run(fmt.Sprintf("invalid_%s", invalidID), func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/v1/jobs/%s", invalidID), nil)
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			// Should return 400 Bad Request
			if status := rr.Code; status != http.StatusBadRequest {
				t.Errorf("expected status 400 for invalid id %q, got %v", invalidID, status)
			}
		})
	}
}

func TestHandleJobGetPathTraversal(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Use encoded path traversal to bypass router's path cleaning
	traversalIDs := []string{
		"..%2f..%2f..%2fetc%2fpasswd",
		".%2f..%2fetc",
	}

	for _, invalidID := range traversalIDs {
		t.Run(fmt.Sprintf("traversal_%s", invalidID), func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/v1/jobs/%s", invalidID), nil)
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			// Should return 400 Bad Request
			if status := rr.Code; status != http.StatusBadRequest {
				t.Errorf("expected status 400 for path traversal id %q, got %v", invalidID, status)
			}
		})
	}
}
