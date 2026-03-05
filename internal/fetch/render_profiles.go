// Package fetch provides HTTP and headless browser content fetching capabilities.
// It handles request routing, rate limiting, retry logic, and render profiles.
// It does NOT handle content extraction or parsing.
package fetch

import (
	"path/filepath"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/hostmatch"
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
// This is a thin wrapper around hostmatch.HostMatchesAnyPattern for backward compatibility.
// Supported patterns:
//   - exact match: "example.com"
//   - wildcard subdomain: "*.example.com"
//   - wildcard prefix: "example.*" (not recommended for security, but supported for completeness)
func HostMatchesAnyPattern(host string, patterns []string) bool {
	return hostmatch.HostMatchesAnyPattern(host, patterns)
}
