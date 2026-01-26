// Package api provides integration tests for health check endpoint (/healthz).
// Tests verify health endpoint returns correct status and includes required fields.
// Does NOT test individual component health checks in depth.
package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealth(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/healthz", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if !strings.Contains(rr.Body.String(), `"status":"ok"`) {
		t.Errorf("handler returned unexpected body: got %v", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"database"`) {
		t.Errorf("handler missing database status: got %v", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"queue"`) {
		t.Errorf("handler missing queue status: got %v", rr.Body.String())
	}
}
