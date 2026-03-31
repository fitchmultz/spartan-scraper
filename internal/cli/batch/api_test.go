// Package batch verifies REST batch transport helpers.
//
// Purpose:
// - Prove CLI batch HTTP helpers surface response-body read failures instead of silently ignoring them.
//
// Responsibilities:
// - Exercise malformed/truncated HTTP responses for create and list flows.
// - Guard the shared response-body reader used by batch API helpers.
//
// Scope:
// - HTTP response handling only; command parsing and direct-mode execution live in other tests.
//
// Usage:
// - Run with `go test ./internal/cli/batch`.
//
// Invariants/Assumptions:
// - Test servers may deliberately send truncated HTTP bodies.
// - Returned errors should preserve the response-body read failure text for operators.
package batch

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func writeTruncatedJSONResponse(t *testing.T, w http.ResponseWriter, statusCode int) {
	t.Helper()

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		t.Fatal("response writer does not support hijacking")
	}

	conn, buf, err := hijacker.Hijack()
	if err != nil {
		t.Fatalf("failed to hijack response: %v", err)
	}
	defer conn.Close()

	body := `{}`
	if _, err := fmt.Fprintf(
		buf,
		"HTTP/1.1 %d %s\r\nContent-Type: application/json\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s",
		statusCode,
		http.StatusText(statusCode),
		len(body)+8,
		body,
	); err != nil {
		t.Fatalf("failed to write truncated response: %v", err)
	}
	if err := buf.Flush(); err != nil {
		t.Fatalf("failed to flush truncated response: %v", err)
	}
}

func TestSubmitBatchViaAPISurfacesResponseBodyReadFailures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTruncatedJSONResponse(t, w, http.StatusCreated)
	}))
	defer server.Close()

	_, err := submitBatchViaAPI(context.Background(), server.URL, BatchScrapeRequest{
		Jobs: []BatchJobRequest{{URL: "https://example.com"}},
	})
	if err == nil {
		t.Fatal("expected response-body read error")
	}
	if !strings.Contains(err.Error(), "failed to read response body") {
		t.Fatalf("expected read failure in error, got %v", err)
	}
}

func TestListBatchesViaAPISurfacesResponseBodyReadFailures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTruncatedJSONResponse(t, w, http.StatusOK)
	}))
	defer server.Close()

	_, port, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		t.Fatalf("failed to resolve test server port: %v", err)
	}

	_, err = listBatchesViaAPI(context.Background(), port, 10, 0)
	if err == nil {
		t.Fatal("expected response-body read error")
	}
	if !strings.Contains(err.Error(), "failed to read response body") {
		t.Fatalf("expected read failure in error, got %v", err)
	}
}
