// Package cli provides tests for the top-level help text.
//
// Purpose:
//   - Keep the published command index aligned with the actual CLI router.
//
// Responsibilities:
//   - Verify top-level help includes commands that are routed by cli.Run.
//
// Scope:
//   - Top-level help output only.
//
// Usage:
//   - Run with `go test ./internal/cli`.
//
// Invariants/Assumptions:
//   - Top-level help should expose real user-facing commands instead of hiding them.
package cli

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestPrintHelp_IncludesRoutedCommands(t *testing.T) {
	output := captureStdout(t, func() {
		printHelp()
	})

	for _, expected := range []string{
		"  ai           AI authoring utilities (preview, templates, render profiles, pipeline JS, research refinement, transforms)",
		"  proxy-pool   Inspect proxy-pool configuration and runtime status",
		"  webhook      Inspect webhook delivery history",
		"  export       Export job results (jsonl, json, md, csv, xlsx)",
		"  export-schedule Manage automated export schedules",
		"  spartan ai preview --url https://example.com --prompt \"Extract the main product facts\"",
		"  spartan scrape --url https://example.com --proxy-region us-east --proxy-tag residential --out ./out/example.json",
		"  spartan webhook deliveries list --job-id <job-id>",
		"  spartan export --job-id <id> --format md --out ./out/report.md",
		"  spartan export --job-id <id> --schedule-id <export-schedule-id> --out ./out/projected.csv",
		"  spartan export-schedule list",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("printHelp() missing %q\noutput:\n%s", expected, output)
		}
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() failed: %v", err)
	}
	os.Stdout = writer
	t.Cleanup(func() {
		os.Stdout = original
	})

	fn()

	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() failed: %v", err)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		t.Fatalf("io.Copy() failed: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("reader.Close() failed: %v", err)
	}

	return buf.String()
}
