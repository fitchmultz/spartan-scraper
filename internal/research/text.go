// Package research provides text processing utilities for research workflows.
//
// Purpose: Normalize research query text, filter low-signal boilerplate, and
// build concise snippets that keep deterministic summaries readable.
// Responsibilities: Tokenize operator queries, score text against query terms,
// detect navigation boilerplate, and trim evidence snippets to useful prose.
// Scope: Shared text helpers for deterministic research ranking and summary
// generation.
// Usage: Called by evidence gathering and summarization helpers inside the
// research package.
// Invariants/Assumptions: Returned snippets are whitespace-normalized,
// boilerplate-heavy navigation text is rejected, and tokenization preserves a
// meaningful fallback if stop-word filtering removes every term.
package research

import (
	"regexp"
	"strings"
)

var (
	nonAlphaNumericPattern = regexp.MustCompile(`[^a-z0-9\s]+`)
	sentenceSplitPattern   = regexp.MustCompile(`[.!?]+`)

	researchStopWords = map[string]struct{}{
		"a": {}, "an": {}, "and": {}, "are": {}, "as": {}, "at": {},
		"be": {}, "by": {}, "do": {}, "for": {}, "from": {}, "how": {},
		"in": {}, "into": {}, "is": {}, "it": {}, "of": {}, "on": {},
		"or": {}, "that": {}, "the": {}, "their": {}, "this": {},
		"to": {}, "vs": {}, "what": {}, "when": {}, "where": {},
		"which": {}, "who": {}, "why": {}, "with": {},
	}

	researchBoilerplatePhrases = []string{
		"skip to main content",
		"skip to search",
		"press enter to activate/deactivate dropdown",
		"press enter to activate dropdown",
		"press enter to deactivate dropdown",
		"arrow_drop_down",
		"toggle navigation",
		"main navigation",
	}
)

// tokenize converts a query string into a list of unique, lowercase tokens.
// Removes punctuation and common stop words while preserving a non-empty
// fallback token set when all remaining words would otherwise be filtered out.
func tokenize(query string) []string {
	clean := strings.ToLower(query)
	clean = nonAlphaNumericPattern.ReplaceAllString(clean, " ")
	parts := strings.Fields(clean)

	filtered := make([]string, 0, len(parts))
	fallback := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		if part == "" || seen[part] {
			continue
		}
		seen[part] = true
		fallback = append(fallback, part)
		if _, stopWord := researchStopWords[part]; stopWord {
			continue
		}
		filtered = append(filtered, part)
	}
	if len(filtered) > 0 {
		return filtered
	}
	return fallback
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

// makeSnippet creates a concise snippet from candidate page text.
// Navigation-heavy boilerplate is rejected so field summaries can take over.
func makeSnippet(text string) string {
	normalized := normalizeWhitespace(text)
	if normalized == "" {
		return ""
	}

	sentences := splitSentences(normalized)
	if len(sentences) == 0 {
		return trimText(normalized, 300)
	}

	selected := make([]string, 0, 2)
	totalChars := 0
	for _, sentence := range sentences {
		clean := normalizeWhitespace(sentence)
		if clean == "" || looksLikeBoilerplateText(clean) {
			continue
		}
		projected := totalChars + len(clean)
		if len(selected) > 0 {
			projected++
		}
		if len(selected) > 0 && projected > 320 {
			break
		}
		selected = append(selected, clean)
		totalChars = projected
		if len(selected) >= 2 || totalChars >= 220 {
			break
		}
	}
	if len(selected) == 0 {
		return ""
	}
	return trimText(strings.Join(selected, " "), 320)
}

// splitSentences splits text into normalized sentences.
func splitSentences(text string) []string {
	normalized := normalizeWhitespace(text)
	if normalized == "" {
		return nil
	}

	parts := sentenceSplitPattern.Split(normalized, -1)
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trim := normalizeWhitespace(part)
		if trim == "" {
			continue
		}
		out = append(out, ensureSentence(trim))
	}
	return out
}

func normalizeWhitespace(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

func looksLikeBoilerplateText(text string) bool {
	lower := strings.ToLower(normalizeWhitespace(text))
	if lower == "" {
		return false
	}
	for _, phrase := range researchBoilerplatePhrases {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

func trimText(text string, max int) string {
	trimmed := normalizeWhitespace(text)
	if trimmed == "" || len(trimmed) <= max {
		return trimmed
	}
	cutoff := strings.LastIndex(trimmed[:max], " ")
	if cutoff < max/2 {
		cutoff = max
	}
	return strings.TrimSpace(trimmed[:cutoff]) + "…"
}

func ensureSentence(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}
	if strings.HasSuffix(trimmed, ".") || strings.HasSuffix(trimmed, "!") || strings.HasSuffix(trimmed, "?") || strings.HasSuffix(trimmed, "…") {
		return trimmed
	}
	return trimmed + "."
}
