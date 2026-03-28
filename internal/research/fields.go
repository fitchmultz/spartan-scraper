// Package research provides evidence-field summarization helpers used by the
// deterministic research pipeline.
//
// Purpose: Normalize extracted evidence fields into concise human-readable
// sentences that can backfill research snippets and summaries.
// Responsibilities: Clone mutable field maps, choose the strongest field text,
// and assemble sentence candidates for summary generation.
// Scope: Research evidence shaping only.
// Usage: Called when turning scrape/crawl outputs into deterministic research
// evidence items.
// Invariants/Assumptions: Field summaries stay compact, preferred semantic
// fields win over generic keys, and repeated values are deduplicated.
package research

import (
	"sort"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/extract"
)

var preferredResearchFieldKeys = []string{
	"description",
	"summary",
	"excerpt",
	"title",
	"h1",
}

func cloneEvidenceFields(fields map[string]extract.FieldValue) map[string]extract.FieldValue {
	if len(fields) == 0 {
		return nil
	}

	out := make(map[string]extract.FieldValue, len(fields))
	for key, field := range fields {
		copied := field
		if len(field.Values) > 0 {
			copied.Values = append([]string(nil), field.Values...)
		}
		out[key] = copied
	}
	return out
}

func makeEvidenceSnippet(text string, fields map[string]extract.FieldValue) string {
	if snippet := makeSnippet(text); snippet != "" {
		return snippet
	}
	return summarizeEvidenceFields(fields)
}

func evidenceSearchText(title, text string, fields map[string]extract.FieldValue) string {
	parts := make([]string, 0, 3)
	if trimmed := normalizeWhitespace(title); trimmed != "" {
		parts = append(parts, trimmed)
	}
	if summary := summarizeEvidenceFields(fields); summary != "" {
		parts = append(parts, summary)
	}
	if snippet := makeSnippet(text); snippet != "" {
		parts = append(parts, snippet)
	}
	return strings.Join(parts, " ")
}

func evidenceSentences(item Evidence) []string {
	sentences := make([]string, 0, 10)
	if title := normalizeWhitespace(item.Title); title != "" && !looksLikeBoilerplateText(title) {
		sentences = append(sentences, ensureSentence(title))
	}
	if summary := summarizeEvidenceFields(item.Fields); summary != "" {
		sentences = append(sentences, splitSentences(summary)...)
	}
	sentences = append(sentences, splitSentences(item.Snippet)...)
	return dedupeSentences(sentences)
}

func summarizeEvidenceFields(fields map[string]extract.FieldValue) string {
	if len(fields) == 0 {
		return ""
	}

	parts := make([]string, 0, 3)
	seen := map[string]struct{}{}
	appendPart := func(raw string) {
		trimmed := trimText(raw, 220)
		if trimmed == "" || looksLikeBoilerplateText(trimmed) {
			return
		}
		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		parts = append(parts, ensureSentence(trimmed))
	}

	for _, key := range preferredResearchFieldKeys {
		field, ok := fields[key]
		if !ok {
			continue
		}
		appendPart(fieldValueText(field))
		if len(parts) >= 3 {
			return strings.Join(parts, " ")
		}
	}

	keys := make([]string, 0, len(fields))
	for key := range fields {
		skip := false
		for _, preferred := range preferredResearchFieldKeys {
			if key == preferred {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		valueText := fieldValueText(fields[key])
		if valueText == "" {
			continue
		}
		appendPart(normalizeFieldLabel(key) + ": " + valueText)
		if len(parts) >= 3 {
			break
		}
	}

	return strings.Join(parts, " ")
}

func fieldValueText(field extract.FieldValue) string {
	parts := make([]string, 0, len(field.Values)+1)
	for _, value := range field.Values {
		trimmed := normalizeWhitespace(value)
		if trimmed == "" {
			continue
		}
		parts = append(parts, trimmed)
	}
	if raw := normalizeWhitespace(field.RawObject); raw != "" {
		parts = append(parts, raw)
	}
	return strings.Join(parts, ", ")
}

func normalizeFieldLabel(name string) string {
	replacer := strings.NewReplacer("_", " ", "-", " ")
	return replacer.Replace(strings.TrimSpace(name))
}

func dedupeSentences(sentences []string) []string {
	if len(sentences) == 0 {
		return nil
	}

	out := make([]string, 0, len(sentences))
	seen := map[string]struct{}{}
	for _, sentence := range sentences {
		normalized := strings.ToLower(normalizeWhitespace(sentence))
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, ensureSentence(strings.TrimSpace(sentence)))
	}
	return out
}
