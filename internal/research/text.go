// Package research provides text processing utilities for research workflows.
package research

import (
	"regexp"
	"strings"
)

// tokenize converts a query string into a list of unique, lowercase tokens.
// Removes punctuation and special characters, deduplicates tokens.
func tokenize(query string) []string {
	clean := strings.ToLower(query)
	re := regexp.MustCompile(`[^a-z0-9\s]+`)
	clean = re.ReplaceAllString(clean, " ")
	parts := strings.Fields(clean)
	uniq := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		if part == "" || seen[part] {
			continue
		}
		seen[part] = true
		uniq = append(uniq, part)
	}
	return uniq
}

// scoreText calculates relevance score of text against query tokens.
// Higher score indicates more matches with the query tokens.
func scoreText(tokens []string, text string) float64 {
	lower := strings.ToLower(text)
	score := 0.0
	for _, token := range tokens {
		score += float64(strings.Count(lower, token))
	}
	return score
}

// makeSnippet creates a text snippet, truncating to 300 characters if necessary.
func makeSnippet(text string) string {
	trimmed := strings.TrimSpace(text)
	if len(trimmed) <= 300 {
		return trimmed
	}
	return trimmed[:300] + "..."
}
