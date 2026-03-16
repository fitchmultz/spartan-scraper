// Package webhook provides tests for shared delivery inspection payload shaping.
//
// Purpose:
//   - Prove operator-facing delivery inspection payloads redact secrets before they
//     reach API, CLI, or MCP surfaces.
//
// Responsibilities:
// - Verify URL sanitization removes credentials, query strings, and fragments.
// - Verify error strings redact obvious secrets.
// - Verify delivery timestamps remain intact after shaping.
//
// Scope:
// - Shared delivery inspection helpers only.
//
// Usage:
// - Run with `go test ./internal/webhook`.
//
// Invariants/Assumptions:
// - Inspectable delivery payloads should never expose raw credentials or tokens.
package webhook

import (
	"strings"
	"testing"
	"time"
)

func TestToInspectableDeliverySanitizesSensitiveFields(t *testing.T) {
	now := time.Date(2026, 3, 16, 12, 0, 0, 0, time.UTC)
	deliveredAt := now.Add(-time.Minute)

	record := &DeliveryRecord{
		ID:           "delivery-1",
		EventID:      "event-1",
		EventType:    EventJobCompleted,
		JobID:        "job-1",
		URL:          "https://user:pass@example.com/hooks/job?token=secret#frag",
		Status:       DeliveryStatusFailed,
		Attempts:     3,
		LastError:    `Authorization: Bearer abc123 password=hunter2`,
		CreatedAt:    now.Add(-2 * time.Minute),
		UpdatedAt:    now,
		DeliveredAt:  &deliveredAt,
		ResponseCode: 500,
	}

	got := ToInspectableDelivery(record)

	if got.URL != "https://example.com/hooks/job" {
		t.Fatalf("unexpected sanitized url: %q", got.URL)
	}
	if strings.Contains(got.LastError, "abc123") {
		t.Fatalf("expected bearer token to be redacted, got %q", got.LastError)
	}
	if strings.Contains(got.LastError, "hunter2") {
		t.Fatalf("expected password to be redacted, got %q", got.LastError)
	}
	if got.DeliveredAt != deliveredAt.Format(time.RFC3339) {
		t.Fatalf("unexpected deliveredAt: got %q want %q", got.DeliveredAt, deliveredAt.Format(time.RFC3339))
	}
}
