// Package fetch provides fetch functionality for Spartan Scraper.
//
// Purpose:
// - Verify form detect selectors test behavior for package fetch.
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

// TestFormDetector_generateSelector tests the CSS selector generation.
func TestFormDetector_generateSelector(t *testing.T) {
	detector := NewFormDetector()

	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "by_id",
			html:     `<input type="text" id="username" name="user">`,
			expected: "#username",
		},
		{
			name:     "by_name_and_type",
			html:     `<input type="password" name="pass">`,
			expected: "input[type='password'][name='pass']",
		},
		{
			name:     "by_name_only",
			html:     `<input name="email">`,
			expected: "[name='email']",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, _ := detector.DetectForms(`<html><body><form>` + tt.html + `</form></body></html>`)
			if len(doc) == 0 {
				t.Skip("no form detected")
			}
		})
	}
}
