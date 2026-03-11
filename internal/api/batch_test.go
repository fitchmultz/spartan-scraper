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
	if resp.JobCount != 1 {
		t.Fatalf("expected 1 created job, got %d", resp.JobCount)
	}

	jobsByBatch, err := srv.store.ListJobsByBatch(context.Background(), resp.ID, store.ListOptions{})
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
	if resp.JobCount != 1 {
		t.Fatalf("expected one research job for the batch, got %d", resp.JobCount)
	}
	if len(resp.Jobs) != 1 {
		t.Fatalf("expected one sanitized job in response, got %d", len(resp.Jobs))
	}

	jobsByBatch, err := srv.store.ListJobsByBatch(context.Background(), resp.ID, store.ListOptions{})
	if err != nil {
		t.Fatalf("list jobs by batch: %v", err)
	}
	if len(jobsByBatch) != 1 {
		t.Fatalf("expected 1 stored job, got %d", len(jobsByBatch))
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

	req := httptest.NewRequest(http.MethodGet, "/v1/jobs/batch/"+created.ID+"?include_jobs=true&limit=abc", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rr.Code, rr.Body.String())
	}
}
