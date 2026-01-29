package extract

import (
	"testing"
	"time"
)

func TestNormalizeBasic(t *testing.T) {
	extracted := Extracted{
		URL:         "http://example.com",
		Title:       "Original Title",
		Text:        "Original text",
		Links:       []string{"/link1", "/link2"},
		Fields:      map[string]FieldValue{},
		JSONLD:      []map[string]any{{"@type": "Article"}},
		Template:    "template1",
		ExtractedAt: time.Now(),
	}

	template := Template{
		Name:      "template2",
		Normalize: NormalizeSpec{},
	}

	result := Normalize(extracted, template)

	if result.URL != extracted.URL {
		t.Errorf("URL not copied: expected %q, got %q", extracted.URL, result.URL)
	}
	if len(result.Links) != len(extracted.Links) {
		t.Errorf("Links not copied: expected %d, got %d", len(extracted.Links), len(result.Links))
	}
	if result.Template != template.Name {
		t.Errorf("Template name not used: expected %q, got %q", template.Name, result.Template)
	}
	if !result.ExtractedAt.Equal(extracted.ExtractedAt) {
		t.Error("ExtractedAt not copied")
	}
	if result.Metadata == nil {
		t.Error("Metadata should be initialized")
	}
	if len(result.Metadata) != 0 {
		t.Errorf("Metadata should be empty, got %d entries", len(result.Metadata))
	}
	if result.Fields == nil {
		t.Error("Fields not preserved")
	}
	if len(result.JSONLD) != len(extracted.JSONLD) {
		t.Error("JSONLD not preserved")
	}
}

func TestNormalizeAllSpecs(t *testing.T) {
	extracted := Extracted{
		URL:   "http://example.com",
		Title: "Original Title",
		Text:  "Original text",
		Links: []string{},
		Fields: map[string]FieldValue{
			"headline":         {Values: []string{"Field Headline"}, Source: FieldSourceSelector},
			"summary":          {Values: []string{"Field Summary"}, Source: FieldSourceSelector},
			"body":             {Values: []string{"Field Body"}, Source: FieldSourceSelector},
			"author_name":      {Values: []string{"Jane Doe"}, Source: FieldSourceSelector},
			"published_date":   {Values: []string{"2024-01-01"}, Source: FieldSourceSelector},
			"article_category": {Values: []string{"Tech"}, Source: FieldSourceSelector},
		},
	}

	template := Template{
		Name: "test",
		Normalize: NormalizeSpec{
			TitleField:       "headline",
			DescriptionField: "summary",
			TextField:        "body",
			MetaFields: map[string]string{
				"author":        "author_name",
				"publishedDate": "published_date",
				"category":      "article_category",
			},
		},
	}

	result := Normalize(extracted, template)

	if result.Title != "Field Headline" {
		t.Errorf("expected title 'Field Headline', got %q", result.Title)
	}
	if result.Description != "Field Summary" {
		t.Errorf("expected description 'Field Summary', got %q", result.Description)
	}
	if result.Text != "Field Body" {
		t.Errorf("expected text 'Field Body', got %q", result.Text)
	}
	if result.Metadata["author"] != "Jane Doe" {
		t.Errorf("expected metadata author 'Jane Doe', got %q", result.Metadata["author"])
	}
	if result.Metadata["publishedDate"] != "2024-01-01" {
		t.Errorf("expected metadata publishedDate '2024-01-01', got %q", result.Metadata["publishedDate"])
	}
	if result.Metadata["category"] != "Tech" {
		t.Errorf("expected metadata category 'Tech', got %q", result.Metadata["category"])
	}
	if len(result.Metadata) != 3 {
		t.Errorf("expected 3 metadata entries, got %d", len(result.Metadata))
	}
}

func TestNormalizeMissingFields(t *testing.T) {
	tests := []struct {
		name      string
		extracted Extracted
		template  Template
	}{
		{
			name: "normalization when extracted.Fields is nil",
			extracted: Extracted{
				URL:    "http://example.com",
				Title:  "Original Title",
				Text:   "Original text",
				Links:  []string{},
				Fields: nil,
			},
			template: Template{
				Name: "test",
				Normalize: NormalizeSpec{
					TitleField:       "headline",
					DescriptionField: "summary",
					TextField:        "body",
					MetaFields: map[string]string{
						"author": "author_name",
					},
				},
			},
		},
		{
			name: "normalization when extracted.Fields is empty",
			extracted: Extracted{
				URL:    "http://example.com",
				Title:  "Original Title",
				Text:   "Original text",
				Links:  []string{},
				Fields: map[string]FieldValue{},
			},
			template: Template{
				Name: "test",
				Normalize: NormalizeSpec{
					TitleField:       "headline",
					DescriptionField: "summary",
					TextField:        "body",
					MetaFields: map[string]string{
						"author": "author_name",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Normalize(tt.extracted, tt.template)

			if result.Title != "Original Title" {
				t.Errorf("expected title to fall back to 'Original Title', got %q", result.Title)
			}
			if result.Description != "" {
				t.Errorf("expected description to be empty, got %q", result.Description)
			}
			if result.Text != "Original text" {
				t.Errorf("expected text to fall back to 'Original text', got %q", result.Text)
			}
			if len(result.Metadata) != 0 {
				t.Errorf("expected empty metadata, got %d entries", len(result.Metadata))
			}
		})
	}
}

func TestNormalizePreservesNonNormalizeFields(t *testing.T) {
	fields := map[string]FieldValue{
		"headline":    {Values: []string{"Headline"}, Source: FieldSourceSelector},
		"summary":     {Values: []string{"Summary"}, Source: FieldSourceSelector},
		"other_field": {Values: []string{"Other"}, Source: FieldSourceSelector},
	}
	jsonld := []map[string]any{
		{"@type": "Article", "headline": "Headline"},
		{"@type": "Organization", "name": "Org"},
	}

	extracted := Extracted{
		URL:    "http://example.com",
		Title:  "Title",
		Text:   "Text",
		Links:  []string{},
		Fields: fields,
		JSONLD: jsonld,
	}

	template := Template{
		Name: "test_template",
		Normalize: NormalizeSpec{
			TitleField: "headline",
		},
	}

	result := Normalize(extracted, template)

	if result.Fields == nil {
		t.Error("Fields should be preserved")
	}
	if !fieldsEqual(result.Fields, fields) {
		t.Error("Fields map was modified")
	}
	if len(result.JSONLD) != len(jsonld) {
		t.Errorf("JSONLD not preserved: expected %d, got %d", len(jsonld), len(result.JSONLD))
	}
	if result.Template != "test_template" {
		t.Errorf("expected template name 'test_template', got %q", result.Template)
	}
}

func TestNormalizeWithComplexFields(t *testing.T) {
	fields := map[string]FieldValue{
		"simple": {Values: []string{"Simple value"}, Source: FieldSourceSelector},
		"with_raw": {
			Values:    []string{"Raw object field"},
			Source:    FieldSourceJSONLD,
			RawObject: `{"nested":{"key":"value"}}`,
		},
		"multiple":     {Values: []string{"Val1", "Val2", "Val3"}, Source: FieldSourceSelector},
		"empty_values": {Values: []string{}, Source: FieldSourceSelector},
	}

	extracted := Extracted{
		URL:    "http://example.com",
		Title:  "Title",
		Text:   "Text",
		Links:  []string{},
		Fields: fields,
	}

	template := Template{
		Name: "test",
		Normalize: NormalizeSpec{
			TitleField: "simple",
			MetaFields: map[string]string{
				"author": "with_raw",
			},
		},
	}

	result := Normalize(extracted, template)

	if result.Title != "Simple value" {
		t.Errorf("expected title 'Simple value', got %q", result.Title)
	}
	if result.Metadata["author"] != "Raw object field" {
		t.Errorf("expected metadata author 'Raw object field', got %q", result.Metadata["author"])
	}

	if result.Fields == nil {
		t.Error("Fields should be preserved")
	}
	if result.Fields["simple"].Values[0] != "Simple value" {
		t.Error("simple field not preserved correctly")
	}
	if result.Fields["with_raw"].RawObject == "" {
		t.Error("RawObject not preserved")
	}
	if len(result.Fields["multiple"].Values) != 3 {
		t.Error("multiple values not preserved")
	}
}

func fieldsEqual(a, b map[string]FieldValue) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || len(v.Values) != len(bv.Values) {
			return false
		}
	}
	return true
}
