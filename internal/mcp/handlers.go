// Package mcp routes MCP tool calls onto Spartan runtime operations.
//
// Purpose:
// - Execute MCP tool requests for AI authoring, jobs, batches, watches, exports, and observability workflows.
//
// Responsibilities:
// - Decode tool arguments and validate required fields.
// - Reuse canonical submission and response-building paths shared with REST where possible.
// - Return stable transport-safe envelopes for persisted runtime entities.
//
// Scope:
// - Tool execution only; protocol handling lives in server.go and long-running job execution lives elsewhere.
//
// Usage:
// - Called by the MCP server when handling `tools/call` JSON-RPC messages.
//
// Invariants/Assumptions:
// - All handlers validate required parameters before execution.
// - Synchronous scrape/crawl/research tools wait for completion, while persisted job/batch tools return stored envelopes.
// - Error responses use apperrors for consistent classification.
package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/aiauthoring"
	"github.com/fitchmultz/spartan-scraper/internal/api"
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
	"github.com/fitchmultz/spartan-scraper/internal/scheduler"
	"github.com/fitchmultz/spartan-scraper/internal/store"
	"github.com/fitchmultz/spartan-scraper/internal/submission"
	"github.com/fitchmultz/spartan-scraper/internal/watch"
	"github.com/fitchmultz/spartan-scraper/internal/webhook"
)

func decodeToolArguments(args map[string]interface{}, dst any) error {
	payload, err := json.Marshal(args)
	if err != nil {
		return apperrors.Wrap(apperrors.KindValidation, "invalid tool arguments", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return apperrors.Validation("invalid tool arguments: " + err.Error())
	}
	var extra any
	if err := decoder.Decode(&extra); err != nil {
		if err == io.EOF {
			return nil
		}
		return apperrors.Validation("invalid tool arguments: " + err.Error())
	}
	return apperrors.Validation("invalid tool arguments: expected a single JSON object")
}

func (s *Server) batchSubmissionDefaults() submission.BatchDefaults {
	return submission.BatchDefaults{
		Defaults: submission.Defaults{
			DefaultTimeoutSeconds: s.manager.DefaultTimeoutSeconds(),
			DefaultUsePlaywright:  s.manager.DefaultUsePlaywright(),
			ResolveAuth:           true,
		},
		MaxBatchSize: s.cfg.MaxBatchSize,
	}
}

func (s *Server) createAndEnqueueBatch(ctx context.Context, kind model.Kind, specs []jobs.JobSpec) (api.BatchResponse, error) {
	batchID := jobs.GenerateBatchID()
	createdJobs, err := s.manager.CreateBatchJobs(ctx, kind, specs, batchID)
	if err != nil {
		return api.BatchResponse{}, err
	}
	if err := s.manager.EnqueueBatch(createdJobs); err != nil {
		return api.BatchResponse{}, err
	}
	batch, stats, err := s.manager.GetBatchStatus(ctx, batchID)
	if err != nil {
		return api.BatchResponse{}, err
	}
	return api.BuildStoreBackedBatchResponse(ctx, s.store, batch, stats, createdJobs, len(createdJobs), len(createdJobs), 0)
}

func (s *Server) handleToolCall(ctx context.Context, base map[string]json.RawMessage) (interface{}, error) {
	var params callParams
	if raw, ok := base["params"]; ok {
		if err := json.Unmarshal(raw, &params); err != nil {
			return nil, err
		}
	}

	if s.setup != nil && params.Name != "health_status" && params.Name != "diagnostic_check" {
		return nil, apperrors.Permission("setup mode only exposes diagnostics until recovery is completed")
	}

	switch params.Name {
	case "health_status":
		return s.buildHealthStatus(ctx), nil
	case "diagnostic_check":
		component := api.NormalizeDiagnosticTarget(strings.TrimSpace(paramdecode.String(params.Arguments, "component")))
		if component == "" {
			return nil, apperrors.Validation("component is required and must be one of browser, ai, or proxy_pool")
		}
		return s.runDiagnostic(ctx, component), nil
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
			Images:        paramdecode.Decode[[]extract.AIImageInput](params.Arguments, "images"),
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
			Images:        paramdecode.Decode[[]extract.AIImageInput](params.Arguments, "images"),
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
			Images:        paramdecode.Decode[[]extract.AIImageInput](params.Arguments, "images"),
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
			Images:        paramdecode.Decode[[]extract.AIImageInput](params.Arguments, "images"),
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
			Images:        paramdecode.Decode[[]extract.AIImageInput](params.Arguments, "images"),
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
			Images:        paramdecode.Decode[[]extract.AIImageInput](params.Arguments, "images"),
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
			Images:        paramdecode.Decode[[]extract.AIImageInput](params.Arguments, "images"),
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
	case "ai_export_shape":
		jobID := strings.TrimSpace(paramdecode.String(params.Arguments, "jobId"))
		if jobID == "" {
			return nil, apperrors.Validation("jobId is required")
		}
		format := strings.TrimSpace(paramdecode.String(params.Arguments, "format"))
		if format == "" {
			return nil, apperrors.Validation("format is required")
		}
		job, err := s.store.Get(ctx, jobID)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindNotFound, "job not found", err)
		}
		if strings.TrimSpace(job.ResultPath) == "" {
			return nil, apperrors.NotFound("job has no result file")
		}
		rawResult, err := os.ReadFile(job.ResultPath)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to read result file", err)
		}
		currentShape := paramdecode.Decode[exporter.ShapeConfig](params.Arguments, "currentShape")
		result, err := s.aiAuthoring.GenerateExportShape(ctx, aiauthoring.ExportShapeRequest{
			JobKind:      job.Kind,
			Format:       format,
			RawResult:    rawResult,
			CurrentShape: currentShape,
			Instructions: strings.TrimSpace(paramdecode.String(params.Arguments, "instructions")),
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	case "ai_transform_generate":
		jobID := strings.TrimSpace(paramdecode.String(params.Arguments, "jobId"))
		if jobID == "" {
			return nil, apperrors.Validation("jobId is required")
		}
		job, err := s.store.Get(ctx, jobID)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindNotFound, "job not found", err)
		}
		if strings.TrimSpace(job.ResultPath) == "" {
			return nil, apperrors.NotFound("job has no result file")
		}
		rawResult, err := os.ReadFile(job.ResultPath)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to read result file", err)
		}
		currentTransform := paramdecode.Decode[exporter.TransformConfig](params.Arguments, "currentTransform")
		result, err := s.aiAuthoring.GenerateTransform(ctx, aiauthoring.TransformRequest{
			JobKind:           job.Kind,
			RawResult:         rawResult,
			CurrentTransform:  currentTransform,
			PreferredLanguage: strings.TrimSpace(paramdecode.String(params.Arguments, "preferredLanguage")),
			Instructions:      strings.TrimSpace(paramdecode.String(params.Arguments, "instructions")),
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	case "scrape_page":
		var req api.ScrapeRequest
		if err := decodeToolArguments(params.Arguments, &req); err != nil {
			return nil, err
		}
		spec, err := api.JobSpecFromScrapeRequest(s.cfg, api.JobSubmissionDefaults{
			DefaultTimeoutSeconds: s.manager.DefaultTimeoutSeconds(),
			DefaultUsePlaywright:  s.manager.DefaultUsePlaywright(),
			ResolveAuth:           true,
		}, req)
		if err != nil {
			return nil, err
		}
		job, err := s.manager.CreateJob(ctx, spec)
		if err != nil {
			return nil, err
		}
		if err := s.manager.Enqueue(job); err != nil {
			return nil, err
		}
		if err := waitForJob(ctx, s.store, job.ID, spec.TimeoutSeconds); err != nil {
			return nil, err
		}
		return loadResult(ctx, s.store, job.ID)
	case "crawl_site":
		var req api.CrawlRequest
		if err := decodeToolArguments(params.Arguments, &req); err != nil {
			return nil, err
		}
		spec, err := api.JobSpecFromCrawlRequest(s.cfg, api.JobSubmissionDefaults{
			DefaultTimeoutSeconds: s.manager.DefaultTimeoutSeconds(),
			DefaultUsePlaywright:  s.manager.DefaultUsePlaywright(),
			ResolveAuth:           true,
		}, req)
		if err != nil {
			return nil, err
		}
		job, err := s.manager.CreateJob(ctx, spec)
		if err != nil {
			return nil, err
		}
		if err := s.manager.Enqueue(job); err != nil {
			return nil, err
		}
		if err := waitForJob(ctx, s.store, job.ID, spec.TimeoutSeconds); err != nil {
			return nil, err
		}
		return loadResult(ctx, s.store, job.ID)
	case "research":
		var req api.ResearchRequest
		if err := decodeToolArguments(params.Arguments, &req); err != nil {
			return nil, err
		}
		spec, err := api.JobSpecFromResearchRequest(s.cfg, api.JobSubmissionDefaults{
			DefaultTimeoutSeconds: s.manager.DefaultTimeoutSeconds(),
			DefaultUsePlaywright:  s.manager.DefaultUsePlaywright(),
			ResolveAuth:           true,
		}, req)
		if err != nil {
			return nil, err
		}
		job, err := s.manager.CreateJob(ctx, spec)
		if err != nil {
			return nil, err
		}
		if err := s.manager.Enqueue(job); err != nil {
			return nil, err
		}
		if err := waitForJob(ctx, s.store, job.ID, spec.TimeoutSeconds); err != nil {
			return nil, err
		}
		return loadResult(ctx, s.store, job.ID)
	case "job_status":
		id := strings.TrimSpace(paramdecode.String(params.Arguments, "id"))
		if id == "" {
			return nil, apperrors.Validation("id is required")
		}
		job, err := s.store.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		return api.BuildStoreBackedJobResponse(ctx, s.store, job)
	case "job_results":
		id := paramdecode.String(params.Arguments, "id")
		if id == "" {
			return nil, apperrors.Validation("id is required")
		}
		return loadResult(ctx, s.store, id)
	case "job_list":
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
	case "job_failure_list":
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
	case "job_cancel":
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
	case "batch_scrape":
		var req api.BatchScrapeRequest
		if err := decodeToolArguments(params.Arguments, &req); err != nil {
			return nil, err
		}
		specs, err := submission.JobSpecsFromBatchScrapeRequest(s.cfg, s.batchSubmissionDefaults(), req)
		if err != nil {
			return nil, err
		}
		return s.createAndEnqueueBatch(ctx, model.KindScrape, specs)
	case "batch_crawl":
		var req api.BatchCrawlRequest
		if err := decodeToolArguments(params.Arguments, &req); err != nil {
			return nil, err
		}
		specs, err := submission.JobSpecsFromBatchCrawlRequest(s.cfg, s.batchSubmissionDefaults(), req)
		if err != nil {
			return nil, err
		}
		return s.createAndEnqueueBatch(ctx, model.KindCrawl, specs)
	case "batch_research":
		var req api.BatchResearchRequest
		if err := decodeToolArguments(params.Arguments, &req); err != nil {
			return nil, err
		}
		specs, err := submission.JobSpecsFromBatchResearchRequest(s.cfg, s.batchSubmissionDefaults(), req)
		if err != nil {
			return nil, err
		}
		return s.createAndEnqueueBatch(ctx, model.KindResearch, specs)
	case "batch_list":
		limit := paramdecode.PositiveInt(params.Arguments, "limit", 100)
		offset := paramdecode.PositiveInt(params.Arguments, "offset", 0)
		batches, stats, total, err := s.manager.ListBatchStatuses(ctx, store.ListOptions{Limit: limit, Offset: offset})
		if err != nil {
			return nil, err
		}
		return api.BuildBatchListResponse(batches, stats, total, limit, offset), nil
	case "batch_status":
		id := strings.TrimSpace(paramdecode.String(params.Arguments, "id"))
		if id == "" {
			return nil, apperrors.Validation("id is required")
		}
		batch, stats, err := s.manager.GetBatchStatus(ctx, id)
		if err != nil {
			return nil, err
		}
		includeJobs := paramdecode.Bool(params.Arguments, "includeJobs")
		if !includeJobs {
			return api.BuildStoreBackedBatchResponse(ctx, s.store, batch, stats, nil, batch.JobCount, 0, 0)
		}
		limit := paramdecode.PositiveInt(params.Arguments, "limit", 50)
		offset := paramdecode.PositiveInt(params.Arguments, "offset", 0)
		jobs, err := s.store.ListJobsByBatch(ctx, id, store.ListOptions{Limit: limit, Offset: offset})
		if err != nil {
			return nil, err
		}
		return api.BuildStoreBackedBatchResponse(ctx, s.store, batch, stats, jobs, batch.JobCount, limit, offset)
	case "batch_cancel":
		id := strings.TrimSpace(paramdecode.String(params.Arguments, "id"))
		if id == "" {
			return nil, apperrors.Validation("id is required")
		}
		if _, err := s.manager.CancelBatch(ctx, id); err != nil {
			return nil, err
		}
		batch, stats, err := s.manager.GetBatchStatus(ctx, id)
		if err != nil {
			return nil, err
		}
		includeJobs := paramdecode.Bool(params.Arguments, "includeJobs")
		if !includeJobs {
			return api.BuildStoreBackedBatchResponse(ctx, s.store, batch, stats, nil, batch.JobCount, 0, 0)
		}
		limit := paramdecode.PositiveInt(params.Arguments, "limit", 50)
		offset := paramdecode.PositiveInt(params.Arguments, "offset", 0)
		jobs, err := s.store.ListJobsByBatch(ctx, id, store.ListOptions{Limit: limit, Offset: offset})
		if err != nil {
			return nil, err
		}
		return api.BuildStoreBackedBatchResponse(ctx, s.store, batch, stats, jobs, batch.JobCount, limit, offset)
	case "job_export":
		id := strings.TrimSpace(paramdecode.String(params.Arguments, "id"))
		if id == "" {
			return nil, apperrors.Validation("id is required")
		}
		exportConfig := exporter.NormalizeResultExportConfig(exporter.ResultExportConfig{
			Format:    strings.TrimSpace(paramdecode.String(params.Arguments, "format")),
			Shape:     paramdecode.Decode[exporter.ShapeConfig](params.Arguments, "shape"),
			Transform: paramdecode.Decode[exporter.TransformConfig](params.Arguments, "transform"),
		})
		if err := exporter.ValidateResultExportConfig(exportConfig); err != nil {
			return nil, err
		}
		job, err := s.store.Get(ctx, id)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindNotFound, "job not found", err)
		}
		if job.ResultPath == "" {
			return nil, apperrors.NotFound("job has no results")
		}

		historyStore := scheduler.NewExportHistoryStore(s.cfg.DataDir)
		record, err := historyStore.CreateRecord(scheduler.CreateRecordInput{
			JobID:       job.ID,
			Trigger:     exporter.OutcomeTriggerMCP,
			Destination: "mcp response",
			Request:     exportConfig,
		})
		if err != nil {
			return nil, err
		}

		rawBytes, err := os.ReadFile(job.ResultPath)
		if err != nil {
			if markErr := historyStore.MarkFailed(record.ID, apperrors.Wrap(apperrors.KindInternal, "failed to read result file", err)); markErr != nil {
				return nil, markErr
			}
			failed, getErr := historyStore.GetByID(record.ID)
			if getErr != nil {
				return nil, getErr
			}
			return api.ExportOutcomeResponse{Export: api.BuildExportInspection(*failed, nil)}, nil
		}

		rendered, err := exporter.RenderResultExport(job, rawBytes, exportConfig)
		if err != nil {
			if markErr := historyStore.MarkFailed(record.ID, err); markErr != nil {
				return nil, markErr
			}
			failed, getErr := historyStore.GetByID(record.ID)
			if getErr != nil {
				return nil, getErr
			}
			return api.ExportOutcomeResponse{Export: api.BuildExportInspection(*failed, nil)}, nil
		}

		if err := historyStore.MarkSuccess(record.ID, rendered); err != nil {
			return nil, err
		}
		stored, err := historyStore.GetByID(record.ID)
		if err != nil {
			return nil, err
		}
		return api.ExportOutcomeResponse{Export: api.BuildExportInspection(*stored, rendered.Content)}, nil
	case "job_export_history":
		id := strings.TrimSpace(paramdecode.String(params.Arguments, "id"))
		if id == "" {
			return nil, apperrors.Validation("id is required")
		}
		limit := paramdecode.PositiveInt(params.Arguments, "limit", 50)
		offset := paramdecode.PositiveInt(params.Arguments, "offset", 0)
		records, total, err := scheduler.NewExportHistoryStore(s.cfg.DataDir).GetByJob(id, limit, offset)
		if err != nil {
			return nil, err
		}
		return api.BuildExportOutcomeListResponse(records, total, limit, offset), nil
	case "export_outcome_get":
		id := strings.TrimSpace(paramdecode.String(params.Arguments, "id"))
		if id == "" {
			return nil, apperrors.Validation("id is required")
		}
		record, err := scheduler.NewExportHistoryStore(s.cfg.DataDir).GetByID(id)
		if err != nil {
			return nil, apperrors.NotFound("export outcome not found")
		}
		return api.ExportOutcomeResponse{Export: api.BuildExportInspection(*record, nil)}, nil
	case "watch_list":
		watches, err := watch.NewFileStorage(s.cfg.DataDir).List()
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"watches": watches}, nil
	case "watch_get":
		id := strings.TrimSpace(paramdecode.String(params.Arguments, "id"))
		if id == "" {
			return nil, apperrors.Validation("id is required")
		}
		watchItem, err := watch.NewFileStorage(s.cfg.DataDir).Get(id)
		if err != nil {
			if watch.IsNotFoundError(err) {
				return nil, apperrors.NotFound("watch not found")
			}
			return nil, err
		}
		return watchItem, nil
	case "watch_create":
		var args watchCreateArgs
		if err := decodeToolArguments(params.Arguments, &args); err != nil {
			return nil, err
		}
		watchItem, err := s.buildWatchCreate(args)
		if err != nil {
			return nil, err
		}
		created, err := watch.NewFileStorage(s.cfg.DataDir).Add(watchItem)
		if err != nil {
			return nil, err
		}
		return created, nil
	case "watch_update":
		var args watchUpdateArgs
		if err := decodeToolArguments(params.Arguments, &args); err != nil {
			return nil, err
		}
		id := strings.TrimSpace(args.ID)
		if id == "" {
			return nil, apperrors.Validation("id is required")
		}
		watchStore := watch.NewFileStorage(s.cfg.DataDir)
		existing, err := watchStore.Get(id)
		if err != nil {
			if watch.IsNotFoundError(err) {
				return nil, apperrors.NotFound("watch not found")
			}
			return nil, err
		}
		if err := s.applyWatchUpdate(existing, args); err != nil {
			return nil, err
		}
		if err := watchStore.Update(existing); err != nil {
			if watch.IsNotFoundError(err) {
				return nil, apperrors.NotFound("watch not found")
			}
			return nil, err
		}
		return existing, nil
	case "watch_delete":
		id := strings.TrimSpace(paramdecode.String(params.Arguments, "id"))
		if id == "" {
			return nil, apperrors.Validation("id is required")
		}
		if err := watch.NewFileStorage(s.cfg.DataDir).Delete(id); err != nil {
			if watch.IsNotFoundError(err) {
				return nil, apperrors.NotFound("watch not found")
			}
			return nil, err
		}
		return map[string]interface{}{"deleted": true, "id": id}, nil
	case "watch_check":
		id := strings.TrimSpace(paramdecode.String(params.Arguments, "id"))
		if id == "" {
			return nil, apperrors.Validation("id is required")
		}
		watchStore := watch.NewFileStorage(s.cfg.DataDir)
		watchItem, err := watchStore.Get(id)
		if err != nil {
			if watch.IsNotFoundError(err) {
				return nil, apperrors.NotFound("watch not found")
			}
			return nil, err
		}
		watcher := watch.NewWatcher(watchStore, s.store, s.cfg.DataDir, nil, &watch.TriggerRuntime{
			Config:  s.cfg,
			Manager: s.manager,
		})
		result, err := watcher.Check(ctx, watchItem)
		if result != nil {
			return result, nil
		}
		if err != nil {
			return nil, err
		}
		return nil, apperrors.Internal("watch check returned no result")
	case "export_schedule_list":
		schedules, err := scheduler.NewExportStorage(s.cfg.DataDir).List()
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"schedules": schedules}, nil
	case "export_schedule_get":
		id := strings.TrimSpace(paramdecode.String(params.Arguments, "id"))
		if id == "" {
			return nil, apperrors.Validation("id is required")
		}
		schedule, err := scheduler.NewExportStorage(s.cfg.DataDir).Get(id)
		if err != nil {
			if scheduler.IsNotFoundError(err) {
				return nil, apperrors.NotFound("export schedule not found")
			}
			return nil, err
		}
		return schedule, nil
	case "export_schedule_create":
		schedule := scheduler.ExportSchedule{
			Name:    strings.TrimSpace(paramdecode.String(params.Arguments, "name")),
			Enabled: paramdecode.BoolDefault(params.Arguments, "enabled", true),
			Filters: paramdecode.Decode[scheduler.ExportFilters](params.Arguments, "filters"),
			Export:  paramdecode.Decode[scheduler.ExportConfig](params.Arguments, "export"),
			Retry:   paramdecode.Decode[scheduler.ExportRetryConfig](params.Arguments, "retry"),
		}
		created, err := scheduler.NewExportStorage(s.cfg.DataDir).Add(schedule)
		if err != nil {
			return nil, err
		}
		return created, nil
	case "export_schedule_update":
		id := strings.TrimSpace(paramdecode.String(params.Arguments, "id"))
		if id == "" {
			return nil, apperrors.Validation("id is required")
		}
		store := scheduler.NewExportStorage(s.cfg.DataDir)
		existing, err := store.Get(id)
		if err != nil {
			if scheduler.IsNotFoundError(err) {
				return nil, apperrors.NotFound("export schedule not found")
			}
			return nil, err
		}
		existing.Name = strings.TrimSpace(paramdecode.String(params.Arguments, "name"))
		existing.Enabled = paramdecode.BoolDefault(params.Arguments, "enabled", existing.Enabled)
		existing.Filters = paramdecode.Decode[scheduler.ExportFilters](params.Arguments, "filters")
		existing.Export = paramdecode.Decode[scheduler.ExportConfig](params.Arguments, "export")
		if _, ok := params.Arguments["retry"]; ok {
			existing.Retry = paramdecode.Decode[scheduler.ExportRetryConfig](params.Arguments, "retry")
		}
		updated, err := store.Update(*existing)
		if err != nil {
			return nil, err
		}
		return updated, nil
	case "export_schedule_delete":
		id := strings.TrimSpace(paramdecode.String(params.Arguments, "id"))
		if id == "" {
			return nil, apperrors.Validation("id is required")
		}
		if err := scheduler.NewExportStorage(s.cfg.DataDir).Delete(id); err != nil {
			if scheduler.IsNotFoundError(err) {
				return nil, apperrors.NotFound("export schedule not found")
			}
			return nil, err
		}
		return map[string]interface{}{"deleted": true, "id": id}, nil
	case "export_schedule_history":
		id := strings.TrimSpace(paramdecode.String(params.Arguments, "id"))
		if id == "" {
			return nil, apperrors.Validation("id is required")
		}
		limit := paramdecode.PositiveInt(params.Arguments, "limit", 50)
		offset := paramdecode.PositiveInt(params.Arguments, "offset", 0)
		records, total, err := scheduler.NewExportHistoryStore(s.cfg.DataDir).GetBySchedule(id, limit, offset)
		if err != nil {
			return nil, err
		}
		return api.BuildExportOutcomeListResponse(records, total, limit, offset), nil
	case "webhook_delivery_list":
		jobID := strings.TrimSpace(paramdecode.String(params.Arguments, "jobId"))
		if jobID == "" {
			jobID = strings.TrimSpace(paramdecode.String(params.Arguments, "job_id"))
		}
		limit := paramdecode.PositiveInt(params.Arguments, "limit", 100)
		if limit > 1000 {
			limit = 1000
		}
		offset := paramdecode.PositiveInt(params.Arguments, "offset", 0)
		deliveryStore, err := loadWebhookDeliveryStore(s.cfg.DataDir)
		if err != nil {
			return nil, err
		}
		records, err := deliveryStore.ListRecords(ctx, jobID, limit, offset)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to list webhook deliveries", err)
		}
		total, err := deliveryStore.CountRecords(ctx, jobID)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to count webhook deliveries", err)
		}
		return map[string]interface{}{
			"deliveries": webhook.ToInspectableDeliveries(records),
			"total":      total,
			"limit":      limit,
			"offset":     offset,
		}, nil
	case "webhook_delivery_get":
		id := strings.TrimSpace(paramdecode.String(params.Arguments, "id"))
		if id == "" {
			return nil, apperrors.Validation("id is required")
		}
		deliveryStore, err := loadWebhookDeliveryStore(s.cfg.DataDir)
		if err != nil {
			return nil, err
		}
		record, found, err := deliveryStore.GetRecord(ctx, id)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to get webhook delivery", err)
		}
		if !found {
			return nil, apperrors.NotFound("webhook delivery not found")
		}
		return webhook.ToInspectableDelivery(record), nil
	case "proxy_pool_status":
		return api.BuildProxyPoolStatusResponse(s.manager.GetProxyPool()), nil
	default:
		return nil, apperrors.Validation(fmt.Sprintf("unknown tool: %s", params.Name))
	}
}

func loadWebhookDeliveryStore(dataDir string) (*webhook.Store, error) {
	deliveryStore := webhook.NewStore(dataDir)
	if err := deliveryStore.Load(); err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to load webhook deliveries", err)
	}
	return deliveryStore, nil
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

func resolveAuthForTool(cfg config.Config, url string, profile string, override fetch.AuthOptions) (fetch.AuthOptions, error) {
	input := auth.ResolveInput{
		ProfileName: profile,
		URL:         url,
		Env:         &cfg.AuthOverrides,
	}
	resolved, err := auth.Resolve(cfg.DataDir, input)
	if err != nil {
		return fetch.AuthOptions{}, err
	}
	authOptions := auth.ToFetchOptions(resolved)
	if override.Proxy != nil {
		authOptions.Proxy = override.Proxy
	}
	if override.ProxyHints != nil {
		authOptions.ProxyHints = fetch.NormalizeProxySelectionHints(override.ProxyHints)
	}
	authOptions.NormalizeTransport()
	if err := authOptions.ValidateTransport(); err != nil {
		return fetch.AuthOptions{}, err
	}
	return authOptions, nil
}

func decodeTransportOverrides(args map[string]interface{}) fetch.AuthOptions {
	proxyURL := strings.TrimSpace(paramdecode.String(args, "proxy"))
	proxyUsername := strings.TrimSpace(paramdecode.String(args, "proxyUsername"))
	proxyPassword := strings.TrimSpace(paramdecode.String(args, "proxyPassword"))
	var proxy *fetch.ProxyConfig
	if proxyURL != "" || proxyUsername != "" || proxyPassword != "" {
		proxy = &fetch.ProxyConfig{
			URL:      proxyURL,
			Username: proxyUsername,
			Password: proxyPassword,
		}
	}
	return fetch.AuthOptions{
		Proxy: proxy,
		ProxyHints: fetch.NormalizeProxySelectionHints(&fetch.ProxySelectionHints{
			PreferredRegion: strings.TrimSpace(paramdecode.String(args, "proxyRegion")),
			RequiredTags:    paramdecode.StringSlice(args, "proxyTags"),
			ExcludeProxyIDs: paramdecode.StringSlice(args, "excludeProxyIds"),
		}),
	}
}
