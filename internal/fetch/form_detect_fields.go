// Package fetch provides HTTP and headless browser content fetching capabilities.
//
// This file contains field finding functions for detecting username, password,
// and submit button fields within forms.
package fetch

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

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
		FieldType:    FieldTypeSubmit,
	}
}

// findAllFormFields extracts all fields from a form with their semantic types.
func (d *FormDetector) findAllFormFields(formElem *goquery.Selection) []FieldMatch {
	var fields []FieldMatch

	// Find all input elements
	formElem.Find("input").Each(func(i int, input *goquery.Selection) {
		fieldType := d.classifyFieldType(input)
		if fieldType == FieldTypeHidden {
			return // Skip hidden fields
		}

		field := d.createFieldMatch(input, fieldType)
		fields = append(fields, field)
	})

	// Find all textarea elements
	formElem.Find("textarea").Each(func(i int, textarea *goquery.Selection) {
		fieldType := d.classifyTextareaType(textarea)
		field := d.createFieldMatch(textarea, fieldType)
		fields = append(fields, field)
	})

	// Find all select elements
	formElem.Find("select").Each(func(i int, selectElem *goquery.Selection) {
		fieldType := d.classifySelectType(selectElem)
		field := d.createFieldMatch(selectElem, fieldType)
		fields = append(fields, field)
	})

	return fields
}

// createFieldMatch creates a FieldMatch from an element.
func (d *FormDetector) createFieldMatch(elem *goquery.Selection, fieldType FieldType) FieldMatch {
	selector := d.generateSelector(elem)
	placeholder, _ := elem.Attr("placeholder")
	required, _ := elem.Attr("required")

	// Determine field name for API usage
	fieldName := d.determineFieldName(elem, fieldType)

	return FieldMatch{
		Selector:     selector,
		Attribute:    d.determineBestAttribute(elem, 0, nil),
		MatchValue:   d.determineMatchValue(elem, 0, nil),
		Confidence:   0.5, // Base confidence for general fields
		MatchReasons: []string{"detected as " + string(fieldType)},
		FieldType:    fieldType,
		FieldName:    fieldName,
		Required:     required != "",
		Placeholder:  placeholder,
	}
}

// determineFieldName generates a human-readable field name for API usage.
func (d *FormDetector) determineFieldName(elem *goquery.Selection, fieldType FieldType) string {
	// Try name attribute first
	name, _ := elem.Attr("name")
	if name != "" {
		return name
	}

	// Try ID next
	id, _ := elem.Attr("id")
	if id != "" {
		return id
	}

	// Try placeholder as fallback
	placeholder, _ := elem.Attr("placeholder")
	if placeholder != "" {
		return placeholder
	}

	// Fallback to field type with index
	return string(fieldType)
}

// findFieldByType finds a field by its semantic type.
func (d *FormDetector) findFieldByType(formElem *goquery.Selection, fieldType FieldType) *FieldMatch {
	fields := d.findAllFormFields(formElem)
	for _, field := range fields {
		if field.FieldType == fieldType {
			return &field
		}
	}
	return nil
}

// findFieldByPattern finds a field matching name/id patterns.
func (d *FormDetector) findFieldByPattern(formElem *goquery.Selection, patterns []string) *FieldMatch {
	var bestMatch *FieldMatch
	bestScore := 0.0

	fields := d.findAllFormFields(formElem)
	for _, field := range fields {
		score := 0.0
		fieldName := strings.ToLower(field.FieldName)

		for _, pattern := range patterns {
			lowerPattern := strings.ToLower(pattern)
			if fieldName == lowerPattern {
				score = 1.0
				break
			} else if strings.Contains(fieldName, lowerPattern) {
				score = 0.7
			}
		}

		if score > bestScore {
			bestScore = score
			match := field
			match.Confidence = score
			bestMatch = &match
		}
	}

	return bestMatch
}

// findSearchField finds a search query input field.
func (d *FormDetector) findSearchField(formElem *goquery.Selection) *FieldMatch {
	// First try: input[type="search"]
	searchInput := formElem.Find("input[type='search']").First()
	if searchInput.Length() > 0 {
		match := d.createFieldMatch(searchInput, FieldTypeSearch)
		return &match
	}

	// Second try: input with search-like name
	patterns := []string{"search", "query", "q", "keyword", "term"}
	return d.findFieldByPattern(formElem, patterns)
}

// findEmailField finds an email input field.
func (d *FormDetector) findEmailField(formElem *goquery.Selection) *FieldMatch {
	// First try: input[type="email"]
	emailInput := formElem.Find("input[type='email']").First()
	if emailInput.Length() > 0 {
		match := d.createFieldMatch(emailInput, FieldTypeEmail)
		return &match
	}

	// Second try: input with email-like name
	patterns := []string{"email", "e-mail", "mail"}
	return d.findFieldByPattern(formElem, patterns)
}

// findPhoneField finds a phone number input field.
func (d *FormDetector) findPhoneField(formElem *goquery.Selection) *FieldMatch {
	// First try: input[type="tel"]
	phoneInput := formElem.Find("input[type='tel']").First()
	if phoneInput.Length() > 0 {
		match := d.createFieldMatch(phoneInput, FieldTypePhone)
		return &match
	}

	// Second try: input with phone-like name
	patterns := []string{"phone", "tel", "mobile", "cell", "fax"}
	return d.findFieldByPattern(formElem, patterns)
}

// findAddressFields finds all address-related fields.
func (d *FormDetector) findAddressFields(formElem *goquery.Selection) []FieldMatch {
	var addressFields []FieldMatch
	patterns := []string{"address", "street", "city", "state", "province", "zip", "postal", "country"}

	fields := d.findAllFormFields(formElem)
	for _, field := range fields {
		fieldName := strings.ToLower(field.FieldName)
		for _, pattern := range patterns {
			if strings.Contains(fieldName, pattern) {
				field.FieldType = FieldTypeAddress
				addressFields = append(addressFields, field)
				break
			}
		}
	}

	return addressFields
}

// findTextareaField finds a textarea field (for messages, comments, etc).
func (d *FormDetector) findTextareaField(formElem *goquery.Selection) *FieldMatch {
	textarea := formElem.Find("textarea").First()
	if textarea.Length() > 0 {
		match := d.createFieldMatch(textarea, FieldTypeTextarea)
		return &match
	}
	return nil
}
