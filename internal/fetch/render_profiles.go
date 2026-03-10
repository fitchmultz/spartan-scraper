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
