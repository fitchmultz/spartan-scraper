// Package extract provides tests for mixed scalar and nested field validation.
// Tests cover documents with both scalar and nested object fields in the same schema.
// Does NOT test deeply nested objects or arrays of objects.
package extract

import (
	"testing"
)

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
