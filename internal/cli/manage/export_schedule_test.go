// Package manage tests export-schedule CLI behavior.
//
// Purpose:
// - Verify operator-facing export-schedule commands surface schedule validation failures clearly.
//
// Responsibilities:
// - Exercise CLI authoring flows that should reject invalid local destinations before persistence.
//
// Scope:
// - `spartan export-schedule` command handling only.
//
// Usage:
// - Run with `go test ./internal/cli/manage`.
//
// Invariants/Assumptions:
// - Invalid local destinations must fail with a non-zero exit code.
// - CLI stderr should mention the stable DATA_DIR/exports policy.
package manage

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/config"
)

func TestRunExportScheduleRejectsInvalidLocalDestination(t *testing.T) {
	stdout, stderr, code := captureCLIOutputWithCode(t, func() int {
		return RunExportSchedule(context.Background(), config.Config{DataDir: t.TempDir()}, []string{
			"add",
			"--name", "Outside Root",
			"--filter-kinds", "scrape",
			"--format", "json",
			"--destination", "local",
			"--local-path", "/tmp/out.json",
		})
	})

	if code != 1 {
		t.Fatalf("RunExportSchedule() code = %d, want 1", code)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "DATA_DIR/exports") {
		t.Fatalf("stderr = %q, want DATA_DIR/exports policy", stderr)
	}
}

func captureCLIOutputWithCode(t *testing.T, fn func() int) (string, string, int) {
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

	code := fn()

	if err := stdoutWriter.Close(); err != nil {
		t.Fatalf("close stdout writer: %v", err)
	}
	if err := stderrWriter.Close(); err != nil {
		t.Fatalf("close stderr writer: %v", err)
	}
	os.Stdout = originalStdout
	os.Stderr = originalStderr

	return readCLIOutputPipe(t, stdoutReader), readCLIOutputPipe(t, stderrReader), code
}

func readCLIOutputPipe(t *testing.T, reader *os.File) string {
	t.Helper()
	defer reader.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	return buf.String()
}
