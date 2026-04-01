// Package mcp provides mcp functionality for Spartan Scraper.
//
// Purpose:
// - Implement agentic support for package mcp.
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
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/paramdecode"
)

func decodeResearchAgenticOptions(arguments map[string]any) (*model.ResearchAgenticConfig, error) {
	if !paramdecode.Bool(arguments, "agentic") {
		return nil, nil
	}
	cfg := model.NormalizeResearchAgenticConfig(&model.ResearchAgenticConfig{
		Enabled:         true,
		Instructions:    paramdecode.String(arguments, "agenticInstructions"),
		MaxRounds:       paramdecode.PositiveInt(arguments, "agenticMaxRounds", model.DefaultResearchAgenticMaxRounds),
		MaxFollowUpURLs: paramdecode.PositiveInt(arguments, "agenticMaxFollowUpUrls", model.DefaultResearchAgenticMaxFollowUpURLs),
	})
	return cfg, model.ValidateResearchAgenticConfig(cfg)
}
