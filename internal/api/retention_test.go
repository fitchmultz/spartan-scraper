// Package api provides integration tests for retention endpoints.
// Tests cover status retrieval and cleanup operations with dry-run mode.
// Does NOT test the retention engine logic itself (see retention package tests).
package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRetentionStatus(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/v1/retention/status", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var resp RetentionStatusResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Verify response has expected fields
	if resp.TotalJobs != 0 {
		t.Errorf("expected 0 jobs in fresh server, got %d", resp.TotalJobs)
	}
	if resp.Guidance == nil {
		t.Fatal("expected retention guidance in status response")
	}
	if resp.Guidance.Status != "disabled" {
		t.Fatalf("expected disabled guidance for fresh config, got %#v", resp.Guidance)
	}
}

func TestBuildRetentionCapabilityGuidanceDisabledIncludesActions(t *testing.T) {
	guidance := BuildRetentionCapabilityGuidance(RetentionStatusResponse{})
	if guidance == nil {
		t.Fatal("expected guidance")
	}
	if guidance.Status != "disabled" {
		t.Fatalf("status = %q, want disabled", guidance.Status)
	}
	if len(guidance.Actions) == 0 {
		t.Fatal("expected disabled guidance actions")
	}
}

func TestBuildRetentionCapabilityGuidanceWarningWhenJobsEligible(t *testing.T) {
	guidance := BuildRetentionCapabilityGuidance(RetentionStatusResponse{
		Enabled:      true,
		MaxJobs:      100,
		TotalJobs:    20,
		JobsEligible: 5,
	})
	if guidance == nil {
		t.Fatal("expected guidance")
	}
	if guidance.Status != "warning" {
		t.Fatalf("status = %q, want warning", guidance.Status)
	}
}

func TestRetentionStatusMethodNotAllowed(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// POST to status endpoint should fail
	req := httptest.NewRequest("POST", "/v1/retention/status", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("expected status %v, got %v", http.StatusMethodNotAllowed, status)
	}
}

func TestRetentionCleanupDryRun(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{"dryRun": true}`
	req := httptest.NewRequest("POST", "/v1/retention/cleanup", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v, body: %s", status, http.StatusOK, rr.Body.String())
	}

	var resp RetentionCleanupResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if !resp.DryRun {
		t.Error("expected dryRun to be true in response")
	}
}

func TestRetentionCleanupWithKind(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{"dryRun": true, "kind": "scrape"}`
	req := httptest.NewRequest("POST", "/v1/retention/cleanup", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestRetentionCleanupInvalidKind(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{"dryRun": true, "kind": "invalid"}`
	req := httptest.NewRequest("POST", "/v1/retention/cleanup", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("expected status %v, got %v", http.StatusBadRequest, status)
	}
}

func TestRetentionCleanupInvalidContentType(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("POST", "/v1/retention/cleanup", bytes.NewReader([]byte(`{}`)))
	// No Content-Type header
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusUnsupportedMediaType {
		t.Errorf("expected status %v, got %v", http.StatusUnsupportedMediaType, status)
	}
}

func TestRetentionCleanupMethodNotAllowed(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/v1/retention/cleanup", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("expected status %v, got %v", http.StatusMethodNotAllowed, status)
	}
}

func TestRetentionNotFound(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/v1/retention/invalid", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("expected status %v, got %v", http.StatusNotFound, status)
	}
}
