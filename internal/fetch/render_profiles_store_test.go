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

func TestRenderProfileStore_GetRateLimitsForURL(t *testing.T) {
	tmp := t.TempDir()
	jsonContent := `{
		"profiles": [
			{
				"name": "slow-site",
				"hostPatterns": ["slow.example.com"],
				"rateLimitQPS": 1,
				"rateLimitBurst": 2
			},
			{
				"name": "fast-site",
				"hostPatterns": ["fast.example.com"],
				"rateLimitQPS": 100,
				"rateLimitBurst": 100
			},
			{
				"name": "no-rate-limit",
				"hostPatterns": ["default.example.com"]
			}
		]
	}`
	path := filepath.Join(tmp, "render_profiles.json")
	if err := os.WriteFile(path, []byte(jsonContent), 0644); err != nil {
		t.Fatal(err)
	}

	store := NewRenderProfileStore(tmp)

	// Test slow site with custom rate limits
	qps, burst := store.GetRateLimitsForURL("https://slow.example.com/page")
	if qps != 1 {
		t.Errorf("expected QPS 1 for slow site, got %d", qps)
	}
	if burst != 2 {
		t.Errorf("expected burst 2 for slow site, got %d", burst)
	}

	// Test fast site with custom rate limits
	qps, burst = store.GetRateLimitsForURL("https://fast.example.com/page")
	if qps != 100 {
		t.Errorf("expected QPS 100 for fast site, got %d", qps)
	}
	if burst != 100 {
		t.Errorf("expected burst 100 for fast site, got %d", burst)
	}

	// Test site without rate limits (should return 0, 0)
	qps, burst = store.GetRateLimitsForURL("https://default.example.com/page")
	if qps != 0 {
		t.Errorf("expected QPS 0 for default site, got %d", qps)
	}
	if burst != 0 {
		t.Errorf("expected burst 0 for default site, got %d", burst)
	}

	// Test non-matching URL (should return 0, 0)
	qps, burst = store.GetRateLimitsForURL("https://unknown.com/page")
	if qps != 0 {
		t.Errorf("expected QPS 0 for unknown site, got %d", qps)
	}
	if burst != 0 {
		t.Errorf("expected burst 0 for unknown site, got %d", burst)
	}
}
