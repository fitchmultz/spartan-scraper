// Package manage contains CLI commands for configuration/data management.
//
// This file implements the 'spartan form' command for form detection and filling.
// It provides subcommands for:
//   - detect: Preview detected forms and fields on a page
//   - fill: Fill and optionally submit a form
//   - submit: Submit a form (useful when fields are already filled)
//
// It does NOT handle CAPTCHA solving or complex JavaScript interactions.
package manage

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
)

// RunForm executes the 'form' CLI command.
func RunForm(ctx context.Context, cfg config.Config, args []string) int {
	if len(args) < 1 {
		printFormHelp()
		return 1
	}

	if isHelpToken(args[0]) {
		printFormHelp()
		return 0
	}

	switch args[0] {
	case "detect":
		return runFormDetect(ctx, cfg, args[1:])
	case "fill":
		return runFormFill(ctx, cfg, args[1:])
	case "submit":
		return runFormSubmit(ctx, cfg, args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown form subcommand: %s\n", args[0])
		printFormHelp()
		return 1
	}
}

// runFormDetect handles 'spartan form detect' command.
func runFormDetect(_ context.Context, _ config.Config, args []string) int {
	fs := flag.NewFlagSet("form detect", flag.ExitOnError)
	url := fs.String("url", "", "URL to detect forms on (required)")
	formType := fs.String("type", "", "Filter by form type (login, register, search, contact, newsletter, checkout, survey)")
	format := fs.String("format", "table", "Output format: table, json")
	headless := fs.Bool("headless", true, "Use headless browser mode")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing flags: %v\n", err)
		return 1
	}

	if *url == "" {
		fmt.Fprintln(os.Stderr, "--url is required")
		return 1
	}

	// Create form filler
	filler := fetch.NewFormFiller(nil)

	// Detect forms
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := fetch.FormDetectRequest{
		URL:      *url,
		FormType: *formType,
		Headless: *headless,
	}

	result, err := filler.Detect(ctx, req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to detect forms: %v\n", err)
		return 1
	}

	// Output results
	switch *format {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			fmt.Fprintf(os.Stderr, "failed to encode JSON: %v\n", err)
			return 1
		}
	default:
		printFormDetectResults(result)
	}

	return 0
}

// runFormFill handles 'spartan form fill' command.
func runFormFill(_ context.Context, _ config.Config, args []string) int {
	fs := flag.NewFlagSet("form fill", flag.ExitOnError)
	url := fs.String("url", "", "URL containing the form (required)")
	data := fs.String("data", "", "JSON object with field values, e.g. '{\"name\": \"John\", \"email\": \"john@example.com\"}'")
	formSelector := fs.String("form-selector", "", "CSS selector for the form (auto-detect if not specified)")
	submit := fs.Bool("submit", false, "Submit the form after filling")
	waitFor := fs.String("wait-for", "", "CSS selector to wait for after submission")
	timeout := fs.Int("timeout", 30, "Timeout in seconds")
	headless := fs.Bool("headless", true, "Use headless browser mode")
	formType := fs.String("form-type", "", "Expected form type for auto-detection (login, contact, search, etc.)")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing flags: %v\n", err)
		return 1
	}

	if *url == "" {
		fmt.Fprintln(os.Stderr, "--url is required")
		return 1
	}

	if *data == "" {
		fmt.Fprintln(os.Stderr, "--data is required (JSON object with field values)")
		return 1
	}

	// Parse field data
	var fields map[string]string
	if err := json.Unmarshal([]byte(*data), &fields); err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse field data JSON: %v\n", err)
		return 1
	}

	// Create form filler
	filler := fetch.NewFormFiller(nil)

	// Fill form
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeout)*time.Second)
	defer cancel()

	req := fetch.FormFillRequest{
		URL:            *url,
		FormSelector:   *formSelector,
		Fields:         fields,
		Submit:         *submit,
		WaitFor:        *waitFor,
		Timeout:        time.Duration(*timeout) * time.Second,
		Headless:       *headless,
		FormTypeFilter: *formType,
	}

	result, err := filler.FillForm(ctx, req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "form fill failed: %v\n", err)
		// Continue to print result even on error
	}

	// Output results
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode JSON: %v\n", err)
		return 1
	}

	if !result.Success {
		return 1
	}

	return 0
}

// runFormSubmit handles 'spartan form submit' command.
func runFormSubmit(_ context.Context, _ config.Config, args []string) int {
	fs := flag.NewFlagSet("form submit", flag.ExitOnError)
	url := fs.String("url", "", "URL containing the form (required)")
	formSelector := fs.String("form-selector", "", "CSS selector for the form (auto-detect if not specified)")
	waitFor := fs.String("wait-for", "", "CSS selector to wait for after submission")
	timeout := fs.Int("timeout", 30, "Timeout in seconds")
	headless := fs.Bool("headless", true, "Use headless browser mode")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing flags: %v\n", err)
		return 1
	}

	if *url == "" {
		fmt.Fprintln(os.Stderr, "--url is required")
		return 1
	}

	// Create form filler
	filler := fetch.NewFormFiller(nil)

	// Submit form (empty fields, just submit)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeout)*time.Second)
	defer cancel()

	req := fetch.FormFillRequest{
		URL:          *url,
		FormSelector: *formSelector,
		Fields:       map[string]string{},
		Submit:       true,
		WaitFor:      *waitFor,
		Timeout:      time.Duration(*timeout) * time.Second,
		Headless:     *headless,
	}

	result, err := filler.FillForm(ctx, req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "form submit failed: %v\n", err)
	}

	// Output results
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode JSON: %v\n", err)
		return 1
	}

	if !result.Success {
		return 1
	}

	return 0
}

// printFormDetectResults prints form detection results in table format.
func printFormDetectResults(result *fetch.FormDetectResponse) {
	fmt.Printf("Forms detected on %s:\n\n", result.URL)

	if len(result.Forms) == 0 {
		fmt.Println("No forms detected.")
		return
	}

	for i, form := range result.Forms {
		fmt.Printf("Form %d:\n", i+1)
		fmt.Printf("  Selector: %s\n", form.FormSelector)
		fmt.Printf("  Type: %s\n", form.FormType)
		fmt.Printf("  Confidence: %.2f\n", form.Score)
		if form.Action != "" {
			fmt.Printf("  Action: %s\n", form.Action)
		}
		if form.Method != "" {
			fmt.Printf("  Method: %s\n", form.Method)
		}

		if len(form.AllFields) > 0 {
			fmt.Printf("  Fields:\n")
			for _, field := range form.AllFields {
				req := ""
				if field.Required {
					req = " (required)"
				}
				fmt.Printf("    - %s (%s)%s\n", field.FieldName, field.FieldType, req)
				fmt.Printf("      Selector: %s\n", field.Selector)
				if field.Placeholder != "" {
					fmt.Printf("      Placeholder: %s\n", field.Placeholder)
				}
			}
		}

		fmt.Println()
	}
}

// printFormHelp prints help for the form command.
func printFormHelp() {
	help := `Usage: spartan form <command> [options]

Commands:
  detect    Detect forms on a page and show their structure
  fill      Fill form fields with provided data
  submit    Submit a form (useful when fields are already filled)

Detect Options:
  --url string         URL to detect forms on (required)
  --type string        Filter by form type (login, register, search, contact, newsletter, checkout, survey)
  --format string      Output format: table, json (default: table)
  --headless bool      Use headless browser mode (default: true)

Fill Options:
  --url string         URL containing the form (required)
  --data string        JSON object with field values (required)
                       Example: '{"name": "John", "email": "john@example.com", "message": "Hello"}'
  --form-selector string   CSS selector for the form (auto-detect if not specified)
  --submit             Submit the form after filling
  --wait-for string    CSS selector to wait for after submission
  --timeout int        Timeout in seconds (default: 30)
  --headless bool      Use headless browser mode (default: true)
  --form-type string   Expected form type for auto-detection

Submit Options:
  --url string         URL containing the form (required)
  --form-selector string   CSS selector for the form (auto-detect if not specified)
  --wait-for string    CSS selector to wait for after submission
  --timeout int        Timeout in seconds (default: 30)
  --headless bool      Use headless browser mode (default: true)

Examples:
  # Detect all forms on a page
  spartan form detect --url https://example.com/contact

  # Detect only contact forms
  spartan form detect --url https://example.com/contact --type contact

  # Fill a contact form
  spartan form fill --url https://example.com/contact \
    --data '{"name": "John Doe", "email": "john@example.com", "message": "Hello!"}' \
    --submit

  # Fill a search form and submit
  spartan form fill --url https://example.com/search \
    --data '{"q": "search query"}' \
    --submit \
    --wait-for ".results"
`
	fmt.Print(help)
}
