package scrape

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/config"
)

func TestRunCrawl_MissingURLFlag(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{DataDir: t.TempDir()}

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := RunCrawl(ctx, cfg, []string{})

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	stderr := buf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr, "--url is required") {
		t.Errorf("expected stderr to contain '--url is required', got %q", stderr)
	}
}

func TestRunCrawl_InvalidMaxDepthValues(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{DataDir: t.TempDir()}

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := RunCrawl(ctx, cfg, []string{"--url", "https://example.com", "--max-depth", "-1"})

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	stderr := buf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr, "maxDepth") {
		t.Errorf("expected stderr to contain validation error about maxDepth, got %q", stderr)
	}
}

func TestRunCrawl_InvalidMaxPagesValues(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{DataDir: t.TempDir()}

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := RunCrawl(ctx, cfg, []string{"--url", "https://example.com", "--max-pages", "-1"})

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	stderr := buf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr, "maxPages") {
		t.Errorf("expected stderr to contain validation error about maxPages, got %q", stderr)
	}
}

func TestRunCrawl_ValidFlagsCreateJob(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	cfg := config.Config{
		DataDir:            tmpDir,
		UsePlaywright:      false,
		RequestTimeoutSecs: 30,
		MaxConcurrency:     1,
		RateLimitQPS:       10,
		RateLimitBurst:     10,
		MaxRetries:         3,
		RetryBaseMs:        100,
		MaxResponseBytes:   10 * 1024 * 1024,
		UserAgent:          "test-agent",
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := RunCrawl(ctx, cfg, []string{"--url", "https://example.com", "--max-depth", "2"})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(output, "\"kind\": \"crawl\"") {
		t.Errorf("expected output to contain job data, got %q", output)
	}
}
