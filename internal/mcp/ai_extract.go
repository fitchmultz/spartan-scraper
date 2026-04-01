// Package mcp provides mcp functionality for Spartan Scraper.
//
// Purpose:
// - Implement ai extract support for package mcp.
//
// Responsibilities:
// - Define the file-local types, functions, and helpers that belong to this package concern.
//
// Scope:
// - Package-internal behavior owned by this file; broader orchestration stays in adjacent package files.
//
// Usage:
// - Used by other files in package `mcp` and any exported callers that depend on this package.
//
// Invariants/Assumptions:
// - This file should preserve the package contract and rely on surrounding package configuration as the source of truth.

package mcp

import (
	"fmt"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/paramdecode"
)

func decodeAIExtractOptions(arguments map[string]any) (*extract.AIExtractOptions, error) {
	if !paramdecode.Bool(arguments, "aiExtract") {
		return nil, nil
	}

	mode := extract.AIExtractionMode(strings.TrimSpace(paramdecode.String(arguments, "aiMode")))
	if mode == "" {
		mode = extract.AIModeNaturalLanguage
	}

	fields := paramdecode.StringSlice(arguments, "aiFields")
	switch mode {
	case extract.AIModeNaturalLanguage:
		return &extract.AIExtractOptions{
			Enabled: true,
			Mode:    mode,
			Prompt:  strings.TrimSpace(paramdecode.String(arguments, "aiPrompt")),
			Fields:  fields,
		}, nil
	case extract.AIModeSchemaGuided:
		schema := paramdecode.Decode[map[string]interface{}](arguments, "aiSchema")
		if len(schema) == 0 {
			return nil, fmt.Errorf("aiSchema is required when aiMode is schema_guided")
		}
		return &extract.AIExtractOptions{
			Enabled: true,
			Mode:    mode,
			Schema:  schema,
			Fields:  fields,
		}, nil
	default:
		return nil, fmt.Errorf("invalid aiMode %q: must be natural_language or schema_guided", mode)
	}
}
