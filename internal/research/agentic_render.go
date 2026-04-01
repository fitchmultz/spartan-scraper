// Package research provides research functionality for Spartan Scraper.
//
// Purpose:
// - Render bounded agentic research prompts and provide local helper utilities used by the workflow.
//
// Responsibilities:
// - Build planning/synthesis HTML bundles, normalize candidate URLs, and extract typed values from schema-guided AI results.
//
// Scope:
// - Prompt rendering and workflow helper logic only; evidence gathering and round orchestration live in adjacent files.
//
// Usage:
// - Used internally by the agentic research workflow.
//
// Invariants/Assumptions:
// - Rendered HTML bundles must remain deterministic and bounded.
// - URL normalization must only emit http/https follow-up targets.
package research

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func collectCandidateURLs(docs []researchDocument, visited map[string]struct{}) []string {
	candidates := make([]string, 0)
	seen := map[string]struct{}{}
	for _, doc := range docs {
		for _, link := range doc.Links {
			if _, ok := visited[link]; ok {
				continue
			}
			if _, ok := seen[link]; ok {
				continue
			}
			seen[link] = struct{}{}
			candidates = append(candidates, link)
			if len(candidates) >= 50 {
				return candidates
			}
		}
	}
	return candidates
}

func filterSelectedFollowUpURLs(selected []string, candidates []string, maxURLs int) []string {
	candidateSet := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		candidateSet[candidate] = struct{}{}
	}
	filtered := make([]string, 0, minInt(len(selected), maxURLs))
	for _, raw := range selected {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		if _, ok := candidateSet[trimmed]; !ok {
			continue
		}
		filtered = appendUniqueStrings(filtered, trimmed)
		if len(filtered) >= maxURLs {
			break
		}
	}
	return filtered
}

func normalizeDocumentLinks(baseURL string, links []string) []string {
	if len(links) == 0 {
		return nil
	}
	out := make([]string, 0, len(links))
	seen := map[string]struct{}{}
	for _, raw := range links {
		normalized := normalizeFollowUpURL(baseURL, raw)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

func normalizeFollowUpURL(baseURL string, raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return ""
	}
	resolved := base.ResolveReference(parsed)
	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return ""
	}
	resolved.Fragment = ""
	return resolved.String()
}

func researchSourcePriorityBoost(target string, candidate string) float64 {
	const preferredSourceBoost = 500.0
	if normalizeResearchSourceURL(target) == normalizeResearchSourceURL(candidate) {
		return preferredSourceBoost
	}
	return 0
}

func normalizeResearchSourceURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return ""
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ""
	}
	parsed.Fragment = ""
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	if parsed.Path != "/" {
		parsed.Path = strings.TrimRight(parsed.Path, "/")
	}
	parsed.RawQuery = ""
	return parsed.String()
}

func renderAgenticPlanningHTML(req Request, cfg *model.ResearchAgenticConfig, base Result, docs []researchDocument, candidates []string) string {
	var b strings.Builder
	b.WriteString("<html><body>")
	b.WriteString("<h1>Research planning bundle</h1>")
	b.WriteString(tag("p", "Query: "+req.Query))
	if cfg.Instructions != "" {
		b.WriteString(tag("p", "Operator instructions: "+cfg.Instructions))
	}
	b.WriteString(tag("p", fmt.Sprintf("Deterministic summary: %s", base.Summary)))
	b.WriteString("<h2>Evidence</h2><ol>")
	for i, doc := range docs {
		b.WriteString("<li>")
		b.WriteString(tag("strong", fmt.Sprintf("%d. %s", i+1, fallbackString(doc.Evidence.Title, doc.Evidence.URL))))
		b.WriteString(tag("p", doc.Evidence.URL))
		b.WriteString(tag("p", doc.Evidence.Snippet))
		if fieldSummary := summarizeEvidenceFields(doc.Evidence.Fields); fieldSummary != "" {
			b.WriteString(tag("p", fieldSummary))
		}
		b.WriteString("</li>")
		if i >= 11 {
			break
		}
	}
	b.WriteString("</ol>")
	b.WriteString("<h2>Candidate follow-up URLs</h2><ul>")
	for _, candidate := range candidates {
		b.WriteString(tag("li", candidate))
	}
	b.WriteString("</ul></body></html>")
	return b.String()
}

func renderAgenticSynthesisHTML(req Request, cfg *model.ResearchAgenticConfig, base Result, docs []researchDocument, rounds []AgenticResearchRound) string {
	var b strings.Builder
	b.WriteString("<html><body>")
	b.WriteString("<h1>Research synthesis bundle</h1>")
	b.WriteString(tag("p", "Query: "+req.Query))
	if cfg.Instructions != "" {
		b.WriteString(tag("p", "Operator instructions: "+cfg.Instructions))
	}
	b.WriteString(tag("p", fmt.Sprintf("Deterministic summary: %s", base.Summary)))
	if len(rounds) > 0 {
		b.WriteString("<h2>Follow-up rounds</h2><ol>")
		for _, round := range rounds {
			b.WriteString("<li>")
			b.WriteString(tag("p", fmt.Sprintf("Round %d goal: %s", round.Round, round.Goal)))
			if len(round.FocusAreas) > 0 {
				b.WriteString(tag("p", "Focus areas: "+strings.Join(round.FocusAreas, ", ")))
			}
			if len(round.SelectedURLs) > 0 {
				b.WriteString(tag("p", "Selected URLs: "+strings.Join(round.SelectedURLs, ", ")))
			}
			if round.Reasoning != "" {
				b.WriteString(tag("p", "Reasoning: "+round.Reasoning))
			}
			b.WriteString("</li>")
		}
		b.WriteString("</ol>")
	}
	b.WriteString("<h2>Evidence</h2><ol>")
	for i, doc := range docs {
		b.WriteString("<li>")
		b.WriteString(tag("strong", fmt.Sprintf("%d. %s", i+1, fallbackString(doc.Evidence.Title, doc.Evidence.URL))))
		b.WriteString(tag("p", doc.Evidence.URL))
		b.WriteString(tag("p", doc.Evidence.Snippet))
		if fieldSummary := summarizeEvidenceFields(doc.Evidence.Fields); fieldSummary != "" {
			b.WriteString(tag("p", fieldSummary))
		}
		b.WriteString("</li>")
		if i >= 15 {
			break
		}
	}
	b.WriteString("</ol></body></html>")
	return b.String()
}

func tag(name string, content string) string {
	return "<" + name + ">" + html.EscapeString(strings.TrimSpace(content)) + "</" + name + ">"
}

func stringField(fields map[string]extract.FieldValue, key string) string {
	field, ok := fields[key]
	if !ok {
		return ""
	}
	for _, value := range field.Values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	trimmed := strings.TrimSpace(field.RawObject)
	trimmed = strings.Trim(trimmed, `"`)
	return strings.TrimSpace(trimmed)
}

func stringSliceField(fields map[string]extract.FieldValue, key string) []string {
	field, ok := fields[key]
	if !ok {
		return nil
	}
	if len(field.Values) > 0 {
		return appendUniqueStrings(nil, field.Values...)
	}
	if strings.TrimSpace(field.RawObject) == "" {
		return nil
	}
	var decoded []string
	if err := json.Unmarshal([]byte(field.RawObject), &decoded); err == nil {
		return appendUniqueStrings(nil, decoded...)
	}
	return appendUniqueStrings(nil, field.RawObject)
}

func appendUniqueStrings(dst []string, values ...string) []string {
	seen := make(map[string]struct{}, len(dst))
	for _, existing := range dst {
		if trimmed := strings.TrimSpace(existing); trimmed != "" {
			seen[trimmed] = struct{}{}
		}
	}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		dst = append(dst, trimmed)
	}
	return dst
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func firstOrEmpty(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func fallbackString(primary string, fallback string) string {
	if trimmed := strings.TrimSpace(primary); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(fallback)
}

func minInt(a int, b int) int {
	if a <= 0 {
		return b
	}
	if b <= 0 || a < b {
		return a
	}
	return b
}
