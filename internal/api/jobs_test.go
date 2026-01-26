// Package api provides integration tests for jobs endpoints (/v1/jobs).
// Tests cover job listing, filtering by status, deletion, and cancellation.
// Does NOT test individual job creation, execution, or result retrieval handlers.
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

	"spartan-scraper/internal/model"
)

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

func TestHandleJobForceDelete(t *testing.T) {
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
		t.Fatalf("job response missing id field")
	}

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

	st := srv.store
	ctx := context.Background()
	if err := st.UpdateResultPath(ctx, jobID, resultPath); err != nil {
		t.Fatalf("failed to update result path: %v", err)
	}

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

	_, err := st.Get(ctx, jobID)
	if err == nil {
		t.Error("job should be deleted from database after force delete")
	}

	if _, err := os.Stat(resultPath); !os.IsNotExist(err) {
		t.Error("result file should be deleted after force delete")
	}

	if _, err := os.Stat(jobDir); !os.IsNotExist(err) {
		t.Error("job directory should be deleted after force delete")
	}
}

func TestHandleJobCancelNotDelete(t *testing.T) {
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

	st := srv.store
	ctx := context.Background()
	if err := st.UpdateResultPath(ctx, jobID, resultPath); err != nil {
		t.Fatalf("failed to update result path: %v", err)
	}

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

	gotJob, err := st.Get(ctx, jobID)
	if err != nil {
		t.Error("job should still exist in database after cancel")
	}
	if gotJob.Status != model.StatusCanceled {
		t.Errorf("job status should be 'canceled', got %s", gotJob.Status)
	}

	if _, err := os.Stat(resultPath); os.IsNotExist(err) {
		t.Error("result file should still exist after cancel")
	}

	if _, err := os.Stat(jobDir); os.IsNotExist(err) {
		t.Error("job directory should still exist after cancel")
	}
}
