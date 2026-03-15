// Package api provides tests for chain CRUD and submission over the HTTP layer.
//
// Purpose:
//   - Verify chain definitions now accept operator-facing request payloads and reject
//     invalid request shapes before persistence.
//
// Responsibilities:
// - Assert chain creation validates node requests.
// - Assert chain submission creates the expected jobs through the shared request model.
//
// Scope:
// - `/v1/chains` and `/v1/chains/{id}/submit` behavior only.
//
// Usage:
// - Run with `go test ./internal/api`.
//
// Invariants/Assumptions:
// - Chain nodes persist `request`, not typed `spec` payloads.
// - Submission reuses the same operator-facing request conversion as live jobs.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestHandleCreateChainRejectsInvalidNodeRequest(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := []byte(`{
		"name": "bad-chain",
		"definition": {
			"nodes": [
				{
					"id": "node-1",
					"kind": "scrape",
					"request": {"url": "https://example.com", "unknown": true}
				}
			],
			"edges": []
		}
	}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/chains", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleCreateAndSubmitChainWithRequestNodes(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	createBody := []byte(`{
		"name": "request-chain",
		"definition": {
			"nodes": [
				{
					"id": "root",
					"kind": "scrape",
					"request": {"url": "https://example.com/root"}
				},
				{
					"id": "child",
					"kind": "crawl",
					"request": {"url": "https://example.com/root", "maxDepth": 1, "maxPages": 3}
				}
			],
			"edges": [{"from": "root", "to": "child"}]
		}
	}`)
	createReq := httptest.NewRequest(http.MethodPost, "/v1/chains", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	srv.Routes().ServeHTTP(createRR, createReq)
	if createRR.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", createRR.Code, createRR.Body.String())
	}

	var created model.JobChain
	if err := json.Unmarshal(createRR.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to parse create response: %v", err)
	}

	submitReq := httptest.NewRequest(http.MethodPost, "/v1/chains/"+created.ID+"/submit", bytes.NewReader([]byte(`{}`)))
	submitReq.Header.Set("Content-Type", "application/json")
	submitRR := httptest.NewRecorder()
	srv.Routes().ServeHTTP(submitRR, submitReq)
	if submitRR.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", submitRR.Code, submitRR.Body.String())
	}

	jobs, err := srv.store.GetJobsByChain(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("failed to load chain jobs: %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("expected 2 chain jobs, got %d", len(jobs))
	}
	if jobs[0].Kind != model.KindScrape || jobs[1].Kind != model.KindCrawl {
		t.Fatalf("unexpected chain job kinds: %#v", []model.Kind{jobs[0].Kind, jobs[1].Kind})
	}
	if jobs[0].DependencyStatus != model.DependencyStatusReady {
		t.Fatalf("expected root dependency status ready, got %s", jobs[0].DependencyStatus)
	}
	if jobs[1].DependencyStatus != model.DependencyStatusPending {
		t.Fatalf("expected child dependency status pending, got %s", jobs[1].DependencyStatus)
	}
}
