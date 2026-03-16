// Package api provides thin submission-contract adapters for schedules and non-HTTP callers.
//
// Purpose:
//   - Keep schedule CRUD and MCP adapters aligned with the canonical operator-facing
//     submission conversion that now lives in internal/submission.
//
// Responsibilities:
// - Reconstruct public request payloads from persisted typed specs.
// - Convert raw schedule request JSON into typed specs using shared defaults.
// - Expose narrow forwarding helpers for existing non-HTTP API callers.
//
// Scope:
// - Thin adapters only; validation and request-to-spec conversion live in internal/submission.
//
// Usage:
// - Used by REST schedule handlers, MCP handlers, and tests.
//
// Invariants/Assumptions:
// - Schedule payloads must stay on the same operator-facing request contract as live submissions.
// - Behavior changes should land in internal/submission first and flow through these adapters.
package api

import (
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/scheduler"
	"github.com/fitchmultz/spartan-scraper/internal/submission"
)

type JobSubmissionDefaults = submission.Defaults

func requestFromSchedule(schedule scheduler.Schedule) (any, error) {
	return submission.RequestFromTypedSpec(schedule.Spec)
}

func JobSpecFromScrapeRequest(cfg config.Config, defaults JobSubmissionDefaults, req ScrapeRequest) (jobs.JobSpec, error) {
	return submission.JobSpecFromScrapeRequest(cfg, defaults, req)
}

func JobSpecFromCrawlRequest(cfg config.Config, defaults JobSubmissionDefaults, req CrawlRequest) (jobs.JobSpec, error) {
	return submission.JobSpecFromCrawlRequest(cfg, defaults, req)
}

func JobSpecFromResearchRequest(cfg config.Config, defaults JobSubmissionDefaults, req ResearchRequest) (jobs.JobSpec, error) {
	return submission.JobSpecFromResearchRequest(cfg, defaults, req)
}

func convertScheduleRequestToTypedSpec(s *Server, kind model.Kind, rawRequest []byte) (jobs.JobSpec, int, any, error) {
	spec, _, err := submission.JobSpecFromRawRequest(s.cfg, s.nonResolvingSubmissionDefaults(), kind, rawRequest)
	if err != nil {
		return jobs.JobSpec{}, 0, nil, err
	}
	version, typedSpec, err := jobs.TypedSpecFromJobSpec(spec)
	return spec, version, typedSpec, err
}

func unsupportedScheduleSpecError(kind model.Kind) error {
	return apperrors.Validation("unsupported typed schedule spec for kind " + string(kind))
}
