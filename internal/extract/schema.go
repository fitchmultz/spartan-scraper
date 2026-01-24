package extract

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

const maxValidationDepth = 10

func ValidateDocument(doc NormalizedDocument, schema *Schema) ValidationResult {
	return validateDocumentWithDepth(doc, schema, 0)
}

func validateDocumentWithDepth(doc NormalizedDocument, schema *Schema, depth int) ValidationResult {
	if schema == nil {
		return ValidationResult{Valid: true}
	}

	var errors []string

	if schema.Type == SchemaObject {
		// Validate required fields
		for _, req := range schema.Required {
			if _, ok := doc.Fields[req]; !ok {
				errors = append(errors, fmt.Sprintf("missing required field: %s", req))
			}
		}

		// Validate properties
		for fieldName, fieldSchema := range schema.Properties {
			fieldVal, exists := doc.Fields[fieldName]
			if !exists {
				continue
			}
			errs := validateField(fieldName, fieldVal, fieldSchema, depth)
			errors = append(errors, errs...)
		}

		// Additional properties check (simplified: we don't strictly fail on extra fields in scraping usually, but if schema says no...)
		if !schema.AdditionalProperties {
			for fieldName := range doc.Fields {
				if _, ok := schema.Properties[fieldName]; !ok {
					errors = append(errors, fmt.Sprintf("unexpected field: %s", fieldName))
				}
			}
		}
	}

	return ValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

func validateField(name string, val FieldValue, schema *Schema, depth int) []string {
	var errs []string

	// Check validation depth limit to prevent infinite recursion
	if depth > maxValidationDepth {
		return []string{fmt.Sprintf("%s: validation depth exceeded", name)}
	}

	switch schema.Type {
	case SchemaArray:
		if schema.MinLength > 0 && len(val.Values) < schema.MinLength {
			errs = append(errs, fmt.Sprintf("%s: too few items (got %d, min %d)", name, len(val.Values), schema.MinLength))
		}
		if schema.MaxLength > 0 && len(val.Values) > schema.MaxLength {
			errs = append(errs, fmt.Sprintf("%s: too many items (got %d, max %d)", name, len(val.Values), schema.MaxLength))
		}
		if schema.Items != nil {
			// For array items, validate each value against the items schema
			for i, v := range val.Values {
				itemPath := fmt.Sprintf("%s[%d]", name, i)
				itemField := FieldValue{Values: []string{v}, Source: val.Source}

				// If items schema is an object, try to parse the value as JSON
				if schema.Items.Type == SchemaObject {
					itemField.RawObject = v
					itemField.Values = nil
				}

				itemErrs := validateField(itemPath, itemField, schema.Items, depth+1)
				errs = append(errs, itemErrs...)
			}
		}

	case SchemaObject:
		// Validate nested object
		if val.RawObject == "" {
			// No object data to validate - this is acceptable for backwards compatibility
			// Only error if this is a required field with no data at all
			if len(val.Values) == 0 {
				errs = append(errs, fmt.Sprintf("%s: missing object data", name))
			}
			return errs
		}

		// Parse JSON object
		var nestedFields map[string]FieldValue
		if err := json.Unmarshal([]byte(val.RawObject), &nestedFields); err != nil {
			errs = append(errs, fmt.Sprintf("%s: invalid object JSON: %v", name, err))
			return errs
		}

		// Recursively validate the nested object with incremented depth
		nestedDoc := NormalizedDocument{Fields: nestedFields}
		nestedResult := validateDocumentWithDepth(nestedDoc, schema, depth+1)
		errs = append(errs, prefixErrors(nestedResult.Errors, name+".")...)

	default:
		// Scalar check on first value
		if len(val.Values) == 0 {
			// If required, it would have been caught by top level check?
			// No, doc.Fields[req] check only checks key existence.
			// But empty list might be considered "missing" value?
			// Let's assume strict: if key exists but empty values, it's null-ish.
			return errs
		}
		// Warn if multiple? Nah.
		v := val.Values[0]
		errs = append(errs, validateScalar(name, v, schema)...)
	}

	return errs
}

// validateNumericConstraints checks minimum/maximum constraints for a numeric value.
func validateNumericConstraints(path string, f float64, schema *Schema) []string {
	var errs []string
	if schema.Minimum != nil && f < *schema.Minimum {
		errs = append(errs, fmt.Sprintf("%s: below minimum %v", path, *schema.Minimum))
	}
	if schema.Maximum != nil && f > *schema.Maximum {
		errs = append(errs, fmt.Sprintf("%s: above maximum %v", path, *schema.Maximum))
	}
	return errs
}

// prefixErrors adds a prefix to all error messages for nested error reporting.
func prefixErrors(errors []string, prefix string) []string {
	result := make([]string, len(errors))
	for i, err := range errors {
		result[i] = prefix + err
	}
	return result
}

func validateScalar(path string, value string, schema *Schema) []string {
	var errs []string

	switch schema.Type {
	case SchemaString:
		if schema.Pattern != "" {
			re, err := regexp.Compile(schema.Pattern)
			if err == nil && !re.MatchString(value) {
				errs = append(errs, fmt.Sprintf("%s: does not match pattern %s", path, schema.Pattern))
			}
		}
		if schema.MinLength > 0 && len(value) < schema.MinLength {
			errs = append(errs, fmt.Sprintf("%s: too short (len %d, min %d)", path, len(value), schema.MinLength))
		}
		if schema.MaxLength > 0 && len(value) > schema.MaxLength {
			errs = append(errs, fmt.Sprintf("%s: too long (len %d, max %d)", path, len(value), schema.MaxLength))
		}
		if len(schema.Enum) > 0 && !slices.Contains(schema.Enum, value) {
			errs = append(errs, fmt.Sprintf("%s: value not in enum", path))
		}

	case SchemaNumber:
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: not a number", path))
		} else {
			errs = append(errs, validateNumericConstraints(path, f, schema)...)
		}

	case SchemaInteger:
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: not an integer", path))
		} else if math.Trunc(f) != f {
			errs = append(errs, fmt.Sprintf("%s: must be an integer (got %s with decimals)", path, value))
		} else {
			errs = append(errs, validateNumericConstraints(path, f, schema)...)
		}

	case SchemaBool:
		lower := strings.ToLower(value)
		if lower != "true" && lower != "false" && lower != "1" && lower != "0" {
			errs = append(errs, fmt.Sprintf("%s: not a boolean", path))
		}

	default:
		errs = append(errs, fmt.Sprintf("%s: unknown schema type: %s", path, schema.Type))
	}

	return errs
}
