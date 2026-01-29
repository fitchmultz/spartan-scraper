// Package api provides integration tests for job results endpoint routing.
// Tests cover malformed paths, missing segments, and HTTP method restrictions.
// Does NOT test result file generation or export logic handled by exporter package.
package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleJobResultsRouting(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "malformed path double slash",
			method:         "GET",
			path:           "/v1/jobs//results",
			expectedStatus: http.StatusMovedPermanently,
		},
		{
			name:           "missing id segment",
			method:         "GET",
			path:           "/v1/jobs/results",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "method not allowed",
			method:         "POST",
			path:           "/v1/jobs/some-id/results",
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("%s: expected status %v, got %v", tt.name, tt.expectedStatus, status)
			}
		})
	}
}
