// Package watch tests selector extraction failure boundaries for watch checks.
//
// Purpose:
// - Verify selector-driven watches fail deterministically for invalid or non-matching selectors.
//
// Responsibilities:
// - Confirm malformed CSS selectors return errors.
// - Confirm selectors with zero text-bearing matches return errors instead of silent empty baselines.
//
// Scope:
// - Focused unit coverage for selector extraction only.
//
// Usage:
// - Run with `go test ./internal/watch`.
//
// Invariants/Assumptions:
// - Watch selectors must either extract meaningful text or fail with a clear operator-facing error.
package watch

import (
	"strings"
	"testing"
)

func TestExtractSelectorRejectsInvalidSelector(t *testing.T) {
	_, err := extractSelector(`<html><body><main>Hello</main></body></html>`, "[")
	if err == nil {
		t.Fatal("expected invalid selector error")
	}
	if !strings.Contains(err.Error(), "invalid selector") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExtractSelectorRejectsSelectorWithoutMatches(t *testing.T) {
	_, err := extractSelector(`<html><body><main>Hello</main></body></html>`, ".missing")
	if err == nil {
		t.Fatal("expected missing selector match error")
	}
	if !strings.Contains(err.Error(), "no elements matched selector") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExtractSelectorRejectsSelectorWithoutText(t *testing.T) {
	_, err := extractSelector(`<html><body><div><span>   </span></div></body></html>`, "div")
	if err == nil {
		t.Fatal("expected empty text extraction error")
	}
	if !strings.Contains(err.Error(), "extracted no text") {
		t.Fatalf("unexpected error: %v", err)
	}
}
