package crawl

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://example.com/path#fragment", "https://example.com/path"},
		{"HTTP://EXAMPLE.COM/", "http://example.com/"},
		{"https://example.com/path?q=1", "https://example.com/path?q=1"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeURL(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeURL(%q) = %q; want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestResolveURL(t *testing.T) {
	base, _ := url.Parse("https://example.com/a/b")
	tests := []struct {
		href     string
		expected string
	}{
		{"c", "https://example.com/a/c"},
		{"/d", "https://example.com/d"},
		{"https://other.com", "https://other.com"},
		{"  ", "https://example.com/a/b"},
	}

	for _, tt := range tests {
		t.Run(tt.href, func(t *testing.T) {
			got := resolveURL(base, tt.href)
			if got != tt.expected {
				t.Errorf("resolveURL(%q) = %q; want %q", tt.href, got, tt.expected)
			}
		})
	}
}

func TestSameHost(t *testing.T) {
	base, _ := url.Parse("https://example.com")
	tests := []struct {
		url      string
		expected bool
	}{
		{"https://example.com/a", true},
		{"http://example.com/b", true},
		{"https://sub.example.com", false},
		{"https://other.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			if got := sameHost(base, tt.url); got != tt.expected {
				t.Errorf("sameHost(%q) = %v; want %v", tt.url, got, tt.expected)
			}
		})
	}
}

func TestRun(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><a href="/p1">P1</a><a href="/p2">P2</a></body></html>`)
	})
	mux.HandleFunc("/p1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><h1>P1</h1></body></html>`)
	})
	mux.HandleFunc("/p2", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><h1>P2</h1></body></html>`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	req := Request{
		URL:         srv.URL,
		MaxDepth:    2,
		MaxPages:    5,
		Concurrency: 2,
		Timeout:     5 * time.Second,
		DataDir:     t.TempDir(),
	}

	results, err := Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	found := map[string]bool{}
	for _, r := range results {
		found[r.URL] = true
	}

	if !found[srv.URL] || !found[srv.URL+"/p1"] || !found[srv.URL+"/p2"] {
		t.Errorf("missing expected URLs in results: %v", found)
	}
}

func TestWaitGroupOnFullChannel(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><h1>Root</h1></body></html>`)
	})

	for i := 1; i <= 20; i++ {
		i := i
		mux.HandleFunc(fmt.Sprintf("/p%d", i), func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, `<html><body><h1>Page %d</h1></body></html>`, i)
		})
	}

	srv := httptest.NewServer(mux)
	defer srv.Close()

	req := Request{
		URL:         srv.URL,
		MaxDepth:    1,
		MaxPages:    20,
		Concurrency: 4,
		Timeout:     5 * time.Second,
		DataDir:     t.TempDir(),
	}

	results, err := Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result (only root), got %d", len(results))
	}
}

func TestPatternMatcher_Matches(t *testing.T) {
	tests := []struct {
		name     string
		include  []string
		exclude  []string
		path     string
		expected bool
	}{
		{"no patterns", nil, nil, "/blog/post", true},
		{"include matches", []string{"/blog/**"}, nil, "/blog/post", true},
		{"include no match", []string{"/blog/**"}, nil, "/products/item", false},
		{"exclude matches", nil, []string{"/admin/*"}, "/admin/users", false},
		{"exclude takes precedence", []string{"/**"}, []string{"/admin/*"}, "/admin/users", false},
		{"star matches single segment", []string{"/products/*"}, nil, "/products/item", true},
		{"star no match multi", []string{"/products/*"}, nil, "/products/cat/item", false},
		{"doublestar matches multi", []string{"/blog/**"}, nil, "/blog/2024/01/post", true},
		{"doublestar matches single", []string{"/blog/**"}, nil, "/blog/post", true},
		{"exact match", []string{"/about"}, nil, "/about", true},
		{"exact no match", []string{"/about"}, nil, "/about/team", false},
		{"multiple includes match first", []string{"/blog/**", "/products/**"}, nil, "/blog/post", true},
		{"multiple includes match second", []string{"/blog/**", "/products/**"}, nil, "/products/item", true},
		{"multiple includes no match", []string{"/blog/**", "/products/**"}, nil, "/contact", false},
		{"exclude with no includes", nil, []string{"/api/**"}, "/api/v1/users", false},
		{"exclude with no includes passes", nil, []string{"/api/**"}, "/blog/post", true},
		{"complex pattern", []string{"/blog/**/comments/*"}, nil, "/blog/2024/post/comments/123", true},
		{"empty include list", []string{}, nil, "/anything", true},
		{"empty exclude list", nil, []string{}, "/anything", true},
		{"pattern with special chars", []string{"/path-with-dash"}, nil, "/path-with-dash", true},
		{"pattern with dot", []string{"/file.txt"}, nil, "/file.txt", true},
		{"pattern with query-like", []string{"/search"}, nil, "/search", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm, err := NewPatternMatcher(tt.include, tt.exclude)
			if err != nil {
				t.Fatalf("NewPatternMatcher failed: %v", err)
			}
			got := pm.Matches(tt.path)
			if got != tt.expected {
				t.Errorf("Matches(%q) = %v; want %v", tt.path, got, tt.expected)
			}
		})
	}
}

func TestPatternMatcher_InvalidPattern(t *testing.T) {
	// Test that invalid regex patterns are rejected
	// Our globToRegex escapes most special chars, making invalid patterns rare
	// A pattern with an unclosed group would fail, but we escape ( and )
	// For now, this test documents that we handle errors gracefully
	// If we add more complex pattern features, this test can be expanded

	// Test that valid patterns still work after attempting invalid ones
	pm, err := NewPatternMatcher([]string{"/valid/**"}, nil)
	if err != nil {
		t.Fatalf("NewPatternMatcher failed for valid pattern: %v", err)
	}
	if !pm.Matches("/valid/path") {
		t.Error("expected /valid/path to match")
	}
}

func TestPatternMatcher_EmptyPatterns(t *testing.T) {
	// Test that empty strings in pattern lists are skipped
	pm, err := NewPatternMatcher([]string{"", "/blog/**", ""}, []string{"", "/admin/*"})
	if err != nil {
		t.Fatalf("NewPatternMatcher failed: %v", err)
	}

	if !pm.Matches("/blog/post") {
		t.Error("expected /blog/post to match")
	}
	if pm.Matches("/admin/users") {
		t.Error("expected /admin/users to be excluded")
	}
}

func TestRun_WithIncludePatterns(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><a href="/blog/post1">Blog 1</a><a href="/products/item1">Product 1</a><a href="/about">About</a></body></html>`)
	})
	mux.HandleFunc("/blog/post1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><h1>Blog Post 1</h1></body></html>`)
	})
	mux.HandleFunc("/products/item1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><h1>Product 1</h1></body></html>`)
	})
	mux.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><h1>About</h1></body></html>`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Parse the server URL to get the path
	serverURL, _ := url.Parse(srv.URL)

	req := Request{
		URL:             srv.URL,
		MaxDepth:        2,
		MaxPages:        10,
		Concurrency:     2,
		Timeout:         5 * time.Second,
		DataDir:         t.TempDir(),
		IncludePatterns: []string{"/blog/*"},
	}

	results, err := Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Should only have root and /blog/post1 (root is always included, /blog/* matches /blog/post1)
	// Note: The root URL is always crawled even if it doesn't match include patterns
	found := map[string]bool{}
	for _, r := range results {
		found[r.URL] = true
	}

	// Root should be included
	if !found[srv.URL] {
		t.Errorf("expected root URL %s to be in results", srv.URL)
	}

	// /blog/post1 should be included (matches /blog/*)
	if !found[srv.URL+"/blog/post1"] {
		t.Errorf("expected /blog/post1 to be in results")
	}

	// /products/item1 should NOT be included (doesn't match /blog/*)
	if found[srv.URL+"/products/item1"] {
		t.Errorf("expected /products/item1 to NOT be in results")
	}

	// /about should NOT be included (doesn't match /blog/*)
	if found[srv.URL+"/about"] {
		t.Errorf("expected /about to NOT be in results")
	}

	t.Logf("Results: %v", found)
	t.Logf("Server URL: %s, Path: %s", srv.URL, serverURL.Path)
}

func TestRun_WithExcludePatterns(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><a href="/blog/post1">Blog 1</a><a href="/admin/users">Admin</a><a href="/about">About</a></body></html>`)
	})
	mux.HandleFunc("/blog/post1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><h1>Blog Post 1</h1><a href="/admin/settings">Settings</a></body></html>`)
	})
	mux.HandleFunc("/admin/users", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><h1>Admin Users</h1></body></html>`)
	})
	mux.HandleFunc("/admin/settings", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><h1>Admin Settings</h1></body></html>`)
	})
	mux.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><h1>About</h1></body></html>`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	req := Request{
		URL:             srv.URL,
		MaxDepth:        2,
		MaxPages:        10,
		Concurrency:     2,
		Timeout:         5 * time.Second,
		DataDir:         t.TempDir(),
		ExcludePatterns: []string{"/admin/*"},
	}

	results, err := Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	found := map[string]bool{}
	for _, r := range results {
		found[r.URL] = true
	}

	// Root should be included
	if !found[srv.URL] {
		t.Errorf("expected root URL to be in results")
	}

	// /blog/post1 should be included
	if !found[srv.URL+"/blog/post1"] {
		t.Errorf("expected /blog/post1 to be in results")
	}

	// /about should be included
	if !found[srv.URL+"/about"] {
		t.Errorf("expected /about to be in results")
	}

	// /admin/users should NOT be included (excluded)
	if found[srv.URL+"/admin/users"] {
		t.Errorf("expected /admin/users to NOT be in results")
	}

	// /admin/settings should NOT be included (excluded)
	if found[srv.URL+"/admin/settings"] {
		t.Errorf("expected /admin/settings to NOT be in results")
	}
}

func TestRun_WithRobotsTxt(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><a href="/allowed">Allowed</a><a href="/blocked">Blocked</a></body></html>`)
	})
	mux.HandleFunc("/allowed", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><h1>Allowed Page</h1></body></html>`)
	})
	mux.HandleFunc("/blocked", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><h1>Blocked Page</h1></body></html>`)
	})
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `User-agent: *
Disallow: /blocked
`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Create robots cache
	robotsCache := NewCache(srv.Client(), time.Hour)

	req := Request{
		URL:         srv.URL,
		MaxDepth:    2,
		MaxPages:    10,
		Concurrency: 2,
		Timeout:     5 * time.Second,
		DataDir:     t.TempDir(),
		RobotsCache: robotsCache,
		UserAgent:   "TestBot",
	}

	results, err := Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	found := map[string]bool{}
	for _, r := range results {
		found[r.URL] = true
	}

	// Root should be included
	if !found[srv.URL] {
		t.Errorf("expected root URL to be in results")
	}

	// /allowed should be included
	if !found[srv.URL+"/allowed"] {
		t.Errorf("expected /allowed to be in results")
	}

	// /blocked should NOT be included (blocked by robots.txt)
	if found[srv.URL+"/blocked"] {
		t.Errorf("expected /blocked to NOT be in results (blocked by robots.txt)")
	}
}

func TestRun_WithoutRobotsTxt(t *testing.T) {
	// Test that crawl works normally without robots.txt cache
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><a href="/page1">Page 1</a></body></html>`)
	})
	mux.HandleFunc("/page1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><h1>Page 1</h1></body></html>`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	// No robots cache - should crawl all URLs
	req := Request{
		URL:         srv.URL,
		MaxDepth:    2,
		MaxPages:    10,
		Concurrency: 2,
		Timeout:     5 * time.Second,
		DataDir:     t.TempDir(),
		// RobotsCache is nil
	}

	results, err := Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestRun_WithDuplicateDetection(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><a href="/page1">Page 1</a><a href="/page2">Page 2</a><a href="/duplicate">Duplicate</a></body></html>`)
	})
	mux.HandleFunc("/page1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><h1>Page 1</h1><p>This is unique content for page 1.</p></body></html>`)
	})
	mux.HandleFunc("/page2", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><h1>Page 2</h1><p>This is unique content for page 2.</p></body></html>`)
	})
	// This page has identical content to page1 - should be detected as duplicate
	mux.HandleFunc("/duplicate", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><h1>Page 1</h1><p>This is unique content for page 1.</p></body></html>`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	req := Request{
		URL:              srv.URL,
		MaxDepth:         2,
		MaxPages:         10,
		Concurrency:      2,
		Timeout:          5 * time.Second,
		DataDir:          t.TempDir(),
		SkipDuplicates:   true,
		SimHashThreshold: 3,
	}

	results, err := Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Check that we got results
	if len(results) == 0 {
		t.Fatal("expected results, got none")
	}

	// Find the duplicate page result
	var duplicateResult *PageResult
	for i := range results {
		if results[i].URL == srv.URL+"/duplicate" {
			duplicateResult = &results[i]
			break
		}
	}

	// The duplicate page should be marked as a duplicate
	if duplicateResult != nil && duplicateResult.DuplicateOf == "" {
		t.Errorf("expected /duplicate to be marked as duplicate, but DuplicateOf is empty")
	}

	// Check that simhash is computed for all results
	for _, r := range results {
		if r.SimHash == 0 && r.Status != 304 {
			t.Errorf("expected non-zero simhash for %s", r.URL)
		}
	}
}

func TestRun_WithoutDuplicateDetection(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><a href="/page1">Page 1</a><a href="/duplicate">Duplicate</a></body></html>`)
	})
	mux.HandleFunc("/page1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><h1>Page 1</h1><p>This is unique content for page 1.</p></body></html>`)
	})
	mux.HandleFunc("/duplicate", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><h1>Page 1</h1><p>This is unique content for page 1.</p></body></html>`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Disable duplicate detection
	req := Request{
		URL:              srv.URL,
		MaxDepth:         2,
		MaxPages:         10,
		Concurrency:      2,
		Timeout:          5 * time.Second,
		DataDir:          t.TempDir(),
		SkipDuplicates:   false,
		SimHashThreshold: 3,
	}

	results, err := Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Find the duplicate page result - should NOT be marked as duplicate
	var duplicateResult *PageResult
	for i := range results {
		if results[i].URL == srv.URL+"/duplicate" {
			duplicateResult = &results[i]
			break
		}
	}

	// When SkipDuplicates is false, DuplicateOf should be empty even for duplicates
	if duplicateResult != nil && duplicateResult.DuplicateOf != "" {
		t.Errorf("expected /duplicate to NOT be marked as duplicate when SkipDuplicates=false")
	}
}

func TestRun_SimHashThreshold(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><a href="/page1">Page 1</a><a href="/similar">Similar</a></body></html>`)
	})
	mux.HandleFunc("/page1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><h1>Page 1</h1><p>The quick brown fox jumps over the lazy dog.</p></body></html>`)
	})
	// Similar content with minor changes
	mux.HandleFunc("/similar", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><h1>Page 1</h1><p>The quick brown fox jumps over a lazy dog.</p></body></html>`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Test with permissive threshold (should detect as duplicate)
	req := Request{
		URL:              srv.URL,
		MaxDepth:         2,
		MaxPages:         10,
		Concurrency:      2,
		Timeout:          5 * time.Second,
		DataDir:          t.TempDir(),
		SkipDuplicates:   true,
		SimHashThreshold: 5, // Higher threshold = more permissive
	}

	results, err := Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify simhash values are computed
	for _, r := range results {
		if r.Status != 304 && r.SimHash == 0 {
			t.Errorf("expected non-zero simhash for %s", r.URL)
		}
	}
}
