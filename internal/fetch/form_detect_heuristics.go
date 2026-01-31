// Package fetch provides HTTP and headless browser content fetching capabilities.
//
// This file contains form detection heuristics for analyzing form elements
// and div-based forms (common in modern SPAs).
package fetch

import (
	"github.com/PuerkitoBio/goquery"
)

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
