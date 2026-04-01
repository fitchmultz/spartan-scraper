// Package scheduler provides automated export scheduling functionality.
//
// Purpose:
// - Define the 1.0 export schedule model for event-driven job-result exports.
//
// Responsibilities:
// - Describe export filters, destination configuration, and retry settings.
// - Keep the supported format/destination contract centralized for validation and UI.
//
// Scope:
//   - Types and helpers only. Persistence, validation, trigger execution, and history
//     tracking live in separate files.
//
// Usage:
// - Used by the scheduler, API handlers, CLI commands, and generated OpenAPI-backed UI.
//
// Invariants/Assumptions:
// - ExportSchedule IDs are UUIDs generated on creation.
// - Filters must specify at least one criteria (job kind, status, or has-results).
// - ExportConfig.Format is one of json, jsonl, md, csv, or xlsx.
// - ExportConfig.DestinationType is either local or webhook.
// - ExportConfig.Shape and ExportConfig.Transform are mutually exclusive.
package scheduler

import (
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/exporter"
)

// ExportSchedule represents an automated export configuration that triggers
// when jobs complete matching specified filter criteria.
type ExportSchedule struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Filter criteria - which jobs to export
	Filters ExportFilters `json:"filters"`

	// Export configuration
	Export ExportConfig `json:"export"`

	// Retry configuration
	Retry ExportRetryConfig `json:"retry"`
}

// ExportFilters defines which jobs match this export schedule.
// All specified criteria must match (AND logic).
type ExportFilters struct {
	// JobKinds filters by job type (scrape, crawl, research).
	// Empty means match all kinds.
	JobKinds []string `json:"job_kinds,omitempty"`

	// JobStatus filters by terminal job status (completed, failed).
	// Empty defaults to ["completed"].
	JobStatus []string `json:"job_status,omitempty"`

	// HasResults when true, only exports jobs with non-empty results.
	HasResults bool `json:"has_results,omitempty"`
}

// ExportConfig defines where and how to export job results.
type ExportConfig struct {
	// Format is the export format: json, jsonl, md, csv, xlsx
	Format string `json:"format"`

	// DestinationType is the export destination: local or webhook
	DestinationType string `json:"destination_type"`

	// LocalPath is the local file path template (for local destination).
	// Supports variables: {job_id}, {timestamp}, {kind}, {format}
	LocalPath string `json:"local_path,omitempty"`

	// WebhookURL is the webhook endpoint (for webhook destination).
	WebhookURL string `json:"webhook_url,omitempty"`

	// PathTemplate is the path template with variables for local destinations.
	// Default: "exports/{kind}/{job_id}.{format}"
	// Supported variables: {job_id}, {timestamp}, {kind}, {format}
	PathTemplate string `json:"path_template,omitempty"`

	// Shape applies deterministic export shaping for markdown and tabular exports.
	Shape exporter.ShapeConfig `json:"shape,omitempty"`

	// Transform applies a deterministic result transformation before export.
	Transform exporter.TransformConfig `json:"transform,omitempty"`
}

// ExportRetryConfig defines retry behavior for failed exports.
type ExportRetryConfig struct {
	// MaxRetries is the maximum number of retry attempts (default: 3).
	MaxRetries int `json:"max_retries"`

	// BaseDelayMs is the initial retry delay in milliseconds (default: 1000).
	BaseDelayMs int `json:"base_delay_ms"`
}

// DefaultRetryConfig returns the default retry configuration.
func DefaultRetryConfig() ExportRetryConfig {
	return ExportRetryConfig{
		MaxRetries:  3,
		BaseDelayMs: 1000,
	}
}

// GetMaxRetries returns the effective max retries (with default).
func (r ExportRetryConfig) GetMaxRetries() int {
	if r.MaxRetries <= 0 {
		return 3
	}
	return r.MaxRetries
}

// GetBaseDelay returns the effective base delay (with default).
func (r ExportRetryConfig) GetBaseDelay() time.Duration {
	if r.BaseDelayMs <= 0 {
		return time.Second
	}
	return time.Duration(r.BaseDelayMs) * time.Millisecond
}

// exportScheduleStore is the JSON persistence format for export schedules.
type exportScheduleStore struct {
	Schedules []ExportSchedule `json:"schedules"`
}

// SupportedExportFormats returns the list of supported export formats.
func SupportedExportFormats() []string {
	return []string{"json", "jsonl", "md", "csv", "xlsx"}
}

// SupportedDestinationTypes returns the list of supported destination types.
func SupportedDestinationTypes() []string {
	return []string{"local", "webhook"}
}

// IsValidExportFormat returns true if the format is supported.
func IsValidExportFormat(format string) bool {
	switch format {
	case "json", "jsonl", "md", "csv", "xlsx":
		return true
	}
	return false
}

// IsValidDestinationType returns true if the destination type is supported.
func IsValidDestinationType(dest string) bool {
	switch dest {
	case "local", "webhook":
		return true
	}
	return false
}

// IsCloudDestination reports whether the destination is a removed legacy cloud target.
// Balanced 1.0 does not support cloud exporters, so this always returns false.
func IsCloudDestination(dest string) bool {
	return false
}
