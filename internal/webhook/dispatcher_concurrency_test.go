// Package webhook provides tests for dispatcher concurrency limits and bounded queue behavior.
//
// Tests cover:
// - Worker concurrency enforcement (max concurrent dispatches)
// - Queue timeout behavior (dropped webhooks)
// - Context cancellation while waiting for queue capacity
// - DroppedCount tracking
//
// Does NOT test:
// - Retry logic (see dispatcher_retry_test.go)
// - Successful dispatch (see dispatcher_success_test.go)
// - SSRF protection (see dispatcher_ssrf_test.go)
//
// Assumes:
// - Queue wait timeout is 5 seconds in the dispatcher implementation
// - DroppedCount tracks webhooks dropped due to queue backpressure
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

	d := newTestDispatcher(t, Config{
		MaxConcurrentDispatches: 5,
		AllowInternal:           true,
	})
	payload := testPayload()

	for i := 0; i < 20; i++ {
		d.Dispatch(context.Background(), server.URL, payload, "")
	}

	time.Sleep(time.Second)

	if maxActive.Load() > 5 {
		t.Errorf("max concurrent %d exceeded limit 5", maxActive.Load())
	}
}

func TestDispatch_QueueTimeoutDropsWebhook(t *testing.T) {
	block := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-block
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	d := newTestDispatcher(t, Config{
		MaxConcurrentDispatches: 1,
		MaxQueuedDispatches:     1,
		AllowInternal:           true,
	})
	payload := testPayload()

	d.Dispatch(context.Background(), server.URL, payload, "")
	time.Sleep(50 * time.Millisecond)
	d.Dispatch(context.Background(), server.URL, payload, "")
	d.Dispatch(context.Background(), server.URL, payload, "")

	time.Sleep(5500 * time.Millisecond)

	if d.DroppedCount() != 1 {
		t.Errorf("expected 1 dropped webhook, got %d", d.DroppedCount())
	}
	stats := d.Stats()
	if stats.QueueCapacity != 1 {
		t.Fatalf("expected queue capacity 1, got %#v", stats)
	}

	close(block)
}

func TestDispatch_ContextCancellationDuringQueueWait(t *testing.T) {
	block := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-block
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	d := newTestDispatcher(t, Config{
		MaxConcurrentDispatches: 1,
		MaxQueuedDispatches:     1,
		AllowInternal:           true,
	})
	payload := testPayload()

	d.Dispatch(context.Background(), server.URL, payload, "")
	time.Sleep(50 * time.Millisecond)
	d.Dispatch(context.Background(), server.URL, payload, "")

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	d.Dispatch(ctx, server.URL, payload, "")
	time.Sleep(300 * time.Millisecond)

	if d.DroppedCount() != 0 {
		t.Errorf("expected 0 dropped webhooks (context cancelled), got %d", d.DroppedCount())
	}

	close(block)
}

func TestDroppedCount_InitialValue(t *testing.T) {
	d := newTestDispatcher(t, Config{})

	if d.DroppedCount() != 0 {
		t.Errorf("expected initial dropped count 0, got %d", d.DroppedCount())
	}
}
