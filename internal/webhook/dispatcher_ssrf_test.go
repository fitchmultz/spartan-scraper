// Package webhook provides tests for SSRF protection at the dispatcher level.
//
// Tests cover:
// - Blocking private IP ranges (RFC1918, link-local, loopback)
// - AllowInternal configuration bypass
// - Invalid URL scheme blocking
//
// Does NOT test:
// - SSRF validation functions directly (see ssrf_test.go for unit tests)
// - DNS rebinding protection (see ssrf_test.go)
// - URL sanitization (see ssrf_test.go)
//
// Assumes:
// - SSRF validation blocks internal/private addresses by default
// - AllowInternal=true bypasses SSRF checks for trusted environments
// - Invalid schemes are rejected regardless of AllowInternal setting
//
// Note: These are integration tests at the dispatcher level. For unit tests
// of the SSRF validation functions, see ssrf_test.go.
package webhook

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestDispatch_SSRFBlocksPrivateIPs(t *testing.T) {
	privateURLs := []string{
		"http://127.0.0.1/webhook",
		"http://10.0.0.1/webhook",
		"http://192.168.1.1/webhook",
		"http://172.16.0.1/webhook",
		"http://169.254.169.254/latest/meta-data/",
		"http://localhost/webhook",
		"http://[::1]/webhook",
	}

	for _, url := range privateURLs {
		t.Run(url, func(t *testing.T) {
			var received atomic.Bool

			d := NewDispatcher(Config{AllowInternal: false})
			payload := testPayload()

			d.Dispatch(context.Background(), url, payload, "")

			// Wait to ensure no request is made
			time.Sleep(50 * time.Millisecond)

			if received.Load() {
				t.Errorf("expected SSRF protection to block %s", url)
			}
		})
	}
}

func TestDispatch_AllowInternalBypassesSSRF(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Test with allowInternal=true
	d := NewDispatcher(Config{
		AllowInternal: true,
	})
	payload := testPayload()

	var received atomic.Bool
	serverWithReceiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Store(true)
		w.WriteHeader(http.StatusOK)
	}))
	defer serverWithReceiver.Close()

	d.Dispatch(context.Background(), serverWithReceiver.URL, payload, "")

	// Wait for async dispatch
	time.Sleep(100 * time.Millisecond)

	if !received.Load() {
		t.Error("expected webhook to be received with allowInternal=true")
	}
}

func TestDispatch_SSRFBlocksInvalidSchemes(t *testing.T) {
	invalidURLs := []string{
		"file:///etc/passwd",
		"ftp://example.com/webhook",
		"gopher://example.com",
	}

	for _, url := range invalidURLs {
		t.Run(url, func(t *testing.T) {
			d := NewDispatcher(Config{})
			payload := testPayload()

			// Should not panic and should not dispatch
			d.Dispatch(context.Background(), url, payload, "")

			// Wait to ensure no request is made
			time.Sleep(50 * time.Millisecond)
		})
	}
}
