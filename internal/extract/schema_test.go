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
