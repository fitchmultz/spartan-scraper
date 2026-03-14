package research

import (
	"sort"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/extract"
)

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
	if trimmed := strings.TrimSpace(title); trimmed != "" {
		parts = append(parts, trimmed)
	}
	if summary := summarizeEvidenceFields(fields); summary != "" {
		parts = append(parts, summary)
	}
	if trimmed := strings.TrimSpace(text); trimmed != "" {
		parts = append(parts, trimmed)
	}
	return strings.Join(parts, " ")
}

func evidenceSentences(item Evidence) []string {
	sentences := make([]string, 0, 8)
	if summary := summarizeEvidenceFields(item.Fields); summary != "" {
		sentences = append(sentences, summary)
	}
	sentences = append(sentences, splitSentences(item.Snippet)...)
	return sentences
}

func summarizeEvidenceFields(fields map[string]extract.FieldValue) string {
	if len(fields) == 0 {
		return ""
	}

	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		valueText := fieldValueText(fields[key])
		if valueText == "" {
			continue
		}
		parts = append(parts, normalizeFieldLabel(key)+": "+valueText)
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "; ") + "."
}

func fieldValueText(field extract.FieldValue) string {
	parts := make([]string, 0, len(field.Values)+1)
	for _, value := range field.Values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		parts = append(parts, trimmed)
	}
	if raw := strings.TrimSpace(field.RawObject); raw != "" {
		parts = append(parts, raw)
	}
	return strings.Join(parts, ", ")
}

func normalizeFieldLabel(name string) string {
	replacer := strings.NewReplacer("_", " ", "-", " ")
	return replacer.Replace(strings.TrimSpace(name))
}
