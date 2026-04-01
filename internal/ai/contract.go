// Package ai provides ai functionality for Spartan Scraper.
//
// Purpose:
// - Implement contract support for package ai.
//
// Responsibilities:
// - Define the file-local types, functions, and helpers that belong to this package concern.
//
// Scope:
// - Package-internal behavior owned by this file; broader orchestration stays in adjacent package files.
//
// Usage:
// - Used by other files in package `ai` and any exported callers that depend on this package.
//
// Invariants/Assumptions:
// - This file should preserve the package contract and rely on surrounding package configuration as the source of truth.

package ai

import (
	"fmt"
	"math"
	"strings"
)

func (r *ExtractResult) Canonicalize() error {
	if r.Fields == nil {
		return fmt.Errorf("extract result missing fields")
	}
	if math.IsNaN(r.Confidence) || math.IsInf(r.Confidence, 0) {
		return fmt.Errorf("extract result confidence must be finite")
	}
	if r.Confidence < 0 {
		r.Confidence = 0
	} else if r.Confidence > 1 {
		r.Confidence = 1
	}
	for name, field := range r.Fields {
		trimmedName := strings.TrimSpace(name)
		if trimmedName == "" {
			return fmt.Errorf("extract result contains empty field name")
		}
		if field.Values == nil {
			field.Values = []string{}
		}
		if strings.TrimSpace(field.Source) == "" {
			field.Source = "llm"
		}
		if trimmedName != name {
			delete(r.Fields, name)
		}
		r.Fields[trimmedName] = field
	}
	return nil
}
