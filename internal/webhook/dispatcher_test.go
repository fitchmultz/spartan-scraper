// Package webhook provides tests for webhook dispatch and delivery.
//
// Tests cover:
// - Dispatcher configuration defaults and custom values
// - Successful webhook dispatch with payload serialization
// - HMAC-SHA256 signature generation and verification
// - Retry logic with exponential backoff
// - Timeout handling for slow endpoints
// - Context cancellation propagation
// - Event type filtering (ShouldSendEvent)
// - HTTP header setting (Content-Type, User-Agent)
//
// Does NOT test:
// - Delivery record persistence (see store_test.go)
// - Actual network calls (uses httptest)
// - Webhook endpoint reliability in production
//
// Assumes:
// - HTTP endpoints follow standard request/response patterns
// - Signatures use HMAC-SHA256 when secret is provided
// - Retries use exponential backoff with configurable limits
package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewDispatcher_Defaults(t *testing.T) {
	d := NewDispatcher(Config{})

	if d.maxRetries != 3 {
		t.Errorf("expected maxRetries=3, got %d", d.maxRetries)
	}
	if d.baseDelay != 1*time.Second {
		t.Errorf("expected baseDelay=1s, got %v", d.baseDelay)
	}
	if d.maxDelay != 30*time.Second {
		t.Errorf("expected maxDelay=30s, got %v", d.maxDelay)
	}
	if d.timeout != 30*time.Second {
		t.Errorf("expected timeout=30s, got %v", d.timeout)
	}
}

func TestNewDispatcher_CustomValues(t *testing.T) {
	cfg := Config{
		Secret:     "test-secret",
		MaxRetries: 5,
		BaseDelay:  500 * time.Millisecond,
		MaxDelay:   60 * time.Second,
		Timeout:    10 * time.Second,
	}
	d := NewDispatcher(cfg)

	if d.maxRetries != 5 {
		t.Errorf("expected maxRetries=5, got %d", d.maxRetries)
	}
	if d.baseDelay != 500*time.Millisecond {
		t.Errorf("expected baseDelay=500ms, got %v", d.baseDelay)
	}
	if d.maxDelay != 60*time.Second {
		t.Errorf("expected maxDelay=60s, got %v", d.maxDelay)
	}
	if d.timeout != 10*time.Second {
		t.Errorf("expected timeout=10s, got %v", d.timeout)
	}
	if d.secret != "test-secret" {
		t.Errorf("expected secret='test-secret', got %q", d.secret)
	}
}

func TestDispatch_Success(t *testing.T) {
	var received atomic.Bool
	var receivedBody []byte
	var receivedSig string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Store(true)
		receivedBody, _ = io.ReadAll(r.Body)
		receivedSig = r.Header.Get("X-Webhook-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	d := NewDispatcher(Config{})
	payload := Payload{
		EventID:   "evt-123",
		EventType: EventJobCompleted,
		Timestamp: time.Now(),
		JobID:     "job-456",
		JobKind:   "scrape",
		Status:    "succeeded",
	}

	d.Dispatch(context.Background(), server.URL, payload, "")

	// Wait for async dispatch
	time.Sleep(100 * time.Millisecond)

	if !received.Load() {
		t.Error("expected webhook to be received")
	}

	var receivedPayload Payload
	if err := json.Unmarshal(receivedBody, &receivedPayload); err != nil {
		t.Fatalf("failed to unmarshal received payload: %v", err)
	}
	if receivedPayload.JobID != "job-456" {
		t.Errorf("expected jobID='job-456', got %q", receivedPayload.JobID)
	}
	if receivedSig != "" {
		t.Error("expected no signature without secret")
	}
}

func TestDispatch_WithSignature(t *testing.T) {
	secret := "my-webhook-secret"
	var receivedSig string
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		receivedSig = r.Header.Get("X-Webhook-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	d := NewDispatcher(Config{})
	payload := Payload{
		EventID:   "evt-123",
		EventType: EventJobCompleted,
		Timestamp: time.Now(),
		JobID:     "job-456",
		JobKind:   "scrape",
		Status:    "succeeded",
	}

	d.Dispatch(context.Background(), server.URL, payload, secret)

	// Wait for async dispatch
	time.Sleep(100 * time.Millisecond)

	if receivedSig == "" {
		t.Fatal("expected signature to be present")
	}

	// Verify signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(receivedBody)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if receivedSig != expectedSig {
		t.Errorf("signature mismatch: got %q, want %q", receivedSig, expectedSig)
	}
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
		MaxRetries: 3,
		BaseDelay:  10 * time.Millisecond,
		MaxDelay:   100 * time.Millisecond,
	})

	payload := Payload{
		EventID:   "evt-123",
		EventType: EventJobCompleted,
		Timestamp: time.Now(),
		JobID:     "job-456",
		JobKind:   "scrape",
		Status:    "succeeded",
	}

	err := d.dispatchWithRetry(context.Background(), server.URL, payload, "")

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
		MaxRetries: 2,
		BaseDelay:  10 * time.Millisecond,
		MaxDelay:   100 * time.Millisecond,
	})

	payload := Payload{
		EventID:   "evt-123",
		EventType: EventJobCompleted,
		Timestamp: time.Now(),
		JobID:     "job-456",
		JobKind:   "scrape",
		Status:    "succeeded",
	}

	err := d.dispatchWithRetry(context.Background(), server.URL, payload, "")

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
		MaxRetries: 1,
		Timeout:    50 * time.Millisecond,
	})

	payload := Payload{
		EventID:   "evt-123",
		EventType: EventJobCompleted,
		Timestamp: time.Now(),
		JobID:     "job-456",
		JobKind:   "scrape",
		Status:    "succeeded",
	}

	err := d.dispatchWithRetry(context.Background(), server.URL, payload, "")

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
		MaxRetries: 3,
		BaseDelay:  50 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())

	payload := Payload{
		EventID:   "evt-123",
		EventType: EventJobCompleted,
		Timestamp: time.Now(),
		JobID:     "job-456",
		JobKind:   "scrape",
		Status:    "succeeded",
	}

	// Cancel context after a short delay
	go func() {
		time.Sleep(25 * time.Millisecond)
		cancel()
	}()

	err := d.dispatchWithRetry(ctx, server.URL, payload, "")

	if err == nil {
		t.Error("expected context cancellation error")
	}
	if ctx.Err() == nil {
		t.Error("expected context to be canceled")
	}
}

func TestShouldSendEvent(t *testing.T) {
	tests := []struct {
		name             string
		eventType        EventType
		status           string
		configuredEvents []string
		want             bool
	}{
		{
			name:             "default empty config sends on completed",
			eventType:        EventJobCompleted,
			status:           "succeeded",
			configuredEvents: nil,
			want:             true,
		},
		{
			name:             "default empty config does not send on started",
			eventType:        EventJobStarted,
			status:           "running",
			configuredEvents: nil,
			want:             false,
		},
		{
			name:             "all sends on any event",
			eventType:        EventJobStarted,
			status:           "running",
			configuredEvents: []string{"all"},
			want:             true,
		},
		{
			name:             "started event matches",
			eventType:        EventJobStarted,
			status:           "running",
			configuredEvents: []string{"started"},
			want:             true,
		},
		{
			name:             "completed matches completed event",
			eventType:        EventJobCompleted,
			status:           "succeeded",
			configuredEvents: []string{"completed"},
			want:             true,
		},
		{
			name:             "failed matches failed status",
			eventType:        EventJobCompleted,
			status:           "failed",
			configuredEvents: []string{"failed"},
			want:             true,
		},
		{
			name:             "failed does not match succeeded status",
			eventType:        EventJobCompleted,
			status:           "succeeded",
			configuredEvents: []string{"failed"},
			want:             false,
		},
		{
			name:             "canceled matches canceled status",
			eventType:        EventJobCompleted,
			status:           "canceled",
			configuredEvents: []string{"canceled"},
			want:             true,
		},
		{
			name:             "succeeded matches succeeded status",
			eventType:        EventJobCompleted,
			status:           "succeeded",
			configuredEvents: []string{"succeeded"},
			want:             true,
		},
		{
			name:             "multiple events - match one",
			eventType:        EventJobStarted,
			status:           "running",
			configuredEvents: []string{"completed", "started"},
			want:             true,
		},
		{
			name:             "multiple events - no match",
			eventType:        EventJobCreated,
			status:           "queued",
			configuredEvents: []string{"completed", "started"},
			want:             false,
		},
		{
			name:             "page_crawled event matches",
			eventType:        EventPageCrawled,
			status:           "",
			configuredEvents: []string{"page_crawled"},
			want:             true,
		},
		{
			name:             "page_crawled event does not match other filters",
			eventType:        EventPageCrawled,
			status:           "",
			configuredEvents: []string{"completed", "started"},
			want:             false,
		},
		{
			name:             "retry_attempted event matches",
			eventType:        EventRetryAttempted,
			status:           "",
			configuredEvents: []string{"retry_attempted"},
			want:             true,
		},
		{
			name:             "export_completed event matches",
			eventType:        EventExportCompleted,
			status:           "",
			configuredEvents: []string{"export_completed"},
			want:             true,
		},
		{
			name:             "all includes new event types",
			eventType:        EventPageCrawled,
			status:           "",
			configuredEvents: []string{"all"},
			want:             true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldSendEvent(tt.eventType, tt.status, tt.configuredEvents)
			if got != tt.want {
				t.Errorf("ShouldSendEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSignPayload(t *testing.T) {
	d := NewDispatcher(Config{})
	payload := []byte(`{"jobId":"test-123"}`)
	secret := "test-secret"

	sig1 := d.signPayload(payload, secret)
	sig2 := d.signPayload(payload, secret)

	// Same payload + secret should produce same signature
	if sig1 != sig2 {
		t.Error("same payload and secret should produce same signature")
	}

	// Different secret should produce different signature
	sig3 := d.signPayload(payload, "different-secret")
	if sig1 == sig3 {
		t.Error("different secret should produce different signature")
	}

	// Different payload should produce different signature
	sig4 := d.signPayload([]byte(`{"jobId":"test-456"}`), secret)
	if sig1 == sig4 {
		t.Error("different payload should produce different signature")
	}

	// Verify it's a valid hex string
	if _, err := hex.DecodeString(sig1); err != nil {
		t.Errorf("signature is not valid hex: %v", err)
	}
}

func TestDispatch_Headers(t *testing.T) {
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	d := NewDispatcher(Config{})
	payload := Payload{
		EventID:   "evt-123",
		EventType: EventJobCompleted,
		Timestamp: time.Now(),
		JobID:     "job-456",
		JobKind:   "scrape",
		Status:    "succeeded",
	}

	d.Dispatch(context.Background(), server.URL, payload, "")

	// Wait for async dispatch
	time.Sleep(100 * time.Millisecond)

	if receivedHeaders.Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type='application/json', got %q", receivedHeaders.Get("Content-Type"))
	}
	if !strings.HasPrefix(receivedHeaders.Get("User-Agent"), "SpartanScraper-Webhook") {
		t.Errorf("expected User-Agent to start with 'SpartanScraper-Webhook', got %q", receivedHeaders.Get("User-Agent"))
	}
}
