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
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/aiauthoring"
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/auth"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/exporter"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/paramdecode"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
	"github.com/fitchmultz/spartan-scraper/internal/research"
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
	case "ai_extract_preview":
		mode := extract.AIExtractionMode(strings.TrimSpace(paramdecode.String(params.Arguments, "mode")))
		if mode == "" {
			mode = extract.AIModeNaturalLanguage
		}
		var schema map[string]interface{}
		if mode == extract.AIModeSchemaGuided {
			schema = paramdecode.Decode[map[string]interface{}](params.Arguments, "schema")
			if len(schema) == 0 {
				return nil, apperrors.Validation("schema is required when mode is schema_guided")
			}
		}
		result, err := s.aiAuthoring.Preview(ctx, aiauthoring.PreviewRequest{
			URL:           paramdecode.String(params.Arguments, "url"),
			HTML:          paramdecode.String(params.Arguments, "html"),
			Mode:          mode,
			Prompt:        strings.TrimSpace(paramdecode.String(params.Arguments, "prompt")),
			Schema:        schema,
			Fields:        paramdecode.StringSlice(params.Arguments, "fields"),
			Headless:      paramdecode.Bool(params.Arguments, "headless"),
			UsePlaywright: paramdecode.Bool(params.Arguments, "playwright"),
			Visual:        paramdecode.Bool(params.Arguments, "visual"),
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	case "ai_template_generate":
		result, err := s.aiAuthoring.GenerateTemplate(ctx, aiauthoring.TemplateRequest{
			URL:           paramdecode.String(params.Arguments, "url"),
			HTML:          paramdecode.String(params.Arguments, "html"),
			Description:   strings.TrimSpace(paramdecode.String(params.Arguments, "description")),
			SampleFields:  paramdecode.StringSlice(params.Arguments, "sampleFields"),
			Headless:      paramdecode.Bool(params.Arguments, "headless"),
			UsePlaywright: paramdecode.Bool(params.Arguments, "playwright"),
			Visual:        paramdecode.Bool(params.Arguments, "visual"),
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	case "ai_template_debug":
		template := paramdecode.Decode[extract.Template](params.Arguments, "template")
		result, err := s.aiAuthoring.DebugTemplate(ctx, aiauthoring.TemplateDebugRequest{
			URL:           paramdecode.String(params.Arguments, "url"),
			HTML:          paramdecode.String(params.Arguments, "html"),
			Template:      template,
			Instructions:  strings.TrimSpace(paramdecode.String(params.Arguments, "instructions")),
			Headless:      paramdecode.Bool(params.Arguments, "headless"),
			UsePlaywright: paramdecode.Bool(params.Arguments, "playwright"),
			Visual:        paramdecode.Bool(params.Arguments, "visual"),
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	case "ai_render_profile_generate":
		result, err := s.aiAuthoring.GenerateRenderProfile(ctx, aiauthoring.RenderProfileRequest{
			URL:           paramdecode.String(params.Arguments, "url"),
			Name:          strings.TrimSpace(paramdecode.String(params.Arguments, "name")),
			HostPatterns:  paramdecode.StringSlice(params.Arguments, "hostPatterns"),
			Instructions:  strings.TrimSpace(paramdecode.String(params.Arguments, "instructions")),
			Headless:      paramdecode.Bool(params.Arguments, "headless"),
			UsePlaywright: paramdecode.Bool(params.Arguments, "playwright"),
			Visual:        paramdecode.Bool(params.Arguments, "visual"),
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	case "ai_render_profile_debug":
		profile := paramdecode.Decode[fetch.RenderProfile](params.Arguments, "profile")
		result, err := s.aiAuthoring.DebugRenderProfile(ctx, aiauthoring.RenderProfileDebugRequest{
			URL:           paramdecode.String(params.Arguments, "url"),
			Profile:       profile,
			Instructions:  strings.TrimSpace(paramdecode.String(params.Arguments, "instructions")),
			Headless:      paramdecode.Bool(params.Arguments, "headless"),
			UsePlaywright: paramdecode.Bool(params.Arguments, "playwright"),
			Visual:        paramdecode.Bool(params.Arguments, "visual"),
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	case "ai_pipeline_js_generate":
		result, err := s.aiAuthoring.GeneratePipelineJS(ctx, aiauthoring.PipelineJSRequest{
			URL:           paramdecode.String(params.Arguments, "url"),
			Name:          strings.TrimSpace(paramdecode.String(params.Arguments, "name")),
			HostPatterns:  paramdecode.StringSlice(params.Arguments, "hostPatterns"),
			Instructions:  strings.TrimSpace(paramdecode.String(params.Arguments, "instructions")),
			Headless:      paramdecode.Bool(params.Arguments, "headless"),
			UsePlaywright: paramdecode.Bool(params.Arguments, "playwright"),
			Visual:        paramdecode.Bool(params.Arguments, "visual"),
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	case "ai_pipeline_js_debug":
		script := paramdecode.Decode[pipeline.JSTargetScript](params.Arguments, "script")
		result, err := s.aiAuthoring.DebugPipelineJS(ctx, aiauthoring.PipelineJSDebugRequest{
			URL:           paramdecode.String(params.Arguments, "url"),
			Script:        script,
			Instructions:  strings.TrimSpace(paramdecode.String(params.Arguments, "instructions")),
			Headless:      paramdecode.Bool(params.Arguments, "headless"),
			UsePlaywright: paramdecode.Bool(params.Arguments, "playwright"),
			Visual:        paramdecode.Bool(params.Arguments, "visual"),
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	case "ai_research_refine":
		researchResult := paramdecode.Decode[research.Result](params.Arguments, "result")
		result, err := s.aiAuthoring.RefineResearch(ctx, aiauthoring.ResearchRefineRequest{
			Result:       researchResult,
			Instructions: strings.TrimSpace(paramdecode.String(params.Arguments, "instructions")),
		})
		if err != nil {
			return nil, err
		}
		return result, nil
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
		aiExtractOpts, err := decodeAIExtractOptions(params.Arguments)
		if err != nil {
			return nil, apperrors.Validation(err.Error())
		}
		if aiExtractOpts != nil {
			extractOpts.AI = aiExtractOpts
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
		aiExtractOpts, err := decodeAIExtractOptions(params.Arguments)
		if err != nil {
			return nil, apperrors.Validation(err.Error())
		}
		if aiExtractOpts != nil {
			extractOpts.AI = aiExtractOpts
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
		aiExtractOpts, err := decodeAIExtractOptions(params.Arguments)
		if err != nil {
			return nil, err
		}
		agenticOpts, err := decodeResearchAgenticOptions(params.Arguments)
		if err != nil {
			return nil, apperrors.Validation(err.Error())
		}
		extractOpts := extract.ExtractOptions{
			Template: paramdecode.String(params.Arguments, "extractTemplate"),
			Validate: paramdecode.Bool(params.Arguments, "extractValidate"),
			AI:       aiExtractOpts,
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
			Agentic:        agenticOpts,
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
