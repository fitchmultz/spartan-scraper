// Package api provides server-level security tests for WebSocket upgrade origin validation.
//
// Purpose: Verify that WebSocket origin enforcement protects localhost deployments from
// cross-site browser origins while preserving local and non-browser client workflows.
// Responsibilities: Assert origin allow/deny rules and route-level behavior in /v1/ws.
// Scope: Origin validation logic only; does not validate full WebSocket session messaging.
// Usage: Executed by `go test ./internal/api` and `make test-ci`.
// Invariants/Assumptions: Empty Origin is treated as a non-browser client and is allowed.
package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIsAllowedWebSocketOrigin(t *testing.T) {
	tests := []struct {
		name   string
		origin string
		want   bool
	}{
		{name: "empty origin allowed", origin: "", want: true},
		{name: "localhost allowed", origin: "http://localhost:5173", want: true},
		{name: "127 loopback allowed", origin: "https://127.0.0.1:3000", want: true},
		{name: "ipv6 loopback allowed", origin: "http://[::1]:8080", want: true},
		{name: "remote host denied", origin: "https://example.com", want: false},
		{name: "invalid origin denied", origin: "not-a-url", want: false},
		{name: "missing host denied", origin: "http:///missing-host", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAllowedWebSocketOrigin(tt.origin)
			if got != tt.want {
				t.Fatalf("isAllowedWebSocketOrigin(%q) = %v, want %v", tt.origin, got, tt.want)
			}
		})
	}
}

func TestHandleWebSocketRejectsNonLocalOrigin(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/v1/ws", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "forbidden websocket origin") {
		t.Fatalf("expected forbidden websocket origin response body, got %q", rr.Body.String())
	}
}

func TestHandleWebSocketAllowsLocalOrMissingOrigin(t *testing.T) {
	tests := []struct {
		name   string
		origin string
	}{
		{name: "missing origin", origin: ""},
		{name: "localhost origin", origin: "http://localhost:5173"},
		{name: "127 origin", origin: "http://127.0.0.1:5173"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, cleanup := setupTestServer(t)
			defer cleanup()

			req := httptest.NewRequest(http.MethodGet, "/v1/ws", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rr := httptest.NewRecorder()

			srv.Routes().ServeHTTP(rr, req)

			if rr.Code == http.StatusForbidden {
				t.Fatalf("expected local/missing origin to pass origin gate, got forbidden")
			}
		})
	}
}
