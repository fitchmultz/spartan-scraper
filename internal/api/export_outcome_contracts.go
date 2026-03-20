// Package api builds canonical export-outcome response envelopes.
//
// Purpose:
// - Translate persisted export history records into operator-facing inspection payloads shared by REST, Web, CLI, and MCP.
//
// Responsibilities:
// - Shape artifact metadata and optional inline content.
// - Derive export titles, messages, and recommended recovery actions.
// - Keep list and single-item export responses transport-consistent.
//
// Scope:
// - Export response construction only; persistence and execution live elsewhere.
//
// Usage:
// - Called by direct export handlers, export-history endpoints, CLI rendering, and MCP tools.
//
// Invariants/Assumptions:
// - Inline content is included only when explicitly provided by the caller.
// - Collection responses always emit arrays, never null.
package api

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/exporter"
	"github.com/fitchmultz/spartan-scraper/internal/scheduler"
)

func BuildExportInspection(record scheduler.ExportRecord, content []byte) ExportInspection {
	request := exporter.NormalizeResultExportConfig(record.Request)

	var artifact *ExportArtifact
	if record.Format != "" || record.Filename != "" || record.ContentType != "" {
		artifact = &ExportArtifact{
			Format:      record.Format,
			Filename:    record.Filename,
			ContentType: record.ContentType,
			RecordCount: record.RecordCount,
			Size:        record.ExportSize,
		}
		if len(content) > 0 {
			if exporter.ResultExportIsBinary(record.Format) {
				artifact.Encoding = "base64"
				artifact.Content = base64.StdEncoding.EncodeToString(content)
			} else {
				artifact.Encoding = "utf8"
				artifact.Content = string(content)
			}
		}
	}

	outcome := ExportInspection{
		ID:          record.ID,
		ScheduleID:  record.ScheduleID,
		JobID:       record.JobID,
		Trigger:     string(record.Trigger),
		Status:      string(record.Status),
		Destination: record.Destination,
		ExportedAt:  record.ExportedAt,
		CompletedAt: record.CompletedAt,
		RetryCount:  record.RetryCount,
		Request:     request,
		Artifact:    artifact,
		Failure:     record.Failure,
	}
	outcome.Title, outcome.Message = buildExportNarrative(outcome)
	outcome.Actions = buildExportRecommendedActions(outcome)
	return outcome
}

func BuildExportOutcomeListResponse(records []scheduler.ExportRecord, total, limit, offset int) ExportOutcomeListResponse {
	items := make([]ExportInspection, 0, len(records))
	for _, record := range records {
		items = append(items, BuildExportInspection(record, nil))
	}
	if items == nil {
		items = []ExportInspection{}
	}
	return ExportOutcomeListResponse{
		Exports: items,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	}
}

func buildExportNarrative(outcome ExportInspection) (string, string) {
	switch outcome.Status {
	case string(exporter.OutcomeSucceeded):
		format := strings.ToUpper(strings.TrimSpace(outcome.Request.Format))
		if outcome.Artifact != nil && strings.TrimSpace(outcome.Artifact.Format) != "" {
			format = strings.ToUpper(outcome.Artifact.Format)
		}
		recordCount := 0
		if outcome.Artifact != nil {
			recordCount = outcome.Artifact.RecordCount
		}
		if outcome.Destination != "" {
			return "Export ready", fmt.Sprintf("%s export completed successfully with %d record(s) and is ready at %s.", format, recordCount, outcome.Destination)
		}
		return "Export ready", fmt.Sprintf("%s export completed successfully with %d record(s).", format, recordCount)
	case string(exporter.OutcomeFailed):
		if outcome.Failure != nil {
			return "Export failed", outcome.Failure.Summary
		}
		return "Export failed", "The export failed without a recorded failure summary."
	default:
		return "Export running", "The export has been recorded and is still in progress."
	}
}

func buildExportRecommendedActions(outcome ExportInspection) []RecommendedAction {
	actions := make([]RecommendedAction, 0, 6)
	baseCommand := buildExportRetryCommand(outcome)

	actions = append(actions, RecommendedAction{
		Label: "Inspect export from the CLI",
		Kind:  ActionKindCommand,
		Value: fmt.Sprintf("spartan export --inspect-id %s", outcome.ID),
	})

	actions = append(actions, RecommendedAction{
		Label: "Inspect saved job results",
		Kind:  ActionKindRoute,
		Value: fmt.Sprintf("/jobs/%s", outcome.JobID),
	})

	if outcome.ScheduleID != "" {
		actions = append(actions, RecommendedAction{
			Label: "Inspect schedule history",
			Kind:  ActionKindRoute,
			Value: "/automation/exports",
		})
	}

	if outcome.Status == string(exporter.OutcomeSucceeded) {
		return actions
	}
	if outcome.Failure == nil {
		return actions
	}

	if outcome.Failure.Retryable {
		actions = append(actions, RecommendedAction{
			Label: "Retry export from the CLI",
			Kind:  ActionKindCommand,
			Value: baseCommand,
		})
	}

	switch outcome.Failure.Category {
	case "transform":
		actions = append(actions,
			RecommendedAction{
				Label: "Retry without the transform",
				Kind:  ActionKindCommand,
				Value: fmt.Sprintf("spartan export --job-id %s --format %s", outcome.JobID, outcome.Request.Format),
			},
			RecommendedAction{
				Label: "Retry as JSONL",
				Kind:  ActionKindCommand,
				Value: fmt.Sprintf("spartan export --job-id %s --format jsonl", outcome.JobID),
			},
		)
	case "format":
		actions = append(actions, RecommendedAction{
			Label: "Retry as JSONL",
			Kind:  ActionKindCommand,
			Value: fmt.Sprintf("spartan export --job-id %s --format jsonl", outcome.JobID),
		})
	case "filesystem":
		actions = append(actions, RecommendedAction{
			Label: "Retry to a writable file path",
			Kind:  ActionKindCommand,
			Value: fmt.Sprintf("spartan export --job-id %s --format %s --out ./exports/%s.%s", outcome.JobID, outcome.Request.Format, outcome.JobID, outcome.Request.Format),
		})
	case "webhook", "network", "timeout":
		actions = append(actions, RecommendedAction{
			Label: "Review export automation settings",
			Kind:  ActionKindRoute,
			Value: "/automation/exports",
		})
	case "result":
		actions = append(actions, RecommendedAction{
			Label: "Inspect the saved result in the Web UI",
			Kind:  ActionKindRoute,
			Value: fmt.Sprintf("/jobs/%s", outcome.JobID),
		})
	}

	return actions
}

func buildExportRetryCommand(outcome ExportInspection) string {
	parts := []string{
		"spartan export",
		fmt.Sprintf("--job-id %s", outcome.JobID),
		fmt.Sprintf("--format %s", outcome.Request.Format),
	}
	if strings.TrimSpace(outcome.Destination) != "" && outcome.Trigger == string(exporter.OutcomeTriggerCLI) {
		parts = append(parts, fmt.Sprintf("--out %s", outcome.Destination))
	}
	return strings.Join(parts, " ")
}
