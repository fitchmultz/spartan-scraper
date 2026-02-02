// Package scheduler provides automated export scheduling functionality.
//
// This file defines the ExportSchedule type and related configuration types
// for triggering exports when jobs complete based on filter criteria.
//
// Export schedules are event-driven (triggered on job completion) rather than
// time-driven like regular schedules. They allow automatic export of job
// results to various destinations (S3, GCS, Azure, local file, webhook).
//
// This file does NOT handle:
// - Persistence (export_storage.go handles that)
// - Validation (export_validation.go handles that)
// - Trigger execution (export_trigger.go handles that)
// - History tracking (export_history.go handles that)
//
// Invariants:
// - ExportSchedule IDs are UUIDs generated on creation
// - Filters must specify at least one criteria (job kind, status, or tags)
// - ExportConfig.Format must be a supported format
// - ExportConfig.DestinationType must be a supported destination
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

	// Tags filters by job tags. All specified tags must be present (AND logic).
	// Empty means no tag filtering.
	Tags []string `json:"tags,omitempty"`

	// HasResults when true, only exports jobs with non-empty results.
	HasResults bool `json:"has_results,omitempty"`
}

// ExportConfig defines where and how to export job results.
type ExportConfig struct {
	// Format is the export format: json, jsonl, md, csv, xlsx, parquet, har, pdf
	Format string `json:"format"`

	// DestinationType is the export destination: s3, gcs, azure, local, webhook
	DestinationType string `json:"destination_type"`

	// CloudConfig is the cloud storage configuration (for s3/gcs/azure destinations).
	CloudConfig *exporter.CloudExportConfig `json:"cloud_config,omitempty"`

	// LocalPath is the local file path template (for local destination).
	// Supports variables: {job_id}, {timestamp}, {kind}, {format}
	LocalPath string `json:"local_path,omitempty"`

	// WebhookURL is the webhook endpoint (for webhook destination).
	WebhookURL string `json:"webhook_url,omitempty"`

	// PathTemplate is the path template with variables for cloud/local destinations.
	// Default: "exports/{kind}/{job_id}.{format}"
	// Supported variables: {job_id}, {timestamp}, {kind}, {format}
	PathTemplate string `json:"path_template,omitempty"`
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
	return []string{"json", "jsonl", "md", "csv", "xlsx", "parquet", "har", "pdf"}
}

// SupportedDestinationTypes returns the list of supported destination types.
func SupportedDestinationTypes() []string {
	return []string{"s3", "gcs", "azure", "local", "webhook"}
}

// IsValidExportFormat returns true if the format is supported.
func IsValidExportFormat(format string) bool {
	switch format {
	case "json", "jsonl", "md", "csv", "xlsx", "parquet", "har", "pdf":
		return true
	}
	return false
}

// IsValidDestinationType returns true if the destination type is supported.
func IsValidDestinationType(dest string) bool {
	switch dest {
	case "s3", "gcs", "azure", "local", "webhook":
		return true
	}
	return false
}

// IsCloudDestination returns true if the destination is a cloud storage type.
func IsCloudDestination(dest string) bool {
	switch dest {
	case "s3", "gcs", "azure":
		return true
	}
	return false
}
