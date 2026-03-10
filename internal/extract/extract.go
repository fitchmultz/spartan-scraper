// Package extract runs template-driven extraction and optional AI enhancement.
//
// Purpose:
// - Execute extraction from HTML into normalized structured documents.
//
// Responsibilities:
// - Load template registries when callers do not provide one.
// - Apply extraction, normalization, optional AI enrichment, and validation.
// - Preserve the minimal legacy adapter used by existing external result surfaces.
//
// Scope:
// - Extraction execution entrypoints only.
//
// Usage:
// - Called by scrape, crawl, and other packages through Execute.
//
// Invariants/Assumptions:
// - Template registry load failures are surfaced instead of silently ignored.
// - AI extraction failures do not abort template extraction.
// - Validation failures follow the configured rejection policy.
package extract

import (
	"context"
	"errors"
	"log/slog"
)

// Result is the compact compatibility summary exposed by scrape and crawl results.
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
		loadedRegistry, err := LoadTemplateRegistry(input.DataDir)
		if err != nil {
			return ExecuteOutput{}, err
		}
		registry = loadedRegistry
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
		// Use the caller context when available so AI work shares cancellation.
		ctx := input.Context
		if ctx == nil {
			ctx = context.Background()
		}
		aiResult, err := input.AIExtractor.Extract(ctx, AIExtractRequest{
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

// FromHTML is the minimal compatibility helper for callers that only need the summary result.
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
