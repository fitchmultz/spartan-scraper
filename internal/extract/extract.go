// Package extract provides functionality for extracting structured data from HTML.
// It uses templates to define how data should be extracted and supports
// normalization and schema validation of the extracted results.
package extract

import (
	"context"
	"errors"
	"log/slog"
)

// Result is the legacy extraction result.
type Result struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Text        string   `json:"text"`
	Links       []string `json:"links"`
}

// Execute runs the extraction pipeline.
func Execute(input ExecuteInput) (ExecuteOutput, error) {
	registry := input.Registry
	if registry == nil {
		var err error
		registry, err = LoadTemplateRegistry(input.DataDir)
		if registry == nil {
			// Fallback to built-ins if loading fails or returns nil
			registry = &TemplateRegistry{Templates: make(map[string]Template)}
		}
		_ = err // Note: err from LoadTemplateRegistry is mostly for file read errors, we proceed with defaults/nil registry.
	}

	tmpl, err := ResolveTemplate(input.Options, registry)
	if err != nil {
		return ExecuteOutput{}, err
	}

	extracted, err := ApplyTemplate(input.URL, input.HTML, tmpl)
	if err != nil {
		return ExecuteOutput{}, err
	}

	normalized := Normalize(extracted, tmpl)

	// NEW: If AI extraction is enabled, enhance results
	if input.Options.AI != nil && input.Options.AI.Enabled && input.AIExtractor != nil {
		aiResult, err := input.AIExtractor.Extract(context.Background(), AIExtractRequest{
			HTML:            input.HTML,
			URL:             input.URL,
			Mode:            input.Options.AI.Mode,
			Prompt:          input.Options.AI.Prompt,
			SchemaExample:   input.Options.AI.Schema,
			Fields:          input.Options.AI.Fields,
			MaxContentChars: DefaultMaxContentChars,
		})
		if err == nil {
			// Merge AI-extracted fields with template results
			if extracted.Fields == nil {
				extracted.Fields = make(map[string]FieldValue)
			}
			for name, value := range aiResult.Fields {
				extracted.Fields[name] = value
			}
			// Also update normalized fields
			if normalized.Fields == nil {
				normalized.Fields = make(map[string]FieldValue)
			}
			for name, value := range aiResult.Fields {
				normalized.Fields[name] = value
			}
			slog.Debug("AI extraction enhanced results",
				"url", input.URL,
				"fields", len(aiResult.Fields),
				"confidence", aiResult.Confidence,
				"cached", aiResult.Cached)
		} else {
			// AI errors are logged but don't fail extraction
			slog.Warn("AI extraction failed, continuing with template extraction", "error", err)
		}
	}

	if input.Options.Validate && tmpl.Schema != nil {
		validation := ValidateDocument(normalized, tmpl.Schema)

		// Get effective rejection policy
		policy := GetEffectiveRejectionPolicy(input.Options, tmpl)

		// Apply rejection policy
		result := ApplyRejectionPolicy(normalized, validation, policy)

		// Handle rejection result
		if result.Error != nil {
			return ExecuteOutput{}, result.Error
		}

		if result.Skip {
			// Document was skipped due to validation failure
			return ExecuteOutput{}, errors.Join(ErrDocumentSkipped, ErrValidationFailed)
		}

		normalized = result.Document
	}

	return ExecuteOutput{
		Extracted:  extracted,
		Normalized: normalized,
	}, nil
}

// FromHTML is a legacy helper that uses the default extraction template.
func FromHTML(html string) (Result, error) {
	output, err := Execute(ExecuteInput{
		HTML:    html,
		Options: ExtractOptions{Template: "default"},
	})
	if err != nil {
		return Result{}, err
	}

	return Result{
		Title:       output.Normalized.Title,
		Description: output.Normalized.Description,
		Text:        output.Normalized.Text,
		Links:       output.Normalized.Links,
	}, nil
}
