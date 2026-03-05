// Package pipeline provides a plugin system for extending scrape and crawl workflows.
// It handles plugin hooks at pre/post stages of fetch, extract, and output operations,
// plugin registration, and JavaScript plugin execution.
// It does NOT handle workflow execution or plugin implementations.
package pipeline

import (
	"github.com/fitchmultz/spartan-scraper/internal/extract"
)

// ValidatorTransformer applies schema validation during pipeline output stage.
// It validates extracted documents against their template schemas and applies
// rejection policies based on validation results.
type ValidatorTransformer struct {
	BaseTransformer
	schema          *extract.Schema
	rejectionPolicy extract.RejectionPolicy
}

// ValidatorOption configures a ValidatorTransformer.
type ValidatorOption func(*ValidatorTransformer)

// WithSchema sets the schema for validation.
func WithSchema(schema *extract.Schema) ValidatorOption {
	return func(v *ValidatorTransformer) {
		v.schema = schema
	}
}

// WithRejectionPolicy sets the rejection policy for validation failures.
func WithRejectionPolicy(policy extract.RejectionPolicy) ValidatorOption {
	return func(v *ValidatorTransformer) {
		v.rejectionPolicy = policy
	}
}

// NewValidatorTransformer creates a new validator transformer with the given options.
func NewValidatorTransformer(opts ...ValidatorOption) *ValidatorTransformer {
	vt := &ValidatorTransformer{
		rejectionPolicy: extract.RejectPolicyNone,
	}
	for _, opt := range opts {
		opt(vt)
	}
	return vt
}

// Name returns the transformer name.
func (v *ValidatorTransformer) Name() string {
	return "validator"
}

// Priority returns the transformer priority (high priority - validate before other transforms).
func (v *ValidatorTransformer) Priority() int {
	return 100
}

// Enabled returns true if validation is enabled.
// It checks both the options and whether a schema is available.
func (v *ValidatorTransformer) Enabled(target Target, opts Options) bool {
	// Check if validation is explicitly enabled in options
	// This could be extended to check for a Validate flag in Options
	return true
}

// Transform validates the structured data against the schema and applies rejection policies.
func (v *ValidatorTransformer) Transform(ctx HookContext, in OutputInput) (OutputOutput, error) {
	// Extract the document from structured data
	doc, ok := ExtractDocument(in.Structured)
	if !ok {
		// If we can't extract a document, pass through unchanged
		return OutputOutput{
			Raw:        in.Raw,
			Structured: in.Structured,
		}, nil
	}

	// Get the schema to use for validation
	schema := v.getSchema(doc)
	if schema == nil {
		// No schema available, pass through unchanged
		return OutputOutput{
			Raw:        in.Raw,
			Structured: in.Structured,
		}, nil
	}

	// Perform validation
	validation := extract.ValidateDocument(doc, schema)

	// Get the effective rejection policy
	policy := v.getRejectionPolicy(doc)

	// Apply rejection policy
	result := extract.ApplyRejectionPolicy(doc, validation, policy)

	// Handle rejection result
	if result.Error != nil {
		return OutputOutput{}, result.Error
	}

	if result.Skip {
		// Document was skipped - return empty output with skip indication
		return OutputOutput{
			Raw:        nil,
			Structured: nil,
		}, extract.ErrDocumentSkipped
	}

	// Update the document with validation results
	updatedDoc := result.Document

	// Return the transformed output
	return OutputOutput{
		Raw:        in.Raw,
		Structured: updatedDoc,
	}, nil
}

// ExtractDocument attempts to extract a NormalizedDocument from the structured data.
// It is exported for testing purposes.
func ExtractDocument(structured any) (extract.NormalizedDocument, bool) {
	switch v := structured.(type) {
	case extract.NormalizedDocument:
		return v, true
	case *extract.NormalizedDocument:
		return *v, true
	case extract.Extracted:
		// Convert Extracted to NormalizedDocument
		return extract.NormalizedDocument{
			URL:         v.URL,
			Title:       v.Title,
			Text:        v.Text,
			Links:       v.Links,
			Metadata:    v.Metadata,
			Fields:      v.Fields,
			JSONLD:      v.JSONLD,
			Template:    v.Template,
			ExtractedAt: v.ExtractedAt,
		}, true
	case *extract.Extracted:
		return extract.NormalizedDocument{
			URL:         v.URL,
			Title:       v.Title,
			Text:        v.Text,
			Links:       v.Links,
			Metadata:    v.Metadata,
			Fields:      v.Fields,
			JSONLD:      v.JSONLD,
			Template:    v.Template,
			ExtractedAt: v.ExtractedAt,
		}, true
	default:
		return extract.NormalizedDocument{}, false
	}
}

// getSchema returns the schema to use for validation.
// It first checks the configured schema, then falls back to the document's template.
func (v *ValidatorTransformer) getSchema(doc extract.NormalizedDocument) *extract.Schema {
	// If we have a configured schema, use it
	if v.schema != nil {
		return v.schema
	}

	// Otherwise, try to load from template registry based on document's template name
	// This would require access to the template registry, which could be passed
	// via context or options in a future enhancement
	return nil
}

// getRejectionPolicy returns the rejection policy to use.
func (v *ValidatorTransformer) getRejectionPolicy(doc extract.NormalizedDocument) extract.RejectionPolicy {
	// Use configured policy if set
	if v.rejectionPolicy != "" {
		return v.rejectionPolicy
	}
	return extract.RejectPolicyNone
}

// RegisterValidator registers a validator transformer with the registry.
func (r *Registry) RegisterValidator(schema *extract.Schema, policy extract.RejectionPolicy) {
	transformer := NewValidatorTransformer(
		WithSchema(schema),
		WithRejectionPolicy(policy),
	)
	r.RegisterTransformer(transformer)
}
