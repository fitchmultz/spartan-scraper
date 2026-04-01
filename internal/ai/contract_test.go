// Package ai provides ai functionality for Spartan Scraper.
//
// Purpose:
// - Verify contract test behavior for package ai.
//
// Responsibilities:
// - Define focused Go test coverage, fixtures, and assertions for the package behavior exercised here.
//
// Scope:
// - Automated test coverage only; production behavior stays in non-test package files.
//
// Usage:
// - Run with `go test` for package `ai` or through `make test-ci`/`make ci`.
//
// Invariants/Assumptions:
// - Tests should remain deterministic and describe the package contract they protect.

package ai

import "testing"

func TestExtractResultCanonicalizeDefaultsFields(t *testing.T) {
	result := ExtractResult{
		Fields: map[string]BridgeFieldValue{
			"title": {},
		},
		Confidence: 0.9,
	}

	if err := result.Canonicalize(); err != nil {
		t.Fatalf("Canonicalize() failed: %v", err)
	}

	field := result.Fields["title"]
	if field.Source != "llm" {
		t.Fatalf("expected default source llm, got %q", field.Source)
	}
	if field.Values == nil {
		t.Fatal("expected values to default to empty slice")
	}
	if len(field.Values) != 0 {
		t.Fatalf("expected empty values slice, got %v", field.Values)
	}
}

func TestExtractResultCanonicalizeRejectsEmptyFieldName(t *testing.T) {
	result := ExtractResult{
		Fields: map[string]BridgeFieldValue{
			"   ": {},
		},
		Confidence: 0.9,
	}

	if err := result.Canonicalize(); err == nil {
		t.Fatal("expected Canonicalize() to reject empty field name")
	}
}

func TestExtractResultCanonicalizeClampsConfidence(t *testing.T) {
	result := ExtractResult{
		Fields: map[string]BridgeFieldValue{
			"title": {},
		},
		Confidence: 1.4,
	}

	if err := result.Canonicalize(); err != nil {
		t.Fatalf("Canonicalize() failed: %v", err)
	}
	if result.Confidence != 1 {
		t.Fatalf("expected confidence to clamp to 1, got %v", result.Confidence)
	}

	result.Confidence = -0.2
	if err := result.Canonicalize(); err != nil {
		t.Fatalf("Canonicalize() failed: %v", err)
	}
	if result.Confidence != 0 {
		t.Fatalf("expected confidence to clamp to 0, got %v", result.Confidence)
	}
}
