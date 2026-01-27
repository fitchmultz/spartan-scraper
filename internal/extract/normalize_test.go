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

func TestNormalizeTextFromField(t *testing.T) {
	tests := []struct {
		name      string
		textField string
		extracted Extracted
		expected  string
	}{
		{
			name:      "text from single value field",
			textField: "body",
			extracted: Extracted{
				URL:   "http://example.com",
				Title: "Title",
				Text:  "Original text",
				Links: []string{},
				Fields: map[string]FieldValue{
					"body": {Values: []string{"Body from field"}, Source: FieldSourceSelector},
				},
			},
			expected: "Body from field",
		},
		{
			name:      "text from first value when multiple",
			textField: "body",
			extracted: Extracted{
				URL:   "http://example.com",
				Title: "Title",
				Text:  "Original text",
				Links: []string{},
				Fields: map[string]FieldValue{
					"body": {Values: []string{"First paragraph", "Second paragraph"}, Source: FieldSourceSelector},
				},
			},
			expected: "First paragraph",
		},
		{
			name:      "TextField takes precedence over extracted.Text",
			textField: "text",
			extracted: Extracted{
				URL:   "http://example.com",
				Title: "Title",
				Text:  "Original text",
				Links: []string{},
				Fields: map[string]FieldValue{
					"text": {Values: []string{"Field text"}, Source: FieldSourceSelector},
				},
			},
			expected: "Field text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template := Template{
				Name: "test",
				Normalize: NormalizeSpec{
					TextField: tt.textField,
				},
			}

			result := Normalize(tt.extracted, template)

			if result.Text != tt.expected {
				t.Errorf("expected text %q, got %q", tt.expected, result.Text)
			}
		})
	}
}

func TestNormalizeTextFallback(t *testing.T) {
	tests := []struct {
		name      string
		textField string
		extracted Extracted
		expected  string
	}{
		{
			name:      "text falls back to extracted.Text when field not found",
			textField: "body",
			extracted: Extracted{
				URL:    "http://example.com",
				Title:  "Title",
				Text:   "Fallback text",
				Links:  []string{},
				Fields: map[string]FieldValue{},
			},
			expected: "Fallback text",
		},
		{
			name:      "text falls back to extracted.Text when field has no values",
			textField: "body",
			extracted: Extracted{
				URL:   "http://example.com",
				Title: "Title",
				Text:  "Fallback text",
				Links: []string{},
				Fields: map[string]FieldValue{
					"body": {Values: []string{}, Source: FieldSourceSelector},
				},
			},
			expected: "Fallback text",
		},
		{
			name:      "text falls back to extracted.Text when TextField is empty",
			textField: "",
			extracted: Extracted{
				URL:   "http://example.com",
				Title: "Title",
				Text:  "Fallback text",
				Links: []string{},
				Fields: map[string]FieldValue{
					"body": {Values: []string{"Body"}, Source: FieldSourceSelector},
				},
			},
			expected: "Fallback text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template := Template{
				Name: "test",
				Normalize: NormalizeSpec{
					TextField: tt.textField,
				},
			}

			result := Normalize(tt.extracted, template)

			if result.Text != tt.expected {
				t.Errorf("expected text %q, got %q", tt.expected, result.Text)
			}
		})
	}
}

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
