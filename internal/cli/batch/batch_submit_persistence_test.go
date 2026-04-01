// Package batch contains scenario-focused tests for the batch subcommand.
//
// Purpose:
// - Verify batch submit persistence flows.
//
// Responsibilities:
// - Cover stored research agentic options and AI extraction request persistence.
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
	"context"
	"io"
	"os"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/config"
)

func TestRunBatchSubmitResearch_StoresAgenticOptions(t *testing.T) {
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

	exitCode := RunBatch(ctx, cfg, []string{
		"submit", "research",
		"--urls", "https://example.com,https://example.org",
		"--query", "pricing model",
		"--agentic",
		"--agentic-instructions", "Prioritize pricing and support commitments",
		"--agentic-max-rounds", "2",
		"--agentic-max-follow-up-urls", "4",
	})

	w.Close()
	os.Stdout = old
	io.Copy(io.Discard, r)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	spec := latestBatchJobSpec(t, tmpDir)
	agenticMap, ok := spec["agentic"].(map[string]interface{})
	if !ok {
		t.Fatalf("agentic spec missing: %#v", spec["agentic"])
	}
	if enabled, _ := agenticMap["enabled"].(bool); !enabled {
		t.Fatalf("expected agentic.enabled true, got %v", enabled)
	}
	if instructions, _ := agenticMap["instructions"].(string); instructions != "Prioritize pricing and support commitments" {
		t.Fatalf("expected instructions to be stored, got %q", instructions)
	}
}
func TestRunBatchSubmitResearch_StoresAIExtractOptions(t *testing.T) {
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

	exitCode := RunBatch(ctx, cfg, []string{
		"submit", "research",
		"--urls", "https://example.com,https://example.org",
		"--query", "pricing model",
		"--ai-extract",
		"--ai-prompt", "Extract the pricing model and support terms",
		"--ai-fields", "pricing_model,support_terms",
	})

	w.Close()
	os.Stdout = old
	io.Copy(io.Discard, r)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	spec := latestBatchJobSpec(t, tmpDir)
	extractMap, ok := spec["extract"].(map[string]interface{})
	if !ok {
		t.Fatalf("extract spec missing: %#v", spec["extract"])
	}
	aiMap, ok := extractMap["ai"].(map[string]interface{})
	if !ok {
		t.Fatalf("ai spec missing: %#v", extractMap["ai"])
	}
	if mode, _ := aiMap["mode"].(string); mode != "natural_language" {
		t.Fatalf("expected natural_language mode, got %q", mode)
	}
	if prompt, _ := aiMap["prompt"].(string); prompt != "Extract the pricing model and support terms" {
		t.Fatalf("expected prompt to be stored, got %q", prompt)
	}
}
