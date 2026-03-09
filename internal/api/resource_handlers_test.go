// Package api provides regression tests for shared named-resource API handlers.
//
// Purpose:
// - Verify file-backed named-resource endpoints behave consistently after consolidation.
//
// Responsibilities:
// - Assert create/list/get/update flows for render profiles and pipeline JS scripts.
// - Assert URL path identifiers override mismatched request-body names on update.
//
// Scope:
// - API handler behavior only; underlying package validation and file formats are exercised indirectly.
//
// Usage:
// - Run with `go test ./internal/api/...`.
//
// Invariants/Assumptions:
// - The shared named-resource helper is the only implementation path for these endpoints.
// - Valid payloads require a name and host-pattern configuration.
package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

func TestRenderProfileCRUDUsesSharedNamedResourceFlow(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	createBody, err := json.Marshal(fetch.RenderProfile{
		Name:         "news",
		HostPatterns: []string{"example.com"},
	})
	if err != nil {
		t.Fatalf("marshal create body: %v", err)
	}

	createReq := httptest.NewRequest(http.MethodPost, "/v1/render-profiles", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createResp := httptest.NewRecorder()
	srv.Routes().ServeHTTP(createResp, createReq)

	if createResp.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", createResp.Code, createResp.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/v1/render-profiles", nil)
	listResp := httptest.NewRecorder()
	srv.Routes().ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("expected list status 200, got %d: %s", listResp.Code, listResp.Body.String())
	}

	updateBody, err := json.Marshal(fetch.RenderProfile{
		Name:         "ignored",
		HostPatterns: []string{"news.example.com"},
	})
	if err != nil {
		t.Fatalf("marshal update body: %v", err)
	}

	updateReq := httptest.NewRequest(http.MethodPut, "/v1/render-profiles/news", bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp := httptest.NewRecorder()
	srv.Routes().ServeHTTP(updateResp, updateReq)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("expected update status 200, got %d: %s", updateResp.Code, updateResp.Body.String())
	}

	var updated fetch.RenderProfile
	if err := json.Unmarshal(updateResp.Body.Bytes(), &updated); err != nil {
		t.Fatalf("unmarshal update response: %v", err)
	}
	if updated.Name != "news" {
		t.Fatalf("expected path name to win, got %q", updated.Name)
	}
}

func TestPipelineJSScriptCRUDUsesSharedNamedResourceFlow(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	createBody, err := json.Marshal(pipeline.JSTargetScript{
		Name:         "render",
		HostPatterns: []string{"example.com"},
		PreNav:       "console.log('hi')",
	})
	if err != nil {
		t.Fatalf("marshal create body: %v", err)
	}

	createReq := httptest.NewRequest(http.MethodPost, "/v1/pipeline-js", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createResp := httptest.NewRecorder()
	srv.Routes().ServeHTTP(createResp, createReq)

	if createResp.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", createResp.Code, createResp.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v1/pipeline-js/render", nil)
	getResp := httptest.NewRecorder()
	srv.Routes().ServeHTTP(getResp, getReq)
	if getResp.Code != http.StatusOK {
		t.Fatalf("expected get status 200, got %d: %s", getResp.Code, getResp.Body.String())
	}

	updateBody, err := json.Marshal(pipeline.JSTargetScript{
		Name:         "ignored",
		HostPatterns: []string{"example.org"},
		PostNav:      "console.log('updated')",
	})
	if err != nil {
		t.Fatalf("marshal update body: %v", err)
	}

	updateReq := httptest.NewRequest(http.MethodPut, "/v1/pipeline-js/render", bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp := httptest.NewRecorder()
	srv.Routes().ServeHTTP(updateResp, updateReq)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("expected update status 200, got %d: %s", updateResp.Code, updateResp.Body.String())
	}

	var updated pipeline.JSTargetScript
	if err := json.Unmarshal(updateResp.Body.Bytes(), &updated); err != nil {
		t.Fatalf("unmarshal update response: %v", err)
	}
	if updated.Name != "render" {
		t.Fatalf("expected path name to win, got %q", updated.Name)
	}
}
