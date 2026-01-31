// Package fetch provides tests for HTTP fetcher rate limit header handling.
// Tests cover RateLimit, X-RateLimit-*, and RateLimit-Policy header parsing.
// Does NOT test adaptive rate limiting algorithms (see limiter_test.go).
package fetch

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestHTTPFetch_RateLimitHeaders verifies that RateLimit headers are extracted
// and populated in the Result.
func TestHTTPFetch_RateLimitHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("RateLimit", "limit=100, remaining=50, reset=3600")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	fetcher := &HTTPFetcher{}
	req := Request{
		URL:     server.URL,
		Timeout: 5 * time.Second,
	}

	result, err := fetcher.Fetch(context.TODO(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != http.StatusOK {
		t.Errorf("expected status 200, got %d", result.Status)
	}

	if result.RateLimit == nil {
		t.Fatal("expected RateLimit to be populated")
	}
	if result.RateLimit.Limit != 100 {
		t.Errorf("expected Limit 100, got %d", result.RateLimit.Limit)
	}
	if result.RateLimit.Remaining != 50 {
		t.Errorf("expected Remaining 50, got %d", result.RateLimit.Remaining)
	}
	if result.RateLimit.Reset.IsZero() {
		t.Error("expected Reset to be set")
	}
}

// TestHTTPFetch_XRateLimitHeaders verifies that X-RateLimit-* headers are extracted.
func TestHTTPFetch_XRateLimitHeaders(t *testing.T) {
	resetTime := time.Now().Add(time.Hour).Unix()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Limit", "1000")
		w.Header().Set("X-RateLimit-Remaining", "999")
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	fetcher := &HTTPFetcher{}
	req := Request{
		URL:     server.URL,
		Timeout: 5 * time.Second,
	}

	result, err := fetcher.Fetch(context.TODO(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.RateLimit == nil {
		t.Fatal("expected RateLimit to be populated")
	}
	if result.RateLimit.Limit != 1000 {
		t.Errorf("expected Limit 1000, got %d", result.RateLimit.Limit)
	}
	if result.RateLimit.Remaining != 999 {
		t.Errorf("expected Remaining 999, got %d", result.RateLimit.Remaining)
	}
}

// TestHTTPFetch_RateLimitWithAdaptiveLimiter verifies that rate limit info
// is passed to the adaptive limiter when available.
func TestHTTPFetch_RateLimitWithAdaptiveLimiter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("RateLimit", "limit=10, remaining=5, reset=60")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	cfg := &AdaptiveConfig{
		Enabled: true,
		MinQPS:  0.1,
		MaxQPS:  100,
	}
	limiter := NewAdaptiveHostLimiter(50, 50, cfg)

	fetcher := &HTTPFetcher{}
	req := Request{
		URL:     server.URL,
		Timeout: 5 * time.Second,
		Limiter: limiter,
	}

	result, err := fetcher.Fetch(context.TODO(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.RateLimit == nil {
		t.Fatal("expected RateLimit to be populated")
	}

	// Verify the limiter was updated with server-provided limit
	// The limiter should adjust to 80% of the server limit (8 QPS)
	status := limiter.GetHostStatus()
	if len(status) != 1 {
		t.Fatalf("expected 1 host status, got %d", len(status))
	}

	// CurrentQPS should be adjusted based on server limit
	if status[0].CurrentQPS != 8 {
		t.Errorf("expected CurrentQPS 8 (80%% of server limit 10), got %f", status[0].CurrentQPS)
	}
}

// TestHTTPFetch_NoRateLimitHeaders verifies behavior when no rate limit headers present.
func TestHTTPFetch_NoRateLimitHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	fetcher := &HTTPFetcher{}
	req := Request{
		URL:     server.URL,
		Timeout: 5 * time.Second,
	}

	result, err := fetcher.Fetch(context.TODO(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.RateLimit != nil {
		t.Error("expected RateLimit to be nil when no headers present")
	}
}

// TestHTTPFetch_RateLimitPolicyHeader verifies RateLimit-Policy header parsing.
func TestHTTPFetch_RateLimitPolicyHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("RateLimit-Policy", "100;w=60")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	fetcher := &HTTPFetcher{}
	req := Request{
		URL:     server.URL,
		Timeout: 5 * time.Second,
	}

	result, err := fetcher.Fetch(context.TODO(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.RateLimit == nil {
		t.Fatal("expected RateLimit to be populated from policy header")
	}
	if result.RateLimit.Limit != 100 {
		t.Errorf("expected Limit 100, got %d", result.RateLimit.Limit)
	}
	if result.RateLimit.Window != 60*time.Second {
		t.Errorf("expected Window 60s, got %v", result.RateLimit.Window)
	}
}
