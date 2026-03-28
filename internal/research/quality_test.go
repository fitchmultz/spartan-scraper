// Package research verifies deterministic research-quality helpers.
//
// Purpose: Protect readability-focused heuristics for deterministic research
// summaries.
// Responsibilities: Assert query token filtering, boilerplate snippet rejection,
// source prioritization, and concise multi-source summary generation.
// Scope: Unit coverage for text and summary helpers only.
// Usage: Run with `go test ./internal/research`.
// Invariants/Assumptions: Deterministic summaries should stay compact, prefer
// explicit source pages, and avoid navigation boilerplate.
package research

import (
	"strings"
	"testing"
)

func TestTokenizeSkipsCommonStopWords(t *testing.T) {
	got := tokenize("Compare how the Go docs and JavaScript docs position learning paths for newcomers")

	for _, want := range []string{"compare", "go", "docs", "javascript", "position", "learning", "paths", "newcomers"} {
		if !containsString(got, want) {
			t.Fatalf("tokenize() missing %q in %v", want, got)
		}
	}
	for _, unwanted := range []string{"the", "and", "for", "how"} {
		if containsString(got, unwanted) {
			t.Fatalf("tokenize() unexpectedly kept stop word %q in %v", unwanted, got)
		}
	}
}

func TestMakeSnippetRejectsNavigationBoilerplate(t *testing.T) {
	text := "Skip to Main Content Why Go arrow_drop_down Press Enter to activate/deactivate dropdown Case Studies Common problems companies solve with Go"
	if got := makeSnippet(text); got != "" {
		t.Fatalf("makeSnippet() = %q, want empty string", got)
	}
}

func TestMakeSnippetKeepsMeaningfulContentWhenBoilerplateIsMixedIn(t *testing.T) {
	text := "Skip to Main Content. The Go docs highlight getting started guides, tutorials, and package references for new developers. Press Enter to activate/deactivate dropdown."
	got := makeSnippet(text)
	if got == "" {
		t.Fatal("makeSnippet() returned empty snippet for mixed content")
	}
	if strings.Contains(strings.ToLower(got), "skip to main content") {
		t.Fatalf("makeSnippet() kept boilerplate sentence: %q", got)
	}
	if !strings.Contains(got, "getting started guides") {
		t.Fatalf("makeSnippet() dropped meaningful sentence: %q", got)
	}
}

func TestSummarizePrefersMeaningfulTopSources(t *testing.T) {
	query := "Compare how the Go docs and JavaScript docs position learning paths for newcomers"
	items := []Evidence{
		{
			Title:   "JavaScript | MDN",
			Snippet: "JavaScript explains the language fundamentals first and then branches into guides and references for deeper learning.",
			Score:   740,
		},
		{
			Title:   "Documentation - The Go Programming Language",
			Snippet: "The Go docs highlight getting started material, tutorials, and package documentation for newcomers.",
			Score:   875,
		},
		{
			Title:   "MDN Web Docs",
			Snippet: "MDN also organizes broader platform references and practical guides for web developers.",
			Score:   300,
		},
	}

	summary := summarize(tokenize(query), items)
	if !strings.Contains(summary, "JavaScript") {
		t.Fatalf("summary %q did not mention JavaScript", summary)
	}
	if !strings.Contains(summary, "Go") {
		t.Fatalf("summary %q did not mention Go", summary)
	}
	if strings.Contains(strings.ToLower(summary), "skip to main content") {
		t.Fatalf("summary %q still contains boilerplate", summary)
	}
	if len(summary) > 420 {
		t.Fatalf("summary too long: %d chars", len(summary))
	}
}

func TestResearchSourcePriorityBoostPrefersExplicitSeedURL(t *testing.T) {
	boost := researchSourcePriorityBoost(
		"https://developer.mozilla.org/en-US/docs/Web/JavaScript",
		"https://developer.mozilla.org/en-US/docs/Web/JavaScript/",
	)
	if boost <= 0 {
		t.Fatalf("expected positive boost for canonical seed URL, got %v", boost)
	}
	if researchSourcePriorityBoost(
		"https://developer.mozilla.org/en-US/docs/Web/JavaScript",
		"https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements",
	) != 0 {
		t.Fatal("expected no boost for non-seed evidence")
	}
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
