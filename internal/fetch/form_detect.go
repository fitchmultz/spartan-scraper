// Package fetch provides HTTP and headless browser content fetching capabilities.
//
// This file implements the main entry points for automatic form detection for
// headless login flows. It analyzes HTML to detect login forms, identify input
// fields, and generate CSS selectors for automated login without manual configuration.
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

// DetectAllForms analyzes HTML and returns all detected forms with full field classification.
// This is the general-purpose form detection that supports all form types.
func (d *FormDetector) DetectAllForms(html string) ([]DetectedForm, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var forms []DetectedForm

	doc.Find("form").Each(func(index int, formElem *goquery.Selection) {
		form := d.analyzeFormFull(index, formElem)
		if form != nil {
			forms = append(forms, *form)
		}
	})

	// Also look for div-based forms
	if len(forms) == 0 {
		form := d.analyzeDivBasedFormFull(doc)
		if form != nil {
			forms = append(forms, *form)
		}
	}

	// Sort by score descending
	sortFormsByScore(forms)

	return forms, nil
}

// DetectFormsByType analyzes HTML and returns forms of a specific type.
func (d *FormDetector) DetectFormsByType(html string, formType FormType) ([]DetectedForm, error) {
	forms, err := d.DetectAllForms(html)
	if err != nil {
		return nil, err
	}

	var filtered []DetectedForm
	for _, form := range forms {
		if form.FormType == formType {
			filtered = append(filtered, form)
		}
	}

	return filtered, nil
}

// DetectFormFields extracts all fields from a specific form.
func (d *FormDetector) DetectFormFields(html string, formSelector string) ([]FieldMatch, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	formElem := doc.Find(formSelector).First()
	if formElem.Length() == 0 {
		return nil, fmt.Errorf("form not found with selector: %s", formSelector)
	}

	return d.findAllFormFields(formElem), nil
}

// analyzeFormFull analyzes a form element with full field detection.
func (d *FormDetector) analyzeFormFull(index int, formElem *goquery.Selection) *DetectedForm {
	form := &DetectedForm{
		FormIndex: index,
		FormType:  FormTypeUnknown,
	}

	// Generate form selector
	form.FormSelector = d.generateFormSelector(formElem, index)

	// Extract form attributes
	form.Action, _ = formElem.Attr("action")
	form.Method, _ = formElem.Attr("method")
	form.Name, _ = formElem.Attr("name")
	form.ID, _ = formElem.Attr("id")

	// Find password field (for login forms)
	passField := d.findPasswordField(formElem)
	if passField != nil {
		form.PassField = passField
		form.Score += passField.Confidence

		// Find username field (near the password field)
		form.UserField = d.findUsernameField(formElem, passField)
		if form.UserField != nil {
			form.Score += form.UserField.Confidence
		}

		// Classify form type
		form.FormType = d.classifyForm(formElem, passField)
	} else {
		// No password field - classify without password
		form.FormType = d.classifyFormWithoutPassword(formElem)
	}

	// Find submit button
	form.SubmitField = d.findSubmitButton(formElem)
	if form.SubmitField != nil {
		form.Score += form.SubmitField.Confidence
	}

	// Extract all fields
	form.AllFields = d.findAllFormFields(formElem)

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

// analyzeDivBasedFormFull attempts to detect forms that aren't wrapped in <form> tags.
func (d *FormDetector) analyzeDivBasedFormFull(doc *goquery.Document) *DetectedForm {
	// Look for common form patterns in divs
	var bestContainer *goquery.Selection
	bestScore := 0

	doc.Find("div").Each(func(i int, div *goquery.Selection) {
		score := 0

		// Count form-like inputs
		textCount := div.Find("input[type='text']").Length()
		emailCount := div.Find("input[type='email']").Length()
		passCount := div.Find("input[type='password']").Length()
		buttonCount := div.Find("button, input[type='submit']").Length()

		score = textCount + emailCount*2 + passCount*3 + buttonCount

		if score > bestScore && score >= 2 {
			bestScore = score
			bestContainer = div
		}
	})

	if bestContainer == nil {
		return nil
	}

	// Analyze this container as a form
	form := &DetectedForm{
		FormIndex:    0,
		FormSelector: d.generateContainerSelector(bestContainer),
		FormType:     FormTypeUnknown,
	}

	// Find password field
	passField := d.findPasswordField(bestContainer)
	if passField != nil {
		form.PassField = passField
		form.Score += passField.Confidence
		form.UserField = d.findUsernameField(bestContainer, passField)
		if form.UserField != nil {
			form.Score += form.UserField.Confidence
		}
		form.FormType = d.classifyForm(bestContainer, passField)
	} else {
		form.FormType = d.classifyFormWithoutPassword(bestContainer)
	}

	form.SubmitField = d.findSubmitButton(bestContainer)
	if form.SubmitField != nil {
		form.Score += form.SubmitField.Confidence
	}

	form.AllFields = d.findAllFormFields(bestContainer)
	form.Score = min(form.Score/2.5, 1.0)

	return form
}
