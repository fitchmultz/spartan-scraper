// Package api provides integration tests for crawl endpoint with Playwright option.
// Tests cover Playwright flag behavior (nil, true, false) in crawl requests.
// Does NOT test Playwright fetcher implementation (fetch package handles that).
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleCrawlPlaywright(t *testing.T) {
	// No workers: this test validates request parameter persistence, not crawl execution.
	srv, cleanup := setupTestServerWithConcurrency(t, 0)
	defer cleanup()

	tests := []struct {
		name             string
		body             string
		expectPlaywright bool
	}{
		{
			name:             "playwright nil (omitted) - uses default",
			body:             `{"url": "https://example.com"}`,
			expectPlaywright: srv.manager.DefaultUsePlaywright(),
		},
		{
			name:             "playwright explicitly false",
			body:             `{"url": "https://example.com", "playwright": false}`,
			expectPlaywright: false,
		},
		{
			name:             "playwright explicitly true",
			body:             `{"url": "https://example.com", "playwright": true}`,
			expectPlaywright: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/crawl", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if status := rr.Code; status != http.StatusOK {
				t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse JSON response: %v", err)
			}

			jobID, ok := resp["id"].(string)
			if !ok {
				t.Fatalf("expected job ID in response, got: %v", resp)
			}

			job, err := srv.store.Get(context.Background(), jobID)
			if err != nil {
				t.Fatalf("failed to get job: %v", err)
			}

			playwright, ok := job.SpecMap()["playwright"].(bool)
			if !ok {
				t.Fatalf("expected bool 'playwright' param, got %v", job.SpecMap()["playwright"])
			}

			if playwright != tt.expectPlaywright {
				t.Errorf("playwright = %v, want %v", playwright, tt.expectPlaywright)
			}
		})
	}
}
