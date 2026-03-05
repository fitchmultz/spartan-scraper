// Package fetch provides HTTP and headless browser content fetching capabilities.
//
// This file implements automated form filling and submission for general forms
// (not just login forms). It uses chromedp for headless browser automation.
//
// The form filler supports:
//   - Automatic form detection and field mapping
//   - Filling text, email, phone, textarea, select, checkbox, and radio fields
//   - Form submission with success/failure detection
//   - Multi-step form workflows
//
// It does NOT handle CAPTCHAs or complex JavaScript-dependent forms.
package fetch

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// FormFillRequest represents a form fill operation.
type FormFillRequest struct {
	URL            string            `json:"url"`                      // URL of the page containing the form
	FormSelector   string            `json:"formSelector,omitempty"`   // CSS selector for the form (auto-detect if empty)
	Fields         map[string]string `json:"fields"`                   // field name/selector -> value
	Submit         bool              `json:"submit"`                   // Whether to submit the form
	WaitFor        string            `json:"waitFor,omitempty"`        // Selector to wait for after submit
	Timeout        time.Duration     `json:"timeout,omitempty"`        // Operation timeout
	Headless       bool              `json:"headless"`                 // Use headless mode
	DetectOnly     bool              `json:"detectOnly,omitempty"`     // Only detect forms, don't fill
	FormTypeFilter string            `json:"formTypeFilter,omitempty"` // Filter by form type (e.g., "contact", "search")
}

// FormFillResult represents the result of a form fill operation.
type FormFillResult struct {
	Success       bool           `json:"success"`
	FormSelector  string         `json:"formSelector"`
	FormType      FormType       `json:"formType,omitempty"`
	FilledFields  []string       `json:"filledFields"`
	Errors        []string       `json:"errors,omitempty"`
	PageURL       string         `json:"pageUrl,omitempty"`
	PageHTML      string         `json:"pageHtml,omitempty"`
	DetectedForms []DetectedForm `json:"detectedForms,omitempty"`
}

// FormFiller handles automated form filling and submission.
type FormFiller struct {
	fetcher *ChromedpFetcher
}

// NewFormFiller creates a new form filler using the provided chromedp fetcher.
func NewFormFiller(fetcher *ChromedpFetcher) *FormFiller {
	return &FormFiller{
		fetcher: fetcher,
	}
}

// DetectForms detects all forms on a page and returns their details.
func (f *FormFiller) DetectForms(ctx context.Context, url string, formTypeFilter string) (*FormFillResult, error) {
	result := &FormFillResult{
		Success:       true,
		FilledFields:  []string{},
		DetectedForms: []DetectedForm{},
	}

	// Create a new browser context
	browserCtx, cancel, err := f.createBrowserContext(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create browser context: %w", err)
	}
	defer cancel()

	// Navigate to the page
	if err := chromedp.Run(browserCtx, chromedp.Navigate(url)); err != nil {
		return nil, fmt.Errorf("failed to navigate to %s: %w", url, err)
	}

	// Wait for body to be ready
	if err := chromedp.Run(browserCtx, chromedp.WaitReady("body", chromedp.ByQuery)); err != nil {
		return nil, fmt.Errorf("failed to wait for page load: %w", err)
	}

	// Extract page HTML
	var html string
	if err := chromedp.Run(browserCtx, chromedp.OuterHTML("html", &html)); err != nil {
		return nil, fmt.Errorf("failed to extract page HTML: %w", err)
	}

	// Detect forms
	detector := NewFormDetector()
	forms, err := detector.DetectAllForms(html)
	if err != nil {
		return nil, fmt.Errorf("failed to detect forms: %w", err)
	}

	// Filter by form type if specified
	if formTypeFilter != "" {
		filterType := FormType(formTypeFilter)
		var filtered []DetectedForm
		for _, form := range forms {
			if form.FormType == filterType {
				filtered = append(filtered, form)
			}
		}
		forms = filtered
	}

	result.DetectedForms = forms

	// Get current page URL
	var pageURL string
	if err := chromedp.Run(browserCtx, chromedp.Location(&pageURL)); err == nil {
		result.PageURL = pageURL
	}

	return result, nil
}

// FillForm fills and optionally submits a form.
func (f *FormFiller) FillForm(ctx context.Context, req FormFillRequest) (*FormFillResult, error) {
	result := &FormFillResult{
		Success:      false,
		FilledFields: []string{},
		Errors:       []string{},
	}

	// Set default timeout
	if req.Timeout == 0 {
		req.Timeout = 30 * time.Second
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, req.Timeout)
	defer cancel()

	// Create browser context
	browserCtx, browserCancel, err := f.createBrowserContext(ctx, req.Headless)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to create browser context: %v", err))
		return result, err
	}
	defer browserCancel()

	// Navigate to the page
	if err := chromedp.Run(browserCtx, chromedp.Navigate(req.URL)); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to navigate to %s: %v", req.URL, err))
		return result, err
	}

	// Wait for body to be ready
	if err := chromedp.Run(browserCtx, chromedp.WaitReady("body", chromedp.ByQuery)); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to wait for page load: %v", err))
		return result, err
	}

	// Extract page HTML for form detection
	var html string
	if err := chromedp.Run(browserCtx, chromedp.OuterHTML("html", &html)); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to extract page HTML: %v", err))
		return result, err
	}

	// Determine form selector
	formSelector := req.FormSelector
	if formSelector == "" {
		// Auto-detect form
		detectedForm, err := f.detectBestForm(html, req.FormTypeFilter)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to detect form: %v", err))
			return result, err
		}
		if detectedForm == nil {
			result.Errors = append(result.Errors, "no suitable form detected on page")
			return result, fmt.Errorf("no form detected")
		}
		formSelector = detectedForm.FormSelector
		result.FormType = detectedForm.FormType
		slog.Info("form auto-detected", "selector", formSelector, "type", detectedForm.FormType)
	}

	result.FormSelector = formSelector

	// Wait for form to be visible
	if err := chromedp.Run(browserCtx, chromedp.WaitVisible(formSelector)); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("form not visible: %v", err))
		return result, err
	}

	// Fill each field
	for fieldName, value := range req.Fields {
		if err := f.fillField(browserCtx, formSelector, fieldName, value); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to fill field %s: %v", fieldName, err))
			slog.Warn("failed to fill field", "field", fieldName, "error", err)
		} else {
			result.FilledFields = append(result.FilledFields, fieldName)
		}
	}

	// Submit form if requested
	if req.Submit {
		if err := f.submitForm(browserCtx, formSelector); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to submit form: %v", err))
			return result, err
		}

		// Wait for specified selector or just wait for page to settle
		if req.WaitFor != "" {
			if err := chromedp.Run(browserCtx, chromedp.WaitVisible(req.WaitFor)); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("wait-for selector not found: %v", err))
			}
		} else {
			// Wait a bit for page to settle after submission
			if err := chromedp.Run(browserCtx, chromedp.Sleep(500*time.Millisecond)); err != nil {
				slog.Warn("sleep after submit failed", "error", err)
			}
		}
	}

	// Get final page state
	var pageURL string
	if err := chromedp.Run(browserCtx, chromedp.Location(&pageURL)); err == nil {
		result.PageURL = pageURL
	}

	// Get page HTML if there were errors (for debugging)
	if len(result.Errors) > 0 {
		var pageHTML string
		if err := chromedp.Run(browserCtx, chromedp.OuterHTML("html", &pageHTML)); err == nil {
			// Truncate for result
			if len(pageHTML) > 5000 {
				pageHTML = pageHTML[:5000] + "..."
			}
			result.PageHTML = pageHTML
		}
	}

	result.Success = len(result.Errors) == 0 || len(result.FilledFields) > 0
	return result, nil
}

// detectBestForm detects the best form to fill based on the HTML and optional type filter.
func (f *FormFiller) detectBestForm(html string, formTypeFilter string) (*DetectedForm, error) {
	detector := NewFormDetector()
	forms, err := detector.DetectAllForms(html)
	if err != nil {
		return nil, err
	}

	if len(forms) == 0 {
		return nil, nil
	}

	// If type filter specified, find best match of that type
	if formTypeFilter != "" {
		filterType := FormType(formTypeFilter)
		for _, form := range forms {
			if form.FormType == filterType {
				return &form, nil
			}
		}
		// No form of specified type found
		return nil, fmt.Errorf("no form of type %s found", formTypeFilter)
	}

	// Return highest scoring form
	return &forms[0], nil
}

// fillField fills a single form field.
func (f *FormFiller) fillField(ctx context.Context, formSelector, fieldName, value string) error {
	// Try to find the field within the form
	// First, try as a CSS selector
	fieldSelector := fieldName
	if !strings.Contains(fieldName, "[") && !strings.HasPrefix(fieldName, "#") && !strings.HasPrefix(fieldName, ".") {
		// Try to construct a selector from the field name
		fieldSelector = fmt.Sprintf("%s input[name='%s'], %s textarea[name='%s'], %s select[name='%s']",
			formSelector, fieldName, formSelector, fieldName, formSelector, fieldName)
	} else {
		// Use the selector within the form context
		fieldSelector = fmt.Sprintf("%s %s", formSelector, fieldName)
	}

	// Check if field exists
	var nodeCount int
	if err := chromedp.Run(ctx, chromedp.Evaluate(fmt.Sprintf(`document.querySelectorAll("%s").length`,
		strings.ReplaceAll(fieldSelector, `"`, `\"`)), &nodeCount)); err != nil {
		return fmt.Errorf("field not found: %w", err)
	}

	if nodeCount == 0 {
		// Try alternative selectors
		altSelectors := []string{
			fmt.Sprintf("%s input[id='%s']", formSelector, fieldName),
			fmt.Sprintf("%s textarea[id='%s']", formSelector, fieldName),
			fmt.Sprintf("%s #%s", formSelector, fieldName),
		}

		for _, altSelector := range altSelectors {
			if err := chromedp.Run(ctx, chromedp.Evaluate(fmt.Sprintf(`document.querySelectorAll("%s").length`,
				strings.ReplaceAll(altSelector, `"`, `\"`)), &nodeCount)); err == nil && nodeCount > 0 {
				fieldSelector = altSelector
				break
			}
		}

		if nodeCount == 0 {
			return fmt.Errorf("field not found with name or id: %s", fieldName)
		}
	}

	// Determine field type to handle appropriately
	var inputType string
	chromedp.Run(ctx, chromedp.Evaluate(fmt.Sprintf(`
		var el = document.querySelector("%s");
		el ? el.type || el.tagName.toLowerCase() : ''
	`, strings.ReplaceAll(fieldSelector, `"`, `\"`)), &inputType))

	inputType = strings.ToLower(inputType)

	// Fill the field based on type
	switch inputType {
	case "select", "select-one":
		return f.fillSelectField(ctx, fieldSelector, value)
	case "checkbox":
		return f.fillCheckboxField(ctx, fieldSelector, value)
	case "radio":
		return f.fillRadioField(ctx, fieldSelector, value)
	default:
		return f.fillTextField(ctx, fieldSelector, value)
	}
}

// fillTextField fills a text, email, password, or textarea field.
func (f *FormFiller) fillTextField(ctx context.Context, selector, value string) error {
	return chromedp.Run(ctx,
		chromedp.WaitVisible(selector),
		chromedp.Clear(selector),
		chromedp.SendKeys(selector, value),
	)
}

// fillSelectField fills a select dropdown.
func (f *FormFiller) fillSelectField(ctx context.Context, selector, value string) error {
	// Try to select by value first, then by text
	script := fmt.Sprintf(`
		var select = document.querySelector("%s");
		if (select) {
			// Try to find option by value
			for (var i = 0; i < select.options.length; i++) {
				if (select.options[i].value === "%s") {
					select.selectedIndex = i;
					select.dispatchEvent(new Event('change', { bubbles: true }));
					return true;
				}
			}
			// Try to find option by text
			for (var i = 0; i < select.options.length; i++) {
				if (select.options[i].textContent.trim() === "%s") {
					select.selectedIndex = i;
					select.dispatchEvent(new Event('change', { bubbles: true }));
					return true;
				}
			}
		}
		return false;
	`, strings.ReplaceAll(selector, `"`, `\"`),
		strings.ReplaceAll(value, `"`, `\"`),
		strings.ReplaceAll(value, `"`, `\"`))

	var success bool
	if err := chromedp.Run(ctx, chromedp.Evaluate(script, &success)); err != nil {
		return err
	}

	if !success {
		return fmt.Errorf("could not select value: %s", value)
	}

	return nil
}

// fillCheckboxField fills a checkbox field.
func (f *FormFiller) fillCheckboxField(ctx context.Context, selector, value string) error {
	checked := value == "true" || value == "on" || value == "1" || value == "yes"

	script := fmt.Sprintf(`
		var checkbox = document.querySelector("%s");
		if (checkbox) {
			checkbox.checked = %t;
			checkbox.dispatchEvent(new Event('change', { bubbles: true }));
			return true;
		}
		return false;
	`, strings.ReplaceAll(selector, `"`, `\"`), checked)

	var success bool
	if err := chromedp.Run(ctx, chromedp.Evaluate(script, &success)); err != nil {
		return err
	}

	if !success {
		return fmt.Errorf("could not set checkbox")
	}

	return nil
}

// fillRadioField fills a radio button field.
func (f *FormFiller) fillRadioField(ctx context.Context, selector, value string) error {
	// For radio buttons, we need to find the specific radio with the matching value
	script := fmt.Sprintf(`
		var radios = document.querySelectorAll("%s");
		for (var i = 0; i < radios.length; i++) {
			if (radios[i].value === "%s") {
				radios[i].checked = true;
				radios[i].dispatchEvent(new Event('change', { bubbles: true }));
				return true;
			}
		}
		return false;
	`, strings.ReplaceAll(selector, `"`, `\"`), strings.ReplaceAll(value, `"`, `\"`))

	var success bool
	if err := chromedp.Run(ctx, chromedp.Evaluate(script, &success)); err != nil {
		return err
	}

	if !success {
		return fmt.Errorf("could not select radio value: %s", value)
	}

	return nil
}

// submitForm submits a form.
func (f *FormFiller) submitForm(ctx context.Context, formSelector string) error {
	// Try to find a submit button within the form
	submitSelectors := []string{
		formSelector + " button[type='submit']",
		formSelector + " input[type='submit']",
		formSelector + " button",
	}

	for _, selector := range submitSelectors {
		var exists bool
		if err := chromedp.Run(ctx, chromedp.Evaluate(fmt.Sprintf(`
			document.querySelector("%s") !== null
		`, strings.ReplaceAll(selector, `"`, `\"`)), &exists)); err != nil {
			continue
		}

		if exists {
			return chromedp.Run(ctx, chromedp.Click(selector))
		}
	}

	// Fallback: submit the form via JavaScript
	script := fmt.Sprintf(`
		var form = document.querySelector("%s");
		if (form) {
			form.submit();
			return true;
		}
		return false;
	`, strings.ReplaceAll(formSelector, `"`, `\"`))

	var success bool
	if err := chromedp.Run(ctx, chromedp.Evaluate(script, &success)); err != nil {
		return err
	}

	if !success {
		return fmt.Errorf("could not submit form")
	}

	return nil
}

// createBrowserContext creates a new browser context with the specified options.
func (f *FormFiller) createBrowserContext(ctx context.Context, headless bool) (context.Context, context.CancelFunc, error) {
	// Use the fetcher's allocator if available
	if f.fetcher != nil {
		// Create a new context from the allocator
		browserCtx, cancel := chromedp.NewContext(ctx)
		return browserCtx, cancel, nil
	}

	// Create a new allocator with default options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	browserCtx, browserCancel := chromedp.NewContext(allocCtx)

	return browserCtx, func() {
		browserCancel()
		cancel()
	}, nil
}

// FormDetectRequest represents a request to detect forms on a page.
type FormDetectRequest struct {
	URL      string `json:"url"`
	FormType string `json:"formType,omitempty"`
	Headless bool   `json:"headless"`
}

// FormDetectResponse represents the response from form detection.
type FormDetectResponse struct {
	URL           string         `json:"url"`
	Forms         []DetectedForm `json:"forms"`
	FormCount     int            `json:"formCount"`
	DetectedTypes []string       `json:"detectedTypes"`
}

// Detect forms on a page and return detailed information.
func (f *FormFiller) Detect(ctx context.Context, req FormDetectRequest) (*FormDetectResponse, error) {
	result := &FormDetectResponse{
		URL:           req.URL,
		Forms:         []DetectedForm{},
		DetectedTypes: []string{},
	}

	// Create browser context
	browserCtx, cancel, err := f.createBrowserContext(ctx, req.Headless)
	if err != nil {
		return nil, fmt.Errorf("failed to create browser context: %w", err)
	}
	defer cancel()

	// Navigate to the page
	if err := chromedp.Run(browserCtx, chromedp.Navigate(req.URL)); err != nil {
		return nil, fmt.Errorf("failed to navigate to %s: %w", req.URL, err)
	}

	// Wait for body to be ready
	if err := chromedp.Run(browserCtx, chromedp.WaitReady("body", chromedp.ByQuery)); err != nil {
		return nil, fmt.Errorf("failed to wait for page load: %w", err)
	}

	// Extract page HTML
	var html string
	if err := chromedp.Run(browserCtx, chromedp.OuterHTML("html", &html)); err != nil {
		return nil, fmt.Errorf("failed to extract page HTML: %w", err)
	}

	// Detect forms
	detector := NewFormDetector()
	forms, err := detector.DetectAllForms(html)
	if err != nil {
		return nil, fmt.Errorf("failed to detect forms: %w", err)
	}

	// Filter by form type if specified
	if req.FormType != "" {
		filterType := FormType(req.FormType)
		var filtered []DetectedForm
		for _, form := range forms {
			if form.FormType == filterType {
				filtered = append(filtered, form)
			}
		}
		forms = filtered
	}

	result.Forms = forms
	result.FormCount = len(forms)

	// Collect unique form types
	typeMap := make(map[string]bool)
	for _, form := range forms {
		typeMap[string(form.FormType)] = true
	}
	for formType := range typeMap {
		result.DetectedTypes = append(result.DetectedTypes, formType)
	}

	return result, nil
}

// MarshalJSON implements custom JSON marshaling for FormFillResult.
func (r FormFillResult) MarshalJSON() ([]byte, error) {
	type Alias FormFillResult
	return json.Marshal(&struct {
		Alias
	}{
		Alias: (Alias)(r),
	})
}

// MarshalJSON implements custom JSON marshaling for FormDetectResponse.
func (r FormDetectResponse) MarshalJSON() ([]byte, error) {
	type Alias FormDetectResponse
	return json.Marshal(&struct {
		Alias
	}{
		Alias: (Alias)(r),
	})
}
