// Package research provides summary generation from evidence.
package research

import (
	"regexp"
	"sort"
	"strings"
)

// scoredSentence represents a sentence with its relevance score.
type scoredSentence struct {
	Text  string
	Score float64
}

// summarize generates a summary from evidence by selecting top-scoring sentences.
func summarize(tokens []string, items []Evidence) string {
	if len(items) == 0 {
		return "No evidence gathered."
	}

	max := 5
	if len(items) < max {
		max = len(items)
	}

	sentences := make([]string, 0, len(items))
	for _, item := range items {
		sentences = append(sentences, evidenceSentences(item)...)
		if len(sentences) > 40 {
			break
		}
	}

	scored := make([]scoredSentence, 0, len(sentences))
	for _, sentence := range sentences {
		scored = append(scored, scoredSentence{
			Text:  sentence,
			Score: scoreText(tokens, sentence),
		})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	selected := make([]string, 0, max)
	for i := 0; i < len(scored) && len(selected) < max; i++ {
		if strings.TrimSpace(scored[i].Text) == "" {
			continue
		}
		selected = append(selected, scored[i].Text)
	}

	if len(selected) == 0 {
		return items[0].Snippet
	}
	return strings.Join(selected, " ")
}

// splitSentences splits text into sentences.
func splitSentences(text string) []string {
	parts := regexp.MustCompile(`[.!?]+`).Split(text, -1)
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trim := strings.TrimSpace(part)
		if trim != "" {
			out = append(out, trim+".")
		}
	}
	return out
}
