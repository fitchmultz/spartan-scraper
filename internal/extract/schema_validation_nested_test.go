// Package extract provides tests for nested object and array validation.
// Tests cover nested objects, deeply nested paths, arrays of objects, and additionalProperties constraints.
// Does NOT test basic scalar type validation.
package extract

import (
	"testing"
)

func TestValidateDocumentNestedObjects(t *testing.T) {
	schema := &Schema{
		Type:     SchemaObject,
		Required: []string{"title", "address"},
		Properties: map[string]*Schema{
			"title": {Type: SchemaString, MinLength: 1},
			"address": {
				Type:     SchemaObject,
				Required: []string{"street", "city"},
				Properties: map[string]*Schema{
					"street": {Type: SchemaString, MinLength: 1},
					"city":   {Type: SchemaString, MinLength: 1},
					"zip":    {Type: SchemaString, Pattern: `^\d{5}$`},
				},
				AdditionalProperties: false,
			},
		},
	}

	mustRawObject := func(fields map[string]FieldValue) string {
		fv, err := NewObjectFieldValue(fields, FieldSourceSelector)
		if err != nil {
			t.Fatalf("failed to create RawObject: %v", err)
		}
		return fv.RawObject
	}

	tests := []struct {
		name       string
		fields     map[string]FieldValue
		valid      bool
		errorCount int
	}{
		{
			name: "valid nested object",
			fields: map[string]FieldValue{
				"title": {Values: []string{"Product Title"}},
				"address": {
					RawObject: mustRawObject(map[string]FieldValue{
						"street": {Values: []string{"123 Main St"}},
						"city":   {Values: []string{"Springfield"}},
						"zip":    {Values: []string{"12345"}},
					}),
					Source: FieldSourceSelector,
				},
			},
			valid:      true,
			errorCount: 0,
		},
		{
			name: "missing required nested field",
			fields: map[string]FieldValue{
				"title": {Values: []string{"Product Title"}},
				"address": {
					RawObject: mustRawObject(map[string]FieldValue{
						"street": {Values: []string{"123 Main St"}},
					}),
					Source: FieldSourceSelector,
				},
			},
			valid:      false,
			errorCount: 1,
		},
		{
			name: "invalid nested field constraint",
			fields: map[string]FieldValue{
				"title": {Values: []string{"Product Title"}},
				"address": {
					RawObject: mustRawObject(map[string]FieldValue{
						"street": {Values: []string{"123 Main St"}},
						"city":   {Values: []string{"Springfield"}},
						"zip":    {Values: []string{"abcde"}},
					}),
					Source: FieldSourceSelector,
				},
			},
			valid:      false,
			errorCount: 1,
		},
		{
			name: "unexpected nested field with additionalProperties false",
			fields: map[string]FieldValue{
				"title": {Values: []string{"Product Title"}},
				"address": {
					RawObject: mustRawObject(map[string]FieldValue{
						"street":  {Values: []string{"123 Main St"}},
						"city":    {Values: []string{"Springfield"}},
						"country": {Values: []string{"USA"}},
					}),
					Source: FieldSourceSelector,
				},
			},
			valid:      false,
			errorCount: 1,
		},
		{
			name: "backwards compatibility - empty RawObject with Values",
			fields: map[string]FieldValue{
				"title": {Values: []string{"Product Title"}},
				"address": {
					Values: []string{"some value"},
					Source: FieldSourceSelector,
				},
			},
			valid:      true,
			errorCount: 0,
		},
		{
			name: "invalid JSON in RawObject",
			fields: map[string]FieldValue{
				"title": {Values: []string{"Product Title"}},
				"address": {
					RawObject: "{invalid json",
					Source:    FieldSourceSelector,
				},
			},
			valid:      false,
			errorCount: 1,
		},
		{
			name: "nested object with all optional fields missing",
			fields: map[string]FieldValue{
				"title": {Values: []string{"Product Title"}},
				"address": {
					RawObject: mustRawObject(map[string]FieldValue{
						"street": {Values: []string{"123 Main St"}},
						"city":   {Values: []string{"Springfield"}},
					}),
					Source: FieldSourceSelector,
				},
			},
			valid:      true,
			errorCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := NormalizedDocument{Fields: tt.fields}
			res := ValidateDocument(doc, schema)
			if res.Valid != tt.valid {
				t.Errorf("expected valid=%v, got %v (errors: %v)", tt.valid, res.Valid, res.Errors)
			}
			if tt.errorCount >= 0 && len(res.Errors) != tt.errorCount {
				t.Errorf("expected %d errors, got %d: %v", tt.errorCount, len(res.Errors), res.Errors)
			}
		})
	}
}

func TestValidateDocumentDeeplyNestedObjects(t *testing.T) {
	schema := &Schema{
		Type:     SchemaObject,
		Required: []string{"user"},
		Properties: map[string]*Schema{
			"user": {
				Type:     SchemaObject,
				Required: []string{"profile"},
				Properties: map[string]*Schema{
					"profile": {
						Type:     SchemaObject,
						Required: []string{"name"},
						Properties: map[string]*Schema{
							"name": {Type: SchemaString, MinLength: 1},
							"address": {
								Type: SchemaObject,
								Properties: map[string]*Schema{
									"city": {Type: SchemaString},
								},
							},
						},
					},
				},
			},
		},
	}

	mustRawObject := func(fields map[string]FieldValue) string {
		fv, err := NewObjectFieldValue(fields, FieldSourceSelector)
		if err != nil {
			t.Fatalf("failed to create RawObject: %v", err)
		}
		return fv.RawObject
	}

	t.Run("valid deeply nested object", func(t *testing.T) {
		doc := NormalizedDocument{
			Fields: map[string]FieldValue{
				"user": {
					RawObject: mustRawObject(map[string]FieldValue{
						"profile": {
							RawObject: mustRawObject(map[string]FieldValue{
								"name": {Values: []string{"John Doe"}},
								"address": {
									RawObject: mustRawObject(map[string]FieldValue{
										"city": {Values: []string{"Boston"}},
									}),
								},
							}),
						},
					}),
					Source: FieldSourceSelector,
				},
			},
		}
		res := ValidateDocument(doc, schema)
		if !res.Valid {
			t.Errorf("expected valid=true, got false (errors: %v)", res.Errors)
		}
	})

	t.Run("missing required field at depth 3", func(t *testing.T) {
		doc := NormalizedDocument{
			Fields: map[string]FieldValue{
				"user": {
					RawObject: mustRawObject(map[string]FieldValue{
						"profile": {
							RawObject: mustRawObject(map[string]FieldValue{
								"address": {
									RawObject: mustRawObject(map[string]FieldValue{
										"city": {Values: []string{"Boston"}},
									}),
								},
							}),
						},
					}),
					Source: FieldSourceSelector,
				},
			},
		}
		res := ValidateDocument(doc, schema)
		if res.Valid {
			t.Errorf("expected valid=false, got true")
		}
		if len(res.Errors) != 1 {
			t.Errorf("expected 1 error, got %d: %v", len(res.Errors), res.Errors)
		}
	})

	t.Run("error path includes full nested path", func(t *testing.T) {
		doc := NormalizedDocument{
			Fields: map[string]FieldValue{
				"user": {
					RawObject: mustRawObject(map[string]FieldValue{
						"profile": {
							RawObject: mustRawObject(map[string]FieldValue{
								"name": {Values: []string{""}},
							}),
						},
					}),
					Source: FieldSourceSelector,
				},
			},
		}
		res := ValidateDocument(doc, schema)
		if res.Valid {
			t.Errorf("expected valid=false, got true")
		}
		found := false
		for _, err := range res.Errors {
			if len(err) > 0 && (err[0:len("user.profile.name")] == "user.profile.name") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected error path to contain 'user.profile.name', got: %v", res.Errors)
		}
	})
}

func TestValidateDocumentArrayOfObjects(t *testing.T) {
	schema := &Schema{
		Type: SchemaObject,
		Properties: map[string]*Schema{
			"items": {
				Type: SchemaArray,
				Items: &Schema{
					Type:     SchemaObject,
					Required: []string{"name", "price"},
					Properties: map[string]*Schema{
						"name":  {Type: SchemaString, MinLength: 1},
						"price": {Type: SchemaNumber, Minimum: floatPtr(0)},
					},
				},
			},
		},
	}

	mustRawObject := func(fields map[string]FieldValue) string {
		fv, err := NewObjectFieldValue(fields, FieldSourceSelector)
		if err != nil {
			t.Fatalf("failed to create RawObject: %v", err)
		}
		return fv.RawObject
	}

	t.Run("valid array of objects", func(t *testing.T) {
		doc := NormalizedDocument{
			Fields: map[string]FieldValue{
				"items": {
					Values: []string{
						mustRawObject(map[string]FieldValue{
							"name":  {Values: []string{"item1"}},
							"price": {Values: []string{"10"}},
						}),
						mustRawObject(map[string]FieldValue{
							"name":  {Values: []string{"item2"}},
							"price": {Values: []string{"20"}},
						}),
					},
					Source: FieldSourceSelector,
				},
			},
		}
		res := ValidateDocument(doc, schema)
		if !res.Valid {
			t.Errorf("expected valid=true, got false (errors: %v)", res.Errors)
		}
	})

	t.Run("invalid item in array", func(t *testing.T) {
		doc := NormalizedDocument{
			Fields: map[string]FieldValue{
				"items": {
					Values: []string{
						mustRawObject(map[string]FieldValue{
							"name":  {Values: []string{"item1"}},
							"price": {Values: []string{"10"}},
						}),
						mustRawObject(map[string]FieldValue{
							"name":  {Values: []string{"item2"}},
							"price": {Values: []string{"-5"}},
						}),
					},
					Source: FieldSourceSelector,
				},
			},
		}
		res := ValidateDocument(doc, schema)
		if res.Valid {
			t.Errorf("expected valid=false, got true")
		}
	})

	t.Run("array item missing required field", func(t *testing.T) {
		doc := NormalizedDocument{
			Fields: map[string]FieldValue{
				"items": {
					Values: []string{
						mustRawObject(map[string]FieldValue{
							"name": {Values: []string{"item1"}},
						}),
					},
					Source: FieldSourceSelector,
				},
			},
		}
		res := ValidateDocument(doc, schema)
		if res.Valid {
			t.Errorf("expected valid=false, got true")
		}
	})
}
