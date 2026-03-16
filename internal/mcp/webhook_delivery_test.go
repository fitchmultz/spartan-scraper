// Package mcp provides tests for webhook delivery inspection tools.
//
// Purpose:
// - Prove MCP callers can inspect persisted webhook delivery history.
//
// Responsibilities:
// - Verify tool registration and input schema coverage.
// - Verify list/detail handlers return sanitized delivery data.
// - Verify missing deliveries return not-found errors.
//
// Scope:
// - Webhook inspection MCP tools only.
//
// Usage:
// - Run with `go test ./internal/mcp`.
//
// Invariants/Assumptions:
// - MCP responses must not expose raw webhook credentials or secret-bearing error strings.
package mcp

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/webhook"
)

func TestWebhookDeliveryToolsInToolsList(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	toolMap := make(map[string]tool)
	for _, tool := range srv.toolsList() {
		toolMap[tool.Name] = tool
	}

	listTool, ok := toolMap["webhook_delivery_list"]
	if !ok {
		t.Fatal("expected webhook_delivery_list in toolsList")
	}
	props := listTool.InputSchema["properties"].(map[string]interface{})
	for _, key := range []string{"jobId", "limit", "offset"} {
		if _, ok := props[key]; !ok {
			t.Fatalf("expected %s in webhook_delivery_list schema", key)
		}
	}

	getTool, ok := toolMap["webhook_delivery_get"]
	if !ok {
		t.Fatal("expected webhook_delivery_get in toolsList")
	}
	required := getTool.InputSchema["required"].([]string)
	if len(required) != 1 || required[0] != "id" {
		t.Fatalf("unexpected required fields for webhook_delivery_get: %#v", required)
	}
}

func TestHandleWebhookDeliveryTools(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	seedMCPWebhookDelivery(t, tmpDir, &webhook.DeliveryRecord{
		ID:           "delivery-1",
		EventID:      "event-1",
		EventType:    webhook.EventJobCompleted,
		JobID:        "job-1",
		URL:          "https://user:pass@example.com/hooks/job?token=secret#frag",
		Status:       webhook.DeliveryStatusFailed,
		Attempts:     3,
		LastError:    "Authorization: Bearer abc123 password=hunter2",
		ResponseCode: 500,
		CreatedAt:    time.Date(2026, 3, 16, 12, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2026, 3, 16, 12, 1, 0, 0, time.UTC),
	})

	ctx := context.Background()
	listResult, err := srv.handleToolCall(ctx, map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name":      "webhook_delivery_list",
			"arguments": map[string]interface{}{"jobId": "job-1", "limit": 10, "offset": 0},
		}),
	})
	if err != nil {
		t.Fatalf("webhook_delivery_list failed: %v", err)
	}

	listPayload, ok := listResult.(map[string]interface{})
	if !ok {
		t.Fatalf("expected list result map, got %T", listResult)
	}
	deliveries, ok := listPayload["deliveries"].([]webhook.InspectableDelivery)
	if !ok {
		t.Fatalf("expected deliveries payload, got %#v", listPayload["deliveries"])
	}
	if len(deliveries) != 1 {
		t.Fatalf("expected 1 delivery, got %d", len(deliveries))
	}
	if deliveries[0].URL != "https://example.com/hooks/job" {
		t.Fatalf("expected sanitized url, got %q", deliveries[0].URL)
	}
	if strings.Contains(deliveries[0].LastError, "abc123") || strings.Contains(deliveries[0].LastError, "hunter2") {
		t.Fatalf("expected sanitized lastError, got %q", deliveries[0].LastError)
	}
	if total, ok := listPayload["total"].(int); !ok || total != 1 {
		t.Fatalf("unexpected total payload: %#v", listPayload["total"])
	}

	getResult, err := srv.handleToolCall(ctx, map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name":      "webhook_delivery_get",
			"arguments": map[string]interface{}{"id": "delivery-1"},
		}),
	})
	if err != nil {
		t.Fatalf("webhook_delivery_get failed: %v", err)
	}
	getDelivery, ok := getResult.(webhook.InspectableDelivery)
	if !ok {
		t.Fatalf("expected inspectable delivery, got %T", getResult)
	}
	if getDelivery.URL != "https://example.com/hooks/job" {
		t.Fatalf("expected sanitized url, got %q", getDelivery.URL)
	}
}

func TestHandleWebhookDeliveryGetNotFound(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	_, err := srv.handleToolCall(context.Background(), map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name":      "webhook_delivery_get",
			"arguments": map[string]interface{}{"id": "missing"},
		}),
	})
	if err == nil {
		t.Fatal("expected error for missing webhook delivery")
	}
	if !apperrors.IsKind(err, apperrors.KindNotFound) {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func seedMCPWebhookDelivery(t *testing.T, dataDir string, record *webhook.DeliveryRecord) {
	t.Helper()
	store := webhook.NewStore(dataDir)
	if err := store.Load(); err != nil {
		t.Fatalf("load store: %v", err)
	}
	if err := store.CreateRecord(context.Background(), record); err != nil {
		t.Fatalf("create record: %v", err)
	}
}
