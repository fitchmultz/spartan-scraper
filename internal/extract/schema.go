package extract

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func ValidateDocument(doc NormalizedDocument, schema *Schema) ValidationResult {
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
			errs := validateField(fieldName, fieldVal, fieldSchema)
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

func validateField(name string, val FieldValue, schema *Schema) []string {
	var errs []string

	// Check if array or single
	// Scraping always produces []string.
	// If schema type is array, we treat the whole slice.
	// If schema type is string/number/bool, we typically take the first element (or iterate if user meant "any of").
	// Convention: If schema type != array, we validate the FIRST value. If there are multiple, warn? Or just use first.
	// Let's validate the first value for scalar types.

	if schema.Type == SchemaArray {
		if schema.MinLength > 0 && len(val.Values) < schema.MinLength {
			errs = append(errs, fmt.Sprintf("%s: too few items (got %d, min %d)", name, len(val.Values), schema.MinLength))
		}
		if schema.MaxLength > 0 && len(val.Values) > schema.MaxLength {
			errs = append(errs, fmt.Sprintf("%s: too many items (got %d, max %d)", name, len(val.Values), schema.MaxLength))
		}
		if schema.Items != nil {
			for i, v := range val.Values {
				itemErrs := validateScalar(fmt.Sprintf("%s[%d]", name, i), v, schema.Items)
				errs = append(errs, itemErrs...)
			}
		}
	} else {
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
		if len(schema.Enum) > 0 {
			found := false
			for _, allowed := range schema.Enum {
				if allowed == value {
					found = true
					break
				}
			}
			if !found {
				errs = append(errs, fmt.Sprintf("%s: value not in enum", path))
			}
		}

	case SchemaNumber:
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: not a number", path))
		} else {
			if schema.Minimum != nil && f < *schema.Minimum {
				errs = append(errs, fmt.Sprintf("%s: below minimum %v", path, *schema.Minimum))
			}
			if schema.Maximum != nil && f > *schema.Maximum {
				errs = append(errs, fmt.Sprintf("%s: above maximum %v", path, *schema.Maximum))
			}
		}

	case SchemaBool:
		lower := strings.ToLower(value)
		if lower != "true" && lower != "false" && lower != "1" && lower != "0" {
			errs = append(errs, fmt.Sprintf("%s: not a boolean", path))
		}
	}

	return errs
}
