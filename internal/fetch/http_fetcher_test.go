package fetch

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPFetch_MaxResponseBytes(t *testing.T) {
	tests := []struct {
		name             string
		responseSize     int
		maxResponseBytes int64
		wantErr          bool
		errContains      string
	}{
		{
			name:             "small response succeeds under default limit",
			responseSize:     1024,             // 1KB
			maxResponseBytes: 10 * 1024 * 1024, // 10MB
			wantErr:          false,
		},
		{
			name:             "response exactly at limit succeeds",
			responseSize:     5000,
			maxResponseBytes: 5000,
			wantErr:          false,
		},
		{
			name:             "response exceeding limit fails",
			responseSize:     10 * 1024 * 1024, // 10MB
			maxResponseBytes: 1024 * 1024,      // 1MB limit
			wantErr:          true,
			errContains:      "exceeded maximum size",
		},
		{
			name:             "zero limit means no limit (backward compat)",
			responseSize:     5 * 1024 * 1024, // 5MB
			maxResponseBytes: 0,
			wantErr:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server with sized response
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write(make([]byte, tt.responseSize))
			}))
			defer server.Close()

			fetcher := &HTTPFetcher{}
			req := Request{
				URL:              server.URL,
				Timeout:          5 * time.Second,
				MaxResponseBytes: tt.maxResponseBytes,
			}

			result, err := fetcher.Fetch(context.TODO(), req)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want contains %q", err.Error(), tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if result.Status != http.StatusOK {
					t.Errorf("status = %d, want %d", result.Status, http.StatusOK)
				}
			}
		})
	}
}

func TestHTTPFetch_MaxResponseBytesErrorMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(make([]byte, 10*1024*1024)) // 10MB
	}))
	defer server.Close()

	fetcher := &HTTPFetcher{}
	req := Request{
		URL:              server.URL,
		Timeout:          5 * time.Second,
		MaxResponseBytes: 1024 * 1024, // 1MB limit
	}

	_, err := fetcher.Fetch(context.TODO(), req)

	if err == nil {
		t.Fatal("expected error for oversized response")
	}

	expectedMsg := fmt.Sprintf("exceeded maximum size of %d bytes", 1024*1024)
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("error message = %q, want contains %q", err.Error(), expectedMsg)
	}
}
