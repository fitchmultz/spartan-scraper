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
