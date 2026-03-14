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
