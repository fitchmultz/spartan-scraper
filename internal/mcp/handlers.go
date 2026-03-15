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
	"bytes"
	"context"
	"encoding/base64"
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
	"github.com/fitchmultz/spartan-scraper/internal/paramdecode"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
	"github.com/fitchmultz/spartan-scraper/internal/research"
	"github.com/fitchmultz/spartan-scraper/internal/scheduler"
	"github.com/fitchmultz/spartan-scraper/internal/store"
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
		id := paramdecode.String(params.Arguments, "id")
		if id == "" {
			return nil, apperrors.Validation("id is required")
		}
		job, err := s.store.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		return api.BuildJobResponse(job), nil
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
		total, err := s.store.CountJobs(ctx, "")
		if err != nil {
			return nil, err
		}
		return api.BuildJobListResponse(jobs, total, limit, offset), nil
	case "job_cancel":
		id := paramdecode.String(params.Arguments, "id")
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
		return api.BuildJobResponse(job), nil
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
			return api.BuildBatchResponse(batch, stats, nil, batch.JobCount, 0, 0), nil
		}
		limit := paramdecode.PositiveInt(params.Arguments, "limit", 50)
		offset := paramdecode.PositiveInt(params.Arguments, "offset", 0)
		jobs, err := s.store.ListJobsByBatch(ctx, id, store.ListOptions{Limit: limit, Offset: offset})
		if err != nil {
			return nil, err
		}
		return api.BuildBatchResponse(batch, stats, jobs, batch.JobCount, limit, offset), nil
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
			return api.BuildBatchResponse(batch, stats, nil, batch.JobCount, 0, 0), nil
		}
		limit := paramdecode.PositiveInt(params.Arguments, "limit", 50)
		offset := paramdecode.PositiveInt(params.Arguments, "offset", 0)
		jobs, err := s.store.ListJobsByBatch(ctx, id, store.ListOptions{Limit: limit, Offset: offset})
		if err != nil {
			return nil, err
		}
		return api.BuildBatchResponse(batch, stats, jobs, batch.JobCount, limit, offset), nil
	case "job_export":
		id := paramdecode.String(params.Arguments, "id")
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
		rawBytes, err := os.ReadFile(job.ResultPath)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to read result file", err)
		}
		exported, err := exporter.ExportResult(job, rawBytes, exportConfig)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to export job", err)
		}
		result := map[string]interface{}{
			"format":      exportConfig.Format,
			"filename":    exporter.ResultExportFilename(job, exportConfig),
			"contentType": exporter.ResultExportContentType(exportConfig.Format),
		}
		if exporter.ResultExportIsBinary(exportConfig.Format) {
			result["encoding"] = "base64"
			result["content"] = encodeBase64(exported)
		} else {
			result["encoding"] = "utf8"
			result["content"] = string(exported)
		}
		return result, nil
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
		return map[string]interface{}{"records": records, "total": total}, nil
	case "proxy_pool_status":
		return api.BuildProxyPoolStatusResponse(s.manager.GetProxyPool()), nil
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

func encodeBase64(value []byte) string {
	return base64.StdEncoding.EncodeToString(value)
}
