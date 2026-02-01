// Package api provides tests for traffic replay execution.
//
// This file contains tests for replay execution with traffic data,
// filtering, comparison, and request modifications.
package api

import (
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

func TestHandleTrafficReplayTimeout(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	ctx := t.Context()

	// Create a test server that responds slowly
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second) // Longer than timeout
		w.WriteHeader(http.StatusOK)
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

	jobID := "test-job-timeout"
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

	// Test with short timeout in request
	body := fmt.Sprintf(`{
		"jobId": "%s",
		"targetBaseUrl": "%s",
		"timeout": 1
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

	// Should have 1 failed request due to timeout
	if resp.Failed != 1 {
		t.Errorf("expected 1 failed (timeout), got %d", resp.Failed)
	}

	// Verify error message mentions timeout
	if len(resp.Results) > 0 && !strings.Contains(resp.Results[0].Error, "Client.Timeout") {
		t.Errorf("expected timeout error, got: %s", resp.Results[0].Error)
	}
}
