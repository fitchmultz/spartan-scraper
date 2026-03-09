// Package api provides integration tests for auth session endpoints.
//
// Purpose:
// - Verify session list/create/get/delete behavior through the HTTP layer.
//
// Responsibilities:
// - Assert response envelopes and status codes.
// - Cover missing-resource semantics for delete.
// - Confirm server-side normalization for saved sessions.
//
// Scope:
// - `/v1/auth/sessions` endpoints only.
//
// Usage:
// - Run with `go test ./internal/api`.
//
// Invariants/Assumptions:
// - Tests use the shared setupTestServer helper and isolated temp storage.
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleCreateSessionDefaultsNameToID(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/auth/sessions",
		strings.NewReader(`{"id":"portfolio-session","domain":"example.com"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response struct {
		Session struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"session"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Session.ID != "portfolio-session" {
		t.Fatalf("expected saved session id, got %q", response.Session.ID)
	}
	if response.Session.Name != "portfolio-session" {
		t.Fatalf("expected defaulted session name, got %q", response.Session.Name)
	}
}

func TestHandleDeleteSessionMissingReturnsNotFound(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodDelete, "/v1/auth/sessions/missing", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rr.Code, rr.Body.String())
	}
}
