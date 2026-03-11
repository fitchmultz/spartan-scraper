// Package api provides integration tests for job results endpoint format conversion.
// Tests cover result type detection by file extension, format query parameter (jsonl, json, csv, md),
// invalid format validation, and default behavior.
// Does NOT test result file generation or export logic handled by exporter package.
package api

import (
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

func TestHandleJobResultsMultipleTypes(t *testing.T) {
	tests := []struct {
		name          string
		ext           string
		expectedCT    string
		resultContent string
	}{
		{
			name:       "jsonl",
			ext:        ".jsonl",
			expectedCT: "application/x-ndjson",
			resultContent: `{"field":"value1"}
{"field":"value2"}
`,
		},
		{
			name:          "json",
			ext:           ".json",
			expectedCT:    "application/json",
			resultContent: `{"field":"value"}`,
		},
		{
			name:          "csv",
			ext:           ".csv",
			expectedCT:    "text/csv",
			resultContent: "field1,field2\nvalue1,value2\n",
		},
		{
			name:          "xml",
			ext:           ".xml",
			expectedCT:    "application/xml",
			resultContent: `<?xml version="1.0"?><root><field>value</field></root>`,
		},
		{
			name:          "txt",
			ext:           ".txt",
			expectedCT:    "text/plain; charset=utf-8",
			resultContent: "plain text content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, cleanup := setupTestServer(t)
			defer cleanup()

			ctx := context.Background()
			jobID := "test-job-" + tt.name

			// Create job directly in store to avoid async processing
			job := model.Job{
				ID:        jobID,
				Kind:      model.KindScrape,
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

			resultPath := filepath.Join(resultDir, "results"+tt.ext)
			if err := os.WriteFile(resultPath, []byte(tt.resultContent), 0o644); err != nil {
				t.Fatalf("failed to write result file: %v", err)
			}

			if err := srv.store.UpdateResultPath(ctx, jobID, resultPath); err != nil {
				t.Fatalf("failed to update job result_path: %v", err)
			}
			if err := srv.store.UpdateStatus(ctx, jobID, model.StatusSucceeded, ""); err != nil {
				t.Fatalf("failed to update job status: %v", err)
			}

			req := httptest.NewRequest("GET", fmt.Sprintf("/v1/jobs/%s/results", jobID), nil)
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if status := rr.Code; status != http.StatusOK {
				t.Fatalf("handler returned wrong status code: got %v want %v, body: %s", status, http.StatusOK, rr.Body.String())
			}

			contentType := rr.Header().Get("Content-Type")
			if contentType != tt.expectedCT {
				t.Errorf("expected Content-Type %q, got %q", tt.expectedCT, contentType)
			}

			if rr.Body.String() != tt.resultContent {
				t.Errorf("expected body %q, got %q", tt.resultContent, rr.Body.String())
			}
		})
	}
}

func TestHandleJobResultsWithFormats(t *testing.T) {
	formats := []string{"jsonl", "json", "md", "csv", "xlsx"}

	for _, format := range formats {
		t.Run("format_"+format, func(t *testing.T) {
			srv, cleanup := setupTestServer(t)
			defer cleanup()

			ctx := context.Background()
			jobID := "test-job-format-" + format

			// Create job directly in store to avoid async processing
			job := model.Job{
				ID:        jobID,
				Kind:      model.KindScrape,
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
			resultContent := `{"url":"https://example.com","status":200,"title":"Test Page"}`
			if err := os.WriteFile(resultPath, []byte(resultContent), 0o644); err != nil {
				t.Fatalf("failed to write result file: %v", err)
			}

			if err := srv.store.UpdateResultPath(ctx, jobID, resultPath); err != nil {
				t.Fatalf("failed to update job result_path: %v", err)
			}
			if err := srv.store.UpdateStatus(ctx, jobID, model.StatusSucceeded, ""); err != nil {
				t.Fatalf("failed to update job status: %v", err)
			}

			req := httptest.NewRequest("GET", fmt.Sprintf("/v1/jobs/%s/results?format=%s", jobID, format), nil)
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if status := rr.Code; status != http.StatusOK {
				t.Errorf("expected 200, got %v", status)
			}

			expectedCT := map[string]string{
				"jsonl": "application/x-ndjson",
				"json":  "application/json",
				"md":    "text/markdown; charset=utf-8",
				"csv":   "text/csv; charset=utf-8",
				"xlsx":  "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			}
			if ct := rr.Header().Get("Content-Type"); ct != expectedCT[format] {
				t.Errorf("expected Content-Type %q, got %q", expectedCT[format], ct)
			}

			disposition := rr.Header().Get("Content-Disposition")
			if disposition == "" {
				t.Errorf("expected Content-Disposition header, got none")
			}
			if !strings.Contains(disposition, jobID) {
				t.Errorf("Content-Disposition should contain job ID")
			}
			if !strings.Contains(disposition, format) {
				t.Errorf("Content-Disposition should contain format %s", format)
			}
		})
	}
}

func TestHandleJobResultsInvalidFormat(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	jobID := "test-job-invalid-format"

	// Create job directly in store to avoid async processing
	job := model.Job{
		ID:        jobID,
		Kind:      model.KindScrape,
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
	if err := os.WriteFile(resultPath, []byte(`{"test":"data"}`), 0o644); err != nil {
		t.Fatalf("failed to write result file: %v", err)
	}

	if err := srv.store.UpdateResultPath(ctx, jobID, resultPath); err != nil {
		t.Fatalf("failed to update result path: %v", err)
	}
	if err := srv.store.UpdateStatus(ctx, jobID, model.StatusSucceeded, ""); err != nil {
		t.Fatalf("failed to update status: %v", err)
	}

	req := httptest.NewRequest("GET", "/v1/jobs/"+jobID+"/results?format=xml", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid format, got %v", status)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	if _, ok := resp["error"]; !ok {
		t.Error("expected error field in response")
	}
}

func TestHandleJobResultsNoFormatParameter(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	jobID := "test-job-no-format"

	// Create job directly in store to avoid async processing
	job := model.Job{
		ID:        jobID,
		Kind:      model.KindScrape,
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
	if err := os.WriteFile(resultPath, []byte(`{"test":"data"}`), 0o644); err != nil {
		t.Fatalf("failed to write result file: %v", err)
	}

	if err := srv.store.UpdateResultPath(ctx, jobID, resultPath); err != nil {
		t.Fatalf("failed to update result path: %v", err)
	}
	if err := srv.store.UpdateStatus(ctx, jobID, model.StatusSucceeded, ""); err != nil {
		t.Fatalf("failed to update status: %v", err)
	}

	req := httptest.NewRequest("GET", "/v1/jobs/"+jobID+"/results", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected 200, got %v", status)
	}

	if ct := rr.Header().Get("Content-Type"); ct != "application/x-ndjson" {
		t.Errorf("expected default Content-Type application/x-ndjson, got %v", ct)
	}
}
