// Package watch provides HTML content extraction helpers for watch check processing.
//
// Purpose:
// - Extract text content from HTML using CSS selectors or full-text extraction.
//
// Responsibilities:
// - Compile and apply CSS selectors to extract matched element text.
// - Strip tags and normalize whitespace for full-text extraction mode.
//
// Scope:
// - Content extraction only; watch execution and scheduling live in sibling files.
//
// Usage:
// - Called by Watcher.fetchContentWithScreenshot when a selector or text mode is configured.
//
// Invariants/Assumptions:
// - Input HTML is well-formed enough for goquery parsing.
// - Selectors must be valid CSS selectors accepted by cascadia.
package watch

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/andybalholm/cascadia"
)

// extractSelector extracts content from HTML using a CSS selector.
func extractSelector(html, selector string) (string, error) {
	compiled, err := cascadia.Compile(selector)
	if err != nil {
		return "", fmt.Errorf("invalid selector %q: %w", selector, err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", err
	}

	matches := doc.FindMatcher(compiled)
	if matches.Length() == 0 {
		return "", fmt.Errorf("no elements matched selector %q", selector)
	}

	results := make([]string, 0, matches.Length())
	matches.Each(func(_ int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "" {
			results = append(results, text)
		}
	})
	if len(results) == 0 {
		return "", fmt.Errorf("selector %q matched elements but extracted no text", selector)
	}

	return strings.Join(results, "\n"), nil
}

// extractTextFromHTML extracts clean text from HTML.
func extractTextFromHTML(html string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return ""
	}

	// Remove script and style elements
	doc.Find("script,style,noscript").Remove()

	// Get text from body
	bodyText := strings.TrimSpace(doc.Find("body").Text())
	return strings.Join(strings.Fields(bodyText), " ")
}
