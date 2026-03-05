// Package hostmatch provides centralized host extraction and pattern matching utilities.
// It consolidates the duplicated host matching logic found in fetch, pipeline, and auth packages.
//
// Responsibilities:
// - Extract host from URLs (with or without scheme)
// - Match hosts against glob-style patterns (exact, *.suffix, prefix.*)
// - Validate host pattern lists
//
// This package does NOT:
// - Handle URL path matching
// - Support full regex patterns (only glob-style wildcards)
// - Perform DNS resolution
//
// Invariants:
// - All inputs are normalized to lowercase for comparison
// - Empty hosts never match
// - Empty pattern lists never match
package hostmatch

import (
	"net/url"
	"strconv"
	"strings"
)

// HostFromURL extracts the hostname from a URL string.
// Handles URLs with or without scheme (defaults to https:// if no scheme).
// Returns empty string if parsing fails.
func HostFromURL(rawURL string) string {
	raw := strings.TrimSpace(rawURL)
	if raw == "" {
		return ""
	}

	// Add scheme if missing
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}

	return strings.ToLower(parsed.Hostname())
}

// HostMatchesAnyPattern checks if a host matches any of the provided glob-style patterns.
// Supported patterns:
//   - exact match: "example.com"
//   - wildcard subdomain: "*.example.com" (matches sub.example.com but not example.com)
//   - wildcard prefix: "example.*" (matches example.com, example.org, etc.)
//
// Returns false if host is empty or patterns is empty.
func HostMatchesAnyPattern(host string, patterns []string) bool {
	if host == "" || len(patterns) == 0 {
		return false
	}

	host = strings.ToLower(strings.TrimSpace(host))

	for _, pattern := range patterns {
		pattern = strings.ToLower(strings.TrimSpace(pattern))
		if pattern == "" {
			continue
		}

		// Exact match
		if host == pattern {
			return true
		}

		// Handle *.example.com
		if strings.HasPrefix(pattern, "*.") {
			suffix := strings.TrimPrefix(pattern, "*.")
			// Require a dot boundary: "sub.example.com" ends with ".example.com"
			// This excludes the root domain because it does not end with ".example.com"
			if strings.HasSuffix(host, "."+suffix) {
				return true
			}
			continue
		}

		// Handle example.* (rare, but included for completeness)
		if strings.HasSuffix(pattern, ".*") {
			prefix := strings.TrimSuffix(pattern, ".*")
			if strings.HasPrefix(host, prefix) {
				return true
			}
			continue
		}
	}

	return false
}

// ValidateHostPatterns validates a list of host patterns.
// Returns error if:
//   - patterns is empty
//   - any pattern is empty or whitespace-only
//   - any pattern contains invalid characters (spaces within the pattern)
//
// Note: This does NOT validate that patterns will actually match anything useful,
// only that they are syntactically valid for the matching algorithm.
func ValidateHostPatterns(patterns []string) error {
	if len(patterns) == 0 {
		return nil // Empty is valid (just means no patterns)
	}

	for i, pattern := range patterns {
		trimmed := strings.TrimSpace(pattern)
		if trimmed == "" {
			return &ValidationError{
				Field:   "hostPatterns",
				Index:   i,
				Message: "host pattern cannot be empty or whitespace-only",
			}
		}
		// Check for internal whitespace (spaces, tabs within the pattern itself)
		if strings.ContainsAny(trimmed, " \t\n\r") {
			return &ValidationError{
				Field:   "hostPatterns",
				Index:   i,
				Message: "host pattern cannot contain whitespace",
			}
		}
	}

	return nil
}

// ValidationError represents a validation error for a specific field/index.
type ValidationError struct {
	Field   string
	Index   int
	Message string
}

func (e *ValidationError) Error() string {
	// Use -1 as sentinel for "no index specified"
	if e.Index >= 0 {
		return e.Field + "[" + strconv.Itoa(e.Index) + "]: " + e.Message
	}
	return e.Field + ": " + e.Message
}
