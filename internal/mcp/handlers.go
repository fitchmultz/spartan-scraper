// MCP tool call handlers.
//
// Responsibilities:
// - Process and route MCP tool calls to appropriate handlers
// - Create and enqueue jobs for scrape, crawl, and research tools
// - Handle job management tools (status, results, list, cancel, export)
//
// Does NOT handle:
// - Server lifecycle or protocol-level routing (handled by server.go)
// - Job execution or worker pool management
//
// Invariants:
// - All handlers validate required parameters before execution
// - Jobs are created, enqueued, and waited for synchronously
// - Error responses use apperrors for consistent classification
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/auth"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/exporter"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/paramdecode"
	"github.com/fitchmultz/spartan-scraper/internal/store"
	"github.com/fitchmultz/spartan-scraper/internal/validate"
)

func (s *Server) handleToolCall(ctx context.Context, base map[string]json.RawMessage) (interface{}, error) {
	var params callParams
	if raw, ok := base["params"]; ok {
		if err := json.Unmarshal(raw, &params); err != nil {
			return nil, err
		}
	}

	switch params.Name {
	case "scrape_page":
		url := paramdecode.String(params.Arguments, "url")
		if url == "" {
			return nil, apperrors.Validation("url is required")
		}
		authProfile := paramdecode.String(params.Arguments, "authProfile")
		timeout := paramdecode.PositiveInt(params.Arguments, "timeoutSeconds", s.cfg.RequestTimeoutSecs)
		opts := validate.JobValidationOpts{
			URL:         url,
			Timeout:     timeout,
			AuthProfile: authProfile,
		}
		if err := validate.ValidateJob(opts, model.KindScrape); err != nil {
			return nil, err
		}
		resolvedAuth, err := resolveAuthForTool(s.cfg, url, authProfile)
		if err != nil {
			return nil, err
		}
		extractOpts := extract.ExtractOptions{
			Template: paramdecode.String(params.Arguments, "extractTemplate"),
			Validate: paramdecode.Bool(params.Arguments, "extractValidate"),
		}
		pipelineOpts := getPipelineOptions(params.Arguments)
		spec := jobs.JobSpec{
			Kind:           model.KindScrape,
			URL:            url,
			Headless:       paramdecode.Bool(params.Arguments, "headless"),
			UsePlaywright:  paramdecode.BoolDefault(params.Arguments, "playwright", s.cfg.UsePlaywright),
			AuthProfile:    authProfile,
			Auth:           resolvedAuth,
			TimeoutSeconds: timeout,
			Extract:        extractOpts,
			Pipeline:       pipelineOpts,
			Incremental:    paramdecode.BoolDefault(params.Arguments, "incremental", false),
		}
		job, err := s.manager.CreateJob(ctx, spec)
		if err != nil {
			return nil, err
		}
		if err := s.manager.Enqueue(job); err != nil {
			return nil, err
		}
		if err := waitForJob(ctx, s.store, job.ID, timeout); err != nil {
			return nil, err
		}
		return loadResult(ctx, s.store, job.ID)
	case "crawl_site":
		url := paramdecode.String(params.Arguments, "url")
		if url == "" {
			return nil, apperrors.Validation("url is required")
		}
		authProfile := paramdecode.String(params.Arguments, "authProfile")
		maxDepth := paramdecode.PositiveInt(params.Arguments, "maxDepth", 2)
		maxPages := paramdecode.PositiveInt(params.Arguments, "maxPages", 200)
		timeout := paramdecode.PositiveInt(params.Arguments, "timeoutSeconds", s.cfg.RequestTimeoutSecs)
		opts := validate.JobValidationOpts{
			URL:         url,
			MaxDepth:    maxDepth,
			MaxPages:    maxPages,
			Timeout:     timeout,
			AuthProfile: authProfile,
		}
		if err := validate.ValidateJob(opts, model.KindCrawl); err != nil {
			return nil, err
		}
		resolvedAuth, err := resolveAuthForTool(s.cfg, url, authProfile)
		if err != nil {
			return nil, err
		}
		extractOpts := extract.ExtractOptions{
			Template: paramdecode.String(params.Arguments, "extractTemplate"),
			Validate: paramdecode.Bool(params.Arguments, "extractValidate"),
		}
		pipelineOpts := getPipelineOptions(params.Arguments)
		spec := jobs.JobSpec{
			Kind:           model.KindCrawl,
			URL:            url,
			MaxDepth:       maxDepth,
			MaxPages:       maxPages,
			Headless:       paramdecode.Bool(params.Arguments, "headless"),
			UsePlaywright:  paramdecode.BoolDefault(params.Arguments, "playwright", s.cfg.UsePlaywright),
			AuthProfile:    authProfile,
			Auth:           resolvedAuth,
			TimeoutSeconds: timeout,
			Extract:        extractOpts,
			Pipeline:       pipelineOpts,
			Incremental:    paramdecode.BoolDefault(params.Arguments, "incremental", false),
		}
		job, err := s.manager.CreateJob(ctx, spec)
		if err != nil {
			return nil, err
		}
		if err := s.manager.Enqueue(job); err != nil {
			return nil, err
		}
		if err := waitForJob(ctx, s.store, job.ID, timeout); err != nil {
			return nil, err
		}
		return loadResult(ctx, s.store, job.ID)
	case "research":
		query := paramdecode.String(params.Arguments, "query")
		urls := paramdecode.StringSlice(params.Arguments, "urls")
		if query == "" || len(urls) == 0 {
			return nil, apperrors.Validation("query and urls are required")
		}
		authProfile := paramdecode.String(params.Arguments, "authProfile")
		maxDepth := paramdecode.PositiveInt(params.Arguments, "maxDepth", 2)
		maxPages := paramdecode.PositiveInt(params.Arguments, "maxPages", 200)
		timeout := paramdecode.PositiveInt(params.Arguments, "timeoutSeconds", s.cfg.RequestTimeoutSecs)
		opts := validate.JobValidationOpts{
			Query:       query,
			URLs:        urls,
			MaxDepth:    maxDepth,
			MaxPages:    maxPages,
			Timeout:     timeout,
			AuthProfile: authProfile,
		}
		if err := validate.ValidateJob(opts, model.KindResearch); err != nil {
			return nil, err
		}
		targetURL := ""
		if len(urls) > 0 {
			targetURL = urls[0]
		}
		resolvedAuth, err := resolveAuthForTool(s.cfg, targetURL, authProfile)
		if err != nil {
			return nil, err
		}
		extractOpts := extract.ExtractOptions{
			Template: paramdecode.String(params.Arguments, "extractTemplate"),
			Validate: paramdecode.Bool(params.Arguments, "extractValidate"),
		}
		pipelineOpts := getPipelineOptions(params.Arguments)
		spec := jobs.JobSpec{
			Kind:           model.KindResearch,
			Query:          query,
			URLs:           urls,
			MaxDepth:       maxDepth,
			MaxPages:       maxPages,
			Headless:       paramdecode.Bool(params.Arguments, "headless"),
			UsePlaywright:  paramdecode.BoolDefault(params.Arguments, "playwright", s.cfg.UsePlaywright),
			AuthProfile:    authProfile,
			Auth:           resolvedAuth,
			TimeoutSeconds: timeout,
			Extract:        extractOpts,
			Pipeline:       pipelineOpts,
		}
		job, err := s.manager.CreateJob(ctx, spec)
		if err != nil {
			return nil, err
		}
		if err := s.manager.Enqueue(job); err != nil {
			return nil, err
		}
		if err := waitForJob(ctx, s.store, job.ID, timeout); err != nil {
			return nil, err
		}
		return loadResult(ctx, s.store, job.ID)
	case "job_status":
		id := paramdecode.String(params.Arguments, "id")
		if id == "" {
			return nil, apperrors.Validation("id is required")
		}
		job, err := s.store.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		return model.SanitizeJob(job), nil
	case "job_results":
		id := paramdecode.String(params.Arguments, "id")
		if id == "" {
			return nil, apperrors.Validation("id is required")
		}
		return loadResult(ctx, s.store, id)
	case "job_list":
		limit := paramdecode.PositiveInt(params.Arguments, "limit", 100)
		offset := paramdecode.PositiveInt(params.Arguments, "offset", 0)
		jobs, err := s.store.ListOpts(ctx, store.ListOptions{Limit: limit, Offset: offset})
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"jobs": model.SanitizeJobs(jobs)}, nil
	case "job_cancel":
		id := paramdecode.String(params.Arguments, "id")
		if id == "" {
			return nil, apperrors.Validation("id is required")
		}
		if err := s.manager.CancelJob(ctx, id); err != nil {
			return nil, err
		}
		return map[string]interface{}{"status": "canceled", "id": id}, nil
	case "job_export":
		id := paramdecode.String(params.Arguments, "id")
		if id == "" {
			return nil, apperrors.Validation("id is required")
		}
		format := paramdecode.String(params.Arguments, "format")
		if format == "" {
			format = "jsonl"
		}
		validFormats := map[string]bool{"jsonl": true, "json": true, "md": true, "csv": true}
		if !validFormats[format] {
			return nil, apperrors.Validation("invalid format: must be jsonl, json, md, or csv")
		}
		job, err := s.store.Get(ctx, id)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindNotFound, "job not found", err)
		}
		if job.ResultPath == "" {
			return nil, apperrors.NotFound("job has no results")
		}
		rawBytes, err := os.ReadFile(job.ResultPath)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to read result file", err)
		}
		exported, err := exporter.Export(job, rawBytes, format)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to export job", err)
		}
		return exported, nil
	default:
		return nil, apperrors.Validation(fmt.Sprintf("unknown tool: %s", params.Name))
	}
}

func loadResult(ctx context.Context, store *store.Store, id string) (string, error) {
	job, err := store.Get(ctx, id)
	if err != nil {
		return "", err
	}
	if job.ResultPath == "" {
		return "", apperrors.NotFound("no result path")
	}
	data, err := os.ReadFile(job.ResultPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func resolveAuthForTool(cfg config.Config, url string, profile string) (fetch.AuthOptions, error) {
	input := auth.ResolveInput{
		ProfileName: profile,
		URL:         url,
		Env:         &cfg.AuthOverrides,
	}
	resolved, err := auth.Resolve(cfg.DataDir, input)
	if err != nil {
		return fetch.AuthOptions{}, err
	}
	return auth.ToFetchOptions(resolved), nil
}
