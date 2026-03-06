// Package webassets verifies the tracked web shell entry assets.
//
// These tests ensure the HTML entrypoint references a real favicon asset so the
// browser does not emit a load-time 404 on fresh page visits.
package webassets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppShellLinksTrackedFavicon(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	indexPath := filepath.Join(root, "web", "index.html")
	faviconPath := filepath.Join(root, "web", "public", "favicon.svg")

	html, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read index.html: %v", err)
	}

	if !strings.Contains(string(html), `<link rel="icon" type="image/svg+xml" href="/favicon.svg" />`) {
		t.Fatalf("index.html does not reference /favicon.svg")
	}

	if _, err := os.Stat(faviconPath); err != nil {
		t.Fatalf("favicon asset missing: %v", err)
	}
}
