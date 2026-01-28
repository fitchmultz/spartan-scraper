// Package api provides integration tests for crawl endpoint (/v1/crawl).
// Tests cover validation of crawl parameters (maxDepth, maxPages).
// Does NOT test crawl execution or job processing logic.
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleCrawlValidation(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name           string
		body           string
		contentType    string
		expectedStatus int
	}{
		{
			name:           "missing content-type",
			body:           `{"url": "https://example.com"}`,
			contentType:    "",
			expectedStatus: http.StatusUnsupportedMediaType,
		},
		{
			name:           "invalid json",
			body:           `{"url": "https://example.com"`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing url",
			body:           `{}`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid url scheme",
			body:           `{"url": "ftp://example.com"}`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid url host",
			body:           `{"url": "https://"}`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid auth profile",
			body:           `{"url": "https://example.com", "authProfile": "invalid name!"}`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid maxDepth",
			body:           `{"url": "https://example.com", "maxDepth": 11}`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid maxPages",
			body:           `{"url": "https://example.com", "maxPages": 20000}`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/crawl", strings.NewReader(tt.body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

			if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("expected Content-Type application/json, got %v", ct)
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Errorf("failed to parse JSON response: %v", err)
			}
			if _, ok := resp["error"]; !ok {
				t.Errorf("expected 'error' field in response, got: %v", resp)
			}
		})
	}
}
