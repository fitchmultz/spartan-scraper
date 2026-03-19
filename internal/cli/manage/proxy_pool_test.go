// Package manage tests capability-aware proxy-pool CLI output.
//
// Purpose:
// - Verify proxy-pool status uses the shared recommended-action bullet format.
//
// Responsibilities:
// - Capture terminal output from proxy-pool status rendering.
// - Assert translated one-click actions render as CLI commands.
// - Prevent regressions back to legacy "Next step" phrasing.
//
// Scope:
// - Proxy-pool status rendering only.
//
// Usage:
// - Run with `go test ./internal/cli/manage`.
//
// Invariants/Assumptions:
// - Optional disabled proxy-pool states should still surface next steps clearly.
// - Shared CLI rendering should stay consistent with `spartan health`.
package manage

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/api"
)

func TestPrintProxyPoolStatusUsesSharedBulletActions(t *testing.T) {
	output := captureProxyPoolStdout(t, func() {
		printProxyPoolStatus(api.ProxyPoolStatusResponse{}, false, "/tmp/missing-proxies.json")
	})

	if !strings.Contains(output, "- Inspect configured proxy pool file") {
		t.Fatalf("expected shared bullet action format, got %q", output)
	}
	if strings.Contains(output, "Next step:") {
		t.Fatalf("expected legacy next-step prefix to be removed, got %q", output)
	}
}

func captureProxyPoolStdout(t *testing.T, fn func()) string {
	t.Helper()
	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	defer reader.Close()

	os.Stdout = writer
	defer func() {
		os.Stdout = original
	}()

	fn()
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	var buffer bytes.Buffer
	if _, err := io.Copy(&buffer, reader); err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return buffer.String()
}
