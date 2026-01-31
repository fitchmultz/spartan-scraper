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
	}
}
