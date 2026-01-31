// Package fetch provides HTTP and headless browser content fetching capabilities.
//
// This file contains form classification logic for determining the type of form
// (login, register, password reset, etc.) based on its fields and structure.
package fetch

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

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
