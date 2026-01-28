// Package api provides integration tests for research endpoint (/v1/research).
// Tests cover request validation including URL validation in queries.
// Does NOT test research workflow execution or multi-source coordination.
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleResearch(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{"query": "test query", "urls": ["https://example.com"]}`
	req := httptest.NewRequest("POST", "/v1/research", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestHandleResearchValidation(t *testing.T) {
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
			body:           `{"query": "test"}`,
			contentType:    "",
			expectedStatus: http.StatusUnsupportedMediaType,
		},
		{
			name:           "invalid json",
			body:           `{"query": "test"`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing query",
			body:           `{}`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid url in urls list",
			body:           `{"query": "test", "urls": ["ftp://example.com"]}`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty url in urls list",
			body:           `{"query": "test", "urls": ["", "https://example.com"]}`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid url host in urls list",
			body:           `{"query": "test", "urls": ["https://"]}`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid auth profile",
			body:           `{"query": "test", "authProfile": "invalid name!"}`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/research", strings.NewReader(tt.body))
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
