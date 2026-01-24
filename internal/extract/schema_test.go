package extract

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateDocument(t *testing.T) {
	schema := &Schema{
		Type:     SchemaObject,
		Required: []string{"title", "price"},
		Properties: map[string]*Schema{
			"title": {
				Type:      SchemaString,
				MinLength: 5,
			},
			"price": {
				Type:    SchemaNumber,
				Minimum: floatPtr(0),
			},
			"quantity": {
				Type:    SchemaInteger,
				Minimum: floatPtr(0),
				Maximum: floatPtr(100),
			},
			"tags": {
				Type: SchemaArray,
				Items: &Schema{
					Type: SchemaString,
					Enum: []string{"new", "sale"},
				},
			},
		},
	}

	tests := []struct {
		name   string
		fields map[string]FieldValue
		valid  bool
	}{
		{
			name: "valid document",
			fields: map[string]FieldValue{
				"title":    {Values: []string{"Valid Title"}},
				"price":    {Values: []string{"19.99"}},
				"quantity": {Values: []string{"5"}},
				"tags":     {Values: []string{"new"}},
			},
			valid: true,
		},
		{
			name: "missing required",
			fields: map[string]FieldValue{
				"title": {Values: []string{"Valid Title"}},
			},
			valid: false,
		},
		{
			name: "invalid type (price not number)",
			fields: map[string]FieldValue{
				"title": {Values: []string{"Valid Title"}},
				"price": {Values: []string{"abc"}},
			},
			valid: false,
		},
		{
			name: "invalid constraint (title too short)",
			fields: map[string]FieldValue{
				"title": {Values: []string{"Hi"}},
				"price": {Values: []string{"10"}},
			},
			valid: false,
		},
		{
			name: "invalid enum",
			fields: map[string]FieldValue{
				"title": {Values: []string{"Valid Title"}},
				"price": {Values: []string{"10"}},
				"tags":  {Values: []string{"unknown"}},
			},
			valid: false,
		},
		{
			name: "valid integer field",
			fields: map[string]FieldValue{
				"title":    {Values: []string{"Valid Title"}},
				"price":    {Values: []string{"19.99"}},
				"quantity": {Values: []string{"42"}},
			},
			valid: true,
		},
		{
			name: "invalid integer (decimal provided)",
			fields: map[string]FieldValue{
				"title":    {Values: []string{"Valid Title"}},
				"price":    {Values: []string{"19.99"}},
				"quantity": {Values: []string{"5.5"}},
			},
			valid: false,
		},
		{
			name: "invalid integer (below minimum)",
			fields: map[string]FieldValue{
				"title":    {Values: []string{"Valid Title"}},
				"price":    {Values: []string{"19.99"}},
				"quantity": {Values: []string{"-1"}},
			},
			valid: false,
		},
		{
			name: "invalid integer (above maximum)",
			fields: map[string]FieldValue{
				"title":    {Values: []string{"Valid Title"}},
				"price":    {Values: []string{"19.99"}},
				"quantity": {Values: []string{"101"}},
			},
			valid: false,
		},
		{
			name: "invalid integer (not a number)",
			fields: map[string]FieldValue{
				"title":    {Values: []string{"Valid Title"}},
				"price":    {Values: []string{"19.99"}},
				"quantity": {Values: []string{"abc"}},
			},
			valid: false,
		},
		{
			name: "valid integer zero",
			fields: map[string]FieldValue{
				"title":    {Values: []string{"Valid Title"}},
				"price":    {Values: []string{"19.99"}},
				"quantity": {Values: []string{"0"}},
			},
			valid: true,
		},
		{
			name: "valid integer at boundary",
			fields: map[string]FieldValue{
				"title":    {Values: []string{"Valid Title"}},
				"price":    {Values: []string{"19.99"}},
				"quantity": {Values: []string{"100"}},
			},
			valid: true,
		},
		{
			name: "valid large integer (beyond int32)",
			fields: map[string]FieldValue{
				"title":    {Values: []string{"Valid Title"}},
				"price":    {Values: []string{"19.99"}},
				"quantity": {Values: []string{"2147483648"}}, // max int32 + 1
			},
			valid: false, // exceeds schema maximum of 100
		},
		{
			name: "valid scientific notation as integer",
			fields: map[string]FieldValue{
				"title":    {Values: []string{"Valid Title"}},
				"price":    {Values: []string{"19.99"}},
				"quantity": {Values: []string{"1e2"}}, // 100
			},
			valid: true,
		},
		{
			name: "valid scientific notation that results in whole number",
			fields: map[string]FieldValue{
				"title":    {Values: []string{"Valid Title"}},
				"price":    {Values: []string{"19.99"}},
				"quantity": {Values: []string{"1.5e1"}}, // 15.0
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := NormalizedDocument{
				Fields: tt.fields,
			}
			res := ValidateDocument(doc, schema)
			if res.Valid != tt.valid {
				t.Errorf("expected valid=%v, got %v (errors: %v)", tt.valid, res.Valid, res.Errors)
			}
		})
	}
}

func floatPtr(v float64) *float64 {
	return &v
}

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

	// Helper to create a FieldValue with RawObject
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
		errorCount int // -1 means don't check count
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
						// missing required "city"
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
						"zip":    {Values: []string{"abcde"}}, // invalid pattern
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
						"country": {Values: []string{"USA"}}, // unexpected field
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
			valid:      true, // backwards compatible - no error when RawObject is empty
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
						// "zip" is optional
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
								// missing required "name"
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
								"name": {Values: []string{""}}, // violates MinLength: 1
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
		// Error should include full path like "user.profile.name: too short"
		found := false
		for _, err := range res.Errors {
			if strings.Contains(err, "user.profile.name") {
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
							"price": {Values: []string{"-5"}}, // below minimum
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
							// missing required "price"
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

func TestValidateDocumentMixedNestedAndScalarFields(t *testing.T) {
	schema := &Schema{
		Type:     SchemaObject,
		Required: []string{"title"},
		Properties: map[string]*Schema{
			"title":   {Type: SchemaString, MinLength: 1},
			"price":   {Type: SchemaNumber, Minimum: floatPtr(0)},
			"inStock": {Type: SchemaBool},
			"vendor": {
				Type:     SchemaObject,
				Required: []string{"name"},
				Properties: map[string]*Schema{
					"name":    {Type: SchemaString, MinLength: 1},
					"address": {Type: SchemaString},
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

	t.Run("valid mixed fields", func(t *testing.T) {
		doc := NormalizedDocument{
			Fields: map[string]FieldValue{
				"title":   {Values: []string{"Product"}},
				"price":   {Values: []string{"19.99"}},
				"inStock": {Values: []string{"true"}},
				"vendor": {RawObject: mustRawObject(map[string]FieldValue{
					"name": {Values: []string{"Acme Corp"}},
				}), Source: FieldSourceSelector},
			},
		}
		res := ValidateDocument(doc, schema)
		if !res.Valid {
			t.Errorf("expected valid=true, got false (errors: %v)", res.Errors)
		}
	})

	t.Run("scalar field invalid", func(t *testing.T) {
		doc := NormalizedDocument{
			Fields: map[string]FieldValue{
				"title":   {Values: []string{"Product"}},
				"price":   {Values: []string{"invalid"}},
				"inStock": {Values: []string{"true"}},
				"vendor": {RawObject: mustRawObject(map[string]FieldValue{
					"name": {Values: []string{"Acme Corp"}},
				}), Source: FieldSourceSelector},
			},
		}
		res := ValidateDocument(doc, schema)
		if res.Valid {
			t.Errorf("expected valid=false, got true")
		}
	})
}

func TestNewObjectFieldValue(t *testing.T) {
	t.Run("creates valid RawObject", func(t *testing.T) {
		fields := map[string]FieldValue{
			"name": {Values: []string{"test"}},
		}
		fv, err := NewObjectFieldValue(fields, FieldSourceSelector)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fv.Source != FieldSourceSelector {
			t.Errorf("expected source=%v, got %v", FieldSourceSelector, fv.Source)
		}
		if fv.RawObject == "" {
			t.Error("expected RawObject to be set")
		}

		// Verify it can be unmarshaled back
		var unmarshaled map[string]FieldValue
		if err := json.Unmarshal([]byte(fv.RawObject), &unmarshaled); err != nil {
			t.Errorf("failed to unmarshal RawObject: %v", err)
		}
		if unmarshaled["name"].Values[0] != "test" {
			t.Errorf("unmarshaled data mismatch")
		}
	})
}

func TestValidateDocumentDepthLimit(t *testing.T) {
	// Create a deeply nested schema (more than maxValidationDepth)
	var createDeepSchema func(int) *Schema
	createDeepSchema = func(depth int) *Schema {
		if depth <= 0 {
			return &Schema{
				Type:     SchemaObject,
				Required: []string{"value"},
				Properties: map[string]*Schema{
					"value": {Type: SchemaString, MinLength: 1},
				},
			}
		}
		return &Schema{
			Type:     SchemaObject,
			Required: []string{"nested"},
			Properties: map[string]*Schema{
				"nested": createDeepSchema(depth - 1),
			},
		}
	}

	// Create a deeply nested document matching the schema
	var createDeepDoc func(int) map[string]FieldValue
	createDeepDoc = func(depth int) map[string]FieldValue {
		if depth <= 0 {
			return map[string]FieldValue{
				"value": {Values: []string{"deep value"}},
			}
		}
		nestedFV, _ := NewObjectFieldValue(createDeepDoc(depth-1), FieldSourceSelector)
		return map[string]FieldValue{
			"nested": nestedFV,
		}
	}

	t.Run("within_depth_limit", func(t *testing.T) {
		// 5 levels is within the limit of 10
		schema := createDeepSchema(5)
		doc := NormalizedDocument{Fields: createDeepDoc(5)}
		res := ValidateDocument(doc, schema)
		if !res.Valid {
			t.Errorf("expected valid=true for depth 5, got false (errors: %v)", res.Errors)
		}
	})

	t.Run("exceeds_depth_limit", func(t *testing.T) {
		// 15 levels exceeds the limit of 10
		schema := createDeepSchema(15)
		doc := NormalizedDocument{Fields: createDeepDoc(15)}
		res := ValidateDocument(doc, schema)
		if res.Valid {
			t.Errorf("expected valid=false for depth 15, got true")
		}
		// Should have at least one depth limit error
		found := false
		for _, err := range res.Errors {
			if strings.Contains(err, "validation depth exceeded") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected depth limit error, got: %v", res.Errors)
		}
	})

	t.Run("exactly_at_depth_limit", func(t *testing.T) {
		// 10 levels is exactly at the limit - should work
		schema := createDeepSchema(10)
		doc := NormalizedDocument{Fields: createDeepDoc(10)}
		res := ValidateDocument(doc, schema)
		if !res.Valid {
			t.Errorf("expected valid=true for depth 10, got false (errors: %v)", res.Errors)
		}
	})
}
