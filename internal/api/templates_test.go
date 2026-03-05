// Package api provides integration tests for templates endpoint (/v1/templates).
// Tests cover template listing.
// Does NOT test template creation, modification, or deletion logic.
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleTemplates(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/v1/templates", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %v", ct)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	templates, ok := response["templates"].([]interface{})
	if !ok {
		t.Fatal("expected templates array in response")
	}

	if len(templates) < 3 {
		t.Errorf("expected at least 3 templates, got %d", len(templates))
	}
}
