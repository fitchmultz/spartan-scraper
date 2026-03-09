// Package api provides HTTP handler tests for feed monitoring endpoints.
//
// Purpose:
// - Verify feed CRUD behavior and request normalization through the API.
//
// Responsibilities:
// - Assert create-time schema defaults are applied.
// - Confirm updates preserve omitted optional fields.
// - Cover not-found semantics for delete operations.
//
// Scope:
// - `/v1/feeds` route behavior only.
//
// Usage:
// - Run with `go test ./internal/api`.
//
// Invariants/Assumptions:
// - Tests use isolated temp storage through setupTestServer.
package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/feed"
)

func TestHandleCreateFeedAppliesDefaults(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := []byte(`{"url":"https://example.com/feed","intervalSeconds":3600}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/feeds", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp FeedResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if !resp.Enabled {
		t.Fatalf("expected enabled to default to true")
	}
	if !resp.AutoScrape {
		t.Fatalf("expected autoScrape to default to true")
	}
	if resp.FeedType != "auto" {
		t.Fatalf("expected feedType auto, got %s", resp.FeedType)
	}
}

func TestHandleUpdateFeedPreservesOmittedOptionalFields(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	storage := feed.NewFileStorage(srv.cfg.DataDir)
	created, err := storage.Add(&feed.Feed{
		URL:             "https://example.com/original-feed",
		FeedType:        feed.FeedTypeRSS,
		IntervalSeconds: 1800,
		Enabled:         false,
		AutoScrape:      false,
	})
	if err != nil {
		t.Fatalf("failed to seed feed: %v", err)
	}

	body := []byte(`{"url":"https://example.com/updated-feed"}`)
	req := httptest.NewRequest(http.MethodPut, "/v1/feeds/"+created.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp FeedResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.URL != "https://example.com/updated-feed" {
		t.Fatalf("expected updated URL, got %s", resp.URL)
	}
	if resp.Enabled {
		t.Fatalf("expected enabled to remain false")
	}
	if resp.AutoScrape {
		t.Fatalf("expected autoScrape to remain false")
	}
	if resp.FeedType != "rss" {
		t.Fatalf("expected feedType to remain rss, got %s", resp.FeedType)
	}
	if resp.IntervalSeconds != 1800 {
		t.Fatalf("expected interval to remain 1800, got %d", resp.IntervalSeconds)
	}
}

func TestHandleDeleteFeedMissingReturnsNotFound(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodDelete, "/v1/feeds/missing", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rr.Code, rr.Body.String())
	}
}
