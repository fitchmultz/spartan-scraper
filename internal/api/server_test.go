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

	"spartan-scraper/internal/config"
	"spartan-scraper/internal/jobs"
	"spartan-scraper/internal/model"
	"spartan-scraper/internal/store"
)

func setupTestServer(t *testing.T) (*Server, func()) {
	t.Helper()
	dataDir := t.TempDir()
	cfg := config.Config{
		DataDir:            dataDir,
		RequestTimeoutSecs: 30,
		MaxConcurrency:     4,
		RateLimitQPS:       10,
		RateLimitBurst:     20,
		MaxRetries:         3,
		RetryBaseMs:        100,
		UserAgent:          "SpartanTest/1.0",
		Port:               "0", // not used for Routes() test
	}

	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}

	manager := jobs.NewManager(
		st,
		dataDir,
		cfg.UserAgent,
		time.Duration(cfg.RequestTimeoutSecs)*time.Second,
		cfg.MaxConcurrency,
		cfg.RateLimitQPS,
		cfg.RateLimitBurst,
		cfg.MaxRetries,
		time.Duration(cfg.RetryBaseMs)*time.Millisecond,
		cfg.MaxResponseBytes,
		false,
	)
	ctx, cancel := context.WithCancel(context.Background())
	manager.Start(ctx)

	srv := NewServer(manager, st, cfg)

	cleanup := func() {
		cancel()
		manager.Wait()
		st.Close()
	}

	return srv, cleanup
}

func TestHealth(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/healthz", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if !strings.Contains(rr.Body.String(), `"status":"ok"`) {
		t.Errorf("handler returned unexpected body: got %v", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"database"`) {
		t.Errorf("handler missing database status: got %v", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"queue"`) {
		t.Errorf("handler missing queue status: got %v", rr.Body.String())
	}
}

func TestHandleJobs(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/v1/jobs", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %v", ct)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Errorf("failed to parse JSON response: %v", err)
	}
	if _, ok := resp["jobs"]; !ok {
		t.Errorf("expected 'jobs' field in response, got: %v", resp)
	}
}

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

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("expected status %v for job with no results, got %v", http.StatusNotFound, status)
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
			if err := os.MkdirAll(resultDir, 0o755); err != nil {
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

func TestContentTypeForExtension(t *testing.T) {
	tests := []struct {
		name         string
		ext          string
		expectedType string
	}{
		{name: "jsonl", ext: ".jsonl", expectedType: "application/x-ndjson"},
		{name: "JSONL uppercase", ext: ".JSONL", expectedType: "application/x-ndjson"},
		{name: "json", ext: ".json", expectedType: "application/json"},
		{name: "JSON uppercase", ext: ".JSON", expectedType: "application/json"},
		{name: "csv", ext: ".csv", expectedType: "text/csv"},
		{name: "xml", ext: ".xml", expectedType: "application/xml"},
		{name: "txt", ext: ".txt", expectedType: "text/plain; charset=utf-8"},
		{name: "unknown extension", ext: ".unknown", expectedType: ""},
		{name: "no extension", ext: "", expectedType: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contentTypeForExtension(tt.ext)
			if result != tt.expectedType {
				t.Errorf("contentTypeForExtension(%q) = %q, want %q", tt.ext, result, tt.expectedType)
			}
		})
	}
}

func TestHandleScrape(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{"url": "https://example.com"}`
	req := httptest.NewRequest("POST", "/v1/scrape", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestHandleScrapeValidation(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name           string
		body           string
		contentType    string
		expectedStatus int
	}{
		{
			name:           "missing content-type",
			body:           `{"url": "https://example.com"}`,
			contentType:    "",
			expectedStatus: http.StatusUnsupportedMediaType,
		},
		{
			name:           "invalid json",
			body:           `{"url": "https://example.com"`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing url",
			body:           `{}`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid url scheme",
			body:           `{"url": "ftp://example.com"}`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid url host",
			body:           `{"url": "https://"}`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid auth profile",
			body:           `{"url": "https://example.com", "authProfile": "invalid name!"}`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "timeout too low",
			body:           `{"url": "https://example.com", "timeoutSeconds": 1}`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "timeout too high",
			body:           `{"url": "https://example.com", "timeoutSeconds": 600}`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/scrape", strings.NewReader(tt.body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

			// Verify JSON response
			if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("expected Content-Type application/json, got %v", ct)
			}

			// Verify error field exists
			var resp map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Errorf("failed to parse JSON response: %v", err)
			}
			if _, ok := resp["error"]; !ok {
				t.Errorf("expected 'error' field in response, got: %v", resp)
			}
		})
	}
}

func TestHandleCrawlValidation(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name           string
		body           string
		expectedStatus int
	}{
		{
			name:           "invalid maxDepth",
			body:           `{"url": "https://example.com", "maxDepth": 11}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid maxPages",
			body:           `{"url": "https://example.com", "maxPages": 20000}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/crawl", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

			// Verify JSON response
			if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("expected Content-Type application/json, got %v", ct)
			}

			// Verify error field exists
			var resp map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Errorf("failed to parse JSON response: %v", err)
			}
			if _, ok := resp["error"]; !ok {
				t.Errorf("expected 'error' field in response, got: %v", resp)
			}
		})
	}
}

func TestHandleResearch(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{"query": "test query", "urls": ["https://example.com"]}`
	req := httptest.NewRequest("POST", "/v1/research", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestHandleAuthImportPathTraversal(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name           string
		body           string
		expectedStatus int
	}{
		{
			name:           "valid filename",
			body:           `{"path": "backup.json"}`,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "empty path",
			body:           `{"path": ""}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "absolute path",
			body:           `{"path": "/tmp/backup.json"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "path traversal with ..",
			body:           `{"path": "../backup.json"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "with directory",
			body:           `{"path": "subdir/backup.json"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "backslash",
			body:           `{"path": "subdir\\backup.json"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "double slash",
			body:           `{"path": "sub//backup.json"}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/auth/import", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v, body: %s", status, tt.expectedStatus, rr.Body.String())
			}

			// Verify JSON response
			if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("expected Content-Type application/json, got %v", ct)
			}

			// Verify error field exists
			var resp map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Errorf("failed to parse JSON response: %v", err)
			}
			if _, ok := resp["error"]; !ok {
				t.Errorf("expected 'error' field in response, got: %v", resp)
			}
		})
	}
}

func TestHandleAuthExportPathTraversal(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name           string
		body           string
		expectedStatus int
	}{
		{
			name:           "valid filename",
			body:           `{"path": "backup.json"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "empty path",
			body:           `{"path": ""}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "absolute path",
			body:           `{"path": "/tmp/backup.json"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "path traversal with ..",
			body:           `{"path": "../backup.json"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "with directory",
			body:           `{"path": "subdir/backup.json"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "backslash",
			body:           `{"path": "subdir\\backup.json"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "double slash",
			body:           `{"path": "sub//backup.json"}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/auth/export", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v, body: %s", status, tt.expectedStatus, rr.Body.String())
			}

			// Only verify JSON error response for error status codes
			if tt.expectedStatus != http.StatusOK {
				// Verify JSON response
				if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
					t.Errorf("expected Content-Type application/json, got %v", ct)
				}

				// Verify error field exists
				var resp map[string]interface{}
				if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
					t.Errorf("failed to parse JSON response: %v", err)
				}
				if _, ok := resp["error"]; !ok {
					t.Errorf("expected 'error' field in response, got: %v", resp)
				}
			}
		})
	}
}

func TestHandleJobForceDelete(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a job via API
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
		t.Fatalf("job response missing id field")
	}

	// Create job directory and result file
	dataDir := srv.cfg.DataDir
	jobDir := filepath.Join(dataDir, "jobs", jobID)
	if err := os.MkdirAll(jobDir, 0o755); err != nil {
		t.Fatalf("failed to create job directory: %v", err)
	}

	resultPath := filepath.Join(jobDir, "results.jsonl")
	resultContent := `{"test":"data"}`
	if err := os.WriteFile(resultPath, []byte(resultContent), 0o644); err != nil {
		t.Fatalf("failed to write result file: %v", err)
	}

	// Update job with result path
	st := srv.store
	ctx := context.Background()
	if err := st.UpdateResultPath(ctx, jobID, resultPath); err != nil {
		t.Fatalf("failed to update result path: %v", err)
	}

	// Force delete the job
	req = httptest.NewRequest("DELETE", fmt.Sprintf("/v1/jobs/%s?force=true", jobID), nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("force delete failed: got status %v, body: %s", status, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse delete response: %v", err)
	}

	if status, ok := resp["status"].(string); !ok || status != "deleted" {
		t.Errorf("expected status 'deleted', got %v", resp["status"])
	}

	// Verify job is gone from DB
	_, err := st.Get(ctx, jobID)
	if err == nil {
		t.Error("job should be deleted from database after force delete")
	}

	// Verify result file is deleted
	if _, err := os.Stat(resultPath); !os.IsNotExist(err) {
		t.Error("result file should be deleted after force delete")
	}

	// Verify job directory is deleted
	if _, err := os.Stat(jobDir); !os.IsNotExist(err) {
		t.Error("job directory should be deleted after force delete")
	}
}

func TestHandleJobCancelNotDelete(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a job via API
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

	// Create job directory and result file
	dataDir := srv.cfg.DataDir
	jobDir := filepath.Join(dataDir, "jobs", jobID)
	if err := os.MkdirAll(jobDir, 0o755); err != nil {
		t.Fatalf("failed to create job directory: %v", err)
	}

	resultPath := filepath.Join(jobDir, "results.jsonl")
	resultContent := `{"test":"data"}`
	if err := os.WriteFile(resultPath, []byte(resultContent), 0o644); err != nil {
		t.Fatalf("failed to write result file: %v", err)
	}

	// Update job with result path
	st := srv.store
	ctx := context.Background()
	if err := st.UpdateResultPath(ctx, jobID, resultPath); err != nil {
		t.Fatalf("failed to update result path: %v", err)
	}

	// Cancel the job (without force=true)
	req = httptest.NewRequest("DELETE", fmt.Sprintf("/v1/jobs/%s", jobID), nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("cancel failed: got status %v", status)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse delete response: %v", err)
	}

	if status, ok := resp["status"].(string); !ok || status != "canceled" {
		t.Errorf("expected status 'canceled', got %v", resp["status"])
	}

	// Verify job still exists in DB (canceled, not deleted)
	gotJob, err := st.Get(ctx, jobID)
	if err != nil {
		t.Error("job should still exist in database after cancel")
	}
	if gotJob.Status != model.StatusCanceled {
		t.Errorf("job status should be 'canceled', got %s", gotJob.Status)
	}

	// Verify result file still exists
	if _, err := os.Stat(resultPath); os.IsNotExist(err) {
		t.Error("result file should still exist after cancel")
	}

	// Verify job directory still exists
	if _, err := os.Stat(jobDir); os.IsNotExist(err) {
		t.Error("job directory should still exist after cancel")
	}
}
