// Package api provides unit tests for HTTP middleware functions.
// Tests cover recovery middleware, logging middleware, request ID handling, and middleware chaining.
// Does NOT test API endpoint handlers or integration behavior.
package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/auth"
	"github.com/fitchmultz/spartan-scraper/internal/config"
)

func TestAPIKeyAuthMiddleware(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test keys
	readWriteKey, err := auth.GenerateAPIKey(tmpDir, "ReadWrite Key", auth.APIKeyPermissionReadWrite, nil)
	if err != nil {
		t.Fatalf("Failed to create read-write key: %v", err)
	}

	readOnlyKey, err := auth.GenerateAPIKey(tmpDir, "ReadOnly Key", auth.APIKeyPermissionReadOnly, nil)
	if err != nil {
		t.Fatalf("Failed to create read-only key: %v", err)
	}

	// Create an expired key
	expiredTime := time.Now().Add(-24 * time.Hour)
	expiredKey, err := auth.GenerateAPIKey(tmpDir, "Expired Key", auth.APIKeyPermissionReadWrite, &expiredTime)
	if err != nil {
		t.Fatalf("Failed to create expired key: %v", err)
	}

	tests := []struct {
		name           string
		apiAuthEnabled bool
		bindAddr       string
		apiKey         string
		method         string
		path           string
		wantStatus     int
	}{
		{
			name:           "localhost without auth enabled skips validation",
			apiAuthEnabled: false,
			bindAddr:       "127.0.0.1",
			apiKey:         "",
			method:         "GET",
			path:           "/v1/jobs",
			wantStatus:     http.StatusOK,
		},
		{
			name:           "localhost with ::1 without auth enabled skips validation",
			apiAuthEnabled: false,
			bindAddr:       "::1",
			apiKey:         "",
			method:         "GET",
			path:           "/v1/jobs",
			wantStatus:     http.StatusOK,
		},
		{
			name:           "non-localhost requires auth even when not explicitly enabled",
			apiAuthEnabled: false,
			bindAddr:       "0.0.0.0",
			apiKey:         "",
			method:         "GET",
			path:           "/v1/jobs",
			wantStatus:     http.StatusForbidden,
		},
		{
			name:           "missing API key returns 403",
			apiAuthEnabled: true,
			bindAddr:       "127.0.0.1",
			apiKey:         "",
			method:         "GET",
			path:           "/v1/jobs",
			wantStatus:     http.StatusForbidden,
		},
		{
			name:           "invalid API key returns 403",
			apiAuthEnabled: true,
			bindAddr:       "127.0.0.1",
			apiKey:         "ss_invalid_key_value",
			method:         "GET",
			path:           "/v1/jobs",
			wantStatus:     http.StatusForbidden,
		},
		{
			name:           "expired API key returns 403",
			apiAuthEnabled: true,
			bindAddr:       "127.0.0.1",
			apiKey:         expiredKey,
			method:         "GET",
			path:           "/v1/jobs",
			wantStatus:     http.StatusForbidden,
		},
		{
			name:           "valid read-write key allows GET",
			apiAuthEnabled: true,
			bindAddr:       "127.0.0.1",
			apiKey:         readWriteKey,
			method:         "GET",
			path:           "/v1/jobs",
			wantStatus:     http.StatusOK,
		},
		{
			name:           "valid read-write key allows POST",
			apiAuthEnabled: true,
			bindAddr:       "127.0.0.1",
			apiKey:         readWriteKey,
			method:         "POST",
			path:           "/v1/scrape",
			wantStatus:     http.StatusOK,
		},
		{
			name:           "read-only key allows GET",
			apiAuthEnabled: true,
			bindAddr:       "127.0.0.1",
			apiKey:         readOnlyKey,
			method:         "GET",
			path:           "/v1/jobs",
			wantStatus:     http.StatusOK,
		},
		{
			name:           "read-only key denies POST",
			apiAuthEnabled: true,
			bindAddr:       "127.0.0.1",
			apiKey:         readOnlyKey,
			method:         "POST",
			path:           "/v1/scrape",
			wantStatus:     http.StatusForbidden,
		},
		{
			name:           "read-only key allows HEAD",
			apiAuthEnabled: true,
			bindAddr:       "127.0.0.1",
			apiKey:         readOnlyKey,
			method:         "HEAD",
			path:           "/v1/jobs",
			wantStatus:     http.StatusOK,
		},
		{
			name:           "read-only key allows OPTIONS",
			apiAuthEnabled: true,
			bindAddr:       "127.0.0.1",
			apiKey:         readOnlyKey,
			method:         "OPTIONS",
			path:           "/v1/jobs",
			wantStatus:     http.StatusOK,
		},
		{
			name:           "health check skips auth even with auth enabled",
			apiAuthEnabled: true,
			bindAddr:       "127.0.0.1",
			apiKey:         "",
			method:         "GET",
			path:           "/healthz",
			wantStatus:     http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Config{
				APIAuthEnabled: tt.apiAuthEnabled,
				BindAddr:       tt.bindAddr,
				DataDir:        tmpDir,
			}

			// Create a simple handler that returns 200
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Wrap with auth middleware
			handler := apiKeyAuthMiddleware(cfg, nextHandler)

			// Create request
			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}

			// Record response
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("apiKeyAuthMiddleware() status = %v, want %v", rr.Code, tt.wantStatus)
			}
		})
	}
}

func TestIsLocalhost(t *testing.T) {
	tests := []struct {
		addr string
		want bool
	}{
		{"127.0.0.1", true},
		{"localhost", true},
		{"::1", true},
		{"127.0.0.2", true},
		{"127.255.255.255", true},
		{"0.0.0.0", false},
		{"192.168.1.1", false},
		{"10.0.0.1", false},
		{"", false},
		{"example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			got := isLocalhost(tt.addr)
			if got != tt.want {
				t.Errorf("isLocalhost(%q) = %v, want %v", tt.addr, got, tt.want)
			}
		})
	}
}

func TestGetAPIKeyFromContext(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test key
	key, err := auth.GenerateAPIKey(tmpDir, "Test Key", auth.APIKeyPermissionReadWrite, nil)
	if err != nil {
		t.Fatalf("Failed to create key: %v", err)
	}

	cfg := config.Config{
		APIAuthEnabled: true,
		BindAddr:       "127.0.0.1",
		DataDir:        tmpDir,
	}

	var capturedKey auth.APIKey
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to get key from context
		if k, ok := GetAPIKeyFromContext(r.Context()); ok {
			capturedKey = k
		}
		w.WriteHeader(http.StatusOK)
	})

	handler := apiKeyAuthMiddleware(cfg, nextHandler)

	req := httptest.NewRequest("GET", "/v1/jobs", nil)
	req.Header.Set("X-API-Key", key)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %v", rr.Code)
	}

	if capturedKey.Key != key {
		t.Errorf("GetAPIKeyFromContext() key = %v, want %v", capturedKey.Key, key)
	}

	if capturedKey.Permissions != auth.APIKeyPermissionReadWrite {
		t.Errorf("GetAPIKeyFromContext() permissions = %v, want %v", capturedKey.Permissions, auth.APIKeyPermissionReadWrite)
	}
}
