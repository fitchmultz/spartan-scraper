// Package api builds canonical watch-check inspection response envelopes.
//
// Purpose:
// - Translate persisted watch history records into operator-facing inspection payloads shared by REST, Web, CLI, and MCP.
//
// Responsibilities:
// - Shape check narratives, artifact download metadata, and recommended next steps.
// - Keep list and single-item watch history responses transport-consistent.
// - Preserve watch-history details without exposing host-local artifact paths.
//
// Scope:
// - Watch history response construction only; persistence and execution live in internal/watch.
//
// Usage:
// - Called by watch history API handlers, CLI rendering, and MCP watch inspection tools.
//
// Invariants/Assumptions:
// - Collection responses always emit arrays, never null.
// - Artifact downloads are check-scoped so historical records remain stable after later checks rotate the latest artifacts.
package api

import (
	"fmt"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/watch"
)

func BuildWatchCheckInspection(record watch.WatchCheckRecord) WatchCheckInspection {
	artifacts := make([]WatchArtifactResponse, 0, len(record.Artifacts))
	for _, artifact := range record.Artifacts {
		artifacts = append(artifacts, toWatchArtifactResponseForCheck(record.WatchID, record.ID, artifact))
	}
	if artifacts == nil {
		artifacts = []WatchArtifactResponse{}
	}

	inspection := WatchCheckInspection{
		ID:                 record.ID,
		WatchID:            record.WatchID,
		URL:                record.URL,
		CheckedAt:          record.CheckedAt,
		Status:             string(record.Status),
		Changed:            record.Changed,
		Baseline:           record.Baseline,
		PreviousHash:       record.PreviousHash,
		CurrentHash:        record.CurrentHash,
		DiffText:           record.DiffText,
		DiffHTML:           record.DiffHTML,
		Error:              record.Error,
		Selector:           record.Selector,
		Artifacts:          artifacts,
		VisualHash:         record.VisualHash,
		PreviousVisualHash: record.PreviousVisualHash,
		VisualChanged:      record.VisualChanged,
		VisualSimilarity:   record.VisualSimilarity,
		TriggeredJobs:      append([]string(nil), record.TriggeredJobs...),
	}
	inspection.Title, inspection.Message = buildWatchCheckNarrative(inspection)
	inspection.Actions = buildWatchCheckRecommendedActions(inspection)
	return inspection
}

func BuildWatchCheckHistoryResponse(records []watch.WatchCheckRecord, total, limit, offset int) WatchCheckHistoryResponse {
	checks := make([]WatchCheckInspection, 0, len(records))
	for _, record := range records {
		checks = append(checks, BuildWatchCheckInspection(record))
	}
	if checks == nil {
		checks = []WatchCheckInspection{}
	}
	return WatchCheckHistoryResponse{
		Checks: checks,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}
}

func buildWatchCheckNarrative(inspection WatchCheckInspection) (string, string) {
	switch inspection.Status {
	case string(watch.CheckStatusFailed):
		if strings.TrimSpace(inspection.Error) != "" {
			return "Check failed", inspection.Error
		}
		return "Check failed", "The watch check failed before Spartan could compare a fresh result against the saved baseline."
	case string(watch.CheckStatusBaseline):
		return "Baseline recorded", "The first successful check saved a comparison baseline for this watch. Future checks will surface deltas from this snapshot."
	case string(watch.CheckStatusChanged):
		aspects := []string{}
		if inspection.CurrentHash != "" && inspection.PreviousHash != "" {
			aspects = append(aspects, "content")
		}
		if inspection.VisualChanged {
			aspects = append(aspects, "visual")
		}
		if len(aspects) == 0 {
			aspects = append(aspects, "watch target")
		}
		message := fmt.Sprintf("Spartan detected a %s change for this watch.", joinWithAnd(aspects))
		if count := len(inspection.TriggeredJobs); count > 0 {
			message += fmt.Sprintf(" Triggered %d follow-on job(s).", count)
		}
		return "Change detected", message
	default:
		return "No change detected", "The latest check matched the saved baseline, so there is nothing new to review yet."
	}
}

func buildWatchCheckRecommendedActions(inspection WatchCheckInspection) []RecommendedAction {
	actions := []RecommendedAction{
		{
			Label: "Inspect this check from the CLI",
			Kind:  ActionKindCommand,
			Value: fmt.Sprintf("spartan watch history %s --check-id %s", inspection.WatchID, inspection.ID),
		},
		{
			Label: "Open watch automation workspace",
			Kind:  ActionKindRoute,
			Value: "/automation",
		},
		{
			Label: "Run the watch check again",
			Kind:  ActionKindCommand,
			Value: fmt.Sprintf("spartan watch check %s", inspection.WatchID),
		},
	}
	if len(inspection.TriggeredJobs) > 0 {
		actions = append(actions, RecommendedAction{
			Label: "Inspect the triggered job",
			Kind:  ActionKindRoute,
			Value: fmt.Sprintf("/jobs/%s", inspection.TriggeredJobs[0]),
		})
	}
	return actions
}

func joinWithAnd(parts []string) string {
	trimmed := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			trimmed = append(trimmed, value)
		}
	}
	switch len(trimmed) {
	case 0:
		return ""
	case 1:
		return trimmed[0]
	case 2:
		return trimmed[0] + " and " + trimmed[1]
	default:
		return strings.Join(trimmed[:len(trimmed)-1], ", ") + ", and " + trimmed[len(trimmed)-1]
	}
}
