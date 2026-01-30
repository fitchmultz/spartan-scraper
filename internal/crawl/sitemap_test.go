// Package crawl provides sitemap.xml parsing functionality.
// This file contains tests for the sitemap parser.
package crawl

import (
	"bytes"
	"compress/gzip"
	"context"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/fetch"
)

// mockFetcher is a test fetcher that returns predefined responses.
type mockFetcher struct {
	responses map[string]fetch.Result
}

func (m *mockFetcher) Fetch(ctx context.Context, req fetch.Request) (fetch.Result, error) {
	if res, ok := m.responses[req.URL]; ok {
		return res, nil
	}
	return fetch.Result{Status: 404}, nil
}

func TestParseSitemap_URLSet(t *testing.T) {
	sitemapXML := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url>
		<loc>https://example.com/page1</loc>
	</url>
	<url>
		<loc>https://example.com/page2</loc>
	</url>
	<url>
		<loc>https://example.com/page3</loc>
	</url>
</urlset>`

	fetcher := &mockFetcher{
		responses: map[string]fetch.Result{
			"https://example.com/sitemap.xml": {
				URL:    "https://example.com/sitemap.xml",
				Status: 200,
				HTML:   sitemapXML,
			},
		},
	}

	parser := NewSitemapParser(fetcher)
	urls, err := parser.ParseSitemap(context.Background(), "https://example.com/sitemap.xml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"https://example.com/page1",
		"https://example.com/page2",
		"https://example.com/page3",
	}

	if len(urls) != len(expected) {
		t.Errorf("expected %d URLs, got %d", len(expected), len(urls))
	}

	for i, url := range urls {
		if url != expected[i] {
			t.Errorf("expected URL %d to be %q, got %q", i, expected[i], url)
		}
	}
}

func TestParseSitemap_SitemapIndex(t *testing.T) {
	indexXML := `<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<sitemap>
		<loc>https://example.com/sitemap1.xml</loc>
	</sitemap>
	<sitemap>
		<loc>https://example.com/sitemap2.xml</loc>
	</sitemap>
</sitemapindex>`

	sitemap1XML := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url><loc>https://example.com/page1</loc></url>
	<url><loc>https://example.com/page2</loc></url>
</urlset>`

	sitemap2XML := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url><loc>https://example.com/page3</loc></url>
	<url><loc>https://example.com/page4</loc></url>
</urlset>`

	fetcher := &mockFetcher{
		responses: map[string]fetch.Result{
			"https://example.com/sitemap-index.xml": {
				URL:    "https://example.com/sitemap-index.xml",
				Status: 200,
				HTML:   indexXML,
			},
			"https://example.com/sitemap1.xml": {
				URL:    "https://example.com/sitemap1.xml",
				Status: 200,
				HTML:   sitemap1XML,
			},
			"https://example.com/sitemap2.xml": {
				URL:    "https://example.com/sitemap2.xml",
				Status: 200,
				HTML:   sitemap2XML,
			},
		},
	}

	parser := NewSitemapParser(fetcher)
	urls, err := parser.ParseSitemap(context.Background(), "https://example.com/sitemap-index.xml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"https://example.com/page1",
		"https://example.com/page2",
		"https://example.com/page3",
		"https://example.com/page4",
	}

	if len(urls) != len(expected) {
		t.Errorf("expected %d URLs, got %d", len(expected), len(urls))
	}

	for i, url := range urls {
		if url != expected[i] {
			t.Errorf("expected URL %d to be %q, got %q", i, expected[i], url)
		}
	}
}

func TestParseSitemap_GzipCompressed(t *testing.T) {
	sitemapXML := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url><loc>https://example.com/page1</loc></url>
</urlset>`

	// Compress the XML
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	gzipWriter.Write([]byte(sitemapXML))
	gzipWriter.Close()

	fetcher := &mockFetcher{
		responses: map[string]fetch.Result{
			"https://example.com/sitemap.xml.gz": {
				URL:    "https://example.com/sitemap.xml.gz",
				Status: 200,
				HTML:   buf.String(),
			},
		},
	}

	parser := NewSitemapParser(fetcher)
	urls, err := parser.ParseSitemap(context.Background(), "https://example.com/sitemap.xml.gz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(urls) != 1 {
		t.Errorf("expected 1 URL, got %d", len(urls))
	}

	if urls[0] != "https://example.com/page1" {
		t.Errorf("expected URL to be %q, got %q", "https://example.com/page1", urls[0])
	}
}

func TestParseSitemap_NotFound(t *testing.T) {
	fetcher := &mockFetcher{
		responses: map[string]fetch.Result{},
	}

	parser := NewSitemapParser(fetcher)
	_, err := parser.ParseSitemap(context.Background(), "https://example.com/sitemap.xml")
	if err == nil {
		t.Error("expected error for 404 response, got nil")
	}
}

func TestParseSitemap_InvalidXML(t *testing.T) {
	fetcher := &mockFetcher{
		responses: map[string]fetch.Result{
			"https://example.com/sitemap.xml": {
				URL:    "https://example.com/sitemap.xml",
				Status: 200,
				HTML:   "not valid xml",
			},
		},
	}

	parser := NewSitemapParser(fetcher)
	_, err := parser.ParseSitemap(context.Background(), "https://example.com/sitemap.xml")
	if err == nil {
		t.Error("expected error for invalid XML, got nil")
	}
}

func TestFilterSameHost(t *testing.T) {
	urls := []string{
		"https://example.com/page1",
		"https://example.com/page2",
		"https://other.com/page1",
		"https://example.com/page3",
	}

	filtered, err := FilterSameHost("https://example.com/", urls)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"https://example.com/page1",
		"https://example.com/page2",
		"https://example.com/page3",
	}

	if len(filtered) != len(expected) {
		t.Errorf("expected %d URLs, got %d", len(expected), len(filtered))
	}

	for i, url := range filtered {
		if url != expected[i] {
			t.Errorf("expected URL %d to be %q, got %q", i, expected[i], url)
		}
	}
}

func TestFilterSameHost_InvalidBaseURL(t *testing.T) {
	_, err := FilterSameHost("://invalid-url", []string{"https://example.com/page1"})
	if err == nil {
		t.Error("expected error for invalid base URL, got nil")
	}
}

func TestParseSitemapXML_URLSet(t *testing.T) {
	xml := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url><loc>https://example.com/page1</loc></url>
	<url><loc>https://example.com/page2</loc></url>
</urlset>`)

	urls, sitemaps, err := parseSitemapXML(xml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sitemaps) != 0 {
		t.Errorf("expected 0 sitemaps, got %d", len(sitemaps))
	}

	expected := []string{"https://example.com/page1", "https://example.com/page2"}
	if len(urls) != len(expected) {
		t.Errorf("expected %d URLs, got %d", len(expected), len(urls))
	}

	for i, url := range urls {
		if url != expected[i] {
			t.Errorf("expected URL %d to be %q, got %q", i, expected[i], url)
		}
	}
}

func TestParseSitemapXML_SitemapIndex(t *testing.T) {
	xml := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<sitemap><loc>https://example.com/sitemap1.xml</loc></sitemap>
	<sitemap><loc>https://example.com/sitemap2.xml</loc></sitemap>
</sitemapindex>`)

	urls, sitemaps, err := parseSitemapXML(xml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(urls) != 0 {
		t.Errorf("expected 0 URLs, got %d", len(urls))
	}

	expected := []string{"https://example.com/sitemap1.xml", "https://example.com/sitemap2.xml"}
	if len(sitemaps) != len(expected) {
		t.Errorf("expected %d sitemaps, got %d", len(expected), len(sitemaps))
	}

	for i, sitemap := range sitemaps {
		if sitemap != expected[i] {
			t.Errorf("expected sitemap %d to be %q, got %q", i, expected[i], sitemap)
		}
	}
}

func TestParseSitemapXML_Invalid(t *testing.T) {
	xml := []byte("not valid xml")

	_, _, err := parseSitemapXML(xml)
	if err == nil {
		t.Error("expected error for invalid XML, got nil")
	}
}

func TestIsGzipped(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		data        []byte
		expected    bool
	}{
		{
			name:        "gzip content type",
			contentType: "application/gzip",
			data:        []byte("some data"),
			expected:    true,
		},
		{
			name:        "magic bytes",
			contentType: "application/xml",
			data:        []byte{0x1f, 0x8b, 0x00, 0x00},
			expected:    true,
		},
		{
			name:        "not gzipped",
			contentType: "application/xml",
			data:        []byte("not gzipped"),
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isGzipped(tt.contentType, tt.data)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDecompressGzip(t *testing.T) {
	original := []byte("hello world")

	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	gzipWriter.Write(original)
	gzipWriter.Close()

	compressed := buf.Bytes()
	decompressed, err := decompressGzip(compressed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Equal(decompressed, original) {
		t.Errorf("expected %q, got %q", original, decompressed)
	}
}

func TestDecompressGzip_Invalid(t *testing.T) {
	_, err := decompressGzip([]byte("not gzipped"))
	if err == nil {
		t.Error("expected error for invalid gzip data, got nil")
	}
}
