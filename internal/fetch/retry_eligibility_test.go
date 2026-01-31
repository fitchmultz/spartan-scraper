// Package fetch provides tests for retry eligibility logic.
// Tests cover retry decisions based on errors, status codes, and configuration.
package fetch

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		want   bool
	}{
		{
			name:   "network error with ErrClosed retries",
			err:    net.ErrClosed,
			status: 0,
			want:   true,
		},
		{
			name:   "timeout error retries",
			err:    errors.New("connection timeout"),
			status: 0,
			want:   true,
		},
		{
			name:   "other errors do not retry",
			err:    errors.New("some error"),
			status: 0,
			want:   false,
		},
		{
			name:   "success status does not retry",
			err:    nil,
			status: 200,
			want:   false,
		},
		{
			name:   "403 does not retry",
			err:    nil,
			status: 403,
			want:   false,
		},
		{
			name:   "401 does not retry",
			err:    nil,
			status: 401,
			want:   false,
		},
		{
			name:   "429 rate limit retries",
			err:    nil,
			status: 429,
			want:   true,
		},
		{
			name:   "500 server error retries",
			err:    nil,
			status: 500,
			want:   true,
		},
		{
			name:   "502 bad gateway retries",
			err:    nil,
			status: 502,
			want:   true,
		},
		{
			name:   "503 service unavailable retries",
			err:    nil,
			status: 503,
			want:   true,
		},
		{
			name:   "504 gateway timeout retries",
			err:    nil,
			status: 504,
			want:   true,
		},
		{
			name:   "400 bad request does not retry",
			err:    nil,
			status: 400,
			want:   false,
		},
		{
			name:   "404 not found does not retry",
			err:    nil,
			status: 404,
			want:   false,
		},
		{
			name:   "context deadline exceeded retries",
			err:    context.DeadlineExceeded,
			status: 0,
			want:   true,
		},
		{
			name: "DNS NXDOMAIN does not retry",
			err: &net.DNSError{
				Err:        "no such host",
				Name:       "nonexistent.example.com",
				IsNotFound: true,
			},
			status: 0,
			want:   false,
		},
		{
			name: "DNS timeout retries",
			err: &net.DNSError{
				Err:       "lookup nonexistent.example.com on 127.0.0.53:53: read udp 127.0.0.1:12345->127.0.0.53:53: i/o timeout",
				Name:      "nonexistent.example.com",
				IsTimeout: true,
			},
			status: 0,
			want:   true,
		},
		{
			name:   "invalid URL scheme does not retry",
			err:    apperrors.ErrInvalidURLScheme,
			status: 0,
			want:   false,
		},
		{
			name:   "invalid URL host does not retry",
			err:    apperrors.ErrInvalidURLHost,
			status: 0,
			want:   false,
		},
		{
			name: "connection refused does not retry",
			err: &net.OpError{
				Err: errors.New("connect: connection refused"),
				Op:  "dial",
			},
			status: 0,
			want:   false,
		},
		{
			name: "no such host (DNS lookup failed) does not retry",
			err: &net.OpError{
				Err: &net.DNSError{
					Err:        "no such host",
					Name:       "invalid-host-name",
					IsNotFound: true,
				},
				Op: "dial",
			},
			status: 0,
			want:   false,
		},
		{
			name: "net.Error with Timeout flag retries",
			err: &net.OpError{
				Err: errors.New("i/o timeout"),
				Op:  "read",
			},
			status: 0,
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldRetry(tt.err, tt.status); got != tt.want {
				t.Errorf("shouldRetry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldRetryWithConfig(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		cfg    RetryConfig
		want   bool
	}{
		{
			name:   "configured 429 retryable",
			status: 429,
			cfg:    RetryConfig{RetryableCodes: map[int]bool{429: true}},
			want:   true,
		},
		{
			name:   "configured 429 not retryable",
			status: 429,
			cfg:    RetryConfig{RetryableCodes: map[int]bool{429: false}},
			want:   false,
		},
		{
			name:   "configured 503 retryable",
			status: 503,
			cfg:    RetryConfig{RetryableCodes: map[int]bool{503: true, 504: true}},
			want:   true,
		},
		{
			name:   "status not in configured list",
			status: 500,
			cfg:    RetryConfig{RetryableCodes: map[int]bool{503: true}},
			want:   false,
		},
		{
			name:   "nil RetryableCodes uses defaults - 500",
			status: 500,
			cfg:    RetryConfig{},
			want:   true,
		},
		{
			name:   "nil RetryableCodes uses defaults - 429",
			status: 429,
			cfg:    RetryConfig{},
			want:   true,
		},
		{
			name:   "nil RetryableCodes uses defaults - 400 not retryable",
			status: 400,
			cfg:    RetryConfig{},
			want:   false,
		},
		{
			name:   "5xx uses defaults if not explicitly configured",
			status: 501,
			cfg:    RetryConfig{RetryableCodes: map[int]bool{429: true}},
			want:   false, // 501 is not in the configured list
		},
		{
			name:   "error triggers retry",
			err:    context.DeadlineExceeded,
			status: 0,
			cfg:    RetryConfig{},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldRetryWithConfig(tt.err, tt.status, tt.cfg)
			if got != tt.want {
				t.Errorf("ShouldRetryWithConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsStatusCodeRetryable(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		retryableCodes map[int]bool
		want           bool
	}{
		{
			name:           "retryable in custom set",
			status:         418, // I'm a teapot
			retryableCodes: map[int]bool{418: true},
			want:           true,
		},
		{
			name:           "not retryable in custom set",
			status:         500,
			retryableCodes: map[int]bool{418: true},
			want:           false,
		},
		{
			name:           "nil uses defaults - 429 retryable",
			status:         429,
			retryableCodes: nil,
			want:           true,
		},
		{
			name:           "nil uses defaults - 404 not retryable",
			status:         404,
			retryableCodes: nil,
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsStatusCodeRetryable(tt.status, tt.retryableCodes)
			if got != tt.want {
				t.Errorf("IsStatusCodeRetryable() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestShouldRetryWithConfig_5xxDefaultBehavior tests 5xx handling when configured
func TestShouldRetryWithConfig_5xxDefaultBehavior(t *testing.T) {
	// When RetryableCodes is set, 5xx codes NOT in the map should NOT retry
	// (unless they are in the default list and the code chooses to fall back)
	cfg := RetryConfig{
		RetryableCodes: map[int]bool{
			429: true,
			// 500, 502, 503, 504 NOT included
		},
	}

	// 429 is configured - should retry
	if !ShouldRetryWithConfig(nil, 429, cfg) {
		t.Error("Expected 429 to be retryable when configured")
	}

	// 500 is not in config but is in defaults - behavior depends on implementation
	// Current implementation: 5xx not in config returns false, unless using nil fallback
	result500 := ShouldRetryWithConfig(nil, 500, cfg)
	// With explicit config, 500 should NOT retry
	if result500 {
		t.Error("Expected 500 to NOT be retryable when using explicit config without 500")
	}
}

// TestRetryConfig_CustomRetryableCodes verifies custom retryable codes work
func TestRetryConfig_CustomRetryableCodes(t *testing.T) {
	// Create config with only specific codes
	cfg := RetryConfig{
		RetryableCodes: map[int]bool{
			418: true, // I'm a teapot (custom)
			420: true, // Enhance Your Calm (custom)
		},
	}

	tests := []struct {
		status int
		want   bool
	}{
		{418, true},
		{420, true},
		{429, false}, // Not in custom list
		{500, false}, // Not in custom list
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.status), func(t *testing.T) {
			got := ShouldRetryWithConfig(nil, tt.status, cfg)
			if got != tt.want {
				t.Errorf("ShouldRetryWithConfig(status=%d) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}
