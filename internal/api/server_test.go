// Package api provides cross-cutting integration tests for REST API server.
// These tests verify API-wide behaviors such as request validation consistency
// and zero value handling across multiple endpoints.
// Handler-specific tests are in files like scrape_test.go, crawl_test.go, etc.
package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestZeroValuesAllowed(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name string
		body string
	}{
		{
			name: "scrape with timeout 0",
			body: `{"url": "https://example.com", "timeoutSeconds": 0}`,
		},
		{
			name: "crawl with maxDepth 0",
			body: `{"url": "https://example.com", "maxDepth": 0, "maxPages": 10}`,
		},
		{
			name: "crawl with maxPages 0",
			body: `{"url": "https://example.com", "maxDepth": 2, "maxPages": 0}`,
		},
		{
			name: "research with all zero values",
			body: `{"query": "test", "urls": ["https://example.com"], "timeoutSeconds": 0, "maxDepth": 0, "maxPages": 0}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := "/v1/scrape"
			if tt.name == "crawl with maxDepth 0" || tt.name == "crawl with maxPages 0" {
				endpoint = "/v1/crawl"
			}
			if tt.name == "research with all zero values" {
				endpoint = "/v1/research"
			}
			req := httptest.NewRequest("POST", endpoint, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if status := rr.Code; status != http.StatusOK {
				t.Errorf("handler returned wrong status code for %s: got %v want %v, body: %s", tt.name, status, http.StatusOK, rr.Body.String())
			}
		})
	}
}

func TestRejectUnknownFields(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	type testCase struct {
		name         string
		method       string
		endpoint     string
		validBody    string
		invalidBody  string
		unknownField string
		prepare      func(t *testing.T, srv *Server)
	}

	tests := []testCase{
		{
			name:         "scrape rejects unknown field",
			method:       http.MethodPost,
			endpoint:     "/v1/scrape",
			validBody:    `{"url": "https://example.com"}`,
			invalidBody:  `{"url": "https://example.com", "unknownField": "test"}`,
			unknownField: "unknownField",
		},
		{
			name:         "crawl rejects unknown field",
			method:       http.MethodPost,
			endpoint:     "/v1/crawl",
			validBody:    `{"url": "https://example.com", "maxDepth": 1, "maxPages": 1}`,
			invalidBody:  `{"url": "https://example.com", "maxDepth": 1, "maxPages": 1, "unknownField": "test"}`,
			unknownField: "unknownField",
		},
		{
			name:         "research rejects unknown field",
			method:       http.MethodPost,
			endpoint:     "/v1/research",
			validBody:    `{"query": "test query", "urls": ["https://example.com"]}`,
			invalidBody:  `{"query": "test query", "urls": ["https://example.com"], "unknownField": "test"}`,
			unknownField: "unknownField",
		},
		{
			name:         "auth profile PUT rejects unknown field",
			method:       http.MethodPut,
			endpoint:     "/v1/auth/profiles/test-profile",
			validBody:    `{}`,
			invalidBody:  `{"unknownField": "test"}`,
			unknownField: "unknownField",
		},
		{
			name:         "auth import rejects unknown field",
			method:       http.MethodPost,
			endpoint:     "/v1/auth/import",
			validBody:    `{"path": "backup.json"}`,
			invalidBody:  `{"path": "backup.json", "unknownField": "test"}`,
			unknownField: "unknownField",
			prepare: func(t *testing.T, srv *Server) {
				req := httptest.NewRequest("POST", "/v1/auth/export", strings.NewReader(`{"path": "backup.json"}`))
				req.Header.Set("Content-Type", "application/json")
				rr := httptest.NewRecorder()
				srv.Routes().ServeHTTP(rr, req)
				if rr.Code != http.StatusOK {
					t.Fatalf("failed to prepare auth import by exporting backup: got %v, want %v, body: %s", rr.Code, http.StatusOK, rr.Body.String())
				}
			},
		},
		{
			name:         "auth export rejects unknown field",
			method:       http.MethodPost,
			endpoint:     "/v1/auth/export",
			validBody:    `{"path": "backup-unknown-fields-test.json"}`,
			invalidBody:  `{"path": "backup-unknown-fields-test.json", "unknownField": "test"}`,
			unknownField: "unknownField",
		},
		{
			name:         "schedules POST rejects unknown field",
			method:       http.MethodPost,
			endpoint:     "/v1/schedules",
			validBody:    `{"kind": "scrape", "intervalSeconds": 3600, "url": "https://example.com"}`,
			invalidBody:  `{"kind": "scrape", "intervalSeconds": 3600, "url": "https://example.com", "unknownField": "test"}`,
			unknownField: "unknownField",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepare != nil {
				tt.prepare(t, srv)
			}

			req := httptest.NewRequest(tt.method, tt.endpoint, strings.NewReader(tt.validBody))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("valid request failed: got %v want %v, body: %s", rr.Code, http.StatusOK, rr.Body.String())
			}

			req = httptest.NewRequest(tt.method, tt.endpoint, strings.NewReader(tt.invalidBody))
			req.Header.Set("Content-Type", "application/json")
			rr = httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400 for unknown field, got %v, body: %s", rr.Code, rr.Body.String())
			}

			body := rr.Body.String()
			if !strings.Contains(body, "invalid json:") {
				t.Errorf("expected 'invalid json:' in error response, got: %s", body)
			}
			if !strings.Contains(body, tt.unknownField) {
				t.Errorf("expected error response to mention unknown field %q, got: %s", tt.unknownField, body)
			}
		})
	}
}
