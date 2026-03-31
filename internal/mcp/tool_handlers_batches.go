// Package mcp implements batch-oriented MCP tool handlers.
//
// Purpose:
// - Keep batch submission and inspection logic separate from single-job handlers.
//
// Responsibilities:
// - Decode batch requests into canonical submission specs.
// - Enqueue batch jobs and return stable batch envelopes.
// - Reuse shared batch response helpers for status and cancel flows.
//
// Scope:
// - Batch MCP handlers only.
//
// Usage:
// - Registered through batchToolRegistry in tool_registry.go.
//
// Invariants/Assumptions:
// - Batch defaults stay aligned with submission defaults used by other operator surfaces.
// - includeJobs controls whether job pagination is loaded from the store.
// - Batch envelopes always use the shared REST/MCP response builders.
package mcp

import (
	"context"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/api"
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/paramdecode"
	"github.com/fitchmultz/spartan-scraper/internal/store"
	"github.com/fitchmultz/spartan-scraper/internal/submission"
)

func (s *Server) handleBatchScrapeTool(ctx context.Context, params callParams) (interface{}, error) {
	var req api.BatchScrapeRequest
	if err := decodeToolArguments(params.Arguments, &req); err != nil {
		return nil, err
	}
	specs, err := submission.JobSpecsFromBatchScrapeRequest(s.cfg, s.batchSubmissionDefaults(), req)
	if err != nil {
		return nil, err
	}
	return s.createAndEnqueueBatch(ctx, model.KindScrape, specs)
}

func (s *Server) handleBatchCrawlTool(ctx context.Context, params callParams) (interface{}, error) {
	var req api.BatchCrawlRequest
	if err := decodeToolArguments(params.Arguments, &req); err != nil {
		return nil, err
	}
	specs, err := submission.JobSpecsFromBatchCrawlRequest(s.cfg, s.batchSubmissionDefaults(), req)
	if err != nil {
		return nil, err
	}
	return s.createAndEnqueueBatch(ctx, model.KindCrawl, specs)
}

func (s *Server) handleBatchResearchTool(ctx context.Context, params callParams) (interface{}, error) {
	var req api.BatchResearchRequest
	if err := decodeToolArguments(params.Arguments, &req); err != nil {
		return nil, err
	}
	specs, err := submission.JobSpecsFromBatchResearchRequest(s.cfg, s.batchSubmissionDefaults(), req)
	if err != nil {
		return nil, err
	}
	return s.createAndEnqueueBatch(ctx, model.KindResearch, specs)
}

func (s *Server) handleBatchListTool(ctx context.Context, params callParams) (interface{}, error) {
	limit := paramdecode.PositiveInt(params.Arguments, "limit", 100)
	offset := paramdecode.PositiveInt(params.Arguments, "offset", 0)
	batches, stats, total, err := s.manager.ListBatchStatuses(ctx, store.ListOptions{Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	return api.BuildBatchListResponse(batches, stats, total, limit, offset), nil
}

func (s *Server) handleBatchStatusTool(ctx context.Context, params callParams) (interface{}, error) {
	id := strings.TrimSpace(paramdecode.String(params.Arguments, "id"))
	if id == "" {
		return nil, apperrors.Validation("id is required")
	}
	includeJobs := paramdecode.Bool(params.Arguments, "includeJobs")
	limit := paramdecode.PositiveInt(params.Arguments, "limit", 50)
	offset := paramdecode.PositiveInt(params.Arguments, "offset", 0)
	return s.buildBatchResponse(ctx, id, includeJobs, limit, offset)
}

func (s *Server) handleBatchCancelTool(ctx context.Context, params callParams) (interface{}, error) {
	id := strings.TrimSpace(paramdecode.String(params.Arguments, "id"))
	if id == "" {
		return nil, apperrors.Validation("id is required")
	}
	if _, err := s.manager.CancelBatch(ctx, id); err != nil {
		return nil, err
	}
	includeJobs := paramdecode.Bool(params.Arguments, "includeJobs")
	limit := paramdecode.PositiveInt(params.Arguments, "limit", 50)
	offset := paramdecode.PositiveInt(params.Arguments, "offset", 0)
	return s.buildBatchResponse(ctx, id, includeJobs, limit, offset)
}
