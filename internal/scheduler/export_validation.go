// Package scheduler provides export schedule validation.
//
// This file is responsible for:
// - Validating ExportSchedule on create/update
// - Validating ExportFilters (at least one filter criteria required)
// - Validating ExportConfig (format must be supported, destination must be valid)
// - Validating destination-specific configuration
//
// This file does NOT handle:
// - Persistence (export_storage.go handles that)
// - Export execution (export_trigger.go handles that)
//
// Validation rules:
// - Name is required and must be non-empty
// - Filters must specify at least one criteria (job kind, status, or tags)
// - Export.Format must be a supported format
// - Export.DestinationType must be a supported destination
// - Cloud destinations require valid cloud config
// - Local destinations require a local path
// - Webhook destinations require a valid webhook URL
package scheduler

import (
	"fmt"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/exporter"
	"github.com/fitchmultz/spartan-scraper/internal/webhook"
)

// ValidateExportSchedule validates an export schedule.
func ValidateExportSchedule(schedule ExportSchedule) error {
	schedule = NormalizeExportSchedule(schedule)
	if strings.TrimSpace(schedule.Name) == "" {
		return apperrors.Validation("export schedule name is required")
	}

	if err := ValidateExportFilters(schedule.Filters); err != nil {
		return err
	}

	if err := ValidateExportConfig(schedule.Export); err != nil {
		return err
	}

	return nil
}

// ValidateExportFilters validates export filter criteria.
// At least one filter criteria must be specified.
func ValidateExportFilters(filters ExportFilters) error {
	hasCriteria := len(filters.JobKinds) > 0 ||
		len(filters.JobStatus) > 0 ||
		len(filters.Tags) > 0 ||
		filters.HasResults

	if !hasCriteria {
		return apperrors.Validation("at least one filter criteria must be specified (job_kinds, job_status, tags, or has_results)")
	}

	// Validate job kinds
	validKinds := map[string]bool{"scrape": true, "crawl": true, "research": true}
	for _, kind := range filters.JobKinds {
		if !validKinds[kind] {
			return apperrors.Validation(fmt.Sprintf("invalid job kind: %s (must be scrape, crawl, or research)", kind))
		}
	}

	// Validate job status
	validStatuses := map[string]bool{"completed": true, "failed": true, "succeeded": true, "canceled": true}
	for _, status := range filters.JobStatus {
		if !validStatuses[status] {
			return apperrors.Validation(fmt.Sprintf("invalid job status: %s (must be completed, failed, succeeded, or canceled)", status))
		}
	}

	return nil
}

// ValidateExportConfig validates export configuration.
func ValidateExportConfig(config ExportConfig) error {
	config = NormalizeExportConfig(config)
	if config.Format == "" {
		return apperrors.Validation("export format is required")
	}

	if !IsValidExportFormat(config.Format) {
		return apperrors.Validation(fmt.Sprintf("unsupported export format: %s (must be one of: json, jsonl, md, csv, xlsx)", config.Format))
	}

	if config.DestinationType == "" {
		return apperrors.Validation("destination type is required")
	}

	if !IsValidDestinationType(config.DestinationType) {
		return apperrors.Validation(fmt.Sprintf("unsupported destination type: %s (must be one of: local, webhook)", config.DestinationType))
	}

	if exporter.HasMeaningfulShape(config.Shape) && !exporter.SupportsShapeFormat(config.Format) {
		return apperrors.Validation("export shaping is supported only for md, csv, and xlsx formats")
	}
	if exporter.HasMeaningfulTransform(config.Transform) {
		if err := exporter.ValidateTransformConfig(config.Transform); err != nil {
			return err
		}
	}
	if exporter.HasMeaningfulShape(config.Shape) && exporter.HasMeaningfulTransform(config.Transform) {
		return apperrors.Validation("export shape and transform cannot be combined on the same schedule")
	}

	// Validate destination-specific config
	switch config.DestinationType {
	case "local":
		if strings.TrimSpace(config.LocalPath) == "" && strings.TrimSpace(config.PathTemplate) == "" {
			return apperrors.Validation("local_path or path_template is required for local destination")
		}
	case "webhook":
		if strings.TrimSpace(config.WebhookURL) == "" {
			return apperrors.Validation("webhook_url is required for webhook destination")
		}
		if err := webhook.ValidateConfigURL(config.WebhookURL); err != nil {
			return err
		}
	}

	return nil
}
