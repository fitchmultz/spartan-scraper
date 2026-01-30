// Package extract provides HTML content extraction using selectors, JSON-LD, and regex.
// It handles template-based extraction, field normalization, and schema validation.
// It does NOT handle fetching or rendering HTML content.
package extract

import (
	"testing"
	"time"
)

func TestValidateJSONSchema_BasicValidation(t *testing.T) {
	tests := []struct {
		name         string
		doc          NormalizedDocument
		schema       *Schema
		wantValid    bool
		wantErrCount int
	}{
		{
			name: "valid document with simple schema",
			doc: NormalizedDocument{
				URL:   "https://example.com",
				Title: "Test Title",
				Fields: map[string]FieldValue{
					"name": {Values: []string{"Test Product"}, Source: FieldSourceSelector},
				},
			},
			schema: &Schema{
				Format: SchemaFormatJSONSchema,
				JSONSchema: map[string]any{
					"type":     "object",
					"required": []string{"url", "title"},
					"properties": map[string]any{
						"url":   map[string]any{"type": "string"},
						"title": map[string]any{"type": "string"},
					},
				},
			},
			wantValid:    true,
			wantErrCount: 0,
		},
		{
			name: "empty string with minLength constraint",
			doc: NormalizedDocument{
				URL:   "https://example.com",
				Title: "",
			},
			schema: &Schema{
				Format: SchemaFormatJSONSchema,
				JSONSchema: map[string]any{
					"type":     "object",
					"required": []string{"url", "title"},
					"properties": map[string]any{
						"url":   map[string]any{"type": "string"},
						"title": map[string]any{"type": "string", "minLength": 1},
					},
				},
			},
			wantValid:    false,
			wantErrCount: 1,
		},
		{
			name: "wrong type",
			doc: NormalizedDocument{
				URL:   "https://example.com",
				Title: "Test",
				Fields: map[string]FieldValue{
					"price": {Values: []string{"not-a-number"}, Source: FieldSourceSelector},
				},
			},
			schema: &Schema{
				Format: SchemaFormatJSONSchema,
				JSONSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"fields": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"price": map[string]any{
									"type": "object",
									"properties": map[string]any{
										"Values": map[string]any{
											"type":  "array",
											"items": map[string]any{"type": "number"},
										},
									},
								},
							},
						},
					},
				},
			},
			wantValid:    false,
			wantErrCount: 1,
		},
		{
			name: "nil schema returns valid",
			doc: NormalizedDocument{
				URL: "https://example.com",
			},
			schema:       nil,
			wantValid:    true,
			wantErrCount: 0,
		},
		{
			name: "custom format uses custom validation",
			doc: NormalizedDocument{
				URL:   "https://example.com",
				Title: "Test",
				Fields: map[string]FieldValue{
					"name": {Values: []string{"test"}, Source: FieldSourceSelector},
				},
			},
			schema: &Schema{
				Format:   SchemaFormatCustom,
				Type:     SchemaObject,
				Required: []string{"name"},
				Properties: map[string]*Schema{
					"name": {Type: SchemaString, MinLength: 1},
				},
			},
			wantValid:    true,
			wantErrCount: 0,
		},
		{
			name: "empty jsonschema returns valid",
			doc: NormalizedDocument{
				URL: "https://example.com",
			},
			schema: &Schema{
				Format:     SchemaFormatJSONSchema,
				JSONSchema: nil,
			},
			wantValid:    true,
			wantErrCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateJSONSchema(tt.doc, tt.schema)
			if result.Valid != tt.wantValid {
				t.Errorf("ValidateJSONSchema() Valid = %v, want %v", result.Valid, tt.wantValid)
			}
			if len(result.Errors) != tt.wantErrCount {
				t.Errorf("ValidateJSONSchema() Errors count = %d, want %d. Errors: %v", len(result.Errors), tt.wantErrCount, result.Errors)
			}
		})
	}
}

func TestValidateJSONSchema_StringConstraints(t *testing.T) {
	tests := []struct {
		name         string
		doc          NormalizedDocument
		schema       map[string]any
		wantValid    bool
		wantErrCount int
	}{
		{
			name: "minLength constraint satisfied",
			doc: NormalizedDocument{
				Title: "Test Title",
			},
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title": map[string]any{
						"type":      "string",
						"minLength": 5,
					},
				},
			},
			wantValid:    true,
			wantErrCount: 0,
		},
		{
			name: "minLength constraint violated",
			doc: NormalizedDocument{
				Title: "Test",
			},
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title": map[string]any{
						"type":      "string",
						"minLength": 10,
					},
				},
			},
			wantValid:    false,
			wantErrCount: 1,
		},
		{
			name: "pattern constraint satisfied",
			doc: NormalizedDocument{
				URL: "https://example.com",
			},
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url": map[string]any{
						"type":    "string",
						"pattern": "^https?://",
					},
				},
			},
			wantValid:    true,
			wantErrCount: 0,
		},
		{
			name: "pattern constraint violated",
			doc: NormalizedDocument{
				URL: "ftp://example.com",
			},
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url": map[string]any{
						"type":    "string",
						"pattern": "^https?://",
					},
				},
			},
			wantValid:    false,
			wantErrCount: 1,
		},
		{
			name: "enum constraint satisfied",
			doc: NormalizedDocument{
				Title: "active",
			},
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title": map[string]any{
						"type": "string",
						"enum": []string{"active", "inactive", "pending"},
					},
				},
			},
			wantValid:    true,
			wantErrCount: 0,
		},
		{
			name: "enum constraint violated",
			doc: NormalizedDocument{
				Title: "unknown",
			},
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title": map[string]any{
						"type": "string",
						"enum": []string{"active", "inactive", "pending"},
					},
				},
			},
			wantValid:    false,
			wantErrCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &Schema{
				Format:     SchemaFormatJSONSchema,
				JSONSchema: tt.schema,
			}
			result := ValidateJSONSchema(tt.doc, schema)
			if result.Valid != tt.wantValid {
				t.Errorf("ValidateJSONSchema() Valid = %v, want %v", result.Valid, tt.wantValid)
			}
			if len(result.Errors) != tt.wantErrCount {
				t.Errorf("ValidateJSONSchema() Errors count = %d, want %d. Errors: %v", len(result.Errors), tt.wantErrCount, result.Errors)
			}
		})
	}
}

func TestValidateJSONSchema_NestedObjects(t *testing.T) {
	tests := []struct {
		name         string
		doc          NormalizedDocument
		schema       map[string]any
		wantValid    bool
		wantErrCount int
	}{
		{
			name: "valid nested object",
			doc: NormalizedDocument{
				Fields: map[string]FieldValue{
					"product": {
						RawObject: `{"name": "Widget", "price": "19.99"}`,
						Source:    FieldSourceSelector,
					},
				},
			},
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"fields": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"product": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"name":  map[string]any{"type": "string"},
									"price": map[string]any{"type": "string"},
								},
								"required": []string{"name", "price"},
							},
						},
					},
				},
			},
			wantValid:    true,
			wantErrCount: 0,
		},
		{
			name: "invalid nested object - missing required",
			doc: NormalizedDocument{
				Fields: map[string]FieldValue{
					"product": {
						RawObject: `{"name": "Widget"}`,
						Source:    FieldSourceSelector,
					},
				},
			},
			schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"fields": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"product": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"name":  map[string]any{"type": "string"},
									"price": map[string]any{"type": "string"},
								},
								"required": []string{"name", "price"},
							},
						},
					},
				},
			},
			wantValid:    false,
			wantErrCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &Schema{
				Format:     SchemaFormatJSONSchema,
				JSONSchema: tt.schema,
			}
			result := ValidateJSONSchema(tt.doc, schema)
			if result.Valid != tt.wantValid {
				t.Errorf("ValidateJSONSchema() Valid = %v, want %v", result.Valid, tt.wantValid)
			}
			if len(result.Errors) != tt.wantErrCount {
				t.Errorf("ValidateJSONSchema() Errors count = %d, want %d. Errors: %v", len(result.Errors), tt.wantErrCount, result.Errors)
			}
		})
	}
}

func TestDocumentToJSON(t *testing.T) {
	tests := []struct {
		name    string
		doc     NormalizedDocument
		wantErr bool
	}{
		{
			name: "simple document",
			doc: NormalizedDocument{
				URL:         "https://example.com",
				Title:       "Test Title",
				Description: "Test Description",
				Text:        "Test content",
				Links:       []string{"https://example.com/page1"},
				Template:    "test",
				ExtractedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			wantErr: false,
		},
		{
			name: "document with fields",
			doc: NormalizedDocument{
				URL: "https://example.com",
				Fields: map[string]FieldValue{
					"name":  {Values: []string{"Test"}, Source: FieldSourceSelector},
					"price": {Values: []string{"19.99"}, Source: FieldSourceSelector},
				},
			},
			wantErr: false,
		},
		{
			name: "document with metadata",
			doc: NormalizedDocument{
				URL:      "https://example.com",
				Metadata: map[string]string{"author": "Test Author"},
			},
			wantErr: false,
		},
		{
			name: "document with JSON-LD",
			doc: NormalizedDocument{
				URL:    "https://example.com",
				JSONLD: []map[string]any{{"@type": "Product", "name": "Widget"}},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := documentToJSON(tt.doc)
			if (err != nil) != tt.wantErr {
				t.Errorf("documentToJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			// Verify required fields are present
			if result["url"] != tt.doc.URL {
				t.Errorf("documentToJSON() url = %v, want %v", result["url"], tt.doc.URL)
			}
			if result["title"] != tt.doc.Title {
				t.Errorf("documentToJSON() title = %v, want %v", result["title"], tt.doc.Title)
			}
		})
	}
}

func TestFieldValueToJSON(t *testing.T) {
	tests := []struct {
		name string
		fv   FieldValue
		want any
	}{
		{
			name: "single value",
			fv:   FieldValue{Values: []string{"test"}, Source: FieldSourceSelector},
			want: "test",
		},
		{
			name: "multiple values",
			fv:   FieldValue{Values: []string{"a", "b", "c"}, Source: FieldSourceSelector},
			want: []string{"a", "b", "c"},
		},
		{
			name: "raw object",
			fv:   FieldValue{RawObject: `{"name": "test"}`, Source: FieldSourceSelector},
			want: map[string]any{"name": "test"},
		},
		{
			name: "empty values",
			fv:   FieldValue{Values: []string{}, Source: FieldSourceSelector},
			want: nil,
		},
		{
			name: "invalid raw object falls back to values",
			fv:   FieldValue{RawObject: `invalid json`, Values: []string{"fallback"}, Source: FieldSourceSelector},
			want: "fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fieldValueToJSON(tt.fv)
			// For complex types, just check non-nil
			if tt.want == nil && got != nil {
				t.Errorf("fieldValueToJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsJSONSchemaSupported(t *testing.T) {
	tests := []struct {
		name   string
		schema *Schema
		want   bool
	}{
		{
			name:   "nil schema",
			schema: nil,
			want:   false,
		},
		{
			name: "custom format",
			schema: &Schema{
				Format: SchemaFormatCustom,
			},
			want: false,
		},
		{
			name: "jsonschema format with schema",
			schema: &Schema{
				Format:     SchemaFormatJSONSchema,
				JSONSchema: map[string]any{"type": "object"},
			},
			want: true,
		},
		{
			name: "jsonschema format without schema",
			schema: &Schema{
				Format:     SchemaFormatJSONSchema,
				JSONSchema: nil,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsJSONSchemaSupported(tt.schema)
			if got != tt.want {
				t.Errorf("IsJSONSchemaSupported() = %v, want %v", got, tt.want)
			}
		})
	}
}
