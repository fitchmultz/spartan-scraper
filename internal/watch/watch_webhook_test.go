// Package watch provides lifecycle-aware webhook delivery coverage for watch notifications.
//
// Purpose:
// - Prove watch webhook delivery stays attached to the caller lifecycle instead of detached goroutines.
//
// Responsibilities:
// - Verify dispatchWebhook waits for the receiver response before returning.
// - Verify delivered payload metadata matches the requested watch event.
//
// Scope:
// - Direct Watcher webhook-delivery behavior only.
//
// Usage:
// - Run with `go test ./internal/watch`.
//
// Invariants/Assumptions:
// - Watch webhook delivery is synchronous with respect to the caller context.
// - Internal test receivers are allowed via the shared dispatcher config.
package watch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/webhook"
)

func TestWatcherDispatchWebhookWaitsForReceiver(t *testing.T) {
	dispatcher := webhook.NewDispatcher(webhook.Config{
		AllowInternal: true,
		MaxRetries:    1,
		Timeout:       time.Second,
	})

	received := make(chan webhook.Payload, 1)
	const receiverDelay = 100 * time.Millisecond
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var payload webhook.Payload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("Decode(): %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		time.Sleep(receiverDelay)
		select {
		case received <- payload:
		default:
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	watcher := &Watcher{dispatcher: dispatcher}
	watch := &Watch{
		ID:             "watch-webhook-sync",
		URL:            "https://example.com",
		NotifyOnChange: true,
		WebhookConfig:  &model.WebhookSpec{URL: server.URL},
	}
	result := &WatchCheckResult{
		WatchID:      watch.ID,
		URL:          watch.URL,
		CheckedAt:    time.Now(),
		PreviousHash: "prev-hash",
		CurrentHash:  "curr-hash",
	}

	started := time.Now()
	watcher.dispatchWebhook(context.Background(), watch, result, webhook.EventContentChanged)
	elapsed := time.Since(started)
	if elapsed < receiverDelay {
		t.Fatalf("expected dispatchWebhook() to wait at least %v, took %v", receiverDelay, elapsed)
	}

	select {
	case payload := <-received:
		if payload.EventType != webhook.EventContentChanged {
			t.Fatalf("expected event type %q, got %q", webhook.EventContentChanged, payload.EventType)
		}
		if payload.URL != watch.URL {
			t.Fatalf("expected payload URL %q, got %q", watch.URL, payload.URL)
		}
		if payload.PreviousHash != result.PreviousHash || payload.CurrentHash != result.CurrentHash {
			t.Fatalf("unexpected payload hashes: %#v", payload)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for watch webhook payload")
	}
}
