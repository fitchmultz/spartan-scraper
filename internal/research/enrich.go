// Package research provides evidence enrichment with simhash, citations, and confidence.
package research

import (
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/simhash"
)

// enrichEvidence enriches evidence items with simhash, citations, and confidence scores.
func enrichEvidence(items []Evidence) []Evidence {
	if len(items) == 0 {
		return items
	}
	maxScore := 0.0
	for _, item := range items {
		if item.Score > maxScore {
			maxScore = item.Score
		}
	}

	out := make([]Evidence, 0, len(items))
	for _, item := range items {
		text := strings.TrimSpace(item.Title + " " + item.Snippet)
		item.SimHash = simhash.Compute(text)
		citation := normalizeCitation(item.URL, item.Snippet, item.Title)
		item.CitationURL = buildCitationURL(citation.Canonical, citation.Anchor)
		item.Confidence = evidenceConfidence(item, maxScore)
		out = append(out, item)
	}
	return out
}
