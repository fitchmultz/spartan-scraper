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

func TestHandleJobsWithStatusFilter(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create jobs directly in store to avoid job manager race conditions
	jobs := []struct {
		body   string
		status model.Status
	}{
		{`{"url": "https://example1.com"}`, model.StatusQueued},
		{`{"url": "https://example2.com"}`, model.StatusRunning},
		{`{"url": "https://example3.com"}`, model.StatusSucceeded},
		{`{"url": "https://example4.com"}`, model.StatusFailed},
	}

	jobIDs := make([]string, 0, len(jobs))
	for i, job := range jobs {
		var scrapeReq map[string]interface{}
		if err := json.Unmarshal([]byte(job.body), &scrapeReq); err != nil {
			t.Fatalf("failed to unmarshal request: %v", err)
		}

		newJob := model.Job{
			ID:        fmt.Sprintf("test-job-%d", i),
			Kind:      model.KindScrape,
			Status:    job.status,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Params:    scrapeReq,
		}

		if err := srv.store.Create(context.Background(), newJob); err != nil {
			t.Fatalf("failed to create job: %v", err)
		}
		jobIDs = append(jobIDs, newJob.ID)
	}

	req := httptest.NewRequest("GET", "/v1/jobs?status=queued", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %v", status)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	jobsList, ok := resp["jobs"].([]interface{})
	if !ok {
		t.Fatal("expected 'jobs' array in response")
	}

	if len(jobsList) != 1 {
		t.Errorf("expected 1 queued job, got %d", len(jobsList))
	}
}

func TestHandleJobsWithInvalidStatus(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/v1/jobs?status=invalid_status", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid status, got %v", status)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	if _, ok := resp["error"]; !ok {
		t.Error("expected 'error' field in response")
	}
}

func TestHandleJobsNoStatusFilter(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create jobs directly in store to avoid job manager race conditions
	for i := 0; i < 3; i++ {
		body := fmt.Sprintf(`{"url": "https://example%d.com"}`, i)
		var scrapeReq map[string]interface{}
		if err := json.Unmarshal([]byte(body), &scrapeReq); err != nil {
			t.Fatalf("failed to unmarshal request: %v", err)
		}

		newJob := model.Job{
			ID:        fmt.Sprintf("test-job-no-filter-%d", i),
			Kind:      model.KindScrape,
			Status:    model.StatusQueued,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Params:    scrapeReq,
		}

		if err := srv.store.Create(context.Background(), newJob); err != nil {
			t.Fatalf("failed to create job: %v", err)
		}
	}

	req := httptest.NewRequest("GET", "/v1/jobs", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %v", status)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	jobsList, ok := resp["jobs"].([]interface{})
	if !ok {
		t.Fatal("expected 'jobs' array in response")
	}

	if len(jobsList) != 3 {
		t.Errorf("expected all 3 jobs without filter, got %d", len(jobsList))
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
			if err := os.MkdirAll(resultDir, 0o755); err != nil {
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
	if err := os.MkdirAll(resultDir, 0o755); err != nil {
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
	if err := os.MkdirAll(resultDir, 0o755); err != nil {
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

func TestHandleResearchValidation(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name           string
		body           string
		expectedStatus int
	}{
		{
			name:           "invalid url in urls list",
			body:           `{"query": "test", "urls": ["ftp://example.com"]}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty url in urls list",
			body:           `{"query": "test", "urls": ["", "https://example.com"]}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/research", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

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

func TestZeroValuesAllowed(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name string
		body string
	}{
		{
			name: "scrape with timeout 0",
			body: `{"url": "https://example.com", "timeoutSeconds": 0}`,
		},
		{
			name: "crawl with maxDepth 0",
			body: `{"url": "https://example.com", "maxDepth": 0, "maxPages": 10}`,
		},
		{
			name: "crawl with maxPages 0",
			body: `{"url": "https://example.com", "maxDepth": 2, "maxPages": 0}`,
		},
		{
			name: "research with all zero values",
			body: `{"query": "test", "urls": ["https://example.com"], "timeoutSeconds": 0, "maxDepth": 0, "maxPages": 0}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := "/v1/scrape"
			if tt.name == "crawl with maxDepth 0" || tt.name == "crawl with maxPages 0" {
				endpoint = "/v1/crawl"
			}
			if tt.name == "research with all zero values" {
				endpoint = "/v1/research"
			}
			req := httptest.NewRequest("POST", endpoint, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if status := rr.Code; status != http.StatusOK {
				t.Errorf("handler returned wrong status code for %s: got %v want %v, body: %s", tt.name, status, http.StatusOK, rr.Body.String())
			}
		})
	}
}

func TestHandleSchedulesList(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/v1/schedules", nil)
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
	if _, ok := resp["schedules"]; !ok {
		t.Errorf("expected 'schedules' field in response, got: %v", resp)
	}
}

func TestHandleSchedulesAdd(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name           string
		body           string
		expectedStatus int
	}{
		{
			name:           "valid scrape schedule",
			body:           `{"kind": "scrape", "intervalSeconds": 3600, "url": "https://example.com"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "valid crawl schedule",
			body:           `{"kind": "crawl", "intervalSeconds": 7200, "url": "https://example.com", "maxDepth": 2, "maxPages": 200}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "valid research schedule",
			body:           `{"kind": "research", "intervalSeconds": 86400, "query": "test query", "urls": ["https://example.com"]}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing kind",
			body:           `{"intervalSeconds": 3600, "url": "https://example.com"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid interval (negative)",
			body:           `{"kind": "scrape", "intervalSeconds": -1, "url": "https://example.com"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid kind value",
			body:           `{"kind": "invalid", "intervalSeconds": 3600, "url": "https://example.com"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing content-type",
			body:           `{"kind": "scrape", "intervalSeconds": 3600, "url": "https://example.com"}`,
			expectedStatus: http.StatusUnsupportedMediaType,
		},
		{
			name:           "missing url for scrape",
			body:           `{"kind": "scrape", "intervalSeconds": 3600}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing url for crawl",
			body:           `{"kind": "crawl", "intervalSeconds": 7200, "maxDepth": 2, "maxPages": 200}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing query and urls for research",
			body:           `{"kind": "research", "intervalSeconds": 86400}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid timeout too low",
			body:           `{"kind": "scrape", "intervalSeconds": 3600, "url": "https://example.com", "timeoutSeconds": 1}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid timeout too high",
			body:           `{"kind": "scrape", "intervalSeconds": 3600, "url": "https://example.com", "timeoutSeconds": 600}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/schedules", strings.NewReader(tt.body))
			if tt.name != "missing content-type" {
				req.Header.Set("Content-Type", "application/json")
			}
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v, body: %s", status, tt.expectedStatus, rr.Body.String())
			}

			if tt.expectedStatus == http.StatusOK {
				var resp map[string]interface{}
				if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
					t.Errorf("failed to parse JSON response: %v", err)
				}
				if _, ok := resp["id"]; !ok {
					t.Errorf("expected 'id' field in schedule response, got: %v", resp)
				}
				if _, ok := resp["kind"]; !ok {
					t.Errorf("expected 'kind' field in schedule response, got: %v", resp)
				}
				if _, ok := resp["intervalSeconds"]; !ok {
					t.Errorf("expected 'intervalSeconds' field in schedule response, got: %v", resp)
				}
				if _, ok := resp["nextRun"]; !ok {
					t.Errorf("expected 'nextRun' field in schedule response, got: %v", resp)
				}
			} else {
				var resp map[string]interface{}
				if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
					t.Errorf("failed to parse JSON response: %v", err)
				}
				if _, ok := resp["error"]; !ok {
					t.Errorf("expected 'error' field in error response, got: %v", resp)
				}
			}
		})
	}
}

func TestHandleScheduleDelete(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	addBody := `{"kind": "scrape", "intervalSeconds": 3600, "url": "https://example.com", "headless": false}`
	req := httptest.NewRequest("POST", "/v1/schedules", strings.NewReader(addBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("failed to add schedule: got status %v, body: %s", status, rr.Body.String())
	}

	var addResp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &addResp); err != nil {
		t.Fatalf("failed to parse add response: %v, body: %s", err, rr.Body.String())
	}

	scheduleID, ok := addResp["id"].(string)
	if !ok {
		t.Fatalf("add response missing id field, got: %+v", addResp)
	}

	t.Logf("Schedule ID: %s", scheduleID)

	req = httptest.NewRequest("DELETE", fmt.Sprintf("/v1/schedules/%s", scheduleID), nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v, body: %s", status, http.StatusOK, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Errorf("failed to parse delete response: %v", err)
	}

	if status, ok := resp["status"].(string); !ok || status != "ok" {
		t.Errorf("expected status 'ok', got %v", resp["status"])
	}
}

func TestHandleScheduleDeleteNotFound(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("DELETE", "/v1/schedules/nonexistent-id", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("deleting non-existent schedule should succeed (idempotent), got status %v", status)
	}
}

func TestHandleScheduleDeleteInvalidID(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{
			name:           "slash only",
			path:           "/v1/schedules/",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "base path without ID",
			path:           "/v1/schedules",
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("DELETE", tt.path, nil)
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("expected status %v for invalid path %s, got %v", tt.expectedStatus, tt.path, status)
			}
		})
	}
}

func TestHandleTemplates(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/v1/templates", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %v", ct)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	templates, ok := response["templates"].([]interface{})
	if !ok {
		t.Fatal("expected templates array in response")
	}

	// Should have at least built-in templates (default, article, product)
	if len(templates) < 3 {
		t.Errorf("expected at least 3 templates, got %d", len(templates))
	}
}

func TestHandleCrawlStates(t *testing.T) {
	ctx := context.Background()
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Insert test data
	state := model.CrawlState{
		URL:          "https://example.com/test",
		ETag:         "test-etag",
		LastModified: "test-modified",
		ContentHash:  "test-hash",
		LastScraped:  time.Now(),
	}
	err := srv.store.UpsertCrawlState(ctx, state)
	if err != nil {
		t.Fatalf("failed to insert crawl state: %v", err)
	}

	req := httptest.NewRequest("GET", "/v1/crawl-states", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	crawlStates, ok := response["crawlStates"].([]interface{})
	if !ok {
		t.Fatal("expected crawlStates array in response")
	}

	if len(crawlStates) != 1 {
		t.Errorf("expected 1 crawl state, got %d", len(crawlStates))
	}
}

func TestHandleCrawlStatesPagination(t *testing.T) {
	ctx := context.Background()
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Insert multiple states
	for i := 1; i <= 5; i++ {
		state := model.CrawlState{
			URL:         fmt.Sprintf("https://example.com/page%d", i),
			ContentHash: fmt.Sprintf("hash%d", i),
			LastScraped: time.Now(),
		}
		err := srv.store.UpsertCrawlState(ctx, state)
		if err != nil {
			t.Fatalf("failed to insert crawl state: %v", err)
		}
	}

	// Test limit
	req := httptest.NewRequest("GET", "/v1/crawl-states?limit=2", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("expected status 200, got %v", status)
	}

	var response map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &response)
	crawlStates := response["crawlStates"].([]interface{})
	if len(crawlStates) != 2 {
		t.Errorf("expected 2 crawl states with limit, got %d", len(crawlStates))
	}

	// Test offset
	req = httptest.NewRequest("GET", "/v1/crawl-states?offset=3", nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	json.Unmarshal(rr.Body.Bytes(), &response)
	crawlStates = response["crawlStates"].([]interface{})
	if len(crawlStates) != 2 {
		t.Errorf("expected 2 crawl states with offset 3, got %d", len(crawlStates))
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
	if err := os.MkdirAll(resultDir, 0o755); err != nil {
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

	totalCountStr = rr.Header().Get("X-Total-Count")
	if totalCountStr != "" {
		t.Error("expected no X-Total-Count header for non-jsonl format")
	}
}
