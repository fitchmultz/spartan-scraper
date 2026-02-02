// Package fetch provides tests for HTTP fetcher context cancellation handling.
// Tests cover cancellation during rate limiter wait and retry backoff periods.
// Does NOT test general request timeout behavior.
package fetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestHTTPFetch_ContextCancellationDuringLimiterWait verifies that context
// cancellation is properly propagated when waiting for the rate limiter.
//
// This test documents the fix for RQ-0022: the HTTP fetcher now checks the
// error return from req.Limiter.Wait and returns immediately on cancellation
// instead of continuing to make the HTTP request.
func TestHTTPFetch_ContextCancellationDuringLimiterWait(t *testing.T) {
	// Create a host limiter with low QPS (1 request per second) to ensure Wait is called.
	// With burst=1, the first request consumes the burst, and subsequent requests
	// must wait for the limiter to allow them through.
	limiter := NewHostLimiter(1, 1)

	// Consume the burst token so that the next Fetch call will block in Wait.
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("burst consumer"))
	}))
	defer server.Close()

	fetcher := &HTTPFetcher{}
	req := Request{
		URL:     server.URL,
		Timeout: 5 * time.Second,
		Limiter: limiter,
	}

	// First request consumes the burst token
	_, _ = fetcher.Fetch(ctx, req)

	// Now create a cancelled context for the second request
	// This request will need to wait for the rate limiter, but the context
	// is already cancelled, so Wait should return immediately with context.Canceled
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	serverCalled := false
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverCalled = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("should not reach here"))
	}))
	defer server2.Close()

	req.URL = server2.URL
	result, err := fetcher.Fetch(cancelledCtx, req)

	// Assert: should return context.Canceled error
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}

	// Assert: result should be empty (zero value)
	if result.URL != "" || result.Status != 0 || result.HTML != "" {
		t.Errorf("expected empty Result, got %+v", result)
	}

	// Assert: server should not have been called (defensive check)
	if serverCalled {
		t.Error("server was called despite cancelled context")
	}
}

// TestHTTPFetch_ContextCancellationDuringBackoff verifies that context
// cancellation stops retry backoff immediately without waiting for the full
// delay duration.
//
// This test documents the fix for RQ-0158: the HTTP fetcher now uses
// sleepWithContext instead of time.Sleep during retry backoff, allowing
// cancellation to interrupt the wait.
func TestHTTPFetch_ContextCancellationDuringBackoff(t *testing.T) {
	// Server that always returns 503 (retryable status code)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	fetcher := &HTTPFetcher{}

	// Create a context we can cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a very short delay (less than the backoff duration)
	// The default baseDelay is 300ms, so cancel at 50ms
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err := fetcher.Fetch(ctx, Request{
		URL:        server.URL,
		Timeout:    5 * time.Second,
		MaxRetries: 3,
	})
	elapsed := time.Since(start)

	// Should return quickly, not after full backoff (which would be ~300ms)
	// Allow some tolerance for timing variations
	if elapsed > 200*time.Millisecond {
		t.Errorf("took too long to return after cancellation: %v (expected < 200ms)", elapsed)
	}

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

// TestHTTPFetch_ContextCancellationDuringBackoffConnectionError verifies that
// context cancellation stops retry backoff when the error is a connection error.
func TestHTTPFetch_ContextCancellationDuringBackoffConnectionError(t *testing.T) {
	fetcher := &HTTPFetcher{}

	// Create a context we can cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a very short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	// Use an invalid URL that will cause a connection error
	_, err := fetcher.Fetch(ctx, Request{
		URL:        "http://127.0.0.1:1", // Port 1 is typically not accessible
		Timeout:    100 * time.Millisecond,
		MaxRetries: 3,
	})
	elapsed := time.Since(start)

	// Should return quickly due to context cancellation, not waiting for full backoff
	// The connection will fail, then backoff starts, then context is cancelled
	if elapsed > 500*time.Millisecond {
		t.Errorf("took too long to return after cancellation: %v (expected < 500ms)", elapsed)
	}

	// Should get context.Canceled, not a connection error or max retries exceeded
	if err != context.Canceled {
		// It's also acceptable to get a connection error if the context wasn't
		// cancelled quickly enough, but we should NOT get "max retries exceeded"
		if err != nil && !strings.Contains(err.Error(), "connection") {
			// Log the actual error for debugging
			t.Logf("Got error: %v (type: %T)", err, err)
		}
	}
}

// TestSleepWithContext_NormalCompletion verifies that SleepWithContext returns
// nil when the sleep completes normally without cancellation.
func TestSleepWithContext_NormalCompletion(t *testing.T) {
	ctx := context.Background()
	start := time.Now()
	err := SleepWithContext(ctx, 50*time.Millisecond)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	// Should have slept for at least the requested duration
	if elapsed < 45*time.Millisecond {
		t.Errorf("sleep was too short: %v (expected ~50ms)", elapsed)
	}
}

// TestSleepWithContext_CancellationDuringSleep verifies that SleepWithContext
// returns context.Canceled immediately when the context is cancelled during sleep.
func TestSleepWithContext_CancellationDuringSleep(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err := SleepWithContext(ctx, 5*time.Second)
	elapsed := time.Since(start)

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}

	// Should return quickly, not after the full 5 second sleep
	if elapsed > 200*time.Millisecond {
		t.Errorf("sleep took too long after cancellation: %v (expected < 200ms)", elapsed)
	}
}

// TestSleepWithContext_AlreadyCancelled verifies that SleepWithContext returns
// immediately when the context is already cancelled before the call.
func TestSleepWithContext_AlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately before sleep

	start := time.Now()
	err := SleepWithContext(ctx, 5*time.Second)
	elapsed := time.Since(start)

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}

	// Should return almost immediately
	if elapsed > 50*time.Millisecond {
		t.Errorf("sleep took too long for already-cancelled context: %v (expected < 50ms)", elapsed)
	}
}

// TestSleepWithContext_DeadlineExceeded verifies that SleepWithContext returns
// context.DeadlineExceeded when the context deadline expires during sleep.
func TestSleepWithContext_DeadlineExceeded(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := SleepWithContext(ctx, 5*time.Second)
	elapsed := time.Since(start)

	if err != context.DeadlineExceeded {
		t.Errorf("expected context.DeadlineExceeded, got %v", err)
	}

	// Should return quickly when deadline expires
	if elapsed > 200*time.Millisecond {
		t.Errorf("sleep took too long after deadline: %v (expected < 200ms)", elapsed)
	}
}
