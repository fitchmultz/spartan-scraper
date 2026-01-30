// Package fetch provides tests for the render profile store.
// Tests cover host pattern matching and profile loading from JSON configuration.
// Does NOT test file system errors or concurrent store access.
package fetch

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHostMatchesAnyPattern(t *testing.T) {
	tests := []struct {
		host     string
		patterns []string
		match    bool
	}{
		{"example.com", []string{"example.com"}, true},
		{"example.com", []string{"other.com", "example.com"}, true},
		{"sub.example.com", []string{"*.example.com"}, true},
		{"example.com", []string{"*.example.com"}, false}, // * requires dot prefix usually implies subdomain
		{"sub.example.com", []string{"example.com"}, false},
		{"example.org", []string{"*.example.com"}, false},
		{"api.test.co.uk", []string{"*.test.co.uk"}, true},
	}

	for _, tt := range tests {
		if got := HostMatchesAnyPattern(tt.host, tt.patterns); got != tt.match {
			t.Errorf("HostMatchesAnyPattern(%q, %v) = %v, want %v", tt.host, tt.patterns, got, tt.match)
		}
	}
}

func TestRenderProfileStore(t *testing.T) {
	tmp := t.TempDir()
	jsonContent := `{
		"profiles": [
			{
				"name": "heavy-site",
				"hostPatterns": ["*.heavy.com"],
				"forceEngine": "chromedp"
			},
			{
				"name": "specific-page",
				"hostPatterns": ["example.com"],
				"preferHeadless": true
			}
		]
	}`
	path := filepath.Join(tmp, "render_profiles.json")
	if err := os.WriteFile(path, []byte(jsonContent), 0644); err != nil {
		t.Fatal(err)
	}

	store := NewRenderProfileStore(tmp)

	// Match wildcard
	prof, found, err := store.MatchURL("https://www.heavy.com/foo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Fatal("expected match for heavy.com")
	}
	if prof.Name != "heavy-site" {
		t.Errorf("expected heavy-site, got %s", prof.Name)
	}
	if prof.ForceEngine != RenderEngineChromedp {
		t.Errorf("expected chromedp, got %s", prof.ForceEngine)
	}

	// Match exact
	prof, found, err = store.MatchURL("http://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Fatal("expected match for example.com")
	}
	if !prof.PreferHeadless {
		t.Error("expected PreferHeadless true")
	}

	// No match
	_, found, _ = store.MatchURL("https://other.com")
	if found {
		t.Error("expected no match for other.com")
	}
}
