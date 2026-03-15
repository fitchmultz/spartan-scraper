// Package webhook provides tests for successful webhook dispatch scenarios.
//
// Tests cover:
// - Basic successful dispatch with payload serialization
// - HMAC-SHA256 signature generation and verification
// - HTTP header setting (Content-Type, User-Agent, X-Webhook-Signature)
// - Event type filtering (ShouldSendEvent)
//
// Does NOT test:
// - Retry logic (see dispatcher_retry_test.go)
// - Concurrency limits (see dispatcher_concurrency_test.go)
// - SSRF protection (see dispatcher_ssrf_test.go and ssrf_test.go)
//
// Assumes:
// - HTTP endpoints follow standard request/response patterns
// - Signatures use HMAC-SHA256 when secret is provided
// - httptest servers use 127.0.0.1, requiring AllowInternal=true
package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

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

	// Use AllowInternal=true for tests using httptest (which uses 127.0.0.1)
	d := NewDispatcher(Config{AllowInternal: true})
	payload := testPayload()

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

	// Use AllowInternal=true for tests using httptest (which uses 127.0.0.1)
	d := NewDispatcher(Config{AllowInternal: true})
	payload := testPayload()

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

func TestDispatch_Headers(t *testing.T) {
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Use AllowInternal=true for tests using httptest (which uses 127.0.0.1)
	d := NewDispatcher(Config{AllowInternal: true})
	payload := testPayload()

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

func TestDeliverExport_MultipartContract(t *testing.T) {
	payload := Payload{
		EventID:      "evt-export-123",
		EventType:    EventExportCompleted,
		Timestamp:    time.Now(),
		JobID:        "job-export-456",
		JobKind:      "scrape",
		Status:       "succeeded",
		ResultURL:    "/v1/jobs/job-export-456/results",
		ExportFormat: "csv",
		Filename:     "job-export-456.csv",
		ContentType:  "text/csv; charset=utf-8",
		RecordCount:  1,
		ExportSize:   int64(len("title\nExample Domain\n")),
	}
	exportBody := []byte("title\nExample Domain\n")
	received := make(chan struct {
		payload           Payload
		exportBody        []byte
		exportFilename    string
		exportContentType string
		headerContentType string
		payloadType       string
	}, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			t.Errorf("ParseMediaType() failed: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if mediaType != "multipart/form-data" {
			t.Errorf("unexpected media type: %q", mediaType)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		reader := multipart.NewReader(r.Body, params["boundary"])
		var request struct {
			payload           Payload
			exportBody        []byte
			exportFilename    string
			exportContentType string
			headerContentType string
			payloadType       string
		}
		request.headerContentType = r.Header.Get("Content-Type")
		request.payloadType = r.Header.Get("X-Spartan-Webhook-Payload-Type")
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Errorf("NextPart() failed: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			body, err := io.ReadAll(part)
			if err != nil {
				t.Errorf("ReadAll(part) failed: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			switch part.FormName() {
			case "metadata":
				if err := json.Unmarshal(body, &request.payload); err != nil {
					t.Errorf("json.Unmarshal(metadata) failed: %v", err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
			case "export":
				request.exportBody = body
				request.exportFilename = part.FileName()
				request.exportContentType = part.Header.Get("Content-Type")
			}
		}
		received <- request
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	d := NewDispatcher(Config{AllowInternal: true})
	if err := d.DeliverExport(context.Background(), server.URL, payload, exportBody, ""); err != nil {
		t.Fatalf("DeliverExport() failed: %v", err)
	}

	select {
	case request := <-received:
		if request.payload.EventType != EventExportCompleted {
			t.Fatalf("unexpected metadata payload: %#v", request.payload)
		}
		if request.payload.ResultURL != payload.ResultURL {
			t.Fatalf("unexpected result URL: %#v", request.payload)
		}
		if request.payload.Filename != payload.Filename || request.exportFilename != payload.Filename {
			t.Fatalf("unexpected export filename metadata=%q part=%q", request.payload.Filename, request.exportFilename)
		}
		if request.payload.ContentType != payload.ContentType || request.exportContentType != payload.ContentType {
			t.Fatalf("unexpected export content type metadata=%q part=%q", request.payload.ContentType, request.exportContentType)
		}
		if string(request.exportBody) != string(exportBody) {
			t.Fatalf("unexpected export body: %q", string(request.exportBody))
		}
		if request.payloadType != "export-multipart" {
			t.Fatalf("expected export-multipart payload type header, got %q", request.payloadType)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for multipart export delivery")
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
