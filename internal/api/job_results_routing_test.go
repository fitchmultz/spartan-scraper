// Package api provides integration tests for job results endpoint routing.
// Responsibilities: validate malformed path handling, missing segments, and method restrictions.
// Scope: HTTP routing behavior only; result generation/export logic is tested in exporter-focused tests.
// Usage: executed by `go test ./internal/api` and `make ci`.
// Invariants/Assumptions: redirect status for path normalization may vary by router/runtime (301, 307, or 308).
package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func containsStatus(statuses []int, target int) bool {
	for _, status := range statuses {
		if status == target {
			return true
		}
	}
	return false
}

func TestHandleJobResultsRouting(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name             string
		method           string
		path             string
		expectedStatuses []int
	}{
		{
			name:             "malformed path double slash",
			method:           "GET",
			path:             "/v1/jobs//results",
			expectedStatuses: []int{http.StatusMovedPermanently, http.StatusTemporaryRedirect, http.StatusPermanentRedirect},
		},
		{
			name:             "missing id segment",
			method:           "GET",
			path:             "/v1/jobs/results",
			expectedStatuses: []int{http.StatusNotFound},
		},
		{
			name:             "method not allowed",
			method:           "POST",
			path:             "/v1/jobs/some-id/results",
			expectedStatuses: []int{http.StatusMethodNotAllowed},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if !containsStatus(tt.expectedStatuses, rr.Code) {
				t.Fatalf("%s: expected one of %v, got %v", tt.name, tt.expectedStatuses, rr.Code)
			}
		})
	}
}
