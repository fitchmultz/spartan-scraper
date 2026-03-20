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

	"github.com/fitchmultz/spartan-scraper/internal/exporter"
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

func TestExportScheduleCreateRejectsInvalidWebhookURL(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{
		"name": "invalid webhook export",
		"filters": {"job_kinds": ["scrape"]},
		"export": {
			"format": "json",
			"destination_type": "webhook",
			"webhook_url": "ftp://example.com/webhook"
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/export-schedules", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	srv.Routes().ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "webhook URL must use http or https scheme") {
		t.Fatalf("expected webhook URL validation error, got %s", res.Body.String())
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

func TestExportScheduleHistoryUsesExportsRouteActions(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	store := scheduler.NewExportStorage(srv.cfg.DataDir)
	schedule, err := store.Add(scheduler.ExportSchedule{
		Name:    "history route target",
		Enabled: true,
		Filters: scheduler.ExportFilters{JobKinds: []string{"scrape"}},
		Export: scheduler.ExportConfig{
			Format:          "json",
			DestinationType: "local",
			LocalPath:       "exports/{kind}/{job_id}.{format}",
			PathTemplate:    "exports/{kind}/{job_id}.{format}",
		},
	})
	if err != nil {
		t.Fatalf("failed to create export schedule: %v", err)
	}

	historyStore := scheduler.NewExportHistoryStore(srv.cfg.DataDir)
	record, err := historyStore.CreateRecord(scheduler.CreateRecordInput{
		ScheduleID:  schedule.ID,
		JobID:       "job-history-1",
		Trigger:     exporter.OutcomeTriggerSchedule,
		Destination: "exports/scrape/job-history-1.json",
		Request: exporter.ResultExportConfig{
			Format: "json",
		},
	})
	if err != nil {
		t.Fatalf("failed to create export history record: %v", err)
	}
	if err := historyStore.MarkSuccess(record.ID, exporter.RenderedResultExport{
		Format:      "json",
		Filename:    "job-history-1.json",
		ContentType: "application/json",
		Size:        128,
		RecordCount: 1,
	}); err != nil {
		t.Fatalf("failed to mark export success: %v", err)
	}

	historyReq := httptest.NewRequest(http.MethodGet, "/v1/export-schedules/"+schedule.ID+"/history?limit=10&offset=0", nil)
	historyRes := httptest.NewRecorder()
	srv.Routes().ServeHTTP(historyRes, historyReq)
	if historyRes.Code != http.StatusOK {
		t.Fatalf("expected history status 200, got %d: %s", historyRes.Code, historyRes.Body.String())
	}

	var history ExportOutcomeListResponse
	if err := json.Unmarshal(historyRes.Body.Bytes(), &history); err != nil {
		t.Fatalf("failed to decode export history response: %v", err)
	}
	if len(history.Exports) != 1 {
		t.Fatalf("expected 1 export history record, got %#v", history)
	}
	assertActionValue(t, history.Exports[0].Actions, "Inspect schedule history", "/automation/exports")
}

func TestBuildExportRecommendedActionsUseExportsRoute(t *testing.T) {
	successActions := buildExportRecommendedActions(ExportInspection{
		ID:         "export-1",
		ScheduleID: "schedule-1",
		JobID:      "job-1",
		Status:     string(exporter.OutcomeSucceeded),
		Request: exporter.ResultExportConfig{
			Format: "json",
		},
	})
	assertActionValue(t, successActions, "Inspect schedule history", "/automation/exports")

	failureActions := buildExportRecommendedActions(ExportInspection{
		ID:     "export-2",
		JobID:  "job-2",
		Status: string(exporter.OutcomeFailed),
		Request: exporter.ResultExportConfig{
			Format: "json",
		},
		Failure: &exporter.FailureContext{
			Category: "network",
			Summary:  "delivery failed",
		},
	})
	assertActionValue(t, failureActions, "Review export automation settings", "/automation/exports")

	timeoutActions := buildExportRecommendedActions(ExportInspection{
		ID:     "export-3",
		JobID:  "job-3",
		Status: string(exporter.OutcomeFailed),
		Request: exporter.ResultExportConfig{
			Format: "json",
		},
		Failure: &exporter.FailureContext{
			Category: "timeout",
			Summary:  "delivery timed out",
		},
	})
	assertActionValue(t, timeoutActions, "Review export automation settings", "/automation/exports")

	transformActions := buildExportRecommendedActions(ExportInspection{
		ID:     "export-4",
		JobID:  "job-4",
		Status: string(exporter.OutcomeFailed),
		Request: exporter.ResultExportConfig{
			Format: "csv",
		},
		Failure: &exporter.FailureContext{
			Category: "transform",
			Summary:  "invalid jmespath expression",
		},
	})
	assertActionValue(t, transformActions, "Retry without the transform", "spartan export --job-id job-4 --format csv")
	assertActionValue(t, transformActions, "Retry as JSONL", "spartan export --job-id job-4 --format jsonl")
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
