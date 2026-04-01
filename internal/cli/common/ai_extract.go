// Package common provides common functionality for Spartan Scraper.
//
// Purpose:
// - Implement ai extract support for package common.
//
// Responsibilities:
// - Define the file-local types, functions, and helpers that belong to this package concern.
//
// Scope:
// - Package-internal behavior owned by this file; broader orchestration stays in adjacent package files.
//
// Usage:
// - Used by other files in package `common` and any exported callers that depend on this package.
//
// Invariants/Assumptions:
// - This file should preserve the package contract and rely on surrounding package configuration as the source of truth.

package common

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/extract"
)

func BuildAIExtractOptions(cf *CommonFlags) (*extract.AIExtractOptions, error) {
	if cf == nil || cf.AIExtract == nil || !*cf.AIExtract {
		return nil, nil
	}

	mode := extract.AIExtractionMode(strings.TrimSpace(*cf.AIExtractMode))
	switch mode {
	case extract.AIModeNaturalLanguage:
		fields := parseAIFields(*cf.AIExtractFields)
		prompt := strings.TrimSpace(*cf.AIExtractPrompt)
		return &extract.AIExtractOptions{
			Enabled: true,
			Mode:    mode,
			Prompt:  prompt,
			Fields:  fields,
		}, nil
	case extract.AIModeSchemaGuided:
		schema, err := parseAISchema(*cf.AIExtractSchema)
		if err != nil {
			return nil, err
		}
		fields := parseAIFields(*cf.AIExtractFields)
		return &extract.AIExtractOptions{
			Enabled: true,
			Mode:    mode,
			Schema:  schema,
			Fields:  fields,
		}, nil
	default:
		return nil, fmt.Errorf("invalid --ai-mode: %s (must be 'natural_language' or 'schema_guided')", *cf.AIExtractMode)
	}
}

func parseAIFields(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	fields := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			fields = append(fields, trimmed)
		}
	}
	if len(fields) == 0 {
		return nil
	}
	return fields
}

func parseAISchema(raw string) (map[string]interface{}, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, fmt.Errorf("--ai-schema is required when --ai-mode schema_guided")
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return nil, fmt.Errorf("invalid --ai-schema JSON: %w", err)
	}
	if len(decoded) == 0 {
		return nil, fmt.Errorf("--ai-schema must be a non-empty JSON object")
	}
	return decoded, nil
}
