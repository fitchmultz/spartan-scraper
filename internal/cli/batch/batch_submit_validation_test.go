// Package batch contains scenario-focused tests for the batch subcommand.
//
// Purpose:
// - Verify batch submit validation flows.
//
// Responsibilities:
// - Cover missing-kind, unknown-kind, scrape/crawl/research guardrails, and URL-count validation.
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

func TestRunBatchSubmit_NoKind(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := RunBatch(ctx, cfg, []string{"submit"})

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
func TestRunBatchSubmit_UnknownKind(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{}

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := RunBatch(ctx, cfg, []string{"submit", "unknown"})

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	stderr := buf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr, "unknown batch submit kind") {
		t.Errorf("expected error about unknown kind, got %q", stderr)
	}
}
func TestRunBatchSubmitScrape_NoURLs(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{DataDir: t.TempDir()}

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := RunBatch(ctx, cfg, []string{"submit", "scrape"})

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	stderr := buf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr, "batch must contain at least one job") {
		t.Errorf("expected canonical batch-size validation error, got %q", stderr)
	}
}
func TestRunBatchSubmitCrawl_NoURLs(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{DataDir: t.TempDir()}

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := RunBatch(ctx, cfg, []string{"submit", "crawl"})

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	stderr := buf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr, "batch must contain at least one job") {
		t.Errorf("expected canonical batch-size validation error, got %q", stderr)
	}
}
func TestRunBatchSubmitResearch_NoQuery(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{DataDir: t.TempDir()}

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := RunBatch(ctx, cfg, []string{"submit", "research", "--urls", "https://example.com"})

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	stderr := buf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr, "query and urls are required") {
		t.Errorf("expected canonical research validation error, got %q", stderr)
	}
}
func TestRunBatchSubmitResearch_InvalidAgenticOptions(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{DataDir: t.TempDir()}

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := RunBatch(ctx, cfg, []string{
		"submit", "research",
		"--urls", "https://example.com",
		"--query", "pricing model",
		"--agentic",
		"--agentic-max-rounds", "5",
	})

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	stderr := buf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr, "agentic.maxRounds must be between 1 and 3") {
		t.Errorf("expected canonical agentic validation error, got %q", stderr)
	}
}
func TestRunBatchSubmitResearch_NoURLs(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{DataDir: t.TempDir()}

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := RunBatch(ctx, cfg, []string{"submit", "research", "--query", "test"})

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	stderr := buf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr, "batch must contain at least one job") {
		t.Errorf("expected canonical batch-size validation error, got %q", stderr)
	}
}
func TestRunBatchSubmitScrape_TooManyURLs(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{DataDir: t.TempDir(), MaxBatchSize: 2}

	urls := []string{
		"https://example-a.com",
		"https://example-b.com",
		"https://example-c.com",
	}

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := RunBatch(ctx, cfg, []string{"submit", "scrape", "--urls", strings.Join(urls, ",")})

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	stderr := buf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr, "batch size 3 exceeds maximum of 2") {
		t.Errorf("expected config-driven batch-size error, got %q", stderr)
	}
}
