// Package api provides api functionality for Spartan Scraper.
//
// Purpose:
// - Verify job results transform test behavior for package api.
//
// Responsibilities:
// - Define focused Go test coverage, fixtures, and assertions for the package behavior exercised here.
//
// Scope:
// - Automated test coverage only; production behavior stays in non-test package files.
//
// Usage:
// - Run with `go test` for package `api` or through `make test-ci`/`make ci`.
//
// Invariants/Assumptions:
// - Tests should remain deterministic and describe the package contract they protect.

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/exporter"
	"github.com/fitchmultz/spartan-scraper/internal/fsutil"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestHandleJobExportWithTransform_JMESPath(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	jobID := "test-job-export-jmespath"
	createSucceededJobWithResult(t, srv, ctx, jobID, model.KindScrape, `{"url":"https://example.com","status":200,"title":"Test Page","content":"Hello World"}`)

	body := `{"format":"json","transform":{"expression":"{title: title, url: url}","language":"jmespath"}}`
	req := newJSONExportRequest(http.MethodPost, fmt.Sprintf("/v1/jobs/%s/export", jobID), body)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	response := decodeExportOutcomeResponse(t, rr)
	assertExportSucceeded(t, response.Export, "application/json")

	var results []map[string]any
	if err := json.Unmarshal([]byte(response.Export.Artifact.Content), &results); err != nil {
		t.Fatalf("failed to parse export content: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if _, ok := results[0]["title"]; !ok {
		t.Fatal("expected transformed result to contain title")
	}
	if _, ok := results[0]["content"]; ok {
		t.Fatalf("expected transformed result to omit content: %#v", results[0])
	}
}

func TestHandleJobExportWithTransform_JSONata(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	jobID := "test-job-export-jsonata"
	createSucceededJobWithResult(t, srv, ctx, jobID, model.KindScrape, `{"name":"Product A","price":100,"quantity":2}`)

	body := `{"format":"json","transform":{"expression":"{\"item\": name, \"total\": price * quantity}","language":"jsonata"}}`
	req := newJSONExportRequest(http.MethodPost, fmt.Sprintf("/v1/jobs/%s/export", jobID), body)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	response := decodeExportOutcomeResponse(t, rr)
	assertExportSucceeded(t, response.Export, "application/json")

	var results []map[string]any
	if err := json.Unmarshal([]byte(response.Export.Artifact.Content), &results); err != nil {
		t.Fatalf("failed to parse export content: %v", err)
	}
	if len(results) != 1 || results[0]["item"] != "Product A" || results[0]["total"] != float64(200) {
		t.Fatalf("unexpected transformed results: %#v", results)
	}
}

func TestHandleJobExportWithTransform_InvalidLanguage(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	jobID := "test-job-export-invalid-language"
	createSucceededJobWithResult(t, srv, ctx, jobID, model.KindScrape, `{"test":"data"}`)

	body := `{"format":"json","transform":{"expression":"{title: title}","language":"invalid"}}`
	req := newJSONExportRequest(http.MethodPost, fmt.Sprintf("/v1/jobs/%s/export", jobID), body)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleJobExportWithTransform_InvalidExpression(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	jobID := "test-job-export-invalid-expression"
	createSucceededJobWithResult(t, srv, ctx, jobID, model.KindScrape, `{"test":"data"}`)

	body := `{"format":"json","transform":{"expression":"{invalid","language":"jmespath"}}`
	req := newJSONExportRequest(http.MethodPost, fmt.Sprintf("/v1/jobs/%s/export", jobID), body)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleJobExportPersistsNoResultFailure(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	jobID := "test-job-export-no-results"
	createQueuedJobWithoutResult(t, srv, ctx, jobID, model.KindScrape)

	req := newJSONExportRequest(http.MethodPost, fmt.Sprintf("/v1/jobs/%s/export", jobID), `{"format":"json"}`)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	response := decodeExportOutcomeResponse(t, rr)
	if response.Export.Status != "failed" {
		t.Fatalf("expected failed export, got %#v", response.Export)
	}
	if response.Export.Failure == nil {
		t.Fatalf("expected failure context, got %#v", response.Export)
	}
	if response.Export.Failure.Category != "result" {
		t.Fatalf("failure category = %q, want result", response.Export.Failure.Category)
	}
	if response.Export.Failure.Retryable {
		t.Fatalf("expected non-retryable failure, got %#v", response.Export.Failure)
	}
	assertActionValue(t, response.Export.Actions, "Inspect saved job results", "/jobs/"+jobID)

	historyReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/jobs/%s/exports?limit=10&offset=0", jobID), nil)
	historyRR := httptest.NewRecorder()
	srv.Routes().ServeHTTP(historyRR, historyReq)
	if historyRR.Code != http.StatusOK {
		t.Fatalf("expected history 200, got %d: %s", historyRR.Code, historyRR.Body.String())
	}
	var history ExportOutcomeListResponse
	if err := json.Unmarshal(historyRR.Body.Bytes(), &history); err != nil {
		t.Fatalf("decode history response: %v", err)
	}
	if len(history.Exports) != 1 {
		t.Fatalf("expected one persisted failure record, got %#v", history.Exports)
	}
	if history.Exports[0].Failure == nil || history.Exports[0].Failure.Category != "result" {
		t.Fatalf("unexpected persisted failure: %#v", history.Exports[0])
	}
}

func TestBuildExportRecommendedActionsForTransformFailure(t *testing.T) {
	actions := buildExportRecommendedActions(ExportInspection{
		ID:     "export-1",
		JobID:  "job-1",
		Status: string(exporter.OutcomeFailed),
		Request: exporter.ResultExportConfig{
			Format: "csv",
		},
		Failure: &exporter.FailureContext{
			Category:  "transform",
			Summary:   "invalid jmespath expression",
			Retryable: false,
		},
	})

	assertActionValue(t, actions, "Retry without the transform", "spartan export --job-id job-1 --format csv")
	assertActionValue(t, actions, "Retry as JSONL", "spartan export --job-id job-1 --format jsonl")
}

func TestHandleJobExportWithTransform_CSVFormat(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	jobID := "test-job-export-csv-transform"
	createSucceededJobWithResult(t, srv, ctx, jobID, model.KindScrape, `{"url":"https://example.com","status":200,"title":"Test Page"}`)

	body := `{"format":"csv","transform":{"expression":"{title: title}","language":"jmespath"}}`
	req := newJSONExportRequest(http.MethodPost, fmt.Sprintf("/v1/jobs/%s/export", jobID), body)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	response := decodeExportOutcomeResponse(t, rr)
	assertExportSucceeded(t, response.Export, "text/csv; charset=utf-8")
	if body := response.Export.Artifact.Content; !strings.Contains(body, "title") || strings.Contains(body, "status") {
		t.Fatalf("unexpected csv export: %s", body)
	}
}

func TestHandleJobExportWithTransform_JSONL(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	jobID := "test-job-export-jsonl-transform"
	createSucceededJobWithResult(t, srv, ctx, jobID, model.KindCrawl, strings.Join([]string{
		`{"url":"https://example.com/a","title":"A","status":200}`,
		`{"url":"https://example.com/b","title":"B","status":200}`,
	}, "\n"))

	body := `{"format":"jsonl","transform":{"expression":"{title: title, url: url}","language":"jmespath"}}`
	req := newJSONExportRequest(http.MethodPost, fmt.Sprintf("/v1/jobs/%s/export", jobID), body)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	response := decodeExportOutcomeResponse(t, rr)
	assertExportSucceeded(t, response.Export, "application/x-ndjson")
	if strings.Contains(response.Export.Artifact.Content, "status") {
		t.Fatalf("expected transformed jsonl to omit status: %s", response.Export.Artifact.Content)
	}
}

func TestHandleJobExportWithShape(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	jobID := "test-job-export-shape"
	createSucceededJobWithResult(t, srv, ctx, jobID, model.KindScrape, `{"url":"https://example.com","status":200,"title":"Test Page","normalized":{"fields":{"price":{"values":["$10"]}}}}`)

	body := `{"format":"md","shape":{"summaryFields":["title","url"],"normalizedFields":["field.price"]}}`
	req := newJSONExportRequest(http.MethodPost, fmt.Sprintf("/v1/jobs/%s/export", jobID), body)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	response := decodeExportOutcomeResponse(t, rr)
	assertExportSucceeded(t, response.Export, "text/markdown; charset=utf-8")
	if !strings.Contains(response.Export.Artifact.Content, "Test Page") || !strings.Contains(response.Export.Artifact.Content, "$10") {
		t.Fatalf("unexpected shaped markdown: %s", response.Export.Artifact.Content)
	}
}

func TestHandleJobExportRejectsShapeAndTransform(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	jobID := "test-job-export-shape-and-transform"
	createSucceededJobWithResult(t, srv, ctx, jobID, model.KindScrape, `{"url":"https://example.com","title":"Test Page"}`)

	body := `{"format":"csv","shape":{"topLevelFields":["url"]},"transform":{"expression":"{url: url}","language":"jmespath"}}`
	req := newJSONExportRequest(http.MethodPost, fmt.Sprintf("/v1/jobs/%s/export", jobID), body)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleJobExportXLSX(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	jobID := "test-job-export-xlsx"
	createSucceededJobWithResult(t, srv, ctx, jobID, model.KindScrape, `{"url":"https://example.com","status":200,"title":"Test Page"}`)

	req := newJSONExportRequest(http.MethodPost, fmt.Sprintf("/v1/jobs/%s/export", jobID), `{"format":"xlsx"}`)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	response := decodeExportOutcomeResponse(t, rr)
	assertExportSucceeded(t, response.Export, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	if response.Export.Artifact.Encoding != "base64" {
		t.Fatalf("expected base64 xlsx payload, got %#v", response.Export.Artifact)
	}
	if response.Export.Artifact.Content == "" {
		t.Fatal("expected xlsx bytes")
	}
}

func TestHandleJobExportHistoryAndOutcomeLookup(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	jobID := "test-job-export-history"
	createSucceededJobWithResult(t, srv, ctx, jobID, model.KindScrape, `{"url":"https://example.com","status":200,"title":"Test Page"}`)

	req := newJSONExportRequest(http.MethodPost, fmt.Sprintf("/v1/jobs/%s/export", jobID), `{"format":"json"}`)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)
	response := decodeExportOutcomeResponse(t, rr)

	historyReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/jobs/%s/exports?limit=10&offset=0", jobID), nil)
	historyRes := httptest.NewRecorder()
	srv.Routes().ServeHTTP(historyRes, historyReq)
	if historyRes.Code != http.StatusOK {
		t.Fatalf("expected history 200, got %d: %s", historyRes.Code, historyRes.Body.String())
	}
	var history ExportOutcomeListResponse
	if err := json.Unmarshal(historyRes.Body.Bytes(), &history); err != nil {
		t.Fatalf("failed to decode history response: %v", err)
	}
	if len(history.Exports) != 1 || history.Exports[0].ID != response.Export.ID {
		t.Fatalf("unexpected export history: %#v", history)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v1/exports/"+response.Export.ID, nil)
	getRes := httptest.NewRecorder()
	srv.Routes().ServeHTTP(getRes, getReq)
	if getRes.Code != http.StatusOK {
		t.Fatalf("expected outcome 200, got %d: %s", getRes.Code, getRes.Body.String())
	}
	var single ExportOutcomeResponse
	if err := json.Unmarshal(getRes.Body.Bytes(), &single); err != nil {
		t.Fatalf("failed to decode single export response: %v", err)
	}
	if single.Export.ID != response.Export.ID || single.Export.Artifact == nil || single.Export.Artifact.Content != "" {
		t.Fatalf("unexpected export lookup payload: %#v", single)
	}
}

func decodeExportOutcomeResponse(t *testing.T, rr *httptest.ResponseRecorder) ExportOutcomeResponse {
	t.Helper()
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %q", ct)
	}
	var response ExportOutcomeResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode export outcome response: %v", err)
	}
	return response
}

func assertExportSucceeded(t *testing.T, outcome ExportInspection, contentType string) {
	t.Helper()
	if outcome.Status != "succeeded" {
		t.Fatalf("expected succeeded export, got %#v", outcome)
	}
	if outcome.Artifact == nil {
		t.Fatalf("expected export artifact, got %#v", outcome)
	}
	if outcome.Artifact.ContentType != contentType {
		t.Fatalf("expected content type %q, got %#v", contentType, outcome.Artifact)
	}
}

func createQueuedJobWithoutResult(t *testing.T, srv *Server, ctx context.Context, jobID string, kind model.Kind) {
	t.Helper()
	job := model.Job{
		ID:        jobID,
		Kind:      kind,
		Status:    model.StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Spec:      map[string]any{"url": "https://example.com"},
	}
	if err := srv.store.Create(ctx, job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}
}

func createSucceededJobWithResult(t *testing.T, srv *Server, ctx context.Context, jobID string, kind model.Kind, resultContent string) {
	t.Helper()
	createQueuedJobWithoutResult(t, srv, ctx, jobID, kind)
	resultDir := filepath.Join(srv.store.DataDir(), "jobs", jobID)
	if err := fsutil.MkdirAllSecure(resultDir); err != nil {
		t.Fatalf("failed to create result directory: %v", err)
	}
	resultPath := filepath.Join(resultDir, "results.jsonl")
	if err := os.WriteFile(resultPath, []byte(resultContent), 0o644); err != nil {
		t.Fatalf("failed to write result file: %v", err)
	}
	if err := srv.store.UpdateResultPath(ctx, jobID, resultPath); err != nil {
		t.Fatalf("failed to update job result path: %v", err)
	}
	if err := srv.store.UpdateStatus(ctx, jobID, model.StatusSucceeded, ""); err != nil {
		t.Fatalf("failed to update job status: %v", err)
	}
}

func newJSONExportRequest(method, path, body string) *http.Request {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}
