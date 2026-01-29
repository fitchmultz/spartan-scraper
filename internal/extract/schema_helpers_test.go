package extract

import (
	"encoding/json"
	"strings"
	"testing"
)

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
		schema := createDeepSchema(5)
		doc := NormalizedDocument{Fields: createDeepDoc(5)}
		res := ValidateDocument(doc, schema)
		if !res.Valid {
			t.Errorf("expected valid=true for depth 5, got false (errors: %v)", res.Errors)
		}
	})

	t.Run("exceeds_depth_limit", func(t *testing.T) {
		schema := createDeepSchema(15)
		doc := NormalizedDocument{Fields: createDeepDoc(15)}
		res := ValidateDocument(doc, schema)
		if res.Valid {
			t.Errorf("expected valid=false for depth 15, got true")
		}
		found := false
		for _, err := range res.Errors {
			if len(err) > 0 && strings.Contains(err, "validation depth exceeded") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected depth limit error, got: %v", res.Errors)
		}
	})

	t.Run("exactly_at_depth_limit", func(t *testing.T) {
		schema := createDeepSchema(10)
		doc := NormalizedDocument{Fields: createDeepDoc(10)}
		res := ValidateDocument(doc, schema)
		if !res.Valid {
			t.Errorf("expected valid=true for depth 10, got false (errors: %v)", res.Errors)
		}
	})
}
