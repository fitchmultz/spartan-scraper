// Package api builds capability-aware retention guidance shared by operator surfaces.
//
// Purpose:
// - Derive operator-facing retention guidance from retention status metrics.
//
// Responsibilities:
// - Classify disabled, warning, danger, and healthy retention states.
// - Attach actionable next steps that work across API, Web, and CLI surfaces.
// - Keep retention guidance language consistent with Settings-route recovery flows.
//
// Scope:
// - Retention guidance derivation only; retention execution stays in the retention package.
//
// Usage:
// - Called by retention API handlers and CLI status rendering.
//
// Invariants/Assumptions:
// - Disabled retention is optional, not a failure.
// - Warning and danger states should recommend previewing cleanup before destructive action.
package api

import "fmt"

func BuildRetentionCapabilityGuidance(status RetentionStatusResponse) *CapabilityGuidance {
	if !status.Enabled {
		return &CapabilityGuidance{
			Status:  "disabled",
			Title:   "Automatic retention is disabled",
			Message: "Spartan will keep completed jobs and crawl state until you enable automatic cleanup or run targeted cleanup manually. Preview first so you understand the blast radius.",
			Actions: []RecommendedAction{
				{
					Label: "Enable retention in the environment",
					Kind:  ActionKindEnv,
					Value: "RETENTION_ENABLED=true",
				},
				{
					Label: "Preview cleanup from the CLI",
					Kind:  ActionKindCommand,
					Value: "spartan retention cleanup --dry-run",
				},
			},
		}
	}

	storageRatio := 0.0
	if status.MaxStorageGB > 0 {
		storageRatio = float64(status.StorageUsedMB) / 1024 / float64(status.MaxStorageGB)
	}

	jobsRatio := 0.0
	if status.MaxJobs > 0 {
		jobsRatio = float64(status.TotalJobs) / float64(status.MaxJobs)
	}

	previewAction := RecommendedAction{
		Label: "Preview cleanup from the CLI",
		Kind:  ActionKindCommand,
		Value: "spartan retention cleanup --dry-run",
	}

	if storageRatio >= 0.9 || jobsRatio >= 0.9 {
		return &CapabilityGuidance{
			Status:  "danger",
			Title:   "Retention limits are close to being hit",
			Message: "Storage or job-count pressure is high. Preview cleanup now, then run cleanup or raise limits intentionally if this growth is expected.",
			Actions: []RecommendedAction{previewAction},
		}
	}

	if status.JobsEligible > 0 {
		return &CapabilityGuidance{
			Status:  "warning",
			Title:   "Cleanup opportunity detected",
			Message: fmt.Sprintf("%d job(s) already meet the current cleanup policy. Preview a cleanup run before pressure becomes urgent.", status.JobsEligible),
			Actions: []RecommendedAction{previewAction},
		}
	}

	return &CapabilityGuidance{Status: "ok"}
}
