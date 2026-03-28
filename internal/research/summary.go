// Package research provides deterministic summary generation from ranked
// evidence.
//
// Purpose: Build a compact operator-facing synthesis from gathered evidence
// without requiring an external model.
// Responsibilities: Select the strongest sentence from the top evidence items,
// balance coverage across sources, and fall back to stable titles/snippets when
// richer prose is unavailable.
// Scope: Deterministic research summary generation only.
// Usage: Called by `buildResearchResult()` after evidence ranking.
// Invariants/Assumptions: Summaries stay short, skip boilerplate, and prefer
// source diversity over dumping many adjacent sentences from a single page.
package research

import (
	"sort"
	"strings"
)

// scoredSentence represents a sentence with its relevance score.
type scoredSentence struct {
	Text  string
	Score float64
}

// summarize generates a compact summary from the top-ranked evidence items.
func summarize(tokens []string, items []Evidence) string {
	if len(items) == 0 {
		return "No evidence gathered."
	}

	maxItems := minInt(len(items), 3)
	selected := make([]string, 0, maxItems)
	seen := map[string]struct{}{}
	totalChars := 0

	for _, item := range items[:maxItems] {
		sentence := bestEvidenceSentence(tokens, item)
		if sentence == "" {
			continue
		}
		key := strings.ToLower(normalizeWhitespace(sentence))
		if _, exists := seen[key]; exists {
			continue
		}
		projected := totalChars + len(sentence)
		if len(selected) > 0 {
			projected++
		}
		if len(selected) > 0 && projected > 420 {
			break
		}
		seen[key] = struct{}{}
		selected = append(selected, sentence)
		totalChars = projected
	}

	if len(selected) == 0 {
		return fallbackSummary(items)
	}
	return strings.Join(selected, " ")
}

func bestEvidenceSentence(tokens []string, item Evidence) string {
	sentences := evidenceSentences(item)
	if len(sentences) == 0 {
		return ""
	}

	scored := make([]scoredSentence, 0, len(sentences))
	for _, sentence := range sentences {
		trimmed := strings.TrimSpace(sentence)
		if trimmed == "" || looksLikeBoilerplateText(trimmed) {
			continue
		}
		scored = append(scored, scoredSentence{
			Text:  trimmed,
			Score: scoreText(tokens, trimmed) + researchSentenceQualityScore(trimmed),
		})
	}
	if len(scored) == 0 {
		return ""
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].Score == scored[j].Score {
			return len(scored[i].Text) < len(scored[j].Text)
		}
		return scored[i].Score > scored[j].Score
	})
	return scored[0].Text
}

func researchSentenceQualityScore(sentence string) float64 {
	length := len(sentence)
	score := 0.0
	switch {
	case length >= 40 && length <= 180:
		score += 0.6
	case length <= 90:
		score += 0.35
	case length > 260:
		score -= 0.8
	}
	if strings.Contains(sentence, "|") {
		score += 0.2
	}
	if strings.Count(sentence, ";") > 2 {
		score -= 0.4
	}
	return score
}

func fallbackSummary(items []Evidence) string {
	for _, item := range items {
		if title := normalizeWhitespace(item.Title); title != "" && !looksLikeBoilerplateText(title) {
			return ensureSentence(title)
		}
		if snippet := makeSnippet(item.Snippet); snippet != "" {
			return snippet
		}
	}
	return "No evidence gathered."
}
