// Package api provides HTTP handler tests for the deduplication endpoints.
//
// Purpose:
// - Verify the API-owned dedup surface returns the expected success and validation behavior.
//
// Responsibilities:
// - Seed the content index with deterministic fixtures.
// - Assert duplicate lookup, URL history, stats, and job-delete cleanup responses.
// - Confirm validation failures stay readable and stable.
//
// Scope:
// - `/v1/dedup/duplicates`, `/v1/dedup/history`, `/v1/dedup/stats`, and `/v1/dedup/job/{id}` only.
//
// Usage:
// - Run with `go test ./internal/api`.
//
// Invariants/Assumptions:
// - Dedup remains an API-only surface.
// - Job delete validation requires UUID-shaped IDs.
// - Tests share one server instance so the store-backed content index is initialized once.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/dedup"
)

func TestHandleDedupEndpoints(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	index := srv.store.GetContentIndex()
	if index == nil {
		t.Fatal("expected content index to be initialized")
	}

	const (
		jobA     = "11111111-1111-1111-1111-111111111111"
		jobB     = "22222222-2222-2222-2222-222222222222"
		jobC     = "33333333-3333-3333-3333-333333333333"
		exactURL = "https://example.com/duplicate"
		otherURL = "https://example.com/unique"
	)

	for _, entry := range []struct {
		jobID   string
		url     string
		simhash uint64
	}{
		{jobID: jobA, url: exactURL, simhash: 42},
		{jobID: jobB, url: exactURL, simhash: 42},
		{jobID: jobC, url: otherURL, simhash: 99},
	} {
		if err := index.Index(context.Background(), entry.jobID, entry.url, entry.simhash); err != nil {
			t.Fatalf("seed dedup index: %v", err)
		}
	}

	t.Run("duplicates validation", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/dedup/duplicates", nil)
		rr := httptest.NewRecorder()
		srv.Routes().ServeHTTP(rr, req)

		assertAPIError(t, rr, http.StatusBadRequest, "simhash parameter is required")

		req = httptest.NewRequest(http.MethodGet, "/v1/dedup/duplicates?simhash=42&threshold=-1", nil)
		rr = httptest.NewRecorder()
		srv.Routes().ServeHTTP(rr, req)

		assertAPIError(t, rr, http.StatusBadRequest, "invalid threshold value")
	})

	t.Run("duplicates success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/dedup/duplicates?simhash=42&threshold=0", nil)
		rr := httptest.NewRecorder()
		srv.Routes().ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}

		var matches []dedup.DuplicateMatch
		if err := json.Unmarshal(rr.Body.Bytes(), &matches); err != nil {
			t.Fatalf("decode duplicates: %v", err)
		}
		if len(matches) != 2 {
			t.Fatalf("expected 2 duplicate matches, got %d", len(matches))
		}
		assertDuplicateJobs(t, matches, jobA, jobB)
	})

	t.Run("history validation", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/dedup/history", nil)
		rr := httptest.NewRecorder()
		srv.Routes().ServeHTTP(rr, req)

		assertAPIError(t, rr, http.StatusBadRequest, "url parameter is required")
	})

	t.Run("history success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/dedup/history?url=%s", exactURL), nil)
		rr := httptest.NewRecorder()
		srv.Routes().ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}

		var history []dedup.ContentEntry
		if err := json.Unmarshal(rr.Body.Bytes(), &history); err != nil {
			t.Fatalf("decode history: %v", err)
		}
		if len(history) != 2 {
			t.Fatalf("expected 2 history entries, got %d", len(history))
		}
		assertHistoryJobs(t, history, jobA, jobB)
	})

	t.Run("stats success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/dedup/stats", nil)
		rr := httptest.NewRecorder()
		srv.Routes().ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}

		var stats dedup.Stats
		if err := json.Unmarshal(rr.Body.Bytes(), &stats); err != nil {
			t.Fatalf("decode stats: %v", err)
		}
		if stats.TotalIndexed != 3 || stats.UniqueURLs != 2 || stats.UniqueJobs != 3 || stats.DuplicatePairs != 1 {
			t.Fatalf("unexpected stats: %#v", stats)
		}
	})

	t.Run("delete validation", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/v1/dedup/job/not-a-uuid", nil)
		rr := httptest.NewRecorder()
		srv.Routes().ServeHTTP(rr, req)

		assertAPIError(t, rr, http.StatusBadRequest, "invalid job id format")
	})

	t.Run("delete success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/v1/dedup/job/"+jobB, nil)
		rr := httptest.NewRecorder()
		srv.Routes().ServeHTTP(rr, req)

		if rr.Code != http.StatusNoContent {
			t.Fatalf("expected status 204, got %d: %s", rr.Code, rr.Body.String())
		}

		req = httptest.NewRequest(http.MethodGet, "/v1/dedup/stats", nil)
		rr = httptest.NewRecorder()
		srv.Routes().ServeHTTP(rr, req)

		var stats dedup.Stats
		if err := json.Unmarshal(rr.Body.Bytes(), &stats); err != nil {
			t.Fatalf("decode stats after delete: %v", err)
		}
		if stats.TotalIndexed != 2 || stats.UniqueURLs != 2 || stats.UniqueJobs != 2 || stats.DuplicatePairs != 0 {
			t.Fatalf("unexpected stats after delete: %#v", stats)
		}
	})
}

func assertAPIError(t *testing.T, rr *httptest.ResponseRecorder, wantStatus int, wantSubstring string) {
	t.Helper()
	if rr.Code != wantStatus {
		t.Fatalf("expected status %d, got %d: %s", wantStatus, rr.Code, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode error payload: %v", err)
	}
	message, _ := payload["error"].(string)
	if message == "" || !strings.Contains(message, wantSubstring) {
		t.Fatalf("expected error containing %q, got %q", wantSubstring, message)
	}
}

func assertDuplicateJobs(t *testing.T, matches []dedup.DuplicateMatch, wantJobIDs ...string) {
	t.Helper()
	seen := map[string]bool{}
	for _, match := range matches {
		seen[match.JobID] = true
	}
	for _, jobID := range wantJobIDs {
		if !seen[jobID] {
			t.Fatalf("expected duplicate results to contain job %s, got %#v", jobID, matches)
		}
	}
}

func assertHistoryJobs(t *testing.T, entries []dedup.ContentEntry, wantJobIDs ...string) {
	t.Helper()
	seen := map[string]bool{}
	for _, entry := range entries {
		seen[entry.JobID] = true
	}
	for _, jobID := range wantJobIDs {
		if !seen[jobID] {
			t.Fatalf("expected history to contain job %s, got %#v", jobID, entries)
		}
	}
}
