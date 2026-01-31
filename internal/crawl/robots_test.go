// Package crawl provides unit tests for robots.txt parsing and caching.
// Tests cover robots.txt parsing, rule matching, user-agent selection, crawl-delay extraction,
// caching with TTL, and fail-open behavior on errors.
// Does NOT test the actual crawl logic (crawl_test.go covers that).
package crawl

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParseRobotsTxt_Basic(t *testing.T) {
	content := `
User-agent: *
Disallow: /admin/
Disallow: /private
Allow: /admin/public

User-agent: BadBot
Disallow: /
`

	ruleset, err := parseRobotsTxt(strings.NewReader(content))
	if err != nil {
		t.Fatalf("parseRobotsTxt failed: %v", err)
	}

	if len(ruleset.groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(ruleset.groups))
	}

	// Test wildcard user-agent
	wildcardRules := ruleset.ForUserAgent("AnyBot")
	if len(wildcardRules.Rules) != 3 {
		t.Errorf("expected 3 rules for wildcard, got %d", len(wildcardRules.Rules))
	}

	// Test specific user-agent
	badBotRules := ruleset.ForUserAgent("BadBot")
	if len(badBotRules.Rules) != 1 {
		t.Errorf("expected 1 rule for BadBot, got %d", len(badBotRules.Rules))
	}
	if badBotRules.Rules[0].Pattern != "/" {
		t.Errorf("expected BadBot to be blocked from /, got %s", badBotRules.Rules[0].Pattern)
	}
}

func TestParseRobotsTxt_CrawlDelay(t *testing.T) {
	content := `
User-agent: *
Crawl-delay: 5
Disallow: /admin/
`

	ruleset, err := parseRobotsTxt(strings.NewReader(content))
	if err != nil {
		t.Fatalf("parseRobotsTxt failed: %v", err)
	}

	specificRules := ruleset.ForUserAgent("TestBot")
	if specificRules.CrawlDelay != 5*time.Second {
		t.Errorf("expected crawl delay of 5s, got %v", specificRules.CrawlDelay)
	}
}

func TestParseRobotsTxt_EmptyAndComments(t *testing.T) {
	content := `
# This is a comment
User-agent: *

# Another comment
Disallow: /admin/

# Empty lines above and below

Disallow: /private
`

	ruleset, err := parseRobotsTxt(strings.NewReader(content))
	if err != nil {
		t.Fatalf("parseRobotsTxt failed: %v", err)
	}

	specificRules := ruleset.ForUserAgent("TestBot")
	if len(specificRules.Rules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(specificRules.Rules))
	}
}

func TestMatchRule(t *testing.T) {
	tests := []struct {
		path     string
		pattern  string
		expected bool
	}{
		// Basic prefix matching
		{"/admin/users", "/admin/", true},
		{"/admin", "/admin/", false}, // /admin doesn't match /admin/ prefix
		{"/admin", "/admin", true},
		{"/admin/", "/admin", true},

		// Exact matching with end anchor
		{"/admin", "/admin$", true},
		{"/admin/users", "/admin$", false},

		// Wildcard matching
		{"/admin/users", "/admin/*", true},
		{"/admin/users/edit", "/admin/*", true},
		{"/other/users", "/admin/*", false},

		// Wildcard with end anchor - * matches any sequence including /
		{"/file.txt", "/*.txt$", true},
		{"/dir/file.txt", "/*.txt$", true},   // * matches across /
		{"/dir/file.txt", "/*/*.txt$", true}, // Two * patterns can match across /

		// Empty pattern matches all
		{"/anything", "", true},

		// Complex patterns - * matches one path segment
		{"/blog/2024/post", "/blog/*/post", true},
		{"/blog/post", "/blog/*/post", false}, // * needs to match something
	}

	for _, tt := range tests {
		t.Run(tt.path+"_"+tt.pattern, func(t *testing.T) {
			got := matchRule(tt.path, tt.pattern)
			if got != tt.expected {
				t.Errorf("matchRule(%q, %q) = %v; want %v", tt.path, tt.pattern, got, tt.expected)
			}
		})
	}
}

func TestRulesetIsAllowed(t *testing.T) {
	content := `
User-agent: *
Disallow: /admin/
Disallow: /private
Allow: /admin/public
Disallow: /secret.txt$
Disallow: /*.pdf$
`

	ruleset, err := parseRobotsTxt(strings.NewReader(content))
	if err != nil {
		t.Fatalf("parseRobotsTxt failed: %v", err)
	}

	specificRules := ruleset.ForUserAgent("TestBot")

	tests := []struct {
		path     string
		expected bool
	}{
		{"/", true},
		{"/public", true},
		{"/admin/", false},
		{"/admin/users", false},
		{"/admin/public", true}, // Allow overrides earlier Disallow
		{"/private", false},
		{"/private/nested", false},
		{"/secret.txt", false},       // Exact match with $
		{"/secret.txt.backup", true}, // Doesn't match with $
		{"/doc.pdf", false},          // Wildcard with $
		{"/docs/doc.pdf", false},     // Wildcard matches any path segment
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := specificRules.IsAllowed(tt.path)
			if got != tt.expected {
				t.Errorf("IsAllowed(%q) = %v; want %v", tt.path, got, tt.expected)
			}
		})
	}
}

func TestForUserAgent_Matching(t *testing.T) {
	content := `
User-agent: Googlebot
Disallow: /no-google/

User-agent: Googlebot-News
Disallow: /no-news/

User-agent: *
Disallow: /no-all/
`

	ruleset, err := parseRobotsTxt(strings.NewReader(content))
	if err != nil {
		t.Fatalf("parseRobotsTxt failed: %v", err)
	}

	// Exact match
	googleRules := ruleset.ForUserAgent("Googlebot")
	if len(googleRules.Rules) != 1 || googleRules.Rules[0].Pattern != "/no-google/" {
		t.Errorf("Googlebot should match Googlebot group")
	}

	// More specific match (Googlebot-News should match its own group, not Googlebot)
	newsRules := ruleset.ForUserAgent("Googlebot-News")
	if len(newsRules.Rules) != 1 || newsRules.Rules[0].Pattern != "/no-news/" {
		t.Errorf("Googlebot-News should match Googlebot-News group, got: %v", newsRules.Rules)
	}

	// Wildcard fallback
	otherRules := ruleset.ForUserAgent("SomeOtherBot")
	if len(otherRules.Rules) != 1 || otherRules.Rules[0].Pattern != "/no-all/" {
		t.Errorf("SomeOtherBot should match wildcard group")
	}
}

func TestCache_IsAllowed(t *testing.T) {
	// Create a test server
	mux := http.NewServeMux()
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `User-agent: *
Disallow: /admin/
Disallow: /private
`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Create cache with test server URL
	cache := NewCache(srv.Client(), time.Hour)

	// Parse server URL to get host
	serverURL := srv.URL
	host := strings.TrimPrefix(serverURL, "http://")

	// Test allowed URL
	allowed, err := cache.IsAllowed(serverURL+"/public", "TestBot")
	if err != nil {
		t.Errorf("IsAllowed failed: %v", err)
	}
	if !allowed {
		t.Errorf("/public should be allowed")
	}

	// Test disallowed URL
	allowed, err = cache.IsAllowed(serverURL+"/admin/", "TestBot")
	if err != nil {
		t.Errorf("IsAllowed failed: %v", err)
	}
	if allowed {
		t.Errorf("/admin/ should be disallowed")
	}

	// Test that cache is used (second request should not hit server)
	allowed, err = cache.IsAllowed(serverURL+"/private", "TestBot")
	if err != nil {
		t.Errorf("IsAllowed failed: %v", err)
	}
	if allowed {
		t.Errorf("/private should be disallowed")
	}

	// Verify cache has entry
	cache.mu.RLock()
	_, hasEntry := cache.data[host]
	cache.mu.RUnlock()
	if !hasEntry {
		t.Errorf("cache should have entry for host")
	}
}

func TestCache_404MeansAllowAll(t *testing.T) {
	// Create a test server that returns 404 for robots.txt
	mux := http.NewServeMux()
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	cache := NewCache(srv.Client(), time.Hour)

	// All URLs should be allowed when robots.txt returns 404
	allowed, err := cache.IsAllowed(srv.URL+"/anything", "TestBot")
	if err != nil {
		t.Errorf("IsAllowed failed: %v", err)
	}
	if !allowed {
		t.Errorf("all URLs should be allowed when robots.txt is 404")
	}
}

func TestCache_TTLExpiration(t *testing.T) {
	requestCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		fmt.Fprint(w, `User-agent: *
Disallow: /admin/
`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Use very short TTL
	cache := NewCache(srv.Client(), 10*time.Millisecond)

	// First request
	cache.IsAllowed(srv.URL+"/test", "TestBot")
	if requestCount != 1 {
		t.Errorf("expected 1 request, got %d", requestCount)
	}

	// Second request (should use cache)
	cache.IsAllowed(srv.URL+"/test2", "TestBot")
	if requestCount != 1 {
		t.Errorf("expected 1 request (cached), got %d", requestCount)
	}

	// Wait for TTL to expire
	time.Sleep(50 * time.Millisecond)

	// Third request (should fetch again)
	cache.IsAllowed(srv.URL+"/test3", "TestBot")
	if requestCount != 2 {
		t.Errorf("expected 2 requests after TTL, got %d", requestCount)
	}
}

func TestCache_GetCrawlDelay(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `User-agent: *
Crawl-delay: 3
Disallow: /admin/

User-agent: FastBot
Crawl-delay: 1
Disallow: /
`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	cache := NewCache(srv.Client(), time.Hour)

	// Parse server URL to get host
	host := strings.TrimPrefix(srv.URL, "http://")

	// Test default crawl delay
	delay := cache.GetCrawlDelay(host, "SomeBot")
	if delay != 3*time.Second {
		t.Errorf("expected crawl delay of 3s, got %v", delay)
	}

	// Test specific bot crawl delay
	delay = cache.GetCrawlDelay(host, "FastBot")
	if delay != 1*time.Second {
		t.Errorf("expected crawl delay of 1s for FastBot, got %v", delay)
	}
}

func TestMatchWildcard(t *testing.T) {
	tests := []struct {
		path      string
		pattern   string
		endAnchor bool
		expected  bool
	}{
		// Basic wildcard
		{"/admin/users", "/admin/*", false, true},
		{"/admin", "/admin/*", false, false}, // * needs something to match
		{"/other/users", "/admin/*", false, false},

		// Multiple wildcards - * matches any sequence
		{"/a/b/c", "/*/*/*", false, true},
		{"/a/b", "/*/*/*", false, false}, // /a/b has 2 segments, pattern has 3

		// Wildcard with end anchor - * matches any sequence including /
		{"/file.txt", "/*.txt", true, true},
		{"/file.txt", "/*.txt", false, true},
		{"/dir/file.txt", "/*.txt", true, true},  // * matches across /
		{"/dir/file.txt", "/*.txt", false, true}, // * matches across / even without anchor

		// Empty pattern
		{"/anything", "", false, true},
		{"/anything", "", true, false},

		// Only wildcard
		{"/anything", "*", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.path+"_"+tt.pattern, func(t *testing.T) {
			got := matchWildcard(tt.path, tt.pattern, tt.endAnchor)
			if got != tt.expected {
				t.Errorf("matchWildcard(%q, %q, %v) = %v; want %v",
					tt.path, tt.pattern, tt.endAnchor, got, tt.expected)
			}
		})
	}
}

func TestCache_FailOpen(t *testing.T) {
	// Create a test server that returns 500 error
	mux := http.NewServeMux()
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	cache := NewCache(srv.Client(), time.Hour)

	// Should fail open (allow) on server error
	allowed, err := cache.IsAllowed(srv.URL+"/anything", "TestBot")
	if err != nil {
		t.Errorf("IsAllowed should not return error on 500: %v", err)
	}
	if !allowed {
		t.Errorf("should fail open and allow URL on server error")
	}
}
