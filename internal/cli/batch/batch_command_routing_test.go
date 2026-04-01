// Package batch contains scenario-focused tests for the batch subcommand.
//
// Purpose:
// - Verify batch command routing and argument guardrails.
//
// Responsibilities:
// - Cover top-level help, unknown subcommands, empty-list handling, and status/cancel argument validation.
//
// Scope:
// - Batch CLI behavior only.
//
// Usage:
// - Run with `go test ./internal/cli/batch`.
//
// Invariants/Assumptions:
// - Tests use temp files and local stores only.
package batch

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/config"
)

func TestRunBatch_NoSubcommand(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := RunBatch(ctx, cfg, []string{})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(output, "Usage:") {
		t.Errorf("expected help output, got %q", output)
	}
}
func TestRunBatch_Help(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := RunBatch(ctx, cfg, []string{"help"})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(output, "submit") {
		t.Errorf("expected help to mention submit subcommand, got %q", output)
	}
	if !strings.Contains(output, "list") {
		t.Errorf("expected help to mention list subcommand, got %q", output)
	}
}
func TestRunBatch_UnknownSubcommand(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{}

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := RunBatch(ctx, cfg, []string{"unknown"})

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	stderr := buf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr, "unknown batch subcommand") {
		t.Errorf("expected error about unknown subcommand, got %q", stderr)
	}
}
func TestRunBatchList_NoBatches(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{DataDir: t.TempDir()}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := RunBatch(ctx, cfg, []string{"list"})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(output, "No batches found.") {
		t.Errorf("expected empty-state output, got %q", output)
	}
}
func TestRunBatchStatus_NoBatchID(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{}

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := RunBatch(ctx, cfg, []string{"status"})

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	stderr := buf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr, "batch ID required") {
		t.Errorf("expected error about missing batch ID, got %q", stderr)
	}
}
func TestRunBatchCancel_NoBatchID(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{}

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := RunBatch(ctx, cfg, []string{"cancel"})

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	stderr := buf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr, "batch ID required") {
		t.Errorf("expected error about missing batch ID, got %q", stderr)
	}
}
