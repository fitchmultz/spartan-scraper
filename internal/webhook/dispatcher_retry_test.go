// Package webhook verifies retry behavior and failure handling for webhook delivery.
//
// Purpose:
// - Prove delivery retries, timeout handling, and context cancellation remain intact.
//
// Responsibilities:
// - Verify retries stop after success or exhaustion.
// - Verify per-request timeouts are enforced.
// - Verify context cancellation aborts retry loops.
//
// Scope:
// - Dispatcher retry logic only.
//
// Usage:
// - Run with `go test ./internal/webhook`.
//
// Invariants/Assumptions:
// - Tests use AllowInternal=true because httptest listeners bind loopback addresses.
// - Each retry test resolves a pinned client once so all attempts share the same dial plan.
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

func retryTestClient(t *testing.T, d *Dispatcher, rawURL string) (*http.Client, func()) {
	t.Helper()
	target, err := resolveDeliveryTarget(context.Background(), rawURL, d.allowInternal, d.resolver)
	if err != nil {
		t.Fatalf("resolveDeliveryTarget() failed: %v", err)
	}
	return d.clientForTarget(target)
}

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
	client, closeClient := retryTestClient(t, d, server.URL)
	defer closeClient()

	err = d.dispatchWithRetry(context.Background(), client, server.URL, request, "")
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
	client, closeClient := retryTestClient(t, d, server.URL)
	defer closeClient()

	err = d.dispatchWithRetry(context.Background(), client, server.URL, request, "")
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
	client, closeClient := retryTestClient(t, d, server.URL)
	defer closeClient()

	err = d.dispatchWithRetry(context.Background(), client, server.URL, request, "")
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
	client, closeClient := retryTestClient(t, d, server.URL)
	defer closeClient()

	go func() {
		time.Sleep(25 * time.Millisecond)
		cancel()
	}()

	err = d.dispatchWithRetry(ctx, client, server.URL, request, "")
	if err == nil {
		t.Error("expected context cancellation error")
	}
	if ctx.Err() == nil {
		t.Error("expected context to be canceled")
	}
}
