package extract

import (
	"testing"
)

func TestNormalizeDescriptionFromField(t *testing.T) {
	tests := []struct {
		name             string
		descriptionField string
		extracted        Extracted
		expectedDesc     string
	}{
		{
			name:             "description from single value field",
			descriptionField: "summary",
			extracted: Extracted{
				URL:   "http://example.com",
				Title: "Title",
				Text:  "Original text",
				Links: []string{},
				Fields: map[string]FieldValue{
					"summary": {Values: []string{"Summary from field"}, Source: FieldSourceSelector},
				},
			},
			expectedDesc: "Summary from field",
		},
		{
			name:             "description from first value when multiple",
			descriptionField: "summary",
			extracted: Extracted{
				URL:   "http://example.com",
				Title: "Title",
				Text:  "Original text",
				Links: []string{},
				Fields: map[string]FieldValue{
					"summary": {Values: []string{"First", "Second"}, Source: FieldSourceSelector},
				},
			},
			expectedDesc: "First",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template := Template{
				Name: "test",
				Normalize: NormalizeSpec{
					DescriptionField: tt.descriptionField,
				},
			}

			result := Normalize(tt.extracted, template)

			if result.Description != tt.expectedDesc {
				t.Errorf("expected description %q, got %q", tt.expectedDesc, result.Description)
			}
		})
	}
}

func TestNormalizeDescriptionFallback(t *testing.T) {
	tests := []struct {
		name             string
		descriptionField string
		extracted        Extracted
		expectedDesc     string
	}{
		{
			name:             "description falls back to 'description' field",
			descriptionField: "summary",
			extracted: Extracted{
				URL:   "http://example.com",
				Title: "Title",
				Text:  "Original text",
				Links: []string{},
				Fields: map[string]FieldValue{
					"description": {Values: []string{"Fallback description"}, Source: FieldSourceSelector},
				},
			},
			expectedDesc: "Fallback description",
		},
		{
			name:             "description falls back to empty string when no description field",
			descriptionField: "summary",
			extracted: Extracted{
				URL:    "http://example.com",
				Title:  "Title",
				Text:   "Original text",
				Links:  []string{},
				Fields: map[string]FieldValue{},
			},
			expectedDesc: "",
		},
		{
			name:             "no panic when description field has no values",
			descriptionField: "",
			extracted: Extracted{
				URL:   "http://example.com",
				Title: "Title",
				Text:  "Original text",
				Links: []string{},
				Fields: map[string]FieldValue{
					"description": {Values: []string{}, Source: FieldSourceSelector},
				},
			},
			expectedDesc: "",
		},
		{
			name:             "description field exists but has no values",
			descriptionField: "summary",
			extracted: Extracted{
				URL:   "http://example.com",
				Title: "Title",
				Text:  "Original text",
				Links: []string{},
				Fields: map[string]FieldValue{
					"summary": {Values: []string{}, Source: FieldSourceSelector},
				},
			},
			expectedDesc: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template := Template{
				Name: "test",
				Normalize: NormalizeSpec{
					DescriptionField: tt.descriptionField,
				},
			}

			result := Normalize(tt.extracted, template)

			if result.Description != tt.expectedDesc {
				t.Errorf("expected description %q, got %q", tt.expectedDesc, result.Description)
			}
		})
	}
}
