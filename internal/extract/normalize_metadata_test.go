// Package extract provides tests for metadata field normalization.
// Tests cover single and multiple meta field mappings, first-value selection, and missing field handling.
// Does NOT test title, description, or text normalization.
package extract

import (
	"testing"
)

func TestNormalizeMetaFields(t *testing.T) {
	tests := []struct {
		name       string
		metaFields map[string]string
		extracted  Extracted
		wantMeta   map[string]string
	}{
		{
			name: "single meta field mapping",
			metaFields: map[string]string{
				"author": "author_name",
			},
			extracted: Extracted{
				URL:   "http://example.com",
				Title: "Title",
				Text:  "Text",
				Links: []string{},
				Fields: map[string]FieldValue{
					"author_name": {Values: []string{"John Doe"}, Source: FieldSourceSelector},
				},
			},
			wantMeta: map[string]string{
				"author": "John Doe",
			},
		},
		{
			name: "multiple meta field mappings",
			metaFields: map[string]string{
				"author":        "author_name",
				"publishedDate": "date",
				"category":      "article_category",
			},
			extracted: Extracted{
				URL:   "http://example.com",
				Title: "Title",
				Text:  "Text",
				Links: []string{},
				Fields: map[string]FieldValue{
					"author_name":      {Values: []string{"Jane Doe"}, Source: FieldSourceSelector},
					"date":             {Values: []string{"2024-01-01"}, Source: FieldSourceSelector},
					"article_category": {Values: []string{"Tech"}, Source: FieldSourceSelector},
				},
			},
			wantMeta: map[string]string{
				"author":        "Jane Doe",
				"publishedDate": "2024-01-01",
				"category":      "Tech",
			},
		},
		{
			name: "first value is used when field has multiple values",
			metaFields: map[string]string{
				"author": "authors",
			},
			extracted: Extracted{
				URL:   "http://example.com",
				Title: "Title",
				Text:  "Text",
				Links: []string{},
				Fields: map[string]FieldValue{
					"authors": {Values: []string{"Author 1", "Author 2", "Author 3"}, Source: FieldSourceSelector},
				},
			},
			wantMeta: map[string]string{
				"author": "Author 1",
			},
		},
		{
			name: "meta fields not set when source field not found",
			metaFields: map[string]string{
				"author":        "author_name",
				"publishedDate": "date",
				"category":      "article_category",
			},
			extracted: Extracted{
				URL:   "http://example.com",
				Title: "Title",
				Text:  "Text",
				Links: []string{},
				Fields: map[string]FieldValue{
					"author_name": {Values: []string{"Jane Doe"}, Source: FieldSourceSelector},
				},
			},
			wantMeta: map[string]string{
				"author": "Jane Doe",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template := Template{
				Name: "test",
				Normalize: NormalizeSpec{
					MetaFields: tt.metaFields,
				},
			}

			result := Normalize(tt.extracted, template)

			for key, expectedValue := range tt.wantMeta {
				if result.Metadata[key] != expectedValue {
					t.Errorf("metadata[%q]: expected %q, got %q", key, expectedValue, result.Metadata[key])
				}
			}

			if len(result.Metadata) != len(tt.wantMeta) {
				t.Errorf("expected %d metadata entries, got %d", len(tt.wantMeta), len(result.Metadata))
			}
		})
	}
}
