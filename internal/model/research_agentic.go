// Package model provides model functionality for Spartan Scraper.
//
// Purpose:
// - Implement research agentic support for package model.
//
// Responsibilities:
// - Define the file-local types, functions, and helpers that belong to this package concern.
//
// Scope:
// - Package-internal behavior owned by this file; broader orchestration stays in adjacent package files.
//
// Usage:
// - Used by other files in package `model` and any exported callers that depend on this package.
//
// Invariants/Assumptions:
// - This file should preserve the package contract and rely on surrounding package configuration as the source of truth.

package model

import (
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

const (
	DefaultResearchAgenticMaxRounds       = 1
	DefaultResearchAgenticMaxFollowUpURLs = 3
	MaxResearchAgenticMaxRounds           = 3
	MaxResearchAgenticMaxFollowUpURLs     = 10
)

// ResearchAgenticConfig configures bounded pi-powered research refinement.
type ResearchAgenticConfig struct {
	Enabled         bool   `json:"enabled,omitempty"`
	Instructions    string `json:"instructions,omitempty"`
	MaxRounds       int    `json:"maxRounds,omitempty"`
	MaxFollowUpURLs int    `json:"maxFollowUpUrls,omitempty"`
}

// NormalizeResearchAgenticConfig applies stable defaults to a research agentic config.
func NormalizeResearchAgenticConfig(cfg *ResearchAgenticConfig) *ResearchAgenticConfig {
	if cfg == nil {
		return nil
	}
	out := *cfg
	out.Instructions = strings.TrimSpace(out.Instructions)
	if out.MaxRounds <= 0 {
		out.MaxRounds = DefaultResearchAgenticMaxRounds
	}
	if out.MaxFollowUpURLs <= 0 {
		out.MaxFollowUpURLs = DefaultResearchAgenticMaxFollowUpURLs
	}
	return &out
}

// ValidateResearchAgenticConfig validates bounded research-agent settings.
func ValidateResearchAgenticConfig(cfg *ResearchAgenticConfig) error {
	if cfg == nil || !cfg.Enabled {
		return nil
	}
	if cfg.MaxRounds < 0 {
		return apperrors.Validation("agentic.maxRounds must be non-negative")
	}
	if cfg.MaxRounds > MaxResearchAgenticMaxRounds {
		return apperrors.Validation("agentic.maxRounds must be between 1 and 3")
	}
	if cfg.MaxFollowUpURLs < 0 {
		return apperrors.Validation("agentic.maxFollowUpUrls must be non-negative")
	}
	if cfg.MaxFollowUpURLs > MaxResearchAgenticMaxFollowUpURLs {
		return apperrors.Validation("agentic.maxFollowUpUrls must be between 1 and 10")
	}
	return nil
}
