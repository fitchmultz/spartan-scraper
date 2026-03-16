// Package manage provides tests for webhook delivery inspection CLI commands.
//
// Purpose:
// - Verify terminal operators can inspect persisted webhook deliveries without a running API server.
//
// Responsibilities:
// - Assert list output shows sanitized delivery metadata.
// - Assert get output returns sanitized JSON detail.
//
// Scope:
// - Direct-store webhook inspection flows only.
//
// Usage:
// - Run with `go test ./internal/cli/manage`.
//
// Invariants/Assumptions:
// - CLI output must not expose raw webhook credentials, query tokens, or secrets.
package manage

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/webhook"
)

func TestRunWebhookDeliveriesList_DirectStore(t *testing.T) {
	tmpDir := t.TempDir()
	seedWebhookDelivery(t, tmpDir, &webhook.DeliveryRecord{
		ID:           "delivery-list-1",
		EventID:      "event-1",
		EventType:    webhook.EventJobCompleted,
		JobID:        "job-123",
		URL:          "https://user:pass@example.com/hooks/job?token=secret",
		Status:       webhook.DeliveryStatusFailed,
		Attempts:     2,
		LastError:    "Authorization: Bearer abc123 password=hunter2",
		ResponseCode: 500,
		CreatedAt:    time.Date(2026, 3, 16, 15, 4, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2026, 3, 16, 15, 5, 0, 0, time.UTC),
	})

	cfg := config.Config{DataDir: tmpDir, Port: "0"}
	stdout, stderr := captureCLIOutput(t, func() int {
		return RunWebhook(context.Background(), cfg, []string{"deliveries", "list"})
	})

	if stderr != "" {
		t.Fatalf("expected no stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "delivery-list-1") {
		t.Fatalf("expected output to include delivery id, got %q", stdout)
	}
	if !strings.Contains(stdout, "https://example.com/hooks/job") {
		t.Fatalf("expected sanitized url in output, got %q", stdout)
	}
	if strings.Contains(stdout, "token=secret") || strings.Contains(stdout, "hunter2") || strings.Contains(stdout, "abc123") {
		t.Fatalf("expected sensitive values to be redacted, got %q", stdout)
	}
	if !strings.Contains(stdout, "Authorization: Bearer [REDACTED] password=[REDACTED]") {
		t.Fatalf("expected redacted error output, got %q", stdout)
	}
	if !strings.Contains(stdout, "500") {
		t.Fatalf("expected response code in output, got %q", stdout)
	}
}

func TestRunWebhookDeliveriesGet_DirectStore(t *testing.T) {
	tmpDir := t.TempDir()
	seedWebhookDelivery(t, tmpDir, &webhook.DeliveryRecord{
		ID:        "delivery-get-1",
		EventID:   "event-2",
		EventType: webhook.EventExportCompleted,
		JobID:     "job-456",
		URL:       "https://user:pass@example.com/export?signature=secret",
		Status:    webhook.DeliveryStatusDelivered,
		Attempts:  1,
		LastError: "token=secret",
		CreatedAt: time.Date(2026, 3, 16, 16, 4, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 3, 16, 16, 5, 0, 0, time.UTC),
	})

	cfg := config.Config{DataDir: tmpDir, Port: "0"}
	stdout, stderr := captureCLIOutput(t, func() int {
		return RunWebhook(context.Background(), cfg, []string{"deliveries", "get", "delivery-get-1"})
	})

	if stderr != "" {
		t.Fatalf("expected no stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"id": "delivery-get-1"`) {
		t.Fatalf("expected JSON detail output, got %q", stdout)
	}
	if !strings.Contains(stdout, `"url": "https://example.com/export"`) {
		t.Fatalf("expected sanitized url in JSON output, got %q", stdout)
	}
	if strings.Contains(stdout, "signature=secret") || strings.Contains(stdout, `"lastError": "token=secret"`) {
		t.Fatalf("expected sensitive values to be redacted, got %q", stdout)
	}
	if !strings.Contains(stdout, `"lastError": "token=[REDACTED]"`) {
		t.Fatalf("expected redacted lastError in JSON output, got %q", stdout)
	}
}

func seedWebhookDelivery(t *testing.T, dataDir string, record *webhook.DeliveryRecord) {
	t.Helper()
	store := webhook.NewStore(dataDir)
	if err := store.Load(); err != nil {
		t.Fatalf("load store: %v", err)
	}
	if err := store.CreateRecord(context.Background(), record); err != nil {
		t.Fatalf("create record: %v", err)
	}
}

func captureCLIOutput(t *testing.T, fn func() int) (string, string) {
	t.Helper()

	originalStdout := os.Stdout
	originalStderr := os.Stderr
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("stderr pipe: %v", err)
	}
	os.Stdout = stdoutWriter
	os.Stderr = stderrWriter
	t.Cleanup(func() {
		os.Stdout = originalStdout
		os.Stderr = originalStderr
	})

	if code := fn(); code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	if err := stdoutWriter.Close(); err != nil {
		t.Fatalf("close stdout writer: %v", err)
	}
	if err := stderrWriter.Close(); err != nil {
		t.Fatalf("close stderr writer: %v", err)
	}
	os.Stdout = originalStdout
	os.Stderr = originalStderr

	return readPipe(t, stdoutReader), readPipe(t, stderrReader)
}

func readPipe(t *testing.T, reader *os.File) string {
	t.Helper()
	defer reader.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	return buf.String()
}
