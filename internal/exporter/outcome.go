// Package exporter centralizes export-outcome modeling shared across operator surfaces.
//
// Purpose:
// - Define the canonical export outcome lifecycle, triggers, and failure context used by API, Web, CLI, MCP, and scheduler workflows.
//
// Responsibilities:
// - Classify export outcomes as pending, succeeded, or failed.
// - Normalize failure summaries into operator-facing categories and retry guidance.
// - Keep export-outcome metadata independent from transport-specific response envelopes.
//
// Scope:
// - Shared export outcome types and failure classification only.
//
// Usage:
// - Used by export history persistence, API/MCP response builders, and CLI rendering.
//
// Invariants/Assumptions:
// - Failure summaries must stay safe for operator-facing surfaces.
// - Failure categorization is best-effort and intentionally coarse-grained.
package exporter

import (
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

type OutcomeStatus string

const (
	OutcomePending   OutcomeStatus = "pending"
	OutcomeSucceeded OutcomeStatus = "succeeded"
	OutcomeFailed    OutcomeStatus = "failed"
)

type OutcomeTrigger string

const (
	OutcomeTriggerAPI      OutcomeTrigger = "api"
	OutcomeTriggerCLI      OutcomeTrigger = "cli"
	OutcomeTriggerMCP      OutcomeTrigger = "mcp"
	OutcomeTriggerSchedule OutcomeTrigger = "schedule"
)

type FailureContext struct {
	Category  string `json:"category"`
	Summary   string `json:"summary"`
	Retryable bool   `json:"retryable"`
	Terminal  bool   `json:"terminal"`
}

func BuildFailureContext(err error) *FailureContext {
	if err == nil {
		return nil
	}

	summary := strings.TrimSpace(apperrors.SafeMessage(err))
	if summary == "" {
		summary = "export failed without a recorded error message"
	}
	if len(summary) > 240 {
		summary = summary[:239] + "…"
	}

	category, retryable := classifyFailure(summary, apperrors.KindOf(err))
	return &FailureContext{
		Category:  category,
		Summary:   summary,
		Retryable: retryable,
		Terminal:  true,
	}
}

func classifyFailure(summary string, kind apperrors.Kind) (string, bool) {
	lower := strings.ToLower(strings.TrimSpace(summary))

	switch {
	case strings.Contains(lower, "jmespath"),
		strings.Contains(lower, "jsonata"),
		strings.Contains(lower, "transform"):
		return "transform", false
	case strings.Contains(lower, "unsupported format"),
		strings.Contains(lower, "csv"),
		strings.Contains(lower, "xlsx"),
		strings.Contains(lower, "markdown"):
		return "format", false
	case strings.Contains(lower, "no result"),
		strings.Contains(lower, "result file"),
		strings.Contains(lower, "empty"):
		return "result", false
	case strings.Contains(lower, "deadline exceeded"),
		strings.Contains(lower, "timeout"):
		return "timeout", true
	case strings.Contains(lower, "permission denied"),
		strings.Contains(lower, "no space left"),
		strings.Contains(lower, "read-only"),
		strings.Contains(lower, "failed to create export file"),
		strings.Contains(lower, "failed to create export directory"),
		strings.Contains(lower, "failed to write export file"):
		return "filesystem", false
	case strings.Contains(lower, "unauthorized"),
		strings.Contains(lower, "forbidden"),
		strings.Contains(lower, "401"),
		strings.Contains(lower, "403"),
		strings.Contains(lower, "auth"):
		return "auth", false
	case strings.Contains(lower, "webhook"):
		return "webhook", true
	case strings.Contains(lower, "dial tcp"),
		strings.Contains(lower, "connection refused"),
		strings.Contains(lower, "tls"),
		strings.Contains(lower, "network"),
		strings.Contains(lower, "eof"),
		strings.Contains(lower, "no such host"):
		return "network", true
	case kind == apperrors.KindValidation:
		return "validation", false
	default:
		return "unknown", false
	}
}
