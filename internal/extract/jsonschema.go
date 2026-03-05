// Package extract provides HTML content extraction using selectors, JSON-LD, and regex.
// It handles template-based extraction, field normalization, and schema validation.
// It does NOT handle fetching or rendering HTML content.
package extract

import (
	"encoding/json"
	"fmt"

	"github.com/xeipuuv/gojsonschema"
)

// ValidateJSONSchema validates a document against a JSON Schema.
// It supports JSON Schema Draft 7, 2019-09, and 2020-12.
func ValidateJSONSchema(doc NormalizedDocument, schema *Schema) ValidationResult {
	if schema == nil || schema.Format != SchemaFormatJSONSchema {
		return ValidationResult{Valid: true}
	}

	// Convert document to JSON for validation
	docJSON, err := documentToJSON(doc)
	if err != nil {
		return ValidationResult{
			Valid:  false,
			Errors: []string{fmt.Sprintf("failed to convert document to JSON: %v", err)},
		}
	}

	// Get the JSON Schema document
	schemaDoc := schema.JSONSchema
	if schemaDoc == nil {
		return ValidationResult{Valid: true}
	}

	// Load schema and document into gojsonschema
	schemaLoader := gojsonschema.NewGoLoader(schemaDoc)
	documentLoader := gojsonschema.NewGoLoader(docJSON)

	// Perform validation
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return ValidationResult{
			Valid:  false,
			Errors: []string{fmt.Sprintf("validation error: %v", err)},
		}
	}

	// Build validation result
	var errors []string
	if !result.Valid() {
		for _, err := range result.Errors() {
			// Format error with path for consistency with custom schema validation
			errors = append(errors, formatJSONSchemaError(err))
		}
	}

	return ValidationResult{
		Valid:  result.Valid(),
		Errors: errors,
	}
}

// documentToJSON converts a NormalizedDocument to a JSON-serializable map.
func documentToJSON(doc NormalizedDocument) (map[string]any, error) {
	// Build a map that represents the document structure
	result := map[string]any{
		"url":         doc.URL,
		"title":       doc.Title,
		"description": doc.Description,
		"text":        doc.Text,
		"links":       doc.Links,
		"template":    doc.Template,
	}

	if len(doc.Metadata) > 0 {
		result["metadata"] = doc.Metadata
	}

	if len(doc.JSONLD) > 0 {
		result["jsonld"] = doc.JSONLD
	}

	// Convert Fields to a simpler structure for validation
	if len(doc.Fields) > 0 {
		fields := make(map[string]any)
		for name, fieldVal := range doc.Fields {
			fields[name] = fieldValueToJSON(fieldVal)
		}
		result["fields"] = fields
	}

	return result, nil
}

// fieldValueToJSON converts a FieldValue to a JSON-serializable value.
func fieldValueToJSON(fv FieldValue) any {
	// If there's a raw object, try to parse it as JSON
	if fv.RawObject != "" {
		var obj any
		if err := json.Unmarshal([]byte(fv.RawObject), &obj); err == nil {
			return obj
		}
	}

	// If there are multiple values, return as array
	if len(fv.Values) == 1 {
		return fv.Values[0]
	}
	if len(fv.Values) > 1 {
		return fv.Values
	}

	return nil
}

// formatJSONSchemaError formats a gojsonschema error into a consistent string format.
func formatJSONSchemaError(err gojsonschema.ResultError) string {
	// Get the field path
	path := err.Context().String()
	if path == "" {
		path = "root"
	}

	// Format: "path: message (details)"
	msg := err.Description()
	if err.Details()["field"] != nil {
		// Add field-specific details if available
		return fmt.Sprintf("%s: %s", path, msg)
	}

	return fmt.Sprintf("%s: %s", path, msg)
}

// IsJSONSchemaSupported returns true if the schema format is JSON Schema.
func IsJSONSchemaSupported(schema *Schema) bool {
	return schema != nil && schema.Format == SchemaFormatJSONSchema && schema.JSONSchema != nil
}
