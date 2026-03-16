// Package api provides regression tests for batch API handlers.
//
// Purpose:
// - Verify standardized batch submission behavior after handler consolidation.
//
// Responsibilities:
// - Assert shared defaults are applied consistently.
// - Assert research batches create a single research job for all submitted URLs.
//
// Scope:
// - API handler behavior only; job execution is not under test here.
//
// Usage:
// - Run with `go test ./internal/api/...`.
//
// Invariants/Assumptions:
// - Batch responses are returned before workers complete execution.
// - Store reads reflect the jobs created for the batch request.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

func TestHandleBatchScrapeDefaultsMethodToGET(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body, err := json.Marshal(BatchScrapeRequest{
		Jobs: []BatchJobRequest{
			{URL: "https://example.com/articles/1"},
		},
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/jobs/batch/scrape", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp BatchResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Batch.JobCount != 1 {
		t.Fatalf("expected 1 created job, got %d", resp.Batch.JobCount)
	}

	jobsByBatch, err := srv.store.ListJobsByBatch(context.Background(), resp.Batch.ID, store.ListOptions{})
	if err != nil {
		t.Fatalf("list jobs by batch: %v", err)
	}
	if len(jobsByBatch) != 1 {
		t.Fatalf("expected 1 stored job, got %d", len(jobsByBatch))
	}
	if got := jobsByBatch[0].SpecMap()["method"]; got != http.MethodGet {
		t.Fatalf("expected stored method %q, got %#v", http.MethodGet, got)
	}
}

func TestHandleBatchScrapeAppliesEnvAuthOverrides(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	srv.cfg.AuthOverrides.Headers = map[string]string{"X-Batch-Env": "present"}

	body, err := json.Marshal(BatchScrapeRequest{
		Jobs: []BatchJobRequest{{URL: "https://example.com/articles/1"}},
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/jobs/batch/scrape", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp BatchResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	jobsByBatch, err := srv.store.ListJobsByBatch(context.Background(), resp.Batch.ID, store.ListOptions{})
	if err != nil {
		t.Fatalf("list jobs by batch: %v", err)
	}
	if len(jobsByBatch) != 1 {
		t.Fatalf("expected 1 stored job, got %d", len(jobsByBatch))
	}
	authConfig, ok := jobsByBatch[0].SpecMap()["auth"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected persisted auth config, got %#v", jobsByBatch[0].SpecMap())
	}
	headers, ok := authConfig["headers"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected persisted auth headers, got %#v", authConfig)
	}
	if headers["X-Batch-Env"] != "present" {
		t.Fatalf("expected env auth header to be persisted, got %#v", headers)
	}
}

func TestHandleBatchResearchCreatesSingleResearchJob(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body, err := json.Marshal(BatchResearchRequest{
		Query: "recent announcements",
		Jobs: []BatchJobRequest{
			{URL: "https://example.com/one"},
			{URL: "https://example.com/two"},
		},
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/jobs/batch/research", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp BatchResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Batch.JobCount != 1 {
		t.Fatalf("expected one research job for the batch, got %d", resp.Batch.JobCount)
	}
	if len(resp.Jobs) != 1 {
		t.Fatalf("expected one sanitized job in response, got %d", len(resp.Jobs))
	}

	jobsByBatch, err := srv.store.ListJobsByBatch(context.Background(), resp.Batch.ID, store.ListOptions{})
	if err != nil {
		t.Fatalf("list jobs by batch: %v", err)
	}
	if len(jobsByBatch) != 1 {
		t.Fatalf("expected 1 stored job, got %d", len(jobsByBatch))
	}
}

func TestHandleBatchResearchStoresAgenticConfig(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body, err := json.Marshal(BatchResearchRequest{
		Query: "recent announcements",
		Jobs: []BatchJobRequest{
			{URL: "https://example.com/one"},
			{URL: "https://example.com/two"},
		},
		Agentic: &model.ResearchAgenticConfig{
			Enabled:         true,
			Instructions:    "Prioritize pricing and support commitments",
			MaxRounds:       2,
			MaxFollowUpURLs: 4,
		},
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/jobs/batch/research", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp BatchResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	jobsByBatch, err := srv.store.ListJobsByBatch(context.Background(), resp.Batch.ID, store.ListOptions{})
	if err != nil {
		t.Fatalf("list jobs by batch: %v", err)
	}
	if len(jobsByBatch) != 1 {
		t.Fatalf("expected 1 stored job, got %d", len(jobsByBatch))
	}
	if _, ok := jobsByBatch[0].SpecMap()["agentic"].(map[string]interface{}); !ok {
		t.Fatalf("expected agentic config in persisted research spec: %#v", jobsByBatch[0].SpecMap())
	}
}

func TestHandleBatchScrapeRejectsInvalidWebhookURL(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body, err := json.Marshal(BatchScrapeRequest{
		Jobs:    []BatchJobRequest{{URL: "https://example.com/articles/1"}},
		Webhook: &WebhookConfig{URL: "ftp://hooks.example.com/batch"},
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/jobs/batch/scrape", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rr.Code, rr.Body.String())
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("webhook URL must use http or https scheme")) {
		t.Fatalf("expected webhook URL validation error, got %s", rr.Body.String())
	}
}

func TestHandleBatchListReturnsSummaries(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	createBatch := func(url string) BatchResponse {
		t.Helper()
		body, err := json.Marshal(BatchScrapeRequest{Jobs: []BatchJobRequest{{URL: url}}})
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/v1/jobs/batch/scrape", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		srv.Routes().ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("expected create status 201, got %d: %s", rr.Code, rr.Body.String())
		}
		var resp BatchResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal create response: %v", err)
		}
		return resp
	}

	older := createBatch("https://example.com/articles/1")
	newer := createBatch("https://example.com/articles/2")

	req := httptest.NewRequest(http.MethodGet, "/v1/jobs/batch?limit=10&offset=0", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("X-Total-Count"); got != "2" {
		t.Fatalf("expected X-Total-Count header 2, got %q", got)
	}

	var resp BatchListResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal list response: %v", err)
	}
	if resp.Total != 2 || resp.Limit != 10 || resp.Offset != 0 {
		t.Fatalf("unexpected pagination metadata: %+v", resp)
	}
	if len(resp.Batches) != 2 {
		t.Fatalf("expected 2 batch summaries, got %d", len(resp.Batches))
	}
	if resp.Batches[0].ID != newer.Batch.ID || resp.Batches[1].ID != older.Batch.ID {
		t.Fatalf("expected newest batch first, got %+v", resp.Batches)
	}
	if resp.Batches[0].Stats.Queued+resp.Batches[0].Stats.Running != 1 ||
		resp.Batches[1].Stats.Queued+resp.Batches[1].Stats.Running != 1 {
		t.Fatalf("expected active-job stats on list response, got %+v", resp.Batches)
	}
}

func TestHandleBatchGetRejectsInvalidPaginationWhenIncludingJobs(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body, err := json.Marshal(BatchScrapeRequest{
		Jobs: []BatchJobRequest{
			{URL: "https://example.com/articles/1"},
		},
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	createReq := httptest.NewRequest(http.MethodPost, "/v1/jobs/batch/scrape", bytes.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createRes := httptest.NewRecorder()
	srv.Routes().ServeHTTP(createRes, createReq)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d: %s", createRes.Code, createRes.Body.String())
	}

	var created BatchResponse
	if err := json.Unmarshal(createRes.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal batch response: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/jobs/batch/"+created.Batch.ID+"?include_jobs=true&limit=abc", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rr.Code, rr.Body.String())
	}
}
