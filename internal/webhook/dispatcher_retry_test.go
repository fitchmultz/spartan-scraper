// Package webhook provides tests for dispatcher retry logic and failure handling.
//
// Tests cover:
// - Retry on failure with exponential backoff
// - Exhausted retries error handling
// - Timeout handling for slow endpoints
// - Context cancellation propagation
//
// Does NOT test:
// - Successful dispatch (see dispatcher_success_test.go)
// - Concurrency limits (see dispatcher_concurrency_test.go)
// - SSRF protection (see dispatcher_ssrf_test.go)
//
// Assumes:
// - Retries use exponential backoff with configurable limits
// - Timeouts are enforced per-request
// - Context cancellation stops retry attempts
package webhook

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestDispatch_RetryOnFailure(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := attempts.Add(1)
		if count < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Use AllowInternal=true for tests using httptest (which uses 127.0.0.1)
	d := NewDispatcher(Config{
		MaxRetries:    3,
		BaseDelay:     10 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		AllowInternal: true,
	})

	payload := testPayload()
	request, err := jsonDeliveryRequest(payload)
	if err != nil {
		t.Fatalf("jsonDeliveryRequest() failed: %v", err)
	}

	err = d.dispatchWithRetry(context.Background(), server.URL, request, "")

	if err != nil {
		t.Errorf("expected success after retries, got error: %v", err)
	}
	if attempts.Load() != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts.Load())
	}
}

func TestDispatch_ExhaustedRetries(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	// Use AllowInternal=true for tests using httptest (which uses 127.0.0.1)
	d := NewDispatcher(Config{
		MaxRetries:    2,
		BaseDelay:     10 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		AllowInternal: true,
	})

	payload := testPayload()
	request, err := jsonDeliveryRequest(payload)
	if err != nil {
		t.Fatalf("jsonDeliveryRequest() failed: %v", err)
	}

	err = d.dispatchWithRetry(context.Background(), server.URL, request, "")

	if err == nil {
		t.Error("expected error after exhausted retries")
	}
	if attempts.Load() != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts.Load())
	}
	if !strings.Contains(err.Error(), "exhausted retries") {
		t.Errorf("expected 'exhausted retries' in error, got: %v", err)
	}
}

func TestDispatch_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Use AllowInternal=true for tests using httptest (which uses 127.0.0.1)
	d := NewDispatcher(Config{
		MaxRetries:    1,
		Timeout:       50 * time.Millisecond,
		AllowInternal: true,
	})

	payload := testPayload()
	request, err := jsonDeliveryRequest(payload)
	if err != nil {
		t.Fatalf("jsonDeliveryRequest() failed: %v", err)
	}

	err = d.dispatchWithRetry(context.Background(), server.URL, request, "")

	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestDispatch_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Use AllowInternal=true for tests using httptest (which uses 127.0.0.1)
	d := NewDispatcher(Config{
		MaxRetries:    3,
		BaseDelay:     50 * time.Millisecond,
		AllowInternal: true,
	})

	ctx, cancel := context.WithCancel(context.Background())

	payload := testPayload()
	request, err := jsonDeliveryRequest(payload)
	if err != nil {
		t.Fatalf("jsonDeliveryRequest() failed: %v", err)
	}

	// Cancel context after a short delay
	go func() {
		time.Sleep(25 * time.Millisecond)
		cancel()
	}()

	err = d.dispatchWithRetry(ctx, server.URL, request, "")

	if err == nil {
		t.Error("expected context cancellation error")
	}
	if ctx.Err() == nil {
		t.Error("expected context to be canceled")
	}
}
