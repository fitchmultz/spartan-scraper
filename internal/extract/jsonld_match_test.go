// Package extract provides tests for JSON-LD matching logic.
// Tests cover type matching (single/array), case-insensitive matching, path traversal, and the All flag.
// Does NOT test JSON-LD extraction from HTML.
package extract

import (
	"testing"
)

func TestMatchJSONLDByType(t *testing.T) {
	documents := []map[string]any{
		{"@type": "Article", "headline": "Article Headline"},
		{"@type": "Product", "name": "Product Name"},
		{"@type": "Organization", "name": "Org Name"},
	}

	rule := JSONLDRule{
		Name: "headline",
		Type: "Article",
		Path: "headline",
	}

	matches := MatchJSONLD(documents, rule)

	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}

	if matches[0] != "Article Headline" {
		t.Errorf("expected match 'Article Headline', got %q", matches[0])
	}
}

func TestMatchJSONLDByTypeArray(t *testing.T) {
	documents := []map[string]any{
		{"@type": []any{"Article", "NewsArticle"}, "headline": "Multiple Types"},
		{"@type": "Product", "name": "Product Name"},
	}

	tests := []struct {
		name     string
		ruleType string
		expected string
	}{
		{
			name:     "match first type in array",
			ruleType: "Article",
			expected: "Multiple Types",
		},
		{
			name:     "match second type in array",
			ruleType: "NewsArticle",
			expected: "Multiple Types",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := JSONLDRule{
				Name: "headline",
				Type: tt.ruleType,
				Path: "headline",
			}

			matches := MatchJSONLD(documents, rule)

			if len(matches) != 1 {
				t.Fatalf("expected 1 match, got %d", len(matches))
			}

			if matches[0] != tt.expected {
				t.Errorf("expected match %q, got %q", tt.expected, matches[0])
			}
		})
	}
}

func TestMatchJSONLDByTypeCaseInsensitive(t *testing.T) {
	documents := []map[string]any{
		{"@type": "article", "headline": "lowercase type"},
		{"@type": "ARTICLE", "headline": "uppercase type"},
		{"@type": "ArTiClE", "headline": "mixed case type"},
	}

	rule := JSONLDRule{
		Name: "headline",
		Type: "ARTICLE",
		Path: "headline",
	}

	matches := MatchJSONLD(documents, rule)

	if len(matches) != 3 {
		t.Fatalf("expected 3 matches (case-insensitive), got %d", len(matches))
	}

	expectedMatches := []string{"lowercase type", "uppercase type", "mixed case type"}
	for i, match := range matches {
		if match != expectedMatches[i] {
			t.Errorf("match %d: expected %q, got %q", i, expectedMatches[i], match)
		}
	}
}

func TestMatchJSONLDNoType(t *testing.T) {
	documents := []map[string]any{
		{"@type": "Article", "headline": "Article Headline"},
		{"headline": "No Type Object"},
		{"@type": "Product", "name": "Product Name"},
	}

	rule := JSONLDRule{
		Name: "headline",
		Type: "",
		Path: "headline",
	}

	matches := MatchJSONLD(documents, rule)

	if len(matches) != 2 {
		t.Fatalf("expected 2 matches (no type filter), got %d", len(matches))
	}

	if matches[0] != "Article Headline" {
		t.Errorf("expected first match 'Article Headline', got %q", matches[0])
	}

	if matches[1] != "No Type Object" {
		t.Errorf("expected second match 'No Type Object', got %q", matches[1])
	}
}

func TestMatchJSONLDTypeNotMatching(t *testing.T) {
	documents := []map[string]any{
		{"@type": "Article", "headline": "Article Headline"},
		{"@type": "Product", "name": "Product Name"},
	}

	rule := JSONLDRule{
		Name: "headline",
		Type: "Organization",
		Path: "headline",
	}

	matches := MatchJSONLD(documents, rule)

	if len(matches) != 0 {
		t.Errorf("expected 0 matches (no matching type), got %d", len(matches))
	}
}

func TestMatchJSONLDPathTraversal(t *testing.T) {
	tests := []struct {
		name      string
		documents []map[string]any
		rule      JSONLDRule
		expected  []string
	}{
		{
			name: "simple dot path",
			documents: []map[string]any{
				{"@type": "Article", "headline": "Test Headline"},
			},
			rule: JSONLDRule{
				Name: "headline",
				Type: "Article",
				Path: "headline",
			},
			expected: []string{"Test Headline"},
		},
		{
			name: "nested path author.name",
			documents: []map[string]any{
				{"@type": "Article", "author": map[string]any{"@type": "Person", "name": "John Doe"}},
			},
			rule: JSONLDRule{
				Name: "author",
				Type: "Article",
				Path: "author.name",
			},
			expected: []string{"John Doe"},
		},
		{
			name: "nested path offers.price",
			documents: []map[string]any{
				{"@type": "Product", "offers": map[string]any{"@type": "Offer", "price": "99.99"}},
			},
			rule: JSONLDRule{
				Name: "price",
				Type: "Product",
				Path: "offers.price",
			},
			expected: []string{"99.99"},
		},
		{
			name: "deep nested path",
			documents: []map[string]any{
				{"@type": "Article", "author": map[string]any{"address": map[string]any{"city": "New York"}}},
			},
			rule: JSONLDRule{
				Name: "city",
				Type: "Article",
				Path: "author.address.city",
			},
			expected: []string{"New York"},
		},
		{
			name: "path that doesn't exist",
			documents: []map[string]any{
				{"@type": "Article", "headline": "Headline"},
			},
			rule: JSONLDRule{
				Name: "author",
				Type: "Article",
				Path: "author.name",
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := MatchJSONLD(tt.documents, tt.rule)

			if len(matches) != len(tt.expected) {
				t.Fatalf("expected %d matches, got %d", len(tt.expected), len(matches))
			}

			for i, match := range matches {
				if match != tt.expected[i] {
					t.Errorf("match %d: expected %q, got %q", i, tt.expected[i], match)
				}
			}
		})
	}
}

func TestMatchJSONLDAllFlag(t *testing.T) {
	documents := []map[string]any{
		{"@type": "Article", "headline": "First Article"},
		{"@type": "Article", "headline": "Second Article"},
		{"@type": "Article", "headline": "Third Article"},
		{"@type": "Product", "name": "Product"},
	}

	rule := JSONLDRule{
		Name: "headline",
		Type: "Article",
		Path: "headline",
		All:  true,
	}

	matches := MatchJSONLD(documents, rule)

	if len(matches) != 3 {
		t.Fatalf("expected 3 matches (All=true), got %d", len(matches))
	}

	expectedMatches := []string{"First Article", "Second Article", "Third Article"}
	for i, match := range matches {
		if match != expectedMatches[i] {
			t.Errorf("match %d: expected %q, got %q", i, expectedMatches[i], match)
		}
	}
}

func TestMatchJSONLDWithoutAllFlag(t *testing.T) {
	documents := []map[string]any{
		{"@type": "Article", "headline": "First Article"},
		{"@type": "Article", "headline": "Second Article"},
		{"@type": "Article", "headline": "Third Article"},
	}

	rule := JSONLDRule{
		Name: "headline",
		Type: "Article",
		Path: "headline",
		All:  false,
	}

	matches := MatchJSONLD(documents, rule)

	if len(matches) != 3 {
		t.Fatalf("expected 3 matches, got %d", len(matches))
	}
}
