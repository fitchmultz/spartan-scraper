// Package webhook provides tests for dispatcher concurrency limits and semaphore behavior.
//
// Tests cover:
// - Concurrency limit enforcement (max concurrent dispatches)
// - Semaphore timeout behavior (dropped webhooks)
// - Context cancellation during semaphore acquire
// - DroppedCount tracking
//
// Does NOT test:
// - Retry logic (see dispatcher_retry_test.go)
// - Successful dispatch (see dispatcher_success_test.go)
// - SSRF protection (see dispatcher_ssrf_test.go)
//
// Assumes:
// - Semaphore has a 5-second timeout for acquiring slots
// - DroppedCount tracks webhooks dropped due to concurrency limits
// - Context cancellation prevents counting as dropped
package webhook

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestDispatch_ConcurrencyLimit(t *testing.T) {
	// Create server with delay to keep connections open
	var activeCount atomic.Int32
	var maxActive atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := activeCount.Add(1)
		for {
			old := maxActive.Load()
			if current <= old || maxActive.CompareAndSwap(old, current) {
				break
			}
		}
		time.Sleep(200 * time.Millisecond)
		activeCount.Add(-1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := Config{
		MaxConcurrentDispatches: 5,
		AllowInternal:           true,
	}
	d := NewDispatcher(cfg)

	payload := Payload{
		EventID:   "evt-test",
		EventType: EventJobCompleted,
		Timestamp: time.Now(),
		JobID:     "job-test",
		JobKind:   "scrape",
		Status:    "succeeded",
	}

	// Launch 20 concurrent dispatches
	for i := 0; i < 20; i++ {
		d.Dispatch(context.Background(), server.URL, payload, "")
	}

	// Wait for all to complete
	time.Sleep(1 * time.Second)

	if maxActive.Load() > 5 {
		t.Errorf("max concurrent %d exceeded limit 5", maxActive.Load())
	}
}

func TestDispatch_SemaphoreTimeout(t *testing.T) {
	block := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-block // Block until released
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := Config{
		MaxConcurrentDispatches: 1,
		AllowInternal:           true,
	}
	d := NewDispatcher(cfg)

	payload := Payload{
		EventID:   "evt-test",
		EventType: EventJobCompleted,
		Timestamp: time.Now(),
		JobID:     "job-test",
		JobKind:   "scrape",
		Status:    "succeeded",
	}

	// First dispatch blocks
	d.Dispatch(context.Background(), server.URL, payload, "")
	time.Sleep(50 * time.Millisecond)

	// Second dispatch should timeout and be dropped
	d.Dispatch(context.Background(), server.URL, payload, "")

	// Wait for timeout to occur (5 second timeout in implementation)
	time.Sleep(5500 * time.Millisecond)

	if d.DroppedCount() != 1 {
		t.Errorf("expected 1 dropped webhook, got %d", d.DroppedCount())
	}

	close(block)
}

func TestDispatch_ContextCancellationDuringSemaphoreAcquire(t *testing.T) {
	block := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-block // Block until released
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := Config{
		MaxConcurrentDispatches: 1,
		AllowInternal:           true,
	}
	d := NewDispatcher(cfg)

	payload := Payload{
		EventID:   "evt-test",
		EventType: EventJobCompleted,
		Timestamp: time.Now(),
		JobID:     "job-test",
		JobKind:   "scrape",
		Status:    "succeeded",
	}

	// First dispatch blocks
	d.Dispatch(context.Background(), server.URL, payload, "")
	time.Sleep(50 * time.Millisecond)

	// Second dispatch with cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	d.Dispatch(ctx, server.URL, payload, "")

	// Wait for context cancellation to be processed
	time.Sleep(200 * time.Millisecond)

	// Should not have dropped (context was cancelled, not timed out)
	if d.DroppedCount() != 0 {
		t.Errorf("expected 0 dropped webhooks (context cancelled), got %d", d.DroppedCount())
	}

	close(block)
}

func TestDroppedCount_InitialValue(t *testing.T) {
	d := NewDispatcher(Config{})

	if d.DroppedCount() != 0 {
		t.Errorf("expected initial dropped count 0, got %d", d.DroppedCount())
	}
}
