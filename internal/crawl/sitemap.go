// Package crawl provides sitemap.xml parsing for URL discovery.
// Supports standard sitemap.xml, sitemap index files, and gzip-compressed sitemaps.
// Does NOT support robots.txt discovery (out of scope per AGENTS.md).
package crawl

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
)

// SitemapParser handles parsing of XML sitemap files.
type SitemapParser struct {
	fetcher fetch.Fetcher
}

// NewSitemapParser creates a new parser with the given fetcher.
// If fetcher is nil, a default fetcher will be created.
func NewSitemapParser(fetcher fetch.Fetcher) *SitemapParser {
	if fetcher == nil {
		fetcher = fetch.NewFetcher("")
	}
	return &SitemapParser{fetcher: fetcher}
}

// ParseSitemap fetches and parses a sitemap URL, returning all URLs found.
// Handles both urlset and sitemapindex formats.
// Recursively follows sitemapindex entries.
// Respects proxy configuration from the fetcher.
func (p *SitemapParser) ParseSitemap(ctx context.Context, sitemapURL string) ([]string, error) {
	return p.parseSitemapRecursive(ctx, sitemapURL, 0)
}

// parseSitemapRecursive recursively parses sitemaps up to maxDepth levels.
const maxSitemapDepth = 5

func (p *SitemapParser) parseSitemapRecursive(ctx context.Context, sitemapURL string, depth int) ([]string, error) {
	if depth > maxSitemapDepth {
		return nil, apperrors.Validation("sitemap nesting exceeds maximum depth")
	}

	slog.Debug("fetching sitemap", "url", apperrors.SanitizeURL(sitemapURL), "depth", depth)

	// Fetch the sitemap using the fetcher to respect proxy/auth settings
	req := fetch.Request{
		URL:     sitemapURL,
		Timeout: 30 * time.Second,
	}

	res, err := p.fetcher.Fetch(ctx, req)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to fetch sitemap", err)
	}

	if res.Status >= 400 {
		return nil, apperrors.Internal(fmt.Sprintf("sitemap fetch returned status %d", res.Status))
	}

	data := []byte(res.HTML)

	// Check for gzip compression
	if isGzipped(res.URL, data) {
		data, err = decompressGzip(data)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to decompress gzip sitemap", err)
		}
	}

	urls, nestedSitemaps, err := parseSitemapXML(data)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to parse sitemap XML", err)
	}

	// Recursively fetch nested sitemaps
	for _, nestedURL := range nestedSitemaps {
		nestedURLs, err := p.parseSitemapRecursive(ctx, nestedURL, depth+1)
		if err != nil {
			slog.Warn("failed to parse nested sitemap", "url", apperrors.SanitizeURL(nestedURL), "error", err)
			continue
		}
		urls = append(urls, nestedURLs...)
	}

	return urls, nil
}

// sitemapURL represents a URL entry in a sitemap.
type sitemapURL struct {
	Loc string `xml:"loc"`
}

// urlset represents a standard sitemap.xml structure.
type urlset struct {
	XMLName xml.Name     `xml:"urlset"`
	URLs    []sitemapURL `xml:"url"`
}

// sitemapIndexEntry represents a sitemap entry in a sitemap index.
type sitemapIndexEntry struct {
	Loc string `xml:"loc"`
}

// sitemapindex represents a sitemap index structure.
type sitemapindex struct {
	XMLName  xml.Name            `xml:"sitemapindex"`
	Sitemaps []sitemapIndexEntry `xml:"sitemap"`
}

// parseSitemapXML parses raw XML bytes, returns URLs and nested sitemap URLs.
func parseSitemapXML(data []byte) (urls []string, sitemaps []string, err error) {
	// Try parsing as urlset first
	var us urlset
	if decodeErr := xml.Unmarshal(data, &us); decodeErr == nil && len(us.URLs) > 0 {
		for _, u := range us.URLs {
			if u.Loc != "" {
				urls = append(urls, strings.TrimSpace(u.Loc))
			}
		}
		return urls, nil, nil
	}

	// Try parsing as sitemapindex
	var si sitemapindex
	if decodeErr := xml.Unmarshal(data, &si); decodeErr == nil && len(si.Sitemaps) > 0 {
		for _, s := range si.Sitemaps {
			if s.Loc != "" {
				sitemaps = append(sitemaps, strings.TrimSpace(s.Loc))
			}
		}
		return nil, sitemaps, nil
	}

	// If neither format parsed successfully, return error
	return nil, nil, apperrors.Validation("sitemap is neither urlset nor sitemapindex format")
}

// isGzipped detects if content is gzip compressed.
func isGzipped(contentType string, data []byte) bool {
	// Check Content-Type header
	if strings.Contains(contentType, "gzip") {
		return true
	}

	// Check magic bytes for gzip
	if len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b {
		return true
	}

	return false
}

// decompressGzip decompresses gzip data.
func decompressGzip(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

// FilterSameHost filters URLs to only include those matching the base host.
func FilterSameHost(baseURL string, urls []string) ([]string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindValidation, "invalid base URL", err)
	}

	var filtered []string
	for _, u := range urls {
		parsed, err := url.Parse(u)
		if err != nil {
			continue
		}
		if parsed.Host == base.Host {
			filtered = append(filtered, u)
		}
	}

	return filtered, nil
}

// FetchAndParseSitemap is a convenience function that fetches and parses a sitemap.
// It returns URLs filtered to the same host as the baseURL.
func FetchAndParseSitemap(ctx context.Context, sitemapURL, baseURL string, fetcher fetch.Fetcher) ([]string, error) {
	parser := NewSitemapParser(fetcher)
	urls, err := parser.ParseSitemap(ctx, sitemapURL)
	if err != nil {
		return nil, err
	}

	return FilterSameHost(baseURL, urls)
}
