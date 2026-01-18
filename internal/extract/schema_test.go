package extract

import (
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
				"title": {Values: []string{"Valid Title"}},
				"price": {Values: []string{"19.99"}},
				"tags":  {Values: []string{"new"}},
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
