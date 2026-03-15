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

	"github.com/fitchmultz/spartan-scraper/internal/scheduler"
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

func TestExportScheduleCreateAndGetRoundTripsTransform(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{
		"name": "projected export",
		"filters": {"job_kinds": ["scrape"]},
		"export": {
			"format": "csv",
			"destination_type": "local",
			"transform": {
				"expression": "{title: title, url: url}",
				"language": "jmespath"
			}
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
	if created.Export.Transform.Expression != "{title: title, url: url}" {
		t.Fatalf("unexpected transform expression: %#v", created.Export.Transform)
	}
	if created.Export.LocalPath != "exports/{kind}/{job_id}.{format}" {
		t.Fatalf("expected default local path, got %q", created.Export.LocalPath)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v1/export-schedules/"+created.ID, nil)
	getRes := httptest.NewRecorder()
	srv.Routes().ServeHTTP(getRes, getReq)
	if getRes.Code != http.StatusOK {
		t.Fatalf("expected get status 200, got %d: %s", getRes.Code, getRes.Body.String())
	}

	var fetched ExportScheduleResponse
	if err := json.Unmarshal(getRes.Body.Bytes(), &fetched); err != nil {
		t.Fatalf("failed to decode get response: %v", err)
	}
	if fetched.Export.Transform.Language != "jmespath" {
		t.Fatalf("unexpected transform language: %#v", fetched.Export.Transform)
	}
}

func TestExportScheduleRejectsShapeAndTransformCombination(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{
		"name": "invalid export",
		"filters": {"job_kinds": ["scrape"]},
		"export": {
			"format": "md",
			"destination_type": "local",
			"shape": {"topLevelFields": ["url"]},
			"transform": {
				"expression": "{title: title}",
				"language": "jmespath"
			}
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/export-schedules", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	srv.Routes().ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "cannot be combined") {
		t.Fatalf("expected shape/transform validation error, got %s", res.Body.String())
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

func TestExportScheduleHandlersSynchronizeLiveRuntime(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	runtime := &captureExportScheduleRuntime{}
	srv.SetExportScheduleRuntime(runtime)

	createBody := `{
		"name": "runtime target",
		"filters": {"job_kinds": ["scrape"]},
		"export": {
			"format": "json",
			"destination_type": "webhook",
			"webhook_url": "https://example.com/webhook"
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
	if runtime.added == nil || runtime.added.ID != created.ID {
		t.Fatalf("expected runtime add call for %s, got %#v", created.ID, runtime.added)
	}

	updateBody := `{
		"name": "runtime target updated",
		"filters": {"job_kinds": ["scrape"]},
		"export": {
			"format": "csv",
			"destination_type": "webhook",
			"webhook_url": "https://example.com/webhook"
		}
	}`
	updateReq := httptest.NewRequest(http.MethodPut, "/v1/export-schedules/"+created.ID, strings.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateRes := httptest.NewRecorder()
	srv.Routes().ServeHTTP(updateRes, updateReq)
	if updateRes.Code != http.StatusOK {
		t.Fatalf("expected update status 200, got %d: %s", updateRes.Code, updateRes.Body.String())
	}
	if runtime.updated == nil || runtime.updated.ID != created.ID || runtime.updated.Export.Format != "csv" {
		t.Fatalf("expected runtime update call for %s, got %#v", created.ID, runtime.updated)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/v1/export-schedules/"+created.ID, nil)
	deleteRes := httptest.NewRecorder()
	srv.Routes().ServeHTTP(deleteRes, deleteReq)
	if deleteRes.Code != http.StatusNoContent {
		t.Fatalf("expected delete status 204, got %d: %s", deleteRes.Code, deleteRes.Body.String())
	}
	if runtime.removedID != created.ID {
		t.Fatalf("expected runtime remove call for %s, got %q", created.ID, runtime.removedID)
	}
}

type captureExportScheduleRuntime struct {
	added     *scheduler.ExportSchedule
	updated   *scheduler.ExportSchedule
	removedID string
}

func (c *captureExportScheduleRuntime) AddSchedule(schedule *scheduler.ExportSchedule) {
	copied := *schedule
	c.added = &copied
}

func (c *captureExportScheduleRuntime) UpdateSchedule(schedule *scheduler.ExportSchedule) {
	copied := *schedule
	c.updated = &copied
}

func (c *captureExportScheduleRuntime) RemoveSchedule(scheduleID string) {
	c.removedID = scheduleID
}
