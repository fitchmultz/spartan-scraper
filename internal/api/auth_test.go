// Package api provides integration tests for auth profile import/export endpoints.
// Tests cover path traversal protection, validation, and security constraints.
// Does NOT test auth profile CRUD operations (profiles managed by auth package).
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleAuthImportPathTraversal(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name           string
		body           string
		expectedStatus int
	}{
		{
			name:           "valid filename",
			body:           `{"path": "backup.json"}`,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "empty path",
			body:           `{"path": ""}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "absolute path",
			body:           `{"path": "/tmp/backup.json"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "path traversal with ..",
			body:           `{"path": "../backup.json"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "with directory",
			body:           `{"path": "subdir/backup.json"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "backslash",
			body:           `{"path": "subdir\\backup.json"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "double slash",
			body:           `{"path": "sub//backup.json"}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/auth/import", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v, body: %s", status, tt.expectedStatus, rr.Body.String())
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

func TestHandleAuthExportPathTraversal(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name           string
		body           string
		expectedStatus int
	}{
		{
			name:           "valid filename",
			body:           `{"path": "backup.json"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "empty path",
			body:           `{"path": ""}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "absolute path",
			body:           `{"path": "/tmp/backup.json"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "path traversal with ..",
			body:           `{"path": "../backup.json"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "with directory",
			body:           `{"path": "subdir/backup.json"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "backslash",
			body:           `{"path": "subdir\\backup.json"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "double slash",
			body:           `{"path": "sub//backup.json"}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/auth/export", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v, body: %s", status, tt.expectedStatus, rr.Body.String())
			}

			if tt.expectedStatus != http.StatusOK {
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
			}
		})
	}
}
