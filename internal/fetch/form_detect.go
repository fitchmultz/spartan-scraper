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
