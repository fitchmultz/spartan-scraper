// Package scrape contains tests for the research subcommand.
//
// Responsibilities:
// - Testing RunResearch flag validation (query, URLs)
// - Testing job creation with valid flags
// - Validating error handling for missing or invalid flags
//
// Non-goals:
// - Testing actual research/analysis logic
// - Testing LLM integration or external APIs
//
// Assumptions:
// - Tests use temporary directories for data storage
// - Tests capture stdout/stderr for validation
// - No external network access required
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

func TestRunResearch_MissingQueryFlag(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{DataDir: t.TempDir()}

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := RunResearch(ctx, cfg, []string{"--urls", "https://example.com"})

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	stderr := buf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr, "--query and --urls are required") {
		t.Errorf("expected stderr to contain '--query and --urls are required', got %q", stderr)
	}
}

func TestRunResearch_MissingURLsFlag(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{DataDir: t.TempDir()}

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := RunResearch(ctx, cfg, []string{"--query", "test"})

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	stderr := buf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr, "--query and --urls are required") {
		t.Errorf("expected stderr to contain '--query and --urls are required', got %q", stderr)
	}
}

func TestRunResearch_InvalidURLList(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{DataDir: t.TempDir()}

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := RunResearch(ctx, cfg, []string{"--query", "test", "--urls", "invalid,https://example.com"})

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	stderr := buf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stderr, "URL") && !strings.Contains(stderr, "invalid") {
		t.Errorf("expected stderr to contain validation error about invalid URL, got %q", stderr)
	}
}

func TestRunResearch_EmptyURLList(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{DataDir: t.TempDir()}

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := RunResearch(ctx, cfg, []string{"--query", "test", "--urls", ""})

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	stderr := buf.String()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if stderr == "" {
		t.Errorf("expected stderr to have error message, got empty string")
	}
}

func TestRunResearch_ValidFlagsCreateJob(t *testing.T) {
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

	exitCode := RunResearch(ctx, cfg, []string{"--query", "test", "--urls", "https://example.com,https://example.org"})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(output, "\"kind\": \"research\"") {
		t.Errorf("expected output to contain job data, got %q", output)
	}
}
