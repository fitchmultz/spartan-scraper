// Package api provides tests for webhook delivery API handlers.
//
// Purpose:
// - Verify pagination and detail validation behavior for webhook delivery endpoints.
//
// Responsibilities:
// - Assert shared pagination helpers are enforced on delivery listing.
// - Confirm missing IDs on delivery detail routes return validation errors.
//
// Scope:
// - HTTP handler behavior for `/v1/webhooks/deliveries`.
//
// Usage:
// - Run with `go test ./internal/api`.
//
// Invariants/Assumptions:
// - Tests use setupTestServer and do not require a configured webhook dispatcher.
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/webhook"
)

func TestHandleWebhookDeliveriesRejectsInvalidPagination(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/v1/webhooks/deliveries?limit=abc", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleWebhookDeliveryDetailMissingIDReturnsValidation(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/v1/webhooks/deliveries/", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleWebhookDeliveriesSanitizesSensitiveFields(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	deliveryStore := webhook.NewStore(srv.cfg.DataDir)
	if err := deliveryStore.Load(); err != nil {
		t.Fatalf("load webhook store: %v", err)
	}
	record := &webhook.DeliveryRecord{
		ID:           "delivery-1",
		EventID:      "event-1",
		EventType:    webhook.EventJobCompleted,
		JobID:        "job-1",
		URL:          "https://user:pass@example.com/hooks/job?token=secret#frag",
		Status:       webhook.DeliveryStatusFailed,
		Attempts:     2,
		LastError:    "Authorization: Bearer abc123 password=hunter2",
		ResponseCode: 500,
		CreatedAt:    time.Date(2026, 3, 16, 12, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2026, 3, 16, 12, 1, 0, 0, time.UTC),
	}
	if err := deliveryStore.CreateRecord(context.Background(), record); err != nil {
		t.Fatalf("create delivery record: %v", err)
	}
	srv.webhookDispatcher = webhook.NewDispatcherWithStore(webhook.Config{}, deliveryStore)

	req := httptest.NewRequest(http.MethodGet, "/v1/webhooks/deliveries", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var payload struct {
		Deliveries []webhook.InspectableDelivery `json:"deliveries"`
		Total      int                           `json:"total"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Total != 1 || len(payload.Deliveries) != 1 {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	delivery := payload.Deliveries[0]
	if delivery.URL != "https://example.com/hooks/job" {
		t.Fatalf("expected sanitized url, got %q", delivery.URL)
	}
	if strings.Contains(delivery.LastError, "abc123") || strings.Contains(delivery.LastError, "hunter2") {
		t.Fatalf("expected sanitized lastError, got %q", delivery.LastError)
	}
}
