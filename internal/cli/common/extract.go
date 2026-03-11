// Package common provides extraction-template helpers shared by CLI entrypoints.
//
// Purpose:
// - Load inline extraction templates from CLI flags without duplicating parsing logic.
//
// Responsibilities:
// - Build extract.ExtractOptions from template/config/validate flags.
// - Parse inline template JSON files into extract.Template values.
//
// Scope:
// - CLI flag translation only.
//
// Usage:
// - Used by scrape, crawl, research, and schedule commands.
//
// Invariants/Assumptions:
// - Empty config paths do not error.
// - Invalid JSON returns a classified user-facing error.
package common

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/fitchmultz/spartan-scraper/internal/extract"
)

func LoadExtractOptions(template, configPath string, validateSchema bool) (extract.ExtractOptions, error) {
	opts := extract.ExtractOptions{
		Template: template,
		Validate: validateSchema,
	}
	if configPath == "" {
		return opts, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return extract.ExtractOptions{}, err
	}
	var tmpl extract.Template
	if err := json.Unmarshal(data, &tmpl); err != nil {
		return extract.ExtractOptions{}, fmt.Errorf("invalid template JSON: %w", err)
	}
	opts.Inline = &tmpl
	return opts, nil
}
