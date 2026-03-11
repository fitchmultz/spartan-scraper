// Package api provides integration tests for export schedule endpoints.
//
// Purpose:
// - Verify shared handler behavior for export schedule CRUD routes.
//
// Responsibilities:
// - Confirm create/update normalization and JSON response semantics.
//
// Scope:
// - HTTP-level coverage for `/v1/export-schedules`.
//
// Usage:
// - Executed via `go test ./internal/api`.
//
// Invariants/Assumptions:
// - Local export schedules should receive a default path template when callers omit it.
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExportScheduleCreateAndUpdateNormalizeLocalDefaults(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	createBody := `{
		"name": "local export",
		"filters": {"job_kinds": ["scrape"]},
		"export": {
			"format": "json",
			"destination_type": "local"
		}
	}`

	createReq := httptest.NewRequest(http.MethodPost, "/v1/export-schedules", strings.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRes := httptest.NewRecorder()
	srv.Routes().ServeHTTP(createRes, createReq)

	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d: %s", createRes.Code, createRes.Body.String())
	}

	var created ExportScheduleResponse
	if err := json.Unmarshal(createRes.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}

	if got := created.Export.PathTemplate; got != "exports/{kind}/{job_id}.{format}" {
		t.Fatalf("expected default local path template on create, got %q", got)
	}
	if got := created.Export.LocalPath; got != "exports/{kind}/{job_id}.{format}" {
		t.Fatalf("expected default local path on create, got %q", got)
	}

	updateBody := `{
		"name": "local export updated",
		"filters": {"job_kinds": ["research"]},
		"export": {
			"format": "jsonl",
			"destination_type": "local"
		}
	}`

	updateReq := httptest.NewRequest(http.MethodPut, "/v1/export-schedules/"+created.ID, strings.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateRes := httptest.NewRecorder()
	srv.Routes().ServeHTTP(updateRes, updateReq)

	if updateRes.Code != http.StatusOK {
		t.Fatalf("expected update status 200, got %d: %s", updateRes.Code, updateRes.Body.String())
	}

	var updated ExportScheduleResponse
	if err := json.Unmarshal(updateRes.Body.Bytes(), &updated); err != nil {
		t.Fatalf("failed to decode update response: %v", err)
	}

	if got := updated.Export.PathTemplate; got != "exports/{kind}/{job_id}.{format}" {
		t.Fatalf("expected default local path template on update, got %q", got)
	}
	if got := updated.Export.LocalPath; got != "exports/{kind}/{job_id}.{format}" {
		t.Fatalf("expected default local path on update, got %q", got)
	}
}

func TestExportScheduleDeleteMissingReturnsNotFound(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodDelete, "/v1/export-schedules/missing", nil)
	res := httptest.NewRecorder()
	srv.Routes().ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("expected delete status 404, got %d: %s", res.Code, res.Body.String())
	}
}

func TestExportScheduleHistoryRejectsInvalidPagination(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{
		"name": "history target",
		"filters": {"job_kinds": ["scrape"]},
		"export": {
			"format": "json",
			"destination_type": "local",
			"local_path": "/tmp/exports.json"
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/v1/export-schedules", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	srv.Routes().ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d: %s", res.Code, res.Body.String())
	}

	var created ExportScheduleResponse
	if err := json.Unmarshal(res.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}

	historyReq := httptest.NewRequest(http.MethodGet, "/v1/export-schedules/"+created.ID+"/history?limit=abc", nil)
	historyRes := httptest.NewRecorder()
	srv.Routes().ServeHTTP(historyRes, historyReq)

	if historyRes.Code != http.StatusBadRequest {
		t.Fatalf("expected history status 400, got %d: %s", historyRes.Code, historyRes.Body.String())
	}
}
