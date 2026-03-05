// Package fetch provides HTTP and headless browser content fetching capabilities.
//
// This file contains utility functions for form detection, including CSS escaping
// and sorting functions.
package fetch

import (
	"strings"
)

// CSSEscape escapes a string for use in CSS selectors.
// This is a simplified version - handles common cases.
func CSSEscape(s string) string {
	// Replace characters that need escaping in CSS selectors
	s = strings.ReplaceAll(s, "'", "\\'")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

// sortFormsByScore sorts forms by score descending (highest confidence first).
func sortFormsByScore(forms []DetectedForm) {
	// Simple bubble sort for small arrays
	for i := range forms {
		for j := i + 1; j < len(forms); j++ {
			if forms[j].Score > forms[i].Score {
				forms[i], forms[j] = forms[j], forms[i]
			}
		}
	}
}

// min returns the minimum of two float64 values.
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
