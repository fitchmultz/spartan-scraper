// Package fetch provides fetch functionality for Spartan Scraper.
//
// Purpose:
// - Verify form detect utils test behavior for package fetch.
//
// Responsibilities:
// - Define focused Go test coverage, fixtures, and assertions for the package behavior exercised here.
//
// Scope:
// - Automated test coverage only; production behavior stays in non-test package files.
//
// Usage:
// - Run with `go test` for package `fetch` or through `make test-ci`/`make ci`.
//
// Invariants/Assumptions:
// - Tests should remain deterministic and describe the package contract they protect.

package fetch

import (
	"testing"
)

// TestCSSEscape tests the CSS escaping function.
func TestCSSEscape(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with-dash", "with-dash"},
		{"with_underscore", "with_underscore"},
		{"with'quote", "with\\'quote"},
		{"with\"double", "with\\\"double"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := CSSEscape(tt.input)
			if result != tt.expected {
				t.Errorf("CSSEscape(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}
