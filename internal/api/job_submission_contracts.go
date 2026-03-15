// Package api provides HTTP submission helpers for single-job and schedule flows.
//
// Purpose:
// - Keep API handlers aligned with the shared operator-facing submission contract.
//
// Responsibilities:
// - Validate scrape/crawl/research requests.
// - Build create-time jobs.JobSpec values and request-option envelopes for handlers.
// - Convert between persisted typed specs and public request payloads for schedules.
//
// Scope:
// - API-facing glue only; canonical request conversion lives in internal/submission.
//
// Usage:
// - Used by REST handlers, schedule CRUD, and MCP adapters.
//
// Invariants/Assumptions:
// - Single-job API surfaces share the same conversion logic as automation surfaces.
// - Schedule payloads stay on the operator-facing request contract.
package api

import (
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/scheduler"
	"github.com/fitchmultz/spartan-scraper/internal/submission"
)

type JobSubmissionDefaults = submission.Defaults

func validateExtractOptions(opts *extract.ExtractOptions) error {
	return submission.ValidateExtractOptions(opts)
}

func validateScrapeRequest(req ScrapeRequest) error {
	return submission.ValidateScrapeRequest(req)
}

func scrapeJobSpecFromRequest(req ScrapeRequest) jobs.JobSpec {
	spec := jobs.JobSpec{
		Kind:        model.KindScrape,
		URL:         req.URL,
		Method:      req.Method,
		Body:        []byte(req.Body),
		ContentType: req.ContentType,
		Headless:    req.Headless,
	}
	if spec.Method == "" {
		spec.Method = "GET"
	}
	return spec
}

func scrapeJobRequestOptions(requestID string, req ScrapeRequest) jobRequestOptions {
	return jobRequestOptions{
		authURL:          req.URL,
		authProfile:      req.AuthProfile,
		auth:             req.Auth,
		extract:          req.Extract,
		pipeline:         req.Pipeline,
		webhook:          req.Webhook,
		screenshot:       req.Screenshot,
		device:           req.Device,
		networkIntercept: req.NetworkIntercept,
		incremental:      req.Incremental,
		playwright:       req.Playwright,
		timeoutSeconds:   req.TimeoutSeconds,
		requestID:        requestID,
	}
}

func validateCrawlRequest(req CrawlRequest) error {
	return submission.ValidateCrawlRequest(req)
}

func crawlJobSpecFromRequest(req CrawlRequest) jobs.JobSpec {
	return jobs.JobSpec{
		Kind:             model.KindCrawl,
		URL:              req.URL,
		MaxDepth:         req.MaxDepth,
		MaxPages:         req.MaxPages,
		Headless:         req.Headless,
		SitemapURL:       req.SitemapURL,
		SitemapOnly:      valueOr(req.SitemapOnly, false),
		IncludePatterns:  req.IncludePatterns,
		ExcludePatterns:  req.ExcludePatterns,
		RespectRobotsTxt: valueOr(req.RespectRobotsTxt, false),
		SkipDuplicates:   valueOr(req.SkipDuplicates, false),
		SimHashThreshold: valueOr(req.SimHashThreshold, 0),
	}
}

func crawlJobRequestOptions(requestID string, req CrawlRequest) jobRequestOptions {
	return jobRequestOptions{
		authURL:          req.URL,
		authProfile:      req.AuthProfile,
		auth:             req.Auth,
		extract:          req.Extract,
		pipeline:         req.Pipeline,
		webhook:          req.Webhook,
		screenshot:       req.Screenshot,
		device:           req.Device,
		networkIntercept: req.NetworkIntercept,
		incremental:      req.Incremental,
		playwright:       req.Playwright,
		timeoutSeconds:   req.TimeoutSeconds,
		requestID:        requestID,
	}
}

func validateResearchRequest(req ResearchRequest) error {
	return submission.ValidateResearchRequest(req)
}

func researchJobSpecFromRequest(req ResearchRequest) jobs.JobSpec {
	return jobs.JobSpec{
		Kind:     model.KindResearch,
		Query:    req.Query,
		URLs:     req.URLs,
		MaxDepth: req.MaxDepth,
		MaxPages: req.MaxPages,
		Headless: req.Headless,
		Agentic:  req.Agentic,
	}
}

func researchJobRequestOptions(requestID string, req ResearchRequest) jobRequestOptions {
	return jobRequestOptions{
		authURL:          req.URLs[0],
		authProfile:      req.AuthProfile,
		auth:             req.Auth,
		extract:          req.Extract,
		pipeline:         req.Pipeline,
		webhook:          req.Webhook,
		screenshot:       req.Screenshot,
		device:           req.Device,
		networkIntercept: req.NetworkIntercept,
		playwright:       req.Playwright,
		timeoutSeconds:   req.TimeoutSeconds,
		requestID:        requestID,
	}
}

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
	spec, _, err := submission.JobSpecFromRawRequest(s.cfg, submission.Defaults{
		DefaultTimeoutSeconds: s.manager.DefaultTimeoutSeconds(),
		DefaultUsePlaywright:  s.manager.DefaultUsePlaywright(),
		ResolveAuth:           false,
	}, kind, rawRequest)
	if err != nil {
		return jobs.JobSpec{}, 0, nil, err
	}
	version, typedSpec, err := jobs.TypedSpecFromJobSpec(spec)
	return spec, version, typedSpec, err
}

func unsupportedScheduleSpecError(kind model.Kind) error {
	return apperrors.Validation("unsupported typed schedule spec for kind " + string(kind))
}
