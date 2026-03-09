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
//   - Cloud export schedules should receive default path and content format values
//     on both create and update when callers omit them.
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExportScheduleCreateAndUpdateNormalizeCloudDefaults(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	createBody := `{
		"name": "cloud export",
		"filters": {"job_kinds": ["scrape"]},
		"export": {
			"format": "json",
			"destination_type": "s3",
			"cloud_config": {
				"provider": "s3",
				"bucket": "portfolio-bucket"
			}
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

	if created.Export.CloudConfig == nil {
		t.Fatal("expected cloud config in create response")
	}
	if got := created.Export.CloudConfig.Path; got != "exports/{kind}/{job_id}.{format}" {
		t.Fatalf("expected default cloud path on create, got %q", got)
	}
	if got := created.Export.CloudConfig.ContentFormat; got != "json" {
		t.Fatalf("expected default cloud content format on create, got %q", got)
	}

	updateBody := `{
		"name": "cloud export updated",
		"filters": {"job_kinds": ["research"]},
		"export": {
			"format": "jsonl",
			"destination_type": "s3",
			"cloud_config": {
				"provider": "s3",
				"bucket": "portfolio-bucket"
			}
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

	if updated.Export.CloudConfig == nil {
		t.Fatal("expected cloud config in update response")
	}
	if got := updated.Export.CloudConfig.Path; got != "exports/{kind}/{job_id}.{format}" {
		t.Fatalf("expected default cloud path on update, got %q", got)
	}
	if got := updated.Export.CloudConfig.ContentFormat; got != "jsonl" {
		t.Fatalf("expected default cloud content format on update, got %q", got)
	}
}
