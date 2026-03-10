// Package api provides integration tests for templates endpoint (/v1/templates).
// Tests cover template listing.
// Does NOT test template creation, modification, or deletion logic.
package api

import (
	"bytes"
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

func TestHandleTemplateLifecycle(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	createBody := []byte(`{
		"name": "portfolio-template",
		"selectors": [{"name":"title","selector":"h1","attr":"text","trim":true}]
	}`)
	createReq := httptest.NewRequest(http.MethodPost, "/v1/templates", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRes := httptest.NewRecorder()
	srv.Routes().ServeHTTP(createRes, createReq)

	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d: %s", createRes.Code, createRes.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v1/templates/portfolio-template", nil)
	getRes := httptest.NewRecorder()
	srv.Routes().ServeHTTP(getRes, getReq)

	if getRes.Code != http.StatusOK {
		t.Fatalf("expected get status 200, got %d: %s", getRes.Code, getRes.Body.String())
	}

	updateBody := []byte(`{
		"name": "renamed-template",
		"selectors": [{"name":"headline","selector":"h1","attr":"text","trim":true}]
	}`)
	updateReq := httptest.NewRequest(http.MethodPut, "/v1/templates/portfolio-template", bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateRes := httptest.NewRecorder()
	srv.Routes().ServeHTTP(updateRes, updateReq)

	if updateRes.Code != http.StatusOK {
		t.Fatalf("expected update status 200, got %d: %s", updateRes.Code, updateRes.Body.String())
	}

	oldGetReq := httptest.NewRequest(http.MethodGet, "/v1/templates/portfolio-template", nil)
	oldGetRes := httptest.NewRecorder()
	srv.Routes().ServeHTTP(oldGetRes, oldGetReq)
	if oldGetRes.Code != http.StatusNotFound {
		t.Fatalf("expected old name to return 404 after rename, got %d: %s", oldGetRes.Code, oldGetRes.Body.String())
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/v1/templates/renamed-template", nil)
	deleteRes := httptest.NewRecorder()
	srv.Routes().ServeHTTP(deleteRes, deleteReq)

	if deleteRes.Code != http.StatusNoContent {
		t.Fatalf("expected delete status 204, got %d: %s", deleteRes.Code, deleteRes.Body.String())
	}
}

func TestHandleTemplateRejectsReservedRename(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	createBody := []byte(`{
		"name": "custom-template",
		"selectors": [{"name":"title","selector":"h1","attr":"text","trim":true}]
	}`)
	createReq := httptest.NewRequest(http.MethodPost, "/v1/templates", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRes := httptest.NewRecorder()
	srv.Routes().ServeHTTP(createRes, createReq)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d: %s", createRes.Code, createRes.Body.String())
	}

	updateBody := []byte(`{
		"name": "default",
		"selectors": [{"name":"title","selector":"h1","attr":"text","trim":true}]
	}`)
	updateReq := httptest.NewRequest(http.MethodPut, "/v1/templates/custom-template", bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateRes := httptest.NewRecorder()
	srv.Routes().ServeHTTP(updateRes, updateReq)

	if updateRes.Code != http.StatusBadRequest {
		t.Fatalf("expected update status 400, got %d: %s", updateRes.Code, updateRes.Body.String())
	}
}
