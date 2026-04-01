// Package aiauthoring provides aiauthoring functionality for Spartan Scraper.
//
// Purpose:
// - Implement debug support for package aiauthoring.
//
// Responsibilities:
// - Define the file-local types, functions, and helpers that belong to this package concern.
//
// Scope:
// - Package-internal behavior owned by this file; broader orchestration stays in adjacent package files.
//
// Usage:
// - Used by other files in package `aiauthoring` and any exported callers that depend on this package.
//
// Invariants/Assumptions:
// - This file should preserve the package contract and rely on surrounding package configuration as the source of truth.

package aiauthoring

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/andybalholm/cascadia"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
)

type templateDiagnostics struct {
	Issues          []string
	ExtractedFields map[string]extract.FieldValue
}

func (d templateDiagnostics) FieldNames() []string {
	if len(d.ExtractedFields) == 0 {
		return nil
	}
	names := make([]string, 0, len(d.ExtractedFields))
	for name := range d.ExtractedFields {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func templateFieldNames(template extract.Template) []string {
	seen := map[string]struct{}{}
	for _, rule := range template.Selectors {
		if name := strings.TrimSpace(rule.Name); name != "" {
			seen[name] = struct{}{}
		}
	}
	for _, rule := range template.JSONLD {
		if name := strings.TrimSpace(rule.Name); name != "" {
			seen[name] = struct{}{}
		}
	}
	for _, rule := range template.Regex {
		if name := strings.TrimSpace(rule.Name); name != "" {
			seen[name] = struct{}{}
		}
	}
	if len(seen) == 0 {
		return nil
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func analyzeTemplate(pageURL string, html string, template extract.Template) templateDiagnostics {
	issues := validateGeneratedTemplate(html, template)
	out := templateDiagnostics{Issues: issues}
	if len(issues) != 0 {
		return out
	}

	extracted, err := extract.ApplyTemplate(pageURL, html, template)
	if err == nil {
		out.ExtractedFields = extracted.Fields
	}

	jsonldObjects, _ := extract.ExtractJSONLD(html)
	for _, rule := range template.JSONLD {
		matches := extract.MatchJSONLD(jsonldObjects, rule)
		if rule.Required && len(matches) == 0 {
			out.Issues = append(out.Issues, fmt.Sprintf("required JSON-LD rule %s matched no values", rule.Name))
		}
	}

	for _, rule := range template.Regex {
		re, err := regexp.Compile(rule.Pattern)
		if err != nil {
			out.Issues = append(out.Issues, fmt.Sprintf("regex %s is invalid: %s", rule.Name, err.Error()))
			continue
		}
		input := html
		switch rule.Source {
		case extract.RegexSourceURL:
			input = pageURL
		case extract.RegexSourceText:
			if extracted.Text != "" {
				input = extracted.Text
			}
		}
		if rule.Required && re.FindStringSubmatch(input) == nil {
			out.Issues = append(out.Issues, fmt.Sprintf("required regex %s matched no values", rule.Name))
		}
	}

	return out
}

func buildTemplateDebugDescription(template extract.Template, instructions string) string {
	base := fmt.Sprintf("Repair or improve the extraction template named %q while preserving its intent and making selectors more robust against real page structure.", template.Name)
	if strings.TrimSpace(instructions) == "" {
		return base
	}
	return base + " Operator instructions: " + strings.TrimSpace(instructions)
}

func buildTemplateDebugFeedback(template extract.Template, diagnostics templateDiagnostics, instructions string) string {
	parts := []string{}
	if strings.TrimSpace(instructions) != "" {
		parts = append(parts, "Operator instructions: "+strings.TrimSpace(instructions))
	}
	if data, err := json.MarshalIndent(template, "", "  "); err == nil {
		parts = append(parts, "Current template:\n"+string(data))
	}
	if len(diagnostics.Issues) > 0 {
		parts = append(parts, "Local diagnostics:\n- "+strings.Join(diagnostics.Issues, "\n- "))
	}
	if len(diagnostics.ExtractedFields) > 0 {
		if data, err := json.MarshalIndent(diagnostics.ExtractedFields, "", "  "); err == nil {
			parts = append(parts, "Current extracted fields:\n"+string(data))
		}
	}
	return strings.Join(parts, "\n\n")
}

func validateGeneratedTemplate(html string, template extract.Template) []string {
	if strings.TrimSpace(template.Name) == "" {
		return []string{"name is required"}
	}
	if len(template.Selectors) == 0 && len(template.JSONLD) == 0 && len(template.Regex) == 0 {
		return []string{"at least one selector, jsonld rule, or regex rule is required"}
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return []string{"generated template could not be validated because the fetched HTML was not parseable"}
	}

	validationErrors := make([]string, 0)
	for _, rule := range template.Selectors {
		if strings.TrimSpace(rule.Name) == "" {
			validationErrors = append(validationErrors, "selector rule is missing a field name")
			continue
		}
		if strings.TrimSpace(rule.Selector) == "" {
			validationErrors = append(validationErrors, fmt.Sprintf("selector %s is empty", rule.Name))
			continue
		}
		if _, err := cascadia.ParseGroup(rule.Selector); err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("selector %s is invalid: %s", rule.Name, err.Error()))
			continue
		}
		if doc.Find(rule.Selector).Length() == 0 {
			validationErrors = append(validationErrors, fmt.Sprintf("selector %s matched no elements", rule.Name))
		}
	}

	return validationErrors
}
