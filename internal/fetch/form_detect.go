// Package fetch provides HTTP and headless browser content fetching capabilities.
//
// This file implements automatic form detection for headless login flows.
// It analyzes HTML to detect login forms, identify input fields, and generate
// CSS selectors for automated login without manual configuration.
//
// The detection uses heuristics based on:
//   - Input type attributes (password, email)
//   - Autocomplete attributes (username, current-password)
//   - Name/id patterns (user, login, email, pass)
//   - Form structure and field relationships
//
// It does NOT execute JavaScript or handle multi-step flows (MFA/2FA).
package fetch

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// FormType classifies detected forms by their likely purpose.
type FormType string

const (
	FormTypeLogin         FormType = "login"
	FormTypeRegister      FormType = "register"
	FormTypePasswordReset FormType = "password_reset"
	FormTypeUnknown       FormType = "unknown"
)

// FieldMatch represents a detected form field with metadata about how it was identified.
type FieldMatch struct {
	Selector     string   `json:"selector"`               // CSS selector to target this field
	Attribute    string   `json:"attribute"`              // Which attribute matched (type, name, id, etc.)
	MatchValue   string   `json:"matchValue"`             // The value that matched
	Confidence   float64  `json:"confidence"`             // Individual field confidence (0.0-1.0)
	MatchReasons []string `json:"matchReasons,omitempty"` // Why this field was selected
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
	FormIndex    int         `json:"formIndex"`      // Index in document (0 = first form)
	FormSelector string      `json:"formSelector"`   // CSS selector to target this form
	Score        float64     `json:"score"`          // Overall confidence score (0.0-1.0)
	FormType     FormType    `json:"formType"`       // Classified type
	UserField    *FieldMatch `json:"userField"`      // Detected username field (nil if not found)
	PassField    *FieldMatch `json:"passField"`      // Detected password field (nil if not found)
	SubmitField  *FieldMatch `json:"submitField"`    // Detected submit button (nil if not found)
	HTML         string      `json:"html,omitempty"` // Form HTML snippet (for debugging)
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

// FormDetector analyzes HTML to find and classify login forms.
type FormDetector struct {
	Weights DetectionWeights
}

// NewFormDetector creates a new form detector with default weights.
func NewFormDetector() *FormDetector {
	return &FormDetector{
		Weights: DefaultDetectionWeights(),
	}
}

// NewFormDetectorWithWeights creates a form detector with custom weights.
func NewFormDetectorWithWeights(weights DetectionWeights) *FormDetector {
	return &FormDetector{
		Weights: weights,
	}
}

// DetectForms analyzes HTML and returns detected forms sorted by confidence (highest first).
func (d *FormDetector) DetectForms(html string) ([]DetectedForm, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var forms []DetectedForm

	doc.Find("form").Each(func(index int, formElem *goquery.Selection) {
		form := d.analyzeForm(index, formElem)
		if form != nil {
			forms = append(forms, *form)
		}
	})

	// Also look for login forms that might not be wrapped in <form> tags
	// (common in modern SPAs with div-based forms)
	if len(forms) == 0 {
		form := d.analyzeDivBasedForm(doc)
		if form != nil {
			forms = append(forms, *form)
		}
	}

	// Sort by score descending
	sortFormsByScore(forms)

	return forms, nil
}

// DetectLoginForm is a convenience method that returns the highest-confidence login form.
// Returns nil if no suitable login form is detected.
func (d *FormDetector) DetectLoginForm(html string) (*DetectedForm, error) {
	forms, err := d.DetectForms(html)
	if err != nil {
		return nil, err
	}

	for _, form := range forms {
		if form.FormType == FormTypeLogin && form.Score > 0.5 {
			return &form, nil
		}
	}

	// If no clear login form, return the highest scoring form if it has a password field
	if len(forms) > 0 && forms[0].PassField != nil {
		return &forms[0], nil
	}

	return nil, nil
}

// analyzeForm analyzes a single form element and returns detection results.
func (d *FormDetector) analyzeForm(index int, formElem *goquery.Selection) *DetectedForm {
	form := &DetectedForm{
		FormIndex: index,
		FormType:  FormTypeUnknown,
	}

	// Generate form selector
	form.FormSelector = d.generateFormSelector(formElem, index)

	// Find password field (required for login forms)
	passField := d.findPasswordField(formElem)
	if passField == nil {
		// No password field - might be a search form or other type
		// Still check if it could be a password reset form
		form.FormType = d.classifyFormWithoutPassword(formElem)
	} else {
		form.PassField = passField
		form.Score += passField.Confidence

		// Find username field (near the password field)
		form.UserField = d.findUsernameField(formElem, passField)
		if form.UserField != nil {
			form.Score += form.UserField.Confidence
		}

		// Classify form type
		form.FormType = d.classifyForm(formElem, passField)
	}

	// Find submit button
	form.SubmitField = d.findSubmitButton(formElem)
	if form.SubmitField != nil {
		form.Score += form.SubmitField.Confidence
	}

	// Normalize score to 0-1 range (max possible is around 2.5)
	form.Score = min(form.Score/2.5, 1.0)

	// Store HTML snippet for debugging (truncated)
	html, _ := formElem.Html()
	if len(html) > 500 {
		html = html[:500] + "..."
	}
	form.HTML = html

	return form
}

// analyzeDivBasedForm attempts to detect forms that aren't wrapped in <form> tags.
func (d *FormDetector) analyzeDivBasedForm(doc *goquery.Document) *DetectedForm {
	// Look for password inputs anywhere in the document
	var passwordInput *goquery.Selection
	doc.Find("input[type='password']").Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			passwordInput = s
		}
	})

	if passwordInput == nil {
		return nil
	}

	// Find a container that holds both username and password fields
	// Walk up the DOM looking for a suitable container
	container := passwordInput.Parent()
	for i := 0; i < 5 && container.Length() > 0; i++ {
		// Check if this container has both password and text/email inputs
		passCount := container.Find("input[type='password']").Length()
		textCount := container.Find("input[type='text']").Length()
		emailCount := container.Find("input[type='email']").Length()

		if passCount > 0 && (textCount > 0 || emailCount > 0) {
			// Found a likely form container
			form := &DetectedForm{
				FormIndex:    0,
				FormSelector: d.generateContainerSelector(container),
				FormType:     FormTypeLogin,
			}

			form.PassField = d.analyzePasswordField(passwordInput)
			form.UserField = d.findUsernameField(container, form.PassField)
			form.SubmitField = d.findSubmitButton(container)

			// Calculate score
			if form.PassField != nil {
				form.Score += form.PassField.Confidence
			}
			if form.UserField != nil {
				form.Score += form.UserField.Confidence
			}
			if form.SubmitField != nil {
				form.Score += form.SubmitField.Confidence
			}

			form.Score = min(form.Score/2.5, 1.0)
			return form
		}

		container = container.Parent()
	}

	return nil
}

// findPasswordField finds the password input field within a form.
func (d *FormDetector) findPasswordField(formElem *goquery.Selection) *FieldMatch {
	// First try: input[type="password"]
	passInput := formElem.Find("input[type='password']").First()
	if passInput.Length() > 0 {
		return d.analyzePasswordField(passInput)
	}

	// Second try: input with password-like name
	var match *FieldMatch
	formElem.Find("input").Each(func(i int, input *goquery.Selection) {
		if match != nil {
			return
		}
		name, _ := input.Attr("name")
		if name == "" {
			return
		}
		lowerName := strings.ToLower(name)
		if strings.Contains(lowerName, "pass") || strings.Contains(lowerName, "pwd") {
			match = &FieldMatch{
				Selector:     d.generateSelector(input),
				Attribute:    "name",
				MatchValue:   name,
				Confidence:   d.Weights.NamePatternPassword,
				MatchReasons: []string{"name contains pass/pwd pattern"},
			}
		}
	})

	return match
}

// analyzePasswordField creates a FieldMatch for a password input.
func (d *FormDetector) analyzePasswordField(input *goquery.Selection) *FieldMatch {
	selector := d.generateSelector(input)

	// Check for autocomplete attribute
	autocomplete, hasAutocomplete := input.Attr("autocomplete")
	if hasAutocomplete {
		switch autocomplete {
		case "current-password", "new-password":
			return &FieldMatch{
				Selector:     selector,
				Attribute:    "autocomplete",
				MatchValue:   autocomplete,
				Confidence:   d.Weights.AutocompletePassword,
				MatchReasons: []string{"autocomplete=" + autocomplete},
			}
		}
	}

	return &FieldMatch{
		Selector:     selector,
		Attribute:    "type",
		MatchValue:   "password",
		Confidence:   d.Weights.PasswordTypeWeight,
		MatchReasons: []string{"input[type=password]"},
	}
}

// findUsernameField finds the username/email input field within a form.
func (d *FormDetector) findUsernameField(formElem *goquery.Selection, passField *FieldMatch) *FieldMatch {
	// Get password field position for proximity scoring
	passInput := formElem.Find(passField.Selector).First()

	var bestMatch *FieldMatch
	bestScore := 0.0

	formElem.Find("input").Each(func(i int, input *goquery.Selection) {
		// Skip password fields
		inputType, _ := input.Attr("type")
		if inputType == "password" {
			return
		}

		// Skip submit/button/hidden inputs
		if inputType == "submit" || inputType == "button" || inputType == "hidden" {
			return
		}

		score := 0.0
		reasons := []string{}

		// Check type="email"
		if inputType == "email" {
			score += d.Weights.EmailTypeWeight
			reasons = append(reasons, "input[type=email]")
		}

		// Check autocomplete
		autocomplete, hasAutocomplete := input.Attr("autocomplete")
		if hasAutocomplete {
			if autocomplete == "username" {
				score += d.Weights.AutocompleteUsername
				reasons = append(reasons, "autocomplete=username")
			} else if autocomplete == "email" {
				score += d.Weights.EmailTypeWeight * 0.9
				reasons = append(reasons, "autocomplete=email")
			}
		}

		// Check name patterns
		name, hasName := input.Attr("name")
		if hasName {
			lowerName := strings.ToLower(name)
			if lowerName == "username" || lowerName == "user" || lowerName == "login" || lowerName == "email" {
				score += d.Weights.NamePatternUsername
				reasons = append(reasons, "name="+name)
			} else if strings.Contains(lowerName, "user") || strings.Contains(lowerName, "login") || strings.Contains(lowerName, "email") {
				score += d.Weights.NamePatternUsername * 0.7
				reasons = append(reasons, "name contains user/login/email pattern")
			}
		}

		// Check id patterns
		id, hasID := input.Attr("id")
		if hasID {
			lowerID := strings.ToLower(id)
			if lowerID == "username" || lowerID == "user" || lowerID == "login" || lowerID == "email" {
				score += d.Weights.IDPatternUsername
				reasons = append(reasons, "id="+id)
			} else if strings.Contains(lowerID, "user") || strings.Contains(lowerID, "login") || strings.Contains(lowerID, "email") {
				score += d.Weights.IDPatternUsername * 0.7
				reasons = append(reasons, "id contains user/login/email pattern")
			}
		}

		// Proximity bonus if close to password field
		if passInput.Length() > 0 && input.Length() > 0 {
			// Simple heuristic: if this input comes immediately before password
			// In a real implementation, we might use DOM position
		}

		if score > bestScore {
			bestScore = score
			bestMatch = &FieldMatch{
				Selector:     d.generateSelector(input),
				Attribute:    d.determineBestAttribute(input, score, reasons),
				MatchValue:   d.determineMatchValue(input, score, reasons),
				Confidence:   score,
				MatchReasons: reasons,
			}
		}
	})

	return bestMatch
}

// findSubmitButton finds the submit button within a form.
func (d *FormDetector) findSubmitButton(formElem *goquery.Selection) *FieldMatch {
	// First try: button[type="submit"]
	submitBtn := formElem.Find("button[type='submit']").First()
	if submitBtn.Length() > 0 {
		return d.analyzeSubmitButton(submitBtn, "button[type='submit']")
	}

	// Second try: input[type="submit"]
	submitInput := formElem.Find("input[type='submit']").First()
	if submitInput.Length() > 0 {
		return d.analyzeSubmitButton(submitInput, "input[type='submit']")
	}

	// Third try: button without type (defaults to submit in HTML5)
	formElem.Find("button").Each(func(i int, btn *goquery.Selection) {
		if submitBtn.Length() > 0 {
			return
		}
		btnType, hasType := btn.Attr("type")
		if !hasType || btnType == "submit" {
			submitBtn = btn
		}
	})
	if submitBtn.Length() > 0 {
		return d.analyzeSubmitButton(submitBtn, "button")
	}

	// Fourth try: any element with login-like text
	loginPatterns := []string{"login", "sign in", "signin", "log in", "submit"}
	var textMatch *FieldMatch
	formElem.Find("button, input[type='button'], a").Each(func(i int, elem *goquery.Selection) {
		if textMatch != nil {
			return
		}
		text := strings.ToLower(elem.Text())
		if text == "" {
			// Try value attribute for inputs
			val, _ := elem.Attr("value")
			text = strings.ToLower(val)
		}

		for _, pattern := range loginPatterns {
			if strings.Contains(text, pattern) {
				textMatch = &FieldMatch{
					Selector:     d.generateSelector(elem),
					Attribute:    "text",
					MatchValue:   pattern,
					Confidence:   d.Weights.SubmitButtonText,
					MatchReasons: []string{"text contains '" + pattern + "'"},
				}
				return
			}
		}
	})

	return textMatch
}

// analyzeSubmitButton creates a FieldMatch for a submit button.
func (d *FormDetector) analyzeSubmitButton(btn *goquery.Selection, btnType string) *FieldMatch {
	selector := d.generateSelector(btn)

	// Check for login-like text
	text := strings.ToLower(btn.Text())
	if text == "" {
		val, _ := btn.Attr("value")
		text = strings.ToLower(val)
	}

	confidence := d.Weights.SubmitButtonType
	reasons := []string{btnType}

	loginPatterns := []string{"login", "sign in", "signin", "log in"}
	for _, pattern := range loginPatterns {
		if strings.Contains(text, pattern) {
			confidence += d.Weights.SubmitButtonText
			reasons = append(reasons, "text contains '"+pattern+"'")
			break
		}
	}

	return &FieldMatch{
		Selector:     selector,
		Attribute:    "type",
		MatchValue:   "submit",
		Confidence:   confidence,
		MatchReasons: reasons,
	}
}

// classifyForm determines the form type based on its fields and structure.
func (d *FormDetector) classifyForm(formElem *goquery.Selection, _ *FieldMatch) FormType {
	// Check for confirm password (indicates registration)
	passCount := formElem.Find("input[type='password']").Length()
	if passCount > 1 {
		return FormTypeRegister
	}

	// Check for other registration indicators
	formText := strings.ToLower(formElem.Text())
	if strings.Contains(formText, "confirm password") ||
		strings.Contains(formText, "re-enter password") ||
		strings.Contains(formText, "repeat password") {
		return FormTypeRegister
	}

	// Check for password reset indicators
	if strings.Contains(formText, "reset password") ||
		strings.Contains(formText, "forgot password") ||
		strings.Contains(formText, "change password") {
		return FormTypePasswordReset
	}

	// Default to login if we have a password field
	return FormTypeLogin
}

// classifyFormWithoutPassword attempts to classify forms without password fields.
func (d *FormDetector) classifyFormWithoutPassword(formElem *goquery.Selection) FormType {
	formText := strings.ToLower(formElem.Text())

	if strings.Contains(formText, "reset password") ||
		strings.Contains(formText, "forgot password") {
		return FormTypePasswordReset
	}

	// Check for single email field (password reset or newsletter)
	emailCount := formElem.Find("input[type='email']").Length()
	textCount := formElem.Find("input[type='text']").Length()

	if emailCount == 1 && textCount == 0 {
		// Could be password reset or just email subscription
		if strings.Contains(formText, "reset") || strings.Contains(formText, "forgot") {
			return FormTypePasswordReset
		}
	}

	return FormTypeUnknown
}

// generateSelector creates a CSS selector for an element.
// Priority: ID > unique name > name + type > complex selector
func (d *FormDetector) generateSelector(elem *goquery.Selection) string {
	// Try ID first (most reliable)
	id, hasID := elem.Attr("id")
	if hasID && id != "" {
		return "#" + CSSEscape(id)
	}

	// Try name attribute
	name, hasName := elem.Attr("name")
	if hasName && name != "" {
		inputType, hasType := elem.Attr("type")
		if hasType && inputType != "" {
			return fmt.Sprintf("input[type='%s'][name='%s']", inputType, CSSEscape(name))
		}
		return fmt.Sprintf("[name='%s']", CSSEscape(name))
	}

	// Fallback to tag with position
	tag := goquery.NodeName(elem)
	if tag == "" {
		tag = "input"
	}

	// Try to use a class if available
	class, hasClass := elem.Attr("class")
	if hasClass && class != "" {
		// Use first class
		classes := strings.Fields(class)
		if len(classes) > 0 {
			return fmt.Sprintf("%s.%s", tag, CSSEscape(classes[0]))
		}
	}

	return tag
}

// generateFormSelector creates a selector for a form element.
func (d *FormDetector) generateFormSelector(formElem *goquery.Selection, index int) string {
	// Try ID first
	id, hasID := formElem.Attr("id")
	if hasID && id != "" {
		return "#" + CSSEscape(id)
	}

	// Try action attribute
	action, hasAction := formElem.Attr("action")
	if hasAction && action != "" {
		return fmt.Sprintf("form[action='%s']", CSSEscape(action))
	}

	// Try class
	class, hasClass := formElem.Attr("class")
	if hasClass && class != "" {
		classes := strings.Fields(class)
		if len(classes) > 0 {
			return "form." + CSSEscape(classes[0])
		}
	}

	// Fallback to index
	return fmt.Sprintf("form:nth-of-type(%d)", index+1)
}

// generateContainerSelector creates a selector for a div-based form container.
func (d *FormDetector) generateContainerSelector(container *goquery.Selection) string {
	// Try ID first
	id, hasID := container.Attr("id")
	if hasID && id != "" {
		return "#" + CSSEscape(id)
	}

	// Try class
	class, hasClass := container.Attr("class")
	if hasClass && class != "" {
		classes := strings.Fields(class)
		if len(classes) > 0 {
			// Use the most specific-looking class
			for _, c := range classes {
				lowerC := strings.ToLower(c)
				if strings.Contains(lowerC, "form") || strings.Contains(lowerC, "login") || strings.Contains(lowerC, "auth") {
					return "." + CSSEscape(c)
				}
			}
			return "." + CSSEscape(classes[0])
		}
	}

	// Fallback to tag
	tag := goquery.NodeName(container)
	if tag == "" {
		tag = "div"
	}
	return tag
}

// determineBestAttribute determines the best attribute to describe why a field matched.
func (d *FormDetector) determineBestAttribute(input *goquery.Selection, _ float64, _ []string) string {
	// Check what gave the highest score
	autocomplete, hasAutocomplete := input.Attr("autocomplete")
	if hasAutocomplete && (autocomplete == "username" || autocomplete == "email") {
		return "autocomplete"
	}

	inputType, _ := input.Attr("type")
	if inputType == "email" {
		return "type"
	}

	name, hasName := input.Attr("name")
	if hasName && name != "" {
		return "name"
	}

	id, hasID := input.Attr("id")
	if hasID && id != "" {
		return "id"
	}

	return "type"
}

// determineMatchValue determines the match value for a field.
func (d *FormDetector) determineMatchValue(input *goquery.Selection, _ float64, _ []string) string {
	autocomplete, hasAutocomplete := input.Attr("autocomplete")
	if hasAutocomplete {
		return autocomplete
	}

	inputType, hasType := input.Attr("type")
	if hasType && inputType != "" && inputType != "text" {
		return inputType
	}

	name, hasName := input.Attr("name")
	if hasName {
		return name
	}

	id, hasID := input.Attr("id")
	if hasID {
		return id
	}

	return ""
}

// CSSEscape escapes a string for use in CSS selectors.
// This is a simplified version - handles common cases.
func CSSEscape(s string) string {
	// Replace characters that need escaping in CSS selectors
	s = strings.ReplaceAll(s, "'", "\\'")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

// sortFormsByScore sorts forms by score descending (highest confidence first).
func sortFormsByScore(forms []DetectedForm) {
	// Simple bubble sort for small arrays
	for i := range forms {
		for j := i + 1; j < len(forms); j++ {
			if forms[j].Score > forms[i].Score {
				forms[i], forms[j] = forms[j], forms[i]
			}
		}
	}
}

// min returns the minimum of two float64 values.
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
