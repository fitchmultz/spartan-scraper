// Package api provides integration tests for job results endpoint basic retrieval and error handling.
// Tests cover 404, no results (queued/running), and granular error states (failed, canceled, succeeded edge cases).
// Does NOT test result file generation or export logic handled by exporter package.
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

	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestHandleJobResultsNotFound(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/v1/jobs/nonexistent-id/results", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("expected status %v, got %v", http.StatusNotFound, status)
	}

	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %v", ct)
	}
}

func TestHandleJobResultsNoResults(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{"url": "https://example.com"}`
	req := httptest.NewRequest("POST", "/v1/scrape", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Fatalf("failed to create job: got status %v", status)
	}

	var jobResp JobResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &jobResp); err != nil {
		t.Fatalf("failed to parse job response: %v", err)
	}

	jobID := jobResp.Job.ID
	if jobID == "" {
		t.Fatalf("job response missing id field")
	}

	req = httptest.NewRequest("GET", fmt.Sprintf("/v1/jobs/%s/results", jobID), nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("expected status %v for job with no results, got %v", http.StatusBadRequest, status)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	msg, _ := resp["error"].(string)
	isQueued := strings.Contains(msg, "job is queued and has no results yet")
	isRunning := strings.Contains(msg, "job is still running and has no results yet")
	if !isQueued && !isRunning {
		t.Errorf("expected error message to indicate queued or running, got %q", msg)
	}
}

func TestHandleJobResultsGranularErrors(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	tests := []struct {
		name           string
		status         model.Status
		setupFile      func(jobID string) string
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:           "queued",
			status:         model.StatusQueued,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "job is queued and has no results yet",
		},
		{
			name:           "running",
			status:         model.StatusRunning,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "job is still running and has no results yet",
		},
		{
			name:           "failed",
			status:         model.StatusFailed,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "job failed and produced no results",
		},
		{
			name:           "canceled",
			status:         model.StatusCanceled,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "job was canceled and produced no results",
		},
		{
			name:           "succeeded - no result path",
			status:         model.StatusSucceeded,
			setupFile:      func(jobID string) string { return "" },
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "job succeeded but no result path was recorded",
		},
		{
			name:   "succeeded - file missing",
			status: model.StatusSucceeded,
			setupFile: func(jobID string) string {
				// Return a path within the job directory that doesn't exist
				return filepath.Join(srv.store.DataDir(), "jobs", jobID, "results.jsonl")
			},
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "job succeeded but result file is missing",
		},
		{
			name:   "succeeded - file empty",
			status: model.StatusSucceeded,
			setupFile: func(jobID string) string {
				jobDir := filepath.Join(srv.store.DataDir(), "jobs", jobID)
				os.MkdirAll(jobDir, 0755)
				path := filepath.Join(jobDir, "results.jsonl")
				os.WriteFile(path, []byte(""), 0644)
				return path
			},
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "job succeeded but result file is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jobID := "test-job-" + strings.ReplaceAll(tt.name, " ", "-")
			job := model.Job{
				ID:        jobID,
				Kind:      model.KindScrape,
				Status:    tt.status,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if tt.setupFile != nil {
				job.ResultPath = tt.setupFile(jobID)
			}

			if err := srv.store.Create(ctx, job); err != nil {
				t.Fatalf("failed to create job: %v", err)
			}

			req := httptest.NewRequest("GET", fmt.Sprintf("/v1/jobs/%s/results", jobID), nil)
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("expected status %v, got %v", tt.expectedStatus, status)
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse error response: %v", err)
			}
			if msg, ok := resp["error"].(string); !ok || !strings.Contains(msg, tt.expectedMsg) {
				t.Errorf("expected error message to contain %q, got %q", tt.expectedMsg, msg)
			}
		})
	}
}

func TestHandleJobResultsPathTraversal(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	tests := []struct {
		name           string
		resultPath     string
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:           "classic traversal",
			resultPath:     "../../../../etc/passwd",
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "result path outside allowed directory",
		},
		{
			name:           "traversal within jobs dir",
			resultPath:     "jobs/../etc/passwd",
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "result path outside allowed directory",
		},
		{
			name:           "absolute path outside data dir",
			resultPath:     "/etc/passwd",
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "result path outside allowed directory",
		},
		{
			name:           "nested traversal",
			resultPath:     "jobs/valid-job-id/../../../etc/passwd",
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "result path outside allowed directory",
		},
		{
			name:           "traversal with null bytes",
			resultPath:     "../../../../etc/passwd\x00",
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "result path outside allowed directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jobID := "test-job-" + strings.ReplaceAll(tt.name, " ", "-")
			job := model.Job{
				ID:         jobID,
				Kind:       model.KindScrape,
				Status:     model.StatusSucceeded,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
				ResultPath: tt.resultPath,
			}

			if err := srv.store.Create(ctx, job); err != nil {
				t.Fatalf("failed to create job: %v", err)
			}

			req := httptest.NewRequest("GET", fmt.Sprintf("/v1/jobs/%s/results", jobID), nil)
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("expected status %v, got %v", tt.expectedStatus, status)
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse error response: %v", err)
			}
			if msg, ok := resp["error"].(string); !ok || !strings.Contains(msg, tt.expectedMsg) {
				t.Errorf("expected error message to contain %q, got %q", tt.expectedMsg, msg)
			}
		})
	}
}

func TestHandleJobResultsValidPath(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Create a valid job with a proper result path
	jobID := "test-valid-job"
	jobDir := filepath.Join(srv.store.DataDir(), "jobs", jobID)
	if err := os.MkdirAll(jobDir, 0755); err != nil {
		t.Fatalf("failed to create job directory: %v", err)
	}

	resultPath := filepath.Join(jobDir, "results.jsonl")
	if err := os.WriteFile(resultPath, []byte(`{"url":"https://example.com","content":"test"}`), 0644); err != nil {
		t.Fatalf("failed to create result file: %v", err)
	}

	job := model.Job{
		ID:         jobID,
		Kind:       model.KindScrape,
		Status:     model.StatusSucceeded,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		ResultPath: resultPath,
	}

	if err := srv.store.Create(ctx, job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	req := httptest.NewRequest("GET", fmt.Sprintf("/v1/jobs/%s/results", jobID), nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	// Should succeed with 200 OK
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status %v for valid path, got %v", http.StatusOK, status)
	}
}
