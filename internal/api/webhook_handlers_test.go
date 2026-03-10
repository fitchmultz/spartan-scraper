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
	"net/http"
	"net/http/httptest"
	"testing"
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
