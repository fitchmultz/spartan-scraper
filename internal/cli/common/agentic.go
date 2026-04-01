// Package common provides common functionality for Spartan Scraper.
//
// Purpose:
// - Implement agentic support for package common.
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

import "github.com/fitchmultz/spartan-scraper/internal/model"

func BuildResearchAgenticConfig(cf *CommonFlags) *model.ResearchAgenticConfig {
	if cf == nil || cf.AgenticResearch == nil || !*cf.AgenticResearch {
		return nil
	}
	return model.NormalizeResearchAgenticConfig(&model.ResearchAgenticConfig{
		Enabled:         true,
		Instructions:    valueOrEmpty(cf.AgenticResearchInstructions),
		MaxRounds:       valueOrZero(cf.AgenticResearchMaxRounds),
		MaxFollowUpURLs: valueOrZero(cf.AgenticResearchMaxFollowUps),
	})
}

func valueOrEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func valueOrZero(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}
