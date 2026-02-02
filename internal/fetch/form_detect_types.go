// Package fetch provides HTTP and headless browser content fetching capabilities.
//
// This file contains the type definitions for form detection, including form types,
// field matches, detected forms, and detection weights configuration.
//
// The detection uses heuristics based on:
//   - Input type attributes (password, email)
//   - Autocomplete attributes (username, current-password)
//   - Name/id patterns (user, login, email, pass)
//   - Form structure and field relationships
//
// It does NOT execute JavaScript or handle multi-step flows (MFA/2FA).
package fetch

// FormType classifies detected forms by their likely purpose.
type FormType string

const (
	FormTypeLogin         FormType = "login"
	FormTypeRegister      FormType = "register"
	FormTypePasswordReset FormType = "password_reset"
	FormTypeSearch        FormType = "search"
	FormTypeContact       FormType = "contact"
	FormTypeNewsletter    FormType = "newsletter"
	FormTypeCheckout      FormType = "checkout"
	FormTypeSurvey        FormType = "survey"
	FormTypeUnknown       FormType = "unknown"
)

// FieldType classifies form fields by their semantic purpose.
type FieldType string

const (
	FieldTypeText     FieldType = "text"
	FieldTypeEmail    FieldType = "email"
	FieldTypePassword FieldType = "password"
	FieldTypePhone    FieldType = "phone"
	FieldTypeAddress  FieldType = "address"
	FieldTypeSearch   FieldType = "search"
	FieldTypeURL      FieldType = "url"
	FieldTypeNumber   FieldType = "number"
	FieldTypeDate     FieldType = "date"
	FieldTypeSelect   FieldType = "select"
	FieldTypeTextarea FieldType = "textarea"
	FieldTypeCheckbox FieldType = "checkbox"
	FieldTypeRadio    FieldType = "radio"
	FieldTypeSubmit   FieldType = "submit"
	FieldTypeHidden   FieldType = "hidden"
	FieldTypeFile     FieldType = "file"
	FieldTypeUnknown  FieldType = "unknown"
)

// FieldMatch represents a detected form field with metadata about how it was identified.
type FieldMatch struct {
	Selector     string    `json:"selector"`               // CSS selector to target this field
	Attribute    string    `json:"attribute"`              // Which attribute matched (type, name, id, etc.)
	MatchValue   string    `json:"matchValue"`             // The value that matched
	Confidence   float64   `json:"confidence"`             // Individual field confidence (0.0-1.0)
	MatchReasons []string  `json:"matchReasons,omitempty"` // Why this field was selected
	FieldType    FieldType `json:"fieldType,omitempty"`    // Semantic field type classification
	FieldName    string    `json:"fieldName,omitempty"`    // Human-readable field name (e.g., "email", "firstName")
	Required     bool      `json:"required,omitempty"`     // Whether the field is required
	Placeholder  string    `json:"placeholder,omitempty"`  // Placeholder text if available
}

// DetectedFormFields captures the fields detected within a form.
type DetectedFormFields struct {
	UserField   FieldMatch `json:"userField"`   // Detected username/email field
	PassField   FieldMatch `json:"passField"`   // Detected password field
	SubmitField FieldMatch `json:"submitField"` // Detected submit button
	FormType    FormType   `json:"formType"`    // Classified type of form
}

// DetectedForm represents a form with detection metadata.
type DetectedForm struct {
	FormIndex    int          `json:"formIndex"`           // Index in document (0 = first form)
	FormSelector string       `json:"formSelector"`        // CSS selector to target this form
	Score        float64      `json:"score"`               // Overall confidence score (0.0-1.0)
	FormType     FormType     `json:"formType"`            // Classified type
	UserField    *FieldMatch  `json:"userField"`           // Detected username field (nil if not found)
	PassField    *FieldMatch  `json:"passField"`           // Detected password field (nil if not found)
	SubmitField  *FieldMatch  `json:"submitField"`         // Detected submit button (nil if not found)
	AllFields    []FieldMatch `json:"allFields,omitempty"` // All detected fields in the form
	HTML         string       `json:"html,omitempty"`      // Form HTML snippet (for debugging)
	Action       string       `json:"action,omitempty"`    // Form action URL
	Method       string       `json:"method,omitempty"`    // Form method (GET/POST)
	Name         string       `json:"name,omitempty"`      // Form name attribute
	ID           string       `json:"id,omitempty"`        // Form ID attribute
}

// DetectionWeights configures the scoring weights for form detection heuristics.
// Higher weights indicate stronger signals.
type DetectionWeights struct {
	PasswordTypeWeight   float64 // input[type=password] - strongest signal
	EmailTypeWeight      float64 // input[type=email]
	AutocompleteUsername float64 // autocomplete="username"
	AutocompletePassword float64 // autocomplete="current-password"
	NamePatternUsername  float64 // name matches user/login/email patterns
	NamePatternPassword  float64 // name matches pass/pwd patterns
	IDPatternUsername    float64 // id matches user/login/email patterns
	SubmitButtonType     float64 // button[type=submit] or input[type=submit]
	SubmitButtonText     float64 // button text contains "login", "sign in", etc.
}

// DefaultDetectionWeights returns sensible default weights for form detection.
func DefaultDetectionWeights() DetectionWeights {
	return DetectionWeights{
		PasswordTypeWeight:   1.0,  // Required for login forms
		EmailTypeWeight:      0.8,  // Strong signal for username
		AutocompleteUsername: 0.9,  // Very reliable indicator
		AutocompletePassword: 0.95, // Very reliable indicator
		NamePatternUsername:  0.7,  // Good signal
		NamePatternPassword:  0.6,  // Moderate signal
		IDPatternUsername:    0.6,  // Moderate signal
		SubmitButtonType:     0.5,  // Weak but consistent signal
		SubmitButtonText:     0.4,  // Text analysis can be unreliable
	}
}
