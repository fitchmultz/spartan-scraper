// Package fetch provides HTTP and headless browser content fetching capabilities.
//
// This file contains form classification logic for determining the type of form
// (login, register, password reset, search, contact, newsletter, checkout, survey)
// based on its fields and structure.
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

	// Check for checkout indicators
	if d.isCheckoutForm(formElem, formText) {
		return FormTypeCheckout
	}

	// Check for survey indicators
	if d.isSurveyForm(formElem, formText) {
		return FormTypeSurvey
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

	// Check for search form
	if d.isSearchForm(formElem, formText) {
		return FormTypeSearch
	}

	// Check for newsletter form
	if d.isNewsletterForm(formElem, formText) {
		return FormTypeNewsletter
	}

	// Check for contact form
	if d.isContactForm(formElem, formText) {
		return FormTypeContact
	}

	// Check for checkout form
	if d.isCheckoutForm(formElem, formText) {
		return FormTypeCheckout
	}

	// Check for survey form
	if d.isSurveyForm(formElem, formText) {
		return FormTypeSurvey
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

// isSearchForm checks if the form is a search form.
func (d *FormDetector) isSearchForm(formElem *goquery.Selection, formText string) bool {
	// Check for search-related text
	searchTerms := []string{"search", "find", "query", "lookup"}
	for _, term := range searchTerms {
		if strings.Contains(formText, term) {
			return true
		}
	}

	// Check action attribute for search indicators
	action, _ := formElem.Attr("action")
	lowerAction := strings.ToLower(action)
	if strings.Contains(lowerAction, "search") || strings.Contains(lowerAction, "find") {
		return true
	}

	// Check for single text input with search-like placeholder
	textInputs := formElem.Find("input[type='text']")
	if textInputs.Length() == 1 {
		placeholder, _ := textInputs.Attr("placeholder")
		lowerPlaceholder := strings.ToLower(placeholder)
		for _, term := range searchTerms {
			if strings.Contains(lowerPlaceholder, term) {
				return true
			}
		}
	}

	// Check for search input type
	if formElem.Find("input[type='search']").Length() > 0 {
		return true
	}

	return false
}

// isContactForm checks if the form is a contact form.
func (d *FormDetector) isContactForm(formElem *goquery.Selection, formText string) bool {
	// Check for contact-related text
	contactTerms := []string{"contact", "message", "feedback", "inquiry", "get in touch", "reach us"}
	for _, term := range contactTerms {
		if strings.Contains(formText, term) {
			// Must have textarea for message body
			if formElem.Find("textarea").Length() > 0 {
				return true
			}
		}
	}

	// Check for email + textarea combination
	emailCount := formElem.Find("input[type='email']").Length()
	textareaCount := formElem.Find("textarea").Length()
	if emailCount >= 1 && textareaCount >= 1 {
		return true
	}

	return false
}

// isNewsletterForm checks if the form is a newsletter/email subscription form.
func (d *FormDetector) isNewsletterForm(formElem *goquery.Selection, formText string) bool {
	// Check for newsletter-related text
	newsletterTerms := []string{"subscribe", "newsletter", "signup", "sign up", "join", "stay updated"}
	for _, term := range newsletterTerms {
		if strings.Contains(formText, term) {
			// Must have email field
			if formElem.Find("input[type='email']").Length() == 1 {
				return true
			}
		}
	}

	// Single email input with no other text inputs
	emailCount := formElem.Find("input[type='email']").Length()
	textCount := formElem.Find("input[type='text']").Length()
	passCount := formElem.Find("input[type='password']").Length()
	textareaCount := formElem.Find("textarea").Length()

	if emailCount == 1 && textCount == 0 && passCount == 0 && textareaCount == 0 {
		return true
	}

	return false
}

// isCheckoutForm checks if the form is a checkout/payment form.
func (d *FormDetector) isCheckoutForm(formElem *goquery.Selection, formText string) bool {
	// Check for checkout-related text
	checkoutTerms := []string{"checkout", "payment", "billing", "shipping", "credit card", "order"}
	for _, term := range checkoutTerms {
		if strings.Contains(formText, term) {
			return true
		}
	}

	// Check for address-related fields
	addressFields := []string{"address", "city", "state", "zip", "postal", "country"}
	addressCount := 0
	formElem.Find("input").Each(func(i int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		id, _ := s.Attr("id")
		lowerName := strings.ToLower(name)
		lowerID := strings.ToLower(id)

		for _, field := range addressFields {
			if strings.Contains(lowerName, field) || strings.Contains(lowerID, field) {
				addressCount++
				break
			}
		}
	})

	if addressCount >= 2 {
		return true
	}

	return false
}

// isSurveyForm checks if the form is a survey/questionnaire form.
func (d *FormDetector) isSurveyForm(formElem *goquery.Selection, formText string) bool {
	// Check for survey-related text
	surveyTerms := []string{"survey", "questionnaire", "poll", "vote", "rate"}
	for _, term := range surveyTerms {
		if strings.Contains(formText, term) {
			return true
		}
	}

	// Many radio buttons or checkboxes
	radioCount := formElem.Find("input[type='radio']").Length()
	checkboxCount := formElem.Find("input[type='checkbox']").Length()

	if radioCount >= 3 || checkboxCount >= 3 {
		return true
	}

	return false
}

// classifyFieldType determines the semantic type of an input field.
func (d *FormDetector) classifyFieldType(input *goquery.Selection) FieldType {
	inputType, _ := input.Attr("type")
	inputType = strings.ToLower(inputType)

	// Check explicit type attribute first
	switch inputType {
	case "email":
		return FieldTypeEmail
	case "password":
		return FieldTypePassword
	case "tel", "phone":
		return FieldTypePhone
	case "search":
		return FieldTypeSearch
	case "url":
		return FieldTypeURL
	case "number":
		return FieldTypeNumber
	case "date", "datetime-local":
		return FieldTypeDate
	case "checkbox":
		return FieldTypeCheckbox
	case "radio":
		return FieldTypeRadio
	case "file":
		return FieldTypeFile
	case "hidden":
		return FieldTypeHidden
	case "submit", "button":
		return FieldTypeSubmit
	}

	// Check autocomplete attribute
	autocomplete, _ := input.Attr("autocomplete")
	autocomplete = strings.ToLower(autocomplete)
	switch {
	case strings.Contains(autocomplete, "email"):
		return FieldTypeEmail
	case strings.Contains(autocomplete, "tel"):
		return FieldTypePhone
	case strings.Contains(autocomplete, "address"):
		return FieldTypeAddress
	case strings.Contains(autocomplete, "url"):
		return FieldTypeURL
	}

	// Check name attribute patterns
	name, _ := input.Attr("name")
	name = strings.ToLower(name)

	// Email patterns
	if strings.Contains(name, "email") || strings.Contains(name, "e-mail") {
		return FieldTypeEmail
	}

	// Phone patterns
	if strings.Contains(name, "phone") || strings.Contains(name, "tel") || strings.Contains(name, "mobile") || strings.Contains(name, "cell") {
		return FieldTypePhone
	}

	// Address patterns
	if strings.Contains(name, "address") || strings.Contains(name, "street") || strings.Contains(name, "city") ||
		strings.Contains(name, "state") || strings.Contains(name, "zip") || strings.Contains(name, "postal") {
		return FieldTypeAddress
	}

	// Search patterns
	if strings.Contains(name, "search") || strings.Contains(name, "query") || strings.Contains(name, "q") {
		return FieldTypeSearch
	}

	// URL patterns
	if strings.Contains(name, "url") || strings.Contains(name, "website") || strings.Contains(name, "link") {
		return FieldTypeURL
	}

	// Number patterns
	if strings.Contains(name, "number") || strings.Contains(name, "count") || strings.Contains(name, "quantity") ||
		strings.Contains(name, "amount") || strings.Contains(name, "price") {
		return FieldTypeNumber
	}

	// Date patterns
	if strings.Contains(name, "date") || strings.Contains(name, "birth") || strings.Contains(name, "dob") {
		return FieldTypeDate
	}

	// Default to text
	return FieldTypeText
}

// classifyTextareaType determines the semantic type of a textarea.
func (d *FormDetector) classifyTextareaType(textarea *goquery.Selection) FieldType {
	// Check name attribute
	name, _ := textarea.Attr("name")
	name = strings.ToLower(name)

	// Message/body patterns
	if strings.Contains(name, "message") || strings.Contains(name, "body") || strings.Contains(name, "comment") ||
		strings.Contains(name, "description") || strings.Contains(name, "content") || strings.Contains(name, "feedback") {
		return FieldTypeTextarea
	}

	// Check placeholder
	placeholder, _ := textarea.Attr("placeholder")
	placeholder = strings.ToLower(placeholder)
	if strings.Contains(placeholder, "message") || strings.Contains(placeholder, "comment") {
		return FieldTypeTextarea
	}

	return FieldTypeTextarea
}

// classifySelectType determines the semantic type of a select element.
func (d *FormDetector) classifySelectType(selectElem *goquery.Selection) FieldType {
	// Check name attribute
	name, _ := selectElem.Attr("name")
	name = strings.ToLower(name)

	// Address-related selects
	if strings.Contains(name, "country") || strings.Contains(name, "state") || strings.Contains(name, "province") {
		return FieldTypeAddress
	}

	return FieldTypeSelect
}
