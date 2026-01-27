// Package fetch provides HTTP and headless browser content fetching capabilities.
// It handles request routing, rate limiting, retry logic, and render profiles.
// It does NOT handle content extraction or parsing.
package fetch

import (
	"path/filepath"
	"strings"
)

// RenderProfilesPath returns the path to the render profiles JSON file.
func RenderProfilesPath(dataDir string) string {
	base := strings.TrimSpace(dataDir)
	if base == "" {
		base = ".data"
	}
	return filepath.Join(base, "render_profiles.json")
}

// HostMatchesAnyPattern checks if a host matches any of the provided glob-style patterns.
// Supported patterns:
//   - exact match: "example.com"
//   - wildcard subdomain: "*.example.com"
//   - wildcard prefix: "example.*" (not recommended for security, but supported for completeness)
func HostMatchesAnyPattern(host string, patterns []string) bool {
	if host == "" || len(patterns) == 0 {
		return false
	}

	host = strings.ToLower(strings.TrimSpace(host))

	for _, pattern := range patterns {
		if strings.TrimSpace(pattern) == "" {
			continue
		}
		pattern = strings.ToLower(strings.TrimSpace(pattern))

		if host == pattern {
			return true
		}

		// Handle *.example.com
		if strings.HasPrefix(pattern, "*.") {
			suffix := strings.TrimPrefix(pattern, "*.")
			// Require a dot boundary: "sub.example.com" ends with ".example.com".
			// This also excludes the root domain because it does not end with ".example.com".
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
