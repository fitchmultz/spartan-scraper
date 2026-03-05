// Package api provides tests for traffic replay handler validation.
//
// This file contains tests for HTTP handler validation logic including
// method validation, job ID extraction, body parsing, and URL validation.
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestHandleTrafficReplayMethodNotAllowed(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/v1/jobs/replay/test-job", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("expected status %v, got %v", http.StatusMethodNotAllowed, status)
	}
}

func TestHandleTrafficReplayMissingJobID(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("POST", "/v1/jobs/replay/", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("expected status %v, got %v", http.StatusBadRequest, status)
	}
}

func TestHandleTrafficReplayInvalidBody(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("POST", "/v1/jobs/replay/test-job", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("expected status %v, got %v", http.StatusBadRequest, status)
	}
}

func TestHandleTrafficReplayMissingTargetURL(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{"jobId": "test-job"}`
	req := httptest.NewRequest("POST", "/v1/jobs/replay/test-job", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("expected status %v, got %v", http.StatusBadRequest, status)
	}
}

func TestHandleTrafficReplayInvalidTargetURL(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{"jobId": "test-job", "targetBaseUrl": "://invalid-url"}`
	req := httptest.NewRequest("POST", "/v1/jobs/replay/test-job", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("expected status %v, got %v", http.StatusBadRequest, status)
	}
}

func TestHandleTrafficReplayJobNotFound(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{"jobId": "nonexistent-job", "targetBaseUrl": "https://example.com"}`
	req := httptest.NewRequest("POST", "/v1/jobs/replay/nonexistent-job", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("expected status %v, got %v", http.StatusNotFound, status)
	}
}

func TestHandleTrafficReplayJobStatusValidation(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	ctx := t.Context()

	tests := []struct {
		name           string
		status         model.Status
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

			if err := srv.store.Create(ctx, job); err != nil {
				t.Fatalf("failed to create job: %v", err)
			}

			body := fmt.Sprintf(`{"jobId": "%s", "targetBaseUrl": "https://example.com"}`, jobID)
			req := httptest.NewRequest("POST", fmt.Sprintf("/v1/jobs/replay/%s", jobID), strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("expected status %v, got %v", tt.expectedStatus, status)
			}

			var resp map[string]any
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse error response: %v", err)
			}
			if msg, ok := resp["error"].(string); !ok || !strings.Contains(msg, tt.expectedMsg) {
				t.Errorf("expected error message to contain %q, got %q", tt.expectedMsg, msg)
			}
		})
	}
}
