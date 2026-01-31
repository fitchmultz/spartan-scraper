// Package api provides tests for traffic replay functionality.
// Tests cover validation, filtering, URL transformation, and replay execution.
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestHandleTrafficReplayMethodNotAllowed(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/v1/jobs/replay/test-job", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("expected status %v, got %v", http.StatusMethodNotAllowed, status)
	}
}

func TestHandleTrafficReplayMissingJobID(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("POST", "/v1/jobs/replay/", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("expected status %v, got %v", http.StatusBadRequest, status)
	}
}

func TestHandleTrafficReplayInvalidBody(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("POST", "/v1/jobs/replay/test-job", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("expected status %v, got %v", http.StatusBadRequest, status)
	}
}

func TestHandleTrafficReplayMissingTargetURL(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{"jobId": "test-job"}`
	req := httptest.NewRequest("POST", "/v1/jobs/replay/test-job", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("expected status %v, got %v", http.StatusBadRequest, status)
	}
}

func TestHandleTrafficReplayInvalidTargetURL(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{"jobId": "test-job", "targetBaseUrl": "://invalid-url"}`
	req := httptest.NewRequest("POST", "/v1/jobs/replay/test-job", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("expected status %v, got %v", http.StatusBadRequest, status)
	}
}

func TestHandleTrafficReplayJobNotFound(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{"jobId": "nonexistent-job", "targetBaseUrl": "https://example.com"}`
	req := httptest.NewRequest("POST", "/v1/jobs/replay/nonexistent-job", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("expected status %v, got %v", http.StatusNotFound, status)
	}
}

func TestHandleTrafficReplayJobStatusValidation(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	ctx := t.Context()

	tests := []struct {
		name           string
		status         model.Status
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jobID := "test-job-" + strings.ReplaceAll(tt.name, " ", "-")
			job := model.Job{
				ID:        jobID,
				Kind:      model.KindScrape,
				Status:    tt.status,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			if err := srv.store.Create(ctx, job); err != nil {
				t.Fatalf("failed to create job: %v", err)
			}

			body := fmt.Sprintf(`{"jobId": "%s", "targetBaseUrl": "https://example.com"}`, jobID)
			req := httptest.NewRequest("POST", fmt.Sprintf("/v1/jobs/replay/%s", jobID), strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("expected status %v, got %v", tt.expectedStatus, status)
			}

			var resp map[string]any
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse error response: %v", err)
			}
			if msg, ok := resp["error"].(string); !ok || !strings.Contains(msg, tt.expectedMsg) {
				t.Errorf("expected error message to contain %q, got %q", tt.expectedMsg, msg)
			}
		})
	}
}

func TestHandleTrafficReplayNoResultPath(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	ctx := t.Context()

	jobID := "test-job-no-result-path"
	job := model.Job{
		ID:        jobID,
		Kind:      model.KindScrape,
		Status:    model.StatusSucceeded,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		// No ResultPath set
	}

	if err := srv.store.Create(ctx, job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	body := fmt.Sprintf(`{"jobId": "%s", "targetBaseUrl": "https://example.com"}`, jobID)
	req := httptest.NewRequest("POST", fmt.Sprintf("/v1/jobs/replay/%s", jobID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("expected status %v, got %v", http.StatusNotFound, status)
	}
}

func TestHandleTrafficReplayNoTrafficData(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	ctx := t.Context()

	// Create empty result file
	tmpDir := t.TempDir()
	resultPath := filepath.Join(tmpDir, "results.jsonl")
	os.WriteFile(resultPath, []byte(""), 0644)

	jobID := "test-job-no-traffic"
	job := model.Job{
		ID:         jobID,
		Kind:       model.KindScrape,
		Status:     model.StatusSucceeded,
		ResultPath: resultPath,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := srv.store.Create(ctx, job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	body := fmt.Sprintf(`{"jobId": "%s", "targetBaseUrl": "https://example.com"}`, jobID)
	req := httptest.NewRequest("POST", fmt.Sprintf("/v1/jobs/replay/%s", jobID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("expected status %v, got %v", http.StatusNotFound, status)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	if msg, ok := resp["error"].(string); !ok || !strings.Contains(msg, "no intercepted traffic") {
		t.Errorf("expected error message about no traffic, got %q", msg)
	}
}

func TestHandleTrafficReplayWithTrafficData(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	ctx := t.Context()

	// Create a test server to replay against
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"replayed": true}`))
	}))
	defer testServer.Close()

	// Create result file with intercepted traffic
	tmpDir := t.TempDir()
	resultPath := filepath.Join(tmpDir, "results.jsonl")

	entry := struct {
		InterceptedData []fetch.InterceptedEntry `json:"interceptedData"`
	}{
		InterceptedData: []fetch.InterceptedEntry{
			{
				Request: fetch.InterceptedRequest{
					RequestID:    "req-1",
					URL:          testServer.URL + "/api/test",
					Method:       "GET",
					Headers:      map[string]string{"Accept": "application/json"},
					ResourceType: fetch.ResourceTypeXHR,
				},
				Response: &fetch.InterceptedResponse{
					RequestID:  "req-1",
					Status:     200,
					StatusText: "OK",
					Headers:    map[string]string{"Content-Type": "application/json"},
					Body:       `{"original": true}`,
					BodySize:   16,
				},
				Duration: 100 * time.Millisecond,
			},
		},
	}

	data, _ := json.Marshal(entry)
	os.WriteFile(resultPath, append(data, '\n'), 0644)

	jobID := "test-job-with-traffic"
	job := model.Job{
		ID:         jobID,
		Kind:       model.KindScrape,
		Status:     model.StatusSucceeded,
		ResultPath: resultPath,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := srv.store.Create(ctx, job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	body := fmt.Sprintf(`{"jobId": "%s", "targetBaseUrl": "%s"}`, jobID, testServer.URL)
	req := httptest.NewRequest("POST", fmt.Sprintf("/v1/jobs/replay/%s", jobID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status %v, got %v: %s", http.StatusOK, status, rr.Body.String())
	}

	var resp TrafficReplayResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.TotalRequests != 1 {
		t.Errorf("expected 1 total request, got %d", resp.TotalRequests)
	}
	if resp.Successful != 1 {
		t.Errorf("expected 1 successful, got %d", resp.Successful)
	}
	if resp.Failed != 0 {
		t.Errorf("expected 0 failed, got %d", resp.Failed)
	}
}

func TestHandleTrafficReplayWithFilter(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	ctx := t.Context()

	// Create a test server to replay against
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer testServer.Close()

	// Create result file with multiple entries
	tmpDir := t.TempDir()
	resultPath := filepath.Join(tmpDir, "results.jsonl")

	entry := struct {
		InterceptedData []fetch.InterceptedEntry `json:"interceptedData"`
	}{
		InterceptedData: []fetch.InterceptedEntry{
			{
				Request: fetch.InterceptedRequest{
					RequestID:    "req-1",
					URL:          testServer.URL + "/api/users",
					Method:       "GET",
					ResourceType: fetch.ResourceTypeXHR,
				},
				Response: &fetch.InterceptedResponse{
					RequestID: "req-1",
					Status:    200,
					BodySize:  10,
				},
			},
			{
				Request: fetch.InterceptedRequest{
					RequestID:    "req-2",
					URL:          testServer.URL + "/api/posts",
					Method:       "POST",
					ResourceType: fetch.ResourceTypeFetch,
				},
				Response: &fetch.InterceptedResponse{
					RequestID: "req-2",
					Status:    201,
					BodySize:  20,
				},
			},
			{
				Request: fetch.InterceptedRequest{
					RequestID:    "req-3",
					URL:          testServer.URL + "/script.js",
					Method:       "GET",
					ResourceType: fetch.ResourceTypeScript,
				},
				Response: &fetch.InterceptedResponse{
					RequestID: "req-3",
					Status:    200,
					BodySize:  30,
				},
			},
		},
	}

	data, _ := json.Marshal(entry)
	os.WriteFile(resultPath, append(data, '\n'), 0644)

	jobID := "test-job-filter"
	job := model.Job{
		ID:         jobID,
		Kind:       model.KindScrape,
		Status:     model.StatusSucceeded,
		ResultPath: resultPath,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := srv.store.Create(ctx, job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	// Test with method filter
	body := fmt.Sprintf(`{
		"jobId": "%s",
		"targetBaseUrl": "%s",
		"filter": {
			"methods": ["GET"]
		}
	}`, jobID, testServer.URL)

	req := httptest.NewRequest("POST", fmt.Sprintf("/v1/jobs/replay/%s", jobID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status %v, got %v: %s", http.StatusOK, status, rr.Body.String())
	}

	var resp TrafficReplayResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Should only replay GET requests (2 out of 3)
	if resp.TotalRequests != 2 {
		t.Errorf("expected 2 total requests (GET only), got %d", resp.TotalRequests)
	}
}

func TestHandleTrafficReplayWithComparison(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	ctx := t.Context()

	// Create a test server that returns different response
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"different": true}`)) // Different from original
	}))
	defer testServer.Close()

	// Create result file with intercepted traffic
	tmpDir := t.TempDir()
	resultPath := filepath.Join(tmpDir, "results.jsonl")

	entry := struct {
		InterceptedData []fetch.InterceptedEntry `json:"interceptedData"`
	}{
		InterceptedData: []fetch.InterceptedEntry{
			{
				Request: fetch.InterceptedRequest{
					RequestID:    "req-1",
					URL:          testServer.URL + "/api/test",
					Method:       "GET",
					Headers:      map[string]string{"Accept": "application/json"},
					ResourceType: fetch.ResourceTypeXHR,
				},
				Response: &fetch.InterceptedResponse{
					RequestID:  "req-1",
					Status:     200,
					StatusText: "OK",
					Headers:    map[string]string{"Content-Type": "application/json"},
					Body:       `{"original": true}`,
					BodySize:   16,
				},
				Duration: 100 * time.Millisecond,
			},
		},
	}

	data, _ := json.Marshal(entry)
	os.WriteFile(resultPath, append(data, '\n'), 0644)

	jobID := "test-job-compare"
	job := model.Job{
		ID:         jobID,
		Kind:       model.KindScrape,
		Status:     model.StatusSucceeded,
		ResultPath: resultPath,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := srv.store.Create(ctx, job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	body := fmt.Sprintf(`{
		"jobId": "%s",
		"targetBaseUrl": "%s",
		"compareResponses": true
	}`, jobID, testServer.URL)

	req := httptest.NewRequest("POST", fmt.Sprintf("/v1/jobs/replay/%s", jobID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status %v, got %v: %s", http.StatusOK, status, rr.Body.String())
	}

	var resp TrafficReplayResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Comparison == nil {
		t.Fatal("expected comparison data")
	}

	if resp.Comparison.TotalCompared != 1 {
		t.Errorf("expected 1 comparison, got %d", resp.Comparison.TotalCompared)
	}

	if resp.Comparison.Mismatches != 1 {
		t.Errorf("expected 1 mismatch (bodies differ), got %d", resp.Comparison.Mismatches)
	}

	if resp.Comparison.Matches != 0 {
		t.Errorf("expected 0 matches, got %d", resp.Comparison.Matches)
	}
}

func TestHandleTrafficReplayFilterNoMatch(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	ctx := t.Context()

	// Create result file with intercepted traffic
	tmpDir := t.TempDir()
	resultPath := filepath.Join(tmpDir, "results.jsonl")

	entry := struct {
		InterceptedData []fetch.InterceptedEntry `json:"interceptedData"`
	}{
		InterceptedData: []fetch.InterceptedEntry{
			{
				Request: fetch.InterceptedRequest{
					RequestID:    "req-1",
					URL:          "https://example.com/api/test",
					Method:       "GET",
					ResourceType: fetch.ResourceTypeXHR,
				},
				Response: &fetch.InterceptedResponse{
					RequestID: "req-1",
					Status:    200,
					BodySize:  10,
				},
			},
		},
	}

	data, _ := json.Marshal(entry)
	os.WriteFile(resultPath, append(data, '\n'), 0644)

	jobID := "test-job-filter-no-match"
	job := model.Job{
		ID:         jobID,
		Kind:       model.KindScrape,
		Status:     model.StatusSucceeded,
		ResultPath: resultPath,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := srv.store.Create(ctx, job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	// Filter that won't match anything
	body := fmt.Sprintf(`{
		"jobId": "%s",
		"targetBaseUrl": "https://example.com",
		"filter": {
			"methods": ["POST"]
		}
	}`, jobID)

	req := httptest.NewRequest("POST", fmt.Sprintf("/v1/jobs/replay/%s", jobID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("expected status %v, got %v", http.StatusBadRequest, status)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	if msg, ok := resp["error"].(string); !ok || !strings.Contains(msg, "no requests match") {
		t.Errorf("expected error message about no matches, got %q", msg)
	}
}

func TestFilterEntries(t *testing.T) {
	entries := []fetch.InterceptedEntry{
		{
			Request: fetch.InterceptedRequest{
				URL:          "https://example.com/api/users",
				Method:       "GET",
				ResourceType: fetch.ResourceTypeXHR,
			},
			Response: &fetch.InterceptedResponse{Status: 200},
		},
		{
			Request: fetch.InterceptedRequest{
				URL:          "https://example.com/api/posts",
				Method:       "POST",
				ResourceType: fetch.ResourceTypeFetch,
			},
			Response: &fetch.InterceptedResponse{Status: 201},
		},
		{
			Request: fetch.InterceptedRequest{
				URL:          "https://example.com/script.js",
				Method:       "GET",
				ResourceType: fetch.ResourceTypeScript,
			},
			Response: &fetch.InterceptedResponse{Status: 200},
		},
	}

	tests := []struct {
		name     string
		filter   *TrafficReplayFilter
		expected int
	}{
		{
			name:     "no filter",
			filter:   nil,
			expected: 3,
		},
		{
			name: "filter by method GET",
			filter: &TrafficReplayFilter{
				Methods: []string{"GET"},
			},
			expected: 2,
		},
		{
			name: "filter by method POST",
			filter: &TrafficReplayFilter{
				Methods: []string{"POST"},
			},
			expected: 1,
		},
		{
			name: "filter by resource type XHR",
			filter: &TrafficReplayFilter{
				ResourceTypes: []string{"xhr"},
			},
			expected: 1,
		},
		{
			name: "filter by status code 200",
			filter: &TrafficReplayFilter{
				StatusCodes: []int{200},
			},
			expected: 2,
		},
		{
			name: "filter by URL pattern",
			filter: &TrafficReplayFilter{
				URLPatterns: []string{"*api*"},
			},
			expected: 2,
		},
		{
			name: "filter with no matches",
			filter: &TrafficReplayFilter{
				Methods: []string{"DELETE"},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterEntries(entries, tt.filter)
			if len(result) != tt.expected {
				t.Errorf("expected %d entries, got %d", tt.expected, len(result))
			}
		})
	}
}

func TestMatchURLPattern(t *testing.T) {
	tests := []struct {
		url     string
		pattern string
		want    bool
	}{
		{"https://example.com/api/users", "*api*", true},
		{"https://example.com/api/users", "*users", true},
		{"https://example.com/api/users", "*posts", false},
		{"https://example.com/api/users/123", "*api/**", true},
		{"https://example.com/api/users", "https://example.com/api/*", true},
		{"https://example.com/api/users", "https://*.com/api/*", true},
		{"https://example.com/api/users", "*", true},
		{"https://example.com/api/users", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			got := matchURLPattern(tt.url, tt.pattern)
			if got != tt.want {
				t.Errorf("matchURLPattern(%q, %q) = %v, want %v", tt.url, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestTransformURL(t *testing.T) {
	tests := []struct {
		original string
		target   string
		want     string
		wantErr  bool
	}{
		{
			original: "https://prod.example.com/api/users",
			target:   "https://staging.example.com",
			want:     "https://staging.example.com/api/users",
			wantErr:  false,
		},
		{
			original: "https://prod.example.com/api/users?id=123",
			target:   "https://staging.example.com",
			want:     "https://staging.example.com/api/users?id=123",
			wantErr:  false,
		},
		{
			original: "https://prod.example.com/api/users#section",
			target:   "https://staging.example.com",
			want:     "https://staging.example.com/api/users#section",
			wantErr:  false,
		},
		{
			original: "://invalid-url",
			target:   "https://staging.example.com",
			want:     "",
			wantErr:  true,
		},
		{
			original: "https://prod.example.com/api/users",
			target:   "://invalid-target",
			want:     "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.original, func(t *testing.T) {
			got, err := transformURL(tt.original, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("transformURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("transformURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHeadersToMap(t *testing.T) {
	headers := http.Header{
		"Content-Type": []string{"application/json"},
		"X-Custom":     []string{"value1", "value2"}, // Multiple values, should take first
	}

	result := headersToMap(headers)

	if result["Content-Type"] != "application/json" {
		t.Errorf("expected Content-Type to be 'application/json', got %q", result["Content-Type"])
	}

	if result["X-Custom"] != "value1" {
		t.Errorf("expected X-Custom to be 'value1', got %q", result["X-Custom"])
	}
}

func TestCompareResponse(t *testing.T) {
	tests := []struct {
		name     string
		original *fetch.InterceptedResponse
		replayed ReplayResponseInfo
		wantDiff bool
	}{
		{
			name: "identical responses",
			original: &fetch.InterceptedResponse{
				Status:   200,
				Headers:  map[string]string{"Content-Type": "application/json"},
				BodySize: 100,
			},
			replayed: ReplayResponseInfo{
				Status:   200,
				Headers:  map[string]string{"Content-Type": "application/json"},
				BodySize: 100,
			},
			wantDiff: false,
		},
		{
			name: "different status",
			original: &fetch.InterceptedResponse{
				Status:   200,
				Headers:  map[string]string{"Content-Type": "application/json"},
				BodySize: 100,
			},
			replayed: ReplayResponseInfo{
				Status:   404,
				Headers:  map[string]string{"Content-Type": "application/json"},
				BodySize: 100,
			},
			wantDiff: true,
		},
		{
			name: "different body size",
			original: &fetch.InterceptedResponse{
				Status:   200,
				Headers:  map[string]string{"Content-Type": "application/json"},
				BodySize: 100,
				Body:     "original body",
			},
			replayed: ReplayResponseInfo{
				Status:   200,
				Headers:  map[string]string{"Content-Type": "application/json"},
				BodySize: 150,
				Body:     "different body content here",
			},
			wantDiff: true,
		},
		{
			name: "different headers",
			original: &fetch.InterceptedResponse{
				Status:   200,
				Headers:  map[string]string{"Content-Type": "application/json"},
				BodySize: 100,
			},
			replayed: ReplayResponseInfo{
				Status:   200,
				Headers:  map[string]string{"Content-Type": "text/html"},
				BodySize: 100,
			},
			wantDiff: true,
		},
		{
			name: "new header in replay",
			original: &fetch.InterceptedResponse{
				Status:   200,
				Headers:  map[string]string{"Content-Type": "application/json"},
				BodySize: 100,
			},
			replayed: ReplayResponseInfo{
				Status:   200,
				Headers:  map[string]string{"Content-Type": "application/json", "X-New": "value"},
				BodySize: 100,
			},
			wantDiff: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := compareResponse(tt.original, tt.replayed)
			if tt.wantDiff && diff == nil {
				t.Error("expected diff, got nil")
			}
			if !tt.wantDiff && diff != nil {
				t.Errorf("expected no diff, got %+v", diff)
			}
		})
	}
}

func TestGenerateBodyDiffPreview(t *testing.T) {
	tests := []struct {
		name     string
		original string
		replayed string
		want     string
	}{
		{
			name:     "identical bodies",
			original: "hello world",
			replayed: "hello world",
			want:     "Bodies are identical",
		},
		{
			name:     "different length",
			original: "short",
			replayed: "short!",
			want:     "Length differs: 5 vs 6",
		},
		{
			name:     "different content",
			original: "hello world this is a test string",
			replayed: "hello world that is a test string",
			want:     "Diff at position",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateBodyDiffPreview(tt.original, tt.replayed)
			if !strings.Contains(got, tt.want) {
				t.Errorf("generateBodyDiffPreview() = %q, want to contain %q", got, tt.want)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		s      string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hello..."},
		{"", 5, ""},
		{"test", 4, "test"},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			got := truncateString(tt.s, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestHandleTrafficReplayWithModifications(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	ctx := t.Context()

	// Create a test server that echoes headers back
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Echo the X-Custom header if present
		if custom := r.Header.Get("X-Custom"); custom != "" {
			w.Header().Set("X-Echo-Custom", custom)
		}
		// Check that X-Removed is not present
		if r.Header.Get("X-Removed") != "" {
			w.Header().Set("X-Removed-Found", "true")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer testServer.Close()

	// Create result file with intercepted traffic
	tmpDir := t.TempDir()
	resultPath := filepath.Join(tmpDir, "results.jsonl")

	entry := struct {
		InterceptedData []fetch.InterceptedEntry `json:"interceptedData"`
	}{
		InterceptedData: []fetch.InterceptedEntry{
			{
				Request: fetch.InterceptedRequest{
					RequestID: "req-1",
					URL:       testServer.URL + "/api/test",
					Method:    "GET",
					Headers: map[string]string{
						"Accept":    "application/json",
						"X-Removed": "should-be-removed",
					},
					ResourceType: fetch.ResourceTypeXHR,
				},
				Response: &fetch.InterceptedResponse{
					RequestID: "req-1",
					Status:    200,
					BodySize:  10,
				},
			},
		},
	}

	data, _ := json.Marshal(entry)
	os.WriteFile(resultPath, append(data, '\n'), 0644)

	jobID := "test-job-modifications"
	job := model.Job{
		ID:         jobID,
		Kind:       model.KindScrape,
		Status:     model.StatusSucceeded,
		ResultPath: resultPath,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := srv.store.Create(ctx, job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	body := fmt.Sprintf(`{
		"jobId": "%s",
		"targetBaseUrl": "%s",
		"modifications": {
			"headers": {
				"X-Custom": "added-value"
			},
			"removeHeaders": ["X-Removed"]
		}
	}`, jobID, testServer.URL)

	req := httptest.NewRequest("POST", fmt.Sprintf("/v1/jobs/replay/%s", jobID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status %v, got %v: %s", http.StatusOK, status, rr.Body.String())
	}

	var resp TrafficReplayResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Successful != 1 {
		t.Errorf("expected 1 successful, got %d", resp.Successful)
	}
}

func TestLoadInterceptedEntries(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create result file with multiple lines (JSONL format)
	tmpDir := t.TempDir()
	resultPath := filepath.Join(tmpDir, "results.jsonl")

	entries := []struct {
		InterceptedData []fetch.InterceptedEntry `json:"interceptedData"`
	}{
		{
			InterceptedData: []fetch.InterceptedEntry{
				{
					Request: fetch.InterceptedRequest{
						RequestID: "req-1",
						URL:       "https://example.com/api/1",
						Method:    "GET",
					},
				},
			},
		},
		{
			InterceptedData: []fetch.InterceptedEntry{
				{
					Request: fetch.InterceptedRequest{
						RequestID: "req-2",
						URL:       "https://example.com/api/2",
						Method:    "POST",
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	for _, e := range entries {
		data, _ := json.Marshal(e)
		buf.Write(data)
		buf.WriteByte('\n')
	}
	os.WriteFile(resultPath, buf.Bytes(), 0644)

	job := model.Job{
		ID:         "test-job",
		Kind:       model.KindScrape,
		Status:     model.StatusSucceeded,
		ResultPath: resultPath,
	}

	loaded, err := srv.loadInterceptedEntries(job)
	if err != nil {
		t.Fatalf("failed to load entries: %v", err)
	}

	if len(loaded) != 2 {
		t.Errorf("expected 2 entries, got %d", len(loaded))
	}
}

func TestLoadInterceptedEntriesMissingFile(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	job := model.Job{
		ID:         "test-job",
		Kind:       model.KindScrape,
		Status:     model.StatusSucceeded,
		ResultPath: "/nonexistent/path/results.jsonl",
	}

	_, err := srv.loadInterceptedEntries(job)
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadInterceptedEntriesNoResultPath(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	job := model.Job{
		ID:     "test-job",
		Kind:   model.KindScrape,
		Status: model.StatusSucceeded,
		// No ResultPath
	}

	_, err := srv.loadInterceptedEntries(job)
	if err == nil {
		t.Error("expected error for missing result path")
	}
}
