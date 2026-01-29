// Package research provides citation URL normalization and generation.
package research

import (
	"net/url"
	"regexp"
	"strings"
)

// buildCitations creates a list of unique citations from evidence items.
func buildCitations(items []Evidence) []Citation {
	seen := map[string]bool{}
	out := make([]Citation, 0, len(items))
	for _, item := range items {
		citation := normalizeCitation(item.URL, item.Snippet, item.Title)
		key := citation.Canonical + "#" + citation.Anchor
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, citation)
	}
	return out
}

// normalizeCitation creates a normalized citation from URL, snippet, and title.
func normalizeCitation(rawURL string, snippet string, title string) Citation {
	canonical := canonicalizeURL(rawURL)
	anchor := citationAnchor(snippet, title)
	return Citation{
		URL:       rawURL,
		Anchor:    anchor,
		Canonical: canonical,
	}
}

// buildCitationURL builds a citation URL with optional anchor fragment.
func buildCitationURL(canonical string, anchor string) string {
	if canonical == "" {
		return ""
	}
	if anchor == "" {
		return canonical
	}
	return canonical + "#" + anchor
}

// canonicalizeURL converts a URL to its canonical form without fragment.
func canonicalizeURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return raw
	}
	parsed.Fragment = ""
	return parsed.String()
}

// citationAnchor generates a URL-safe anchor from snippet or title text.
func citationAnchor(snippet string, title string) string {
	base := strings.TrimSpace(snippet)
	if base == "" {
		base = strings.TrimSpace(title)
	}
	if base == "" {
		return ""
	}
	re := regexp.MustCompile(`[^a-z0-9\s]+`)
	clean := re.ReplaceAllString(strings.ToLower(base), " ")
	words := strings.Fields(clean)
	if len(words) == 0 {
		return ""
	}
	if len(words) > 8 {
		words = words[:8]
	}
	return strings.Join(words, "-")
}
