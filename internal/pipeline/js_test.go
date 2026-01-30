// Package pipeline provides tests for JavaScript registry loading and matching.
// Tests cover JSRegistry loading from JSON (valid, invalid, missing, empty), URL pattern matching
// with wildcards, host matching logic, script selection by engine, and data directory defaults.
// Does NOT test actual JavaScript execution in headless browsers.
package pipeline

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadJSRegistry_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	registry, err := LoadJSRegistry(tmpDir)
	if err != nil {
		t.Fatalf("expected no error for non-existent file, got %v", err)
	}
	if registry == nil {
		t.Fatal("expected non-nil registry")
	}
	if len(registry.Scripts) != 0 {
		t.Errorf("expected empty scripts, got %d", len(registry.Scripts))
	}
}

func TestLoadJSRegistry_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, jsRegistryFile)
	if err := os.WriteFile(registryPath, []byte("invalid json"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	_, err := LoadJSRegistry(tmpDir)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestLoadJSRegistry_ValidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	jsonContent := `{
		"scripts": [
			{
				"name": "test-script",
				"hostPatterns": ["example.com"],
				"engine": "chromedp",
				"preNav": "console.log('before')",
				"postNav": "console.log('after')",
				"selectors": [".class1", "#id1"]
			}
		]
	}`
	registryPath := filepath.Join(tmpDir, jsRegistryFile)
	if err := os.WriteFile(registryPath, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	registry, err := LoadJSRegistry(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(registry.Scripts) != 1 {
		t.Fatalf("expected 1 script, got %d", len(registry.Scripts))
	}
	script := registry.Scripts[0]
	if script.Name != "test-script" {
		t.Errorf("expected name 'test-script', got '%s'", script.Name)
	}
	if len(script.HostPatterns) != 1 || script.HostPatterns[0] != "example.com" {
		t.Errorf("expected hostPatterns ['example.com'], got %v", script.HostPatterns)
	}
	if script.Engine != "chromedp" {
		t.Errorf("expected engine 'chromedp', got '%s'", script.Engine)
	}
	if script.PreNav != "console.log('before')" {
		t.Errorf("expected preNav 'console.log('before')', got '%s'", script.PreNav)
	}
	if script.PostNav != "console.log('after')" {
		t.Errorf("expected postNav 'console.log('after')', got '%s'", script.PostNav)
	}
	if len(script.Selectors) != 2 || script.Selectors[0] != ".class1" || script.Selectors[1] != "#id1" {
		t.Errorf("expected selectors ['.class1', '#id1'], got %v", script.Selectors)
	}
}

func TestLoadJSRegistry_EmptyJSON(t *testing.T) {
	tmpDir := t.TempDir()
	jsonContent := `{}`
	registryPath := filepath.Join(tmpDir, jsRegistryFile)
	if err := os.WriteFile(registryPath, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	registry, err := LoadJSRegistry(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if registry == nil {
		t.Fatal("expected non-nil registry")
	}
	if len(registry.Scripts) != 0 {
		t.Errorf("expected empty scripts, got %d", len(registry.Scripts))
	}
}

func TestJSRegistryMatch_NilRegistry(t *testing.T) {
	var registry *JSRegistry
	matches := registry.Match("https://example.com")
	if matches != nil {
		t.Errorf("expected nil for nil registry, got %v", matches)
	}
}

func TestJSRegistryMatch_EmptyScripts(t *testing.T) {
	registry := &JSRegistry{Scripts: []JSTargetScript{}}
	matches := registry.Match("https://example.com")
	if matches != nil {
		t.Errorf("expected nil for empty scripts, got %v", matches)
	}
}

func TestJSRegistryMatch_InvalidURL(t *testing.T) {
	registry := &JSRegistry{
		Scripts: []JSTargetScript{
			{Name: "test", HostPatterns: []string{"example.com"}},
		},
	}
	tests := []struct {
		name string
		url  string
	}{
		{"empty url", ""},
		{"whitespace url", "   "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := registry.Match(tt.url)
			if matches != nil {
				t.Errorf("expected nil for invalid url '%s', got %v", tt.url, matches)
			}
		})
	}
}

func TestJSRegistryMatch_MatchingPatterns(t *testing.T) {
	registry := &JSRegistry{
		Scripts: []JSTargetScript{
			{Name: "script1", HostPatterns: []string{"example.com"}},
			{Name: "script2", HostPatterns: []string{"other.com"}},
			{Name: "script3", HostPatterns: []string{"*.sub.example.com"}},
		},
	}
	matches := registry.Match("https://example.com/path")
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].Name != "script1" {
		t.Errorf("expected script1, got %s", matches[0].Name)
	}
}

func TestJSRegistryMatch_NoMatches(t *testing.T) {
	registry := &JSRegistry{
		Scripts: []JSTargetScript{
			{Name: "script1", HostPatterns: []string{"example.com"}},
			{Name: "script2", HostPatterns: []string{"other.com"}},
		},
	}
	matches := registry.Match("https://unmatched.com/path")
	if len(matches) != 0 {
		t.Errorf("expected empty slice for no matches, got %v", matches)
	}
}

func TestSelectScripts_Empty(t *testing.T) {
	pre, post, selectors := SelectScripts([]JSTargetScript{}, "chromedp")
	if pre != nil || post != nil || selectors != nil {
		t.Errorf("expected nil, nil, nil, got %v, %v, %v", pre, post, selectors)
	}
}

func TestSelectScripts_FilterByEngine(t *testing.T) {
	scripts := []JSTargetScript{
		{Name: "script1", Engine: "chromedp", PreNav: "pre1"},
		{Name: "script2", Engine: "playwright", PreNav: "pre2"},
		{Name: "script3", Engine: "chromedp", PostNav: "post3"},
	}
	pre, post, _ := SelectScripts(scripts, "chromedp")
	if len(pre) != 1 || pre[0] != "pre1" {
		t.Errorf("expected pre ['pre1'], got %v", pre)
	}
	if len(post) != 1 || post[0] != "post3" {
		t.Errorf("expected post ['post3'], got %v", post)
	}
}

func TestSelectScripts_NoEngineFilter(t *testing.T) {
	scripts := []JSTargetScript{
		{Name: "script1", Engine: "", PreNav: "pre1"},
		{Name: "script2", Engine: "", PreNav: "pre2"},
		{Name: "script3", Engine: "", PostNav: "post3"},
	}
	pre, post, _ := SelectScripts(scripts, "")
	if len(pre) != 2 || pre[0] != "pre1" || pre[1] != "pre2" {
		t.Errorf("expected pre ['pre1', 'pre2'], got %v", pre)
	}
	if len(post) != 1 || post[0] != "post3" {
		t.Errorf("expected post ['post3'], got %v", post)
	}
}

func TestSelectScripts_CollectFields(t *testing.T) {
	scripts := []JSTargetScript{
		{Name: "script1", PreNav: "pre1", PostNav: "post1", Selectors: []string{".sel1", "#sel2"}},
		{Name: "script2", PreNav: "pre2", Selectors: []string{".sel3"}},
		{Name: "script3", PostNav: "post3"},
	}
	pre, post, selectors := SelectScripts(scripts, "")
	if len(pre) != 2 || pre[0] != "pre1" || pre[1] != "pre2" {
		t.Errorf("expected pre ['pre1', 'pre2'], got %v", pre)
	}
	if len(post) != 2 || post[0] != "post1" || post[1] != "post3" {
		t.Errorf("expected post ['post1', 'post3'], got %v", post)
	}
	if len(selectors) != 3 || selectors[0] != ".sel1" || selectors[1] != "#sel2" || selectors[2] != ".sel3" {
		t.Errorf("expected selectors ['.sel1', '#sel2', '.sel3'], got %v", selectors)
	}
}

func TestSelectScripts_TrimsWhitespace(t *testing.T) {
	scripts := []JSTargetScript{
		{Name: "script1", PreNav: "   pre1   ", PostNav: "  ", Selectors: []string{"  .sel1  "}},
		{Name: "script2", PreNav: "   ", PostNav: "post2   "},
	}
	pre, post, selectors := SelectScripts(scripts, "")
	if len(pre) != 1 || pre[0] != "   pre1   " {
		t.Errorf("expected pre ['   pre1   '], got %v", pre)
	}
	if len(post) != 1 || post[0] != "post2   " {
		t.Errorf("expected post ['post2   '], got %v", post)
	}
	if len(selectors) != 1 {
		t.Errorf("expected 1 selector, got %d: %v", len(selectors), selectors)
	}
	// Verify selector is present (exact whitespace match)
	if selectors[0] != "  .sel1  " {
		t.Errorf("expected selector '  .sel1  ', got %q", selectors[0])
	}
}

func TestHostMatchesAnyPattern_EmptyPatterns(t *testing.T) {
	matches := hostMatchesAnyPattern("example.com", []string{})
	if matches {
		t.Error("expected false for empty patterns")
	}
}

func TestHostMatchesAnyPattern_ExactMatch(t *testing.T) {
	matches := hostMatchesAnyPattern("example.com", []string{"example.com", "other.com"})
	if !matches {
		t.Error("expected true for exact match")
	}
	matches = hostMatchesAnyPattern("example.com", []string{"other.com", "test.com"})
	if matches {
		t.Error("expected false for no exact match")
	}
}

func TestHostMatchesAnyPattern_WildcardPrefix(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		patterns []string
		want     bool
	}{
		{"subdomain matches", "sub.example.com", []string{"*.example.com"}, true},
		{"root does not match", "example.com", []string{"*.example.com"}, false},
		{"multi-level subdomain matches", "sub.sub.example.com", []string{"*.example.com"}, true},
		{"non-matching subdomain", "sub.other.com", []string{"*.example.com"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hostMatchesAnyPattern(tt.host, tt.patterns); got != tt.want {
				t.Errorf("hostMatchesAnyPattern(%q, %v) = %v, want %v", tt.host, tt.patterns, got, tt.want)
			}
		})
	}
}

func TestHostMatchesAnyPattern_WildcardSuffix(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		patterns []string
		want     bool
	}{
		{"com matches", "example.com", []string{"example.*"}, true},
		{"org matches", "example.org", []string{"example.*"}, true},
		{"different prefix does not match", "other.com", []string{"example.*"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hostMatchesAnyPattern(tt.host, tt.patterns); got != tt.want {
				t.Errorf("hostMatchesAnyPattern(%q, %v) = %v, want %v", tt.host, tt.patterns, got, tt.want)
			}
		})
	}
}

func TestHostMatchesAnyPattern_CaseInsensitive(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		patterns []string
		want     bool
	}{
		{"uppercase host", "EXAMPLE.COM", []string{"example.com"}, true},
		{"uppercase pattern", "example.com", []string{"EXAMPLE.COM"}, true},
		{"mixed case", "ExAmPlE.CoM", []string{"eXaMpLe.cOm"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hostMatchesAnyPattern(tt.host, tt.patterns); got != tt.want {
				t.Errorf("hostMatchesAnyPattern(%q, %v) = %v, want %v", tt.host, tt.patterns, got, tt.want)
			}
		})
	}
}

func TestHostMatchesAnyPattern_TrimsWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		patterns []string
		want     bool
	}{
		{"whitespace host", "  example.com  ", []string{"example.com"}, true},
		{"whitespace pattern", "example.com", []string{"  example.com  "}, true},
		{"whitespace both", "  example.com  ", []string{"  example.com  "}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hostMatchesAnyPattern(tt.host, tt.patterns); got != tt.want {
				t.Errorf("hostMatchesAnyPattern(%q, %v) = %v, want %v", tt.host, tt.patterns, got, tt.want)
			}
		})
	}
}

func TestDefaultDataDir_Empty(t *testing.T) {
	result := defaultDataDir("")
	if result != ".data" {
		t.Errorf("expected '.data', got '%s'", result)
	}
}

func TestDefaultDataDir_Whitespace(t *testing.T) {
	result := defaultDataDir("   ")
	if result != ".data" {
		t.Errorf("expected '.data', got '%s'", result)
	}
}

func TestDefaultDataDir_CustomPath(t *testing.T) {
	result := defaultDataDir("/custom/path")
	if result != "/custom/path" {
		t.Errorf("expected '/custom/path', got '%s'", result)
	}
	result = defaultDataDir("  /custom/path  ")
	if result != "/custom/path" {
		t.Errorf("expected '/custom/path', got '%s'", result)
	}
}
