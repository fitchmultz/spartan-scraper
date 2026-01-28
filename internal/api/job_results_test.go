// Package api provides integration tests for job results endpoint (/v1/jobs/{id}/results).
// Tests cover result retrieval, format conversion (jsonl, json, csv, xml, md, txt), and pagination.
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
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/fsutil"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestHandleJobResultsNotFound(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/v1/jobs/nonexistent-id/results", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("expected status %v, got %v", http.StatusNotFound, status)
	}

	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %v", ct)
	}
}

func TestHandleJobResultsNoResults(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{"url": "https://example.com"}`
	req := httptest.NewRequest("POST", "/v1/scrape", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("failed to create job: got status %v", status)
	}

	var job map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &job); err != nil {
		t.Fatalf("failed to parse job response: %v", err)
	}

	jobID, ok := job["id"].(string)
	if !ok {
		t.Fatalf("job response missing id field")
	}

	req = httptest.NewRequest("GET", fmt.Sprintf("/v1/jobs/%s/results", jobID), nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("expected status %v for job with no results, got %v", http.StatusBadRequest, status)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	msg, _ := resp["error"].(string)
	isQueued := strings.Contains(msg, "job is queued and has no results yet")
	isRunning := strings.Contains(msg, "job is still running and has no results yet")
	if !isQueued && !isRunning {
		t.Errorf("expected error message to indicate queued or running, got %q", msg)
	}
}

func TestHandleJobResultsGranularErrors(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	tests := []struct {
		name           string
		status         model.Status
		setupFile      func(jobID string) string
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:           "queued",
			status:         model.StatusQueued,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "job is queued and has no results yet",
		},
		{
			name:           "running",
			status:         model.StatusRunning,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "job is still running and has no results yet",
		},
		{
			name:           "failed",
			status:         model.StatusFailed,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "job failed and produced no results",
		},
		{
			name:           "canceled",
			status:         model.StatusCanceled,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "job was canceled and produced no results",
		},
		{
			name:           "succeeded - no result path",
			status:         model.StatusSucceeded,
			setupFile:      func(jobID string) string { return "" },
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "job succeeded but no result path was recorded",
		},
		{
			name:           "succeeded - file missing",
			status:         model.StatusSucceeded,
			setupFile:      func(jobID string) string { return "/nonexistent/path/results.jsonl" },
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "job succeeded but result file is missing",
		},
		{
			name:   "succeeded - file empty",
			status: model.StatusSucceeded,
			setupFile: func(jobID string) string {
				path := filepath.Join(t.TempDir(), "empty.jsonl")
				os.WriteFile(path, []byte(""), 0644)
				return path
			},
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "job succeeded but result file is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a job directly in store
			jobID := "test-job-" + strings.ReplaceAll(tt.name, " ", "-")
			job := model.Job{
				ID:        jobID,
				Kind:      model.KindScrape,
				Status:    tt.status,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if tt.setupFile != nil {
				job.ResultPath = tt.setupFile(jobID)
			}

			if err := srv.store.Create(ctx, job); err != nil {
				t.Fatalf("failed to create job: %v", err)
			}

			req := httptest.NewRequest("GET", fmt.Sprintf("/v1/jobs/%s/results", jobID), nil)
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("expected status %v, got %v", tt.expectedStatus, status)
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse error response: %v", err)
			}
			if msg, ok := resp["error"].(string); !ok || !strings.Contains(msg, tt.expectedMsg) {
				t.Errorf("expected error message to contain %q, got %q", tt.expectedMsg, msg)
			}
		})
	}
}

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

			body := `{"url": "https://example.com"}`
			req := httptest.NewRequest("POST", "/v1/scrape", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if status := rr.Code; status != http.StatusOK {
				t.Fatalf("failed to create job: got status %v, body: %s", status, rr.Body.String())
			}

			var job map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &job); err != nil {
				t.Fatalf("failed to parse job response: %v", err)
			}

			jobID, ok := job["id"].(string)
			if !ok {
				t.Fatalf("job response missing id field: %v", job)
			}

			dataDir := t.TempDir()
			resultDir := filepath.Join(dataDir, "jobs", jobID)
			if err := fsutil.MkdirAllSecure(resultDir); err != nil {
				t.Fatalf("failed to create result directory: %v", err)
			}

			resultPath := filepath.Join(resultDir, "results"+tt.ext)
			if err := os.WriteFile(resultPath, []byte(tt.resultContent), 0o644); err != nil {
				t.Fatalf("failed to write result file: %v", err)
			}

			st := srv.store
			ctx := context.Background()

			if err := st.UpdateResultPath(ctx, jobID, resultPath); err != nil {
				t.Fatalf("failed to update job result_path: %v", err)
			}
			if err := st.UpdateStatus(ctx, jobID, model.StatusSucceeded, ""); err != nil {
				t.Fatalf("failed to update job status: %v", err)
			}

			req = httptest.NewRequest("GET", fmt.Sprintf("/v1/jobs/%s/results", jobID), nil)
			rr = httptest.NewRecorder()
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
	formats := []string{"jsonl", "json", "md", "csv"}

	for _, format := range formats {
		t.Run("format_"+format, func(t *testing.T) {
			srv, cleanup := setupTestServer(t)
			defer cleanup()

			body := `{"url": "https://example.com"}`
			req := httptest.NewRequest("POST", "/v1/scrape", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if status := rr.Code; status != http.StatusOK {
				t.Fatalf("failed to create job: got status %v", status)
			}

			var job map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &job); err != nil {
				t.Fatalf("failed to parse job response: %v", err)
			}

			jobID, ok := job["id"].(string)
			if !ok {
				t.Fatalf("job response missing id field")
			}

			dataDir := t.TempDir()
			resultDir := filepath.Join(dataDir, "jobs", jobID)
			if err := fsutil.MkdirAllSecure(resultDir); err != nil {
				t.Fatalf("failed to create result directory: %v", err)
			}

			resultPath := filepath.Join(resultDir, "results.jsonl")
			resultContent := `{"url":"https://example.com","status":200,"title":"Test Page"}`
			if err := os.WriteFile(resultPath, []byte(resultContent), 0o644); err != nil {
				t.Fatalf("failed to write result file: %v", err)
			}

			st := srv.store
			ctx := context.Background()

			if err := st.UpdateResultPath(ctx, jobID, resultPath); err != nil {
				t.Fatalf("failed to update job result_path: %v", err)
			}
			if err := st.UpdateStatus(ctx, jobID, model.StatusSucceeded, ""); err != nil {
				t.Fatalf("failed to update job status: %v", err)
			}

			req = httptest.NewRequest("GET", fmt.Sprintf("/v1/jobs/%s/results?format=%s", jobID, format), nil)
			rr = httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if status := rr.Code; status != http.StatusOK {
				t.Errorf("expected 200, got %v", status)
			}

			expectedCT := map[string]string{
				"jsonl": "application/x-ndjson",
				"json":  "application/json",
				"md":    "text/markdown; charset=utf-8",
				"csv":   "text/csv; charset=utf-8",
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

	body := `{"url": "https://example.com"}`
	req := httptest.NewRequest("POST", "/v1/scrape", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("failed to create job: got status %v", status)
	}

	var job map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &job); err != nil {
		t.Fatalf("failed to parse job response: %v", err)
	}

	jobID, ok := job["id"].(string)
	if !ok {
		t.Fatalf("job response missing id field")
	}

	dataDir := t.TempDir()
	resultDir := filepath.Join(dataDir, "jobs", jobID)
	if err := fsutil.MkdirAllSecure(resultDir); err != nil {
		t.Fatalf("failed to create result directory: %v", err)
	}

	resultPath := filepath.Join(resultDir, "results.jsonl")
	if err := os.WriteFile(resultPath, []byte(`{"test":"data"}`), 0o644); err != nil {
		t.Fatalf("failed to write result file: %v", err)
	}

	st := srv.store
	ctx := context.Background()

	if err := st.UpdateResultPath(ctx, jobID, resultPath); err != nil {
		t.Fatalf("failed to update result path: %v", err)
	}
	if err := st.UpdateStatus(ctx, jobID, model.StatusSucceeded, ""); err != nil {
		t.Fatalf("failed to update status: %v", err)
	}

	req = httptest.NewRequest("GET", "/v1/jobs/"+jobID+"/results?format=xml", nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid format, got %v", status)
	}

	var resp map[string]interface{}
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

	body := `{"url": "https://example.com"}`
	req := httptest.NewRequest("POST", "/v1/scrape", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("failed to create job: got status %v", status)
	}

	var job map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &job); err != nil {
		t.Fatalf("failed to parse job response: %v", err)
	}

	jobID, ok := job["id"].(string)
	if !ok {
		t.Fatalf("job response missing id field")
	}

	dataDir := t.TempDir()
	resultDir := filepath.Join(dataDir, "jobs", jobID)
	if err := fsutil.MkdirAllSecure(resultDir); err != nil {
		t.Fatalf("failed to create result directory: %v", err)
	}

	resultPath := filepath.Join(resultDir, "results.jsonl")
	if err := os.WriteFile(resultPath, []byte(`{"test":"data"}`), 0o644); err != nil {
		t.Fatalf("failed to write result file: %v", err)
	}

	st := srv.store
	ctx := context.Background()

	if err := st.UpdateResultPath(ctx, jobID, resultPath); err != nil {
		t.Fatalf("failed to update result path: %v", err)
	}
	if err := st.UpdateStatus(ctx, jobID, model.StatusSucceeded, ""); err != nil {
		t.Fatalf("failed to update status: %v", err)
	}

	req = httptest.NewRequest("GET", "/v1/jobs/"+jobID+"/results", nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected 200, got %v", status)
	}

	if ct := rr.Header().Get("Content-Type"); ct != "application/x-ndjson" {
		t.Errorf("expected default Content-Type application/x-ndjson, got %v", ct)
	}
}

func TestHandleJobResultsWithPagination(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	body := `{"url": "https://example.com"}`
	req := httptest.NewRequest("POST", "/v1/scrape", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("failed to create job: got status %v", status)
	}

	var job map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &job); err != nil {
		t.Fatalf("failed to parse job response: %v", err)
	}

	jobID, ok := job["id"].(string)
	if !ok {
		t.Fatalf("job response missing id field")
	}

	dataDir := t.TempDir()
	resultDir := filepath.Join(dataDir, "jobs", jobID)
	if err := fsutil.MkdirAllSecure(resultDir); err != nil {
		t.Fatalf("failed to create result directory: %v", err)
	}

	resultPath := filepath.Join(resultDir, "results.jsonl")
	var resultLines []string
	for i := 1; i <= 150; i++ {
		resultLines = append(resultLines, fmt.Sprintf(`{"url":"https://example.com/page%d","status":200,"title":"Page %d"}`, i, i))
	}
	resultContent := strings.Join(resultLines, "\n")
	if err := os.WriteFile(resultPath, []byte(resultContent), 0o644); err != nil {
		t.Fatalf("failed to write result file: %v", err)
	}

	if err := srv.store.UpdateResultPath(ctx, jobID, resultPath); err != nil {
		t.Fatalf("failed to update result path: %v", err)
	}
	if err := srv.store.UpdateStatus(ctx, jobID, model.StatusSucceeded, ""); err != nil {
		t.Fatalf("failed to update status: %v", err)
	}

	req = httptest.NewRequest("GET", "/v1/jobs/"+jobID+"/results?format=jsonl&limit=50&offset=0", nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected 200, got %v", status)
	}

	totalCountStr := rr.Header().Get("X-Total-Count")
	if totalCountStr == "" {
		t.Error("expected X-Total-Count header")
	}
	totalCount, _ := strconv.Atoi(totalCountStr)
	if totalCount != 150 {
		t.Errorf("expected total count 150, got %d", totalCount)
	}

	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %v", ct)
	}

	var items []map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &items); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(items) != 50 {
		t.Errorf("expected 50 items, got %d", len(items))
	}

	req = httptest.NewRequest("GET", "/v1/jobs/"+jobID+"/results?format=jsonl&limit=50&offset=50", nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	json.Unmarshal(rr.Body.Bytes(), &items)
	if len(items) != 50 {
		t.Errorf("expected 50 items with offset 50, got %d", len(items))
	}

	req = httptest.NewRequest("GET", "/v1/jobs/"+jobID+"/results?format=jsonl&limit=50&offset=100", nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	json.Unmarshal(rr.Body.Bytes(), &items)
	if len(items) != 50 {
		t.Errorf("expected 50 items with offset 100, got %d", len(items))
	}

	req = httptest.NewRequest("GET", "/v1/jobs/"+jobID+"/results?format=jsonl&limit=50&offset=150", nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	json.Unmarshal(rr.Body.Bytes(), &items)
	if len(items) != 0 {
		t.Errorf("expected 0 items with offset beyond total, got %d", len(items))
	}

	req = httptest.NewRequest("GET", "/v1/jobs/"+jobID+"/results?format=jsonl&limit=-1&offset=0", nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	json.Unmarshal(rr.Body.Bytes(), &items)
	if len(items) != 100 {
		t.Errorf("expected default limit of 100 with invalid limit, got %d", len(items))
	}

	req = httptest.NewRequest("GET", "/v1/jobs/"+jobID+"/results?format=jsonl&limit=2000&offset=0", nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	json.Unmarshal(rr.Body.Bytes(), &items)
	if len(items) != 150 {
		t.Errorf("expected max limit of 1000 with limit > 1000, but only 150 items in file, got %d", len(items))
	}

	req = httptest.NewRequest("GET", "/v1/jobs/"+jobID+"/results?format=jsonl&limit=50&offset=-1", nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	json.Unmarshal(rr.Body.Bytes(), &items)
	if len(items) != 50 {
		t.Errorf("expected default offset of 0 with invalid offset, got %d", len(items))
	}

	req = httptest.NewRequest("GET", "/v1/jobs/"+jobID+"/results?format=jsonl&limit=50&offset=0", nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	json.Unmarshal(rr.Body.Bytes(), &items)
	firstTitle, _ := items[0]["title"].(string)
	if firstTitle != "Page 1" {
		t.Errorf("expected first item to be Page 1, got %v", firstTitle)
	}

	fiftiethTitle, _ := items[49]["title"].(string)
	if fiftiethTitle != "Page 50" {
		t.Errorf("expected 50th item to be Page 50, got %v", fiftiethTitle)
	}

	req = httptest.NewRequest("GET", "/v1/jobs/"+jobID+"/results?format=jsonl&limit=50&offset=50", nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	json.Unmarshal(rr.Body.Bytes(), &items)
	firstTitle, _ = items[0]["title"].(string)
	if firstTitle != "Page 51" {
		t.Errorf("expected first item on second page to be Page 51, got %v", firstTitle)
	}

	req = httptest.NewRequest("GET", "/v1/jobs/"+jobID+"/results?format=json&limit=50&offset=0", nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected 200 for json format, got %v", status)
	}

	if ct := rr.Header().Get("X-Total-Count"); ct != "" {
		t.Error("expected no X-Total-Count header for non-jsonl format")
	}
}

func TestHandleJobResultsRouting(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "malformed path double slash",
			method:         "GET",
			path:           "/v1/jobs//results",
			expectedStatus: http.StatusMovedPermanently, // ServeMux redirects // to /
		},
		{
			name:           "missing id segment",
			method:         "GET",
			path:           "/v1/jobs/results",
			expectedStatus: http.StatusNotFound, // results is treated as ID if it doesn't match /results
		},
		{
			name:           "method not allowed",
			method:         "POST",
			path:           "/v1/jobs/some-id/results",
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("%s: expected status %v, got %v", tt.name, tt.expectedStatus, status)
			}
		})
	}
}
