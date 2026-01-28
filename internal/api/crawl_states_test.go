// Package api provides integration tests for crawl states endpoint (/v1/crawl-states).
// Tests cover crawl state listing and pagination.
// Does NOT test crawl state upsert logic or crawl deduplication.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

func TestHandleCrawlStates(t *testing.T) {
	ctx := context.Background()
	srv, cleanup := setupTestServer(t)
	defer cleanup()

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

	req = httptest.NewRequest("GET", "/v1/crawl-states?offset=3", nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	json.Unmarshal(rr.Body.Bytes(), &response)
	crawlStates = response["crawlStates"].([]interface{})
	if len(crawlStates) != 2 {
		t.Errorf("expected 2 crawl states with offset 3, got %d", len(crawlStates))
	}
}

func TestHandleCrawlStatesDelete(t *testing.T) {
	ctx := context.Background()
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Insert test data
	urls := []string{"https://example.com/1", "https://example.com/2"}
	for _, u := range urls {
		state := model.CrawlState{
			URL:         u,
			LastScraped: time.Now(),
		}
		if err := srv.store.UpsertCrawlState(ctx, state); err != nil {
			t.Fatalf("failed to insert crawl state: %v", err)
		}
	}

	// Test delete single
	req := httptest.NewRequest("DELETE", "/v1/crawl-states?url=https://example.com/1", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	states, _ := srv.store.ListCrawlStates(ctx, store.ListCrawlStatesOptions{})
	if len(states) != 1 {
		t.Errorf("expected 1 crawl state after deletion, got %d", len(states))
	}
	if states[0].URL != "https://example.com/2" {
		t.Errorf("expected URL https://example.com/2 to remain, got %s", states[0].URL)
	}

	// Test clear all
	req = httptest.NewRequest("DELETE", "/v1/crawl-states", nil)
	rr = httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	states, _ = srv.store.ListCrawlStates(ctx, store.ListCrawlStatesOptions{})
	if len(states) != 0 {
		t.Errorf("expected 0 crawl states after clear, got %d", len(states))
	}
}
