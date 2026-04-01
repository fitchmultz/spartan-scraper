// Package scheduler provides scheduler functionality for Spartan Scraper.
//
// Purpose:
// - Implement export normalize support for package scheduler.
//
// Responsibilities:
// - Define the file-local types, functions, and helpers that belong to this package concern.
//
// Scope:
// - Package-internal behavior owned by this file; broader orchestration stays in adjacent package files.
//
// Usage:
// - Used by other files in package `scheduler` and any exported callers that depend on this package.
//
// Invariants/Assumptions:
// - This file should preserve the package contract and rely on surrounding package configuration as the source of truth.

package scheduler

import (
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/exporter"
)

const defaultLocalExportPathTemplate = "exports/{kind}/{job_id}.{format}"

// NormalizeExportSchedule applies shared export-schedule defaults and trimming so
// every interface (API, Web, CLI, MCP, scheduler) persists the same contract.
func NormalizeExportSchedule(schedule ExportSchedule) ExportSchedule {
	schedule.Name = strings.TrimSpace(schedule.Name)
	schedule.Export = NormalizeExportConfig(schedule.Export)
	return schedule
}

// NormalizeExportConfig applies shared export config defaults and trimming.
func NormalizeExportConfig(config ExportConfig) ExportConfig {
	config.Format = strings.TrimSpace(config.Format)
	config.DestinationType = strings.TrimSpace(config.DestinationType)
	config.LocalPath = strings.TrimSpace(config.LocalPath)
	config.WebhookURL = strings.TrimSpace(config.WebhookURL)
	config.PathTemplate = strings.TrimSpace(config.PathTemplate)
	config.Shape = exporter.NormalizeShapeConfig(config.Shape)
	config.Transform = exporter.NormalizeTransformConfig(config.Transform)

	if config.DestinationType == "local" {
		if config.PathTemplate == "" && config.LocalPath == "" {
			config.PathTemplate = defaultLocalExportPathTemplate
		}
		if config.LocalPath == "" {
			config.LocalPath = config.PathTemplate
		}
		if config.PathTemplate == "" {
			config.PathTemplate = config.LocalPath
		}
	}

	return config
}
