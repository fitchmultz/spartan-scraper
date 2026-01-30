// Package extract provides tests for title normalization.
// Tests cover TitleField mapping and fallback to extracted.Title.
// Does NOT test description or text normalization.
package extract

import (
	"testing"
)

func TestNormalizeTitleFromField(t *testing.T) {
	tests := []struct {
		name          string
		titleField    string
		extracted     Extracted
		expectedTitle string
	}{
		{
			name:       "title from single value field",
			titleField: "headline",
			extracted: Extracted{
				URL:   "http://example.com",
				Title: "Original Title",
				Text:  "Original text",
				Links: []string{},
				Fields: map[string]FieldValue{
					"headline": {Values: []string{"Headline from field"}, Source: FieldSourceSelector},
				},
			},
			expectedTitle: "Headline from field",
		},
		{
			name:       "title from first value when multiple",
			titleField: "headline",
			extracted: Extracted{
				URL:   "http://example.com",
				Title: "Original Title",
				Text:  "Original text",
				Links: []string{},
				Fields: map[string]FieldValue{
					"headline": {Values: []string{"First", "Second", "Third"}, Source: FieldSourceSelector},
				},
			},
			expectedTitle: "First",
		},
		{
			name:       "TitleField takes precedence over extracted.Title",
			titleField: "title",
			extracted: Extracted{
				URL:   "http://example.com",
				Title: "Original Title",
				Text:  "Original text",
				Links: []string{},
				Fields: map[string]FieldValue{
					"title": {Values: []string{"Field Title"}, Source: FieldSourceSelector},
				},
			},
			expectedTitle: "Field Title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template := Template{
				Name: "test",
				Normalize: NormalizeSpec{
					TitleField: tt.titleField,
				},
			}

			result := Normalize(tt.extracted, template)

			if result.Title != tt.expectedTitle {
				t.Errorf("expected title %q, got %q", tt.expectedTitle, result.Title)
			}
		})
	}
}

func TestNormalizeTitleFallback(t *testing.T) {
	tests := []struct {
		name          string
		titleField    string
		extracted     Extracted
		expectedTitle string
	}{
		{
			name:       "title falls back to extracted.Title when field not found",
			titleField: "headline",
			extracted: Extracted{
				URL:    "http://example.com",
				Title:  "Fallback Title",
				Text:   "Original text",
				Links:  []string{},
				Fields: map[string]FieldValue{},
			},
			expectedTitle: "Fallback Title",
		},
		{
			name:       "title falls back to extracted.Title when field has no values",
			titleField: "headline",
			extracted: Extracted{
				URL:   "http://example.com",
				Title: "Fallback Title",
				Text:  "Original text",
				Links: []string{},
				Fields: map[string]FieldValue{
					"headline": {Values: []string{}, Source: FieldSourceSelector},
				},
			},
			expectedTitle: "Fallback Title",
		},
		{
			name:       "title falls back to extracted.Title when TitleField is empty",
			titleField: "",
			extracted: Extracted{
				URL:   "http://example.com",
				Title: "Fallback Title",
				Text:  "Original text",
				Links: []string{},
				Fields: map[string]FieldValue{
					"headline": {Values: []string{"Headline"}, Source: FieldSourceSelector},
				},
			},
			expectedTitle: "Fallback Title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template := Template{
				Name: "test",
				Normalize: NormalizeSpec{
					TitleField: tt.titleField,
				},
			}

			result := Normalize(tt.extracted, template)

			if result.Title != tt.expectedTitle {
				t.Errorf("expected title %q, got %q", tt.expectedTitle, result.Title)
			}
		})
	}
}
