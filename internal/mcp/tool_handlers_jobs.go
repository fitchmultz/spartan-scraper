// Package mcp implements job-oriented MCP tool handlers.
//
// Purpose:
// - Isolate synchronous job execution and persisted job inspection from other MCP domains.
//
// Responsibilities:
// - Decode scrape, crawl, and research requests into canonical job specs.
// - Reuse shared manager defaults and result-loading helpers.
// - Return stable job envelopes for status, listing, results, and cancel operations.
//
// Scope:
// - Single-job MCP handlers only; batch flows live in tool_handlers_batches.go.
//
// Usage:
// - Registered through jobToolRegistry in tool_registry.go.
//
// Invariants/Assumptions:
// - Runtime job requests resolve auth using the same defaults as REST.
// - Unknown or invalid status filters return validation errors.
// - Result loading reads the persisted job artifact instead of cached in-memory state.
package mcp

import (
	"context"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/api"
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/paramdecode"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

func (s *Server) handleScrapePageTool(ctx context.Context, params callParams) (interface{}, error) {
	var req api.ScrapeRequest
	if err := decodeToolArguments(params.Arguments, &req); err != nil {
		return nil, err
	}
	spec, err := api.JobSpecFromScrapeRequest(s.cfg, s.jobSubmissionDefaults(), req)
	if err != nil {
		return nil, err
	}
	return s.runJobAndLoadResult(ctx, spec)
}

func (s *Server) handleCrawlSiteTool(ctx context.Context, params callParams) (interface{}, error) {
	var req api.CrawlRequest
	if err := decodeToolArguments(params.Arguments, &req); err != nil {
		return nil, err
	}
	spec, err := api.JobSpecFromCrawlRequest(s.cfg, s.jobSubmissionDefaults(), req)
	if err != nil {
		return nil, err
	}
	return s.runJobAndLoadResult(ctx, spec)
}

func (s *Server) handleResearchTool(ctx context.Context, params callParams) (interface{}, error) {
	var req api.ResearchRequest
	if err := decodeToolArguments(params.Arguments, &req); err != nil {
		return nil, err
	}
	spec, err := api.JobSpecFromResearchRequest(s.cfg, s.jobSubmissionDefaults(), req)
	if err != nil {
		return nil, err
	}
	return s.runJobAndLoadResult(ctx, spec)
}

func (s *Server) handleJobStatusTool(ctx context.Context, params callParams) (interface{}, error) {
	id := strings.TrimSpace(paramdecode.String(params.Arguments, "id"))
	if id == "" {
		return nil, apperrors.Validation("id is required")
	}
	job, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return api.BuildStoreBackedJobResponse(ctx, s.store, job)
}

func (s *Server) handleJobResultsTool(ctx context.Context, params callParams) (interface{}, error) {
	id := paramdecode.String(params.Arguments, "id")
	if id == "" {
		return nil, apperrors.Validation("id is required")
	}
	return loadResult(ctx, s.store, id)
}

func (s *Server) handleJobListTool(ctx context.Context, params callParams) (interface{}, error) {
	limit := paramdecode.PositiveInt(params.Arguments, "limit", 100)
	offset := paramdecode.PositiveInt(params.Arguments, "offset", 0)
	statusArg := strings.TrimSpace(paramdecode.String(params.Arguments, "status"))

	var (
		jobs  []model.Job
		total int
		err   error
	)
	if statusArg != "" {
		status := model.Status(statusArg)
		if !status.IsValid() {
			return nil, apperrors.Validation("status must be one of queued, running, succeeded, failed, or canceled")
		}
		jobs, err = s.store.ListByStatus(ctx, status, store.ListByStatusOptions{Limit: limit, Offset: offset})
		if err != nil {
			return nil, err
		}
		total, err = s.store.CountJobs(ctx, status)
		if err != nil {
			return nil, err
		}
	} else {
		jobs, err = s.store.ListOpts(ctx, store.ListOptions{Limit: limit, Offset: offset})
		if err != nil {
			return nil, err
		}
		total, err = s.store.CountJobs(ctx, "")
		if err != nil {
			return nil, err
		}
	}
	return api.BuildStoreBackedJobListResponse(ctx, s.store, jobs, total, limit, offset)
}

func (s *Server) handleJobFailureListTool(ctx context.Context, params callParams) (interface{}, error) {
	limit := paramdecode.PositiveInt(params.Arguments, "limit", 50)
	offset := paramdecode.PositiveInt(params.Arguments, "offset", 0)
	jobs, err := s.store.ListByStatus(ctx, model.StatusFailed, store.ListByStatusOptions{Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	total, err := s.store.CountJobs(ctx, model.StatusFailed)
	if err != nil {
		return nil, err
	}
	return api.BuildStoreBackedJobListResponse(ctx, s.store, jobs, total, limit, offset)
}

func (s *Server) handleJobCancelTool(ctx context.Context, params callParams) (interface{}, error) {
	id := strings.TrimSpace(paramdecode.String(params.Arguments, "id"))
	if id == "" {
		return nil, apperrors.Validation("id is required")
	}
	if err := s.manager.CancelJob(ctx, id); err != nil {
		return nil, err
	}
	job, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return api.BuildStoreBackedJobResponse(ctx, s.store, job)
}
