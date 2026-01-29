package extract

import (
	"testing"
)

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
