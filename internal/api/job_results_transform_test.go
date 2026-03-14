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

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %q", ct)
	}

	var results []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &results); err != nil {
		t.Fatalf("failed to parse response: %v", err)
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

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var results []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &results); err != nil {
		t.Fatalf("failed to parse response: %v", err)
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

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "text/csv; charset=utf-8" {
		t.Fatalf("expected text/csv; charset=utf-8, got %q", ct)
	}
	if body := rr.Body.String(); !strings.Contains(body, "title") || strings.Contains(body, "status") {
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

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/x-ndjson" {
		t.Fatalf("expected application/x-ndjson, got %q", ct)
	}
	if strings.Contains(rr.Body.String(), "status") {
		t.Fatalf("expected transformed jsonl to omit status: %s", rr.Body.String())
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

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "text/markdown; charset=utf-8" {
		t.Fatalf("expected markdown content type, got %q", ct)
	}
	if !strings.Contains(rr.Body.String(), "Test Page") || !strings.Contains(rr.Body.String(), "$10") {
		t.Fatalf("unexpected shaped markdown: %s", rr.Body.String())
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

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Fatalf("unexpected xlsx content type: %q", ct)
	}
	if len(rr.Body.Bytes()) == 0 {
		t.Fatal("expected xlsx bytes")
	}
}

func createSucceededJobWithResult(t *testing.T, srv *Server, ctx context.Context, jobID string, kind model.Kind, resultContent string) {
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
