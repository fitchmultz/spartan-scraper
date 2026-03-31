// Package mcp implements AI authoring MCP tool handlers.
//
// Purpose:
// - Keep AI authoring request shaping and validation in a focused domain file.
//
// Responsibilities:
// - Decode MCP arguments into bounded AI authoring requests.
// - Enforce required fields for schema-guided and result-backed flows.
// - Load representative job results when AI export/transform generation needs them.
//
// Scope:
// - MCP AI authoring handlers only; the authoring service owns execution.
//
// Usage:
// - Registered through aiToolRegistry in tool_registry.go.
//
// Invariants/Assumptions:
// - AI authoring requests stay aligned with toolsList schemas.
// - Result-backed tools require a stored job result file.
// - Validation errors remain transport-safe via apperrors.
package mcp

import (
	"context"
	"os"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/aiauthoring"
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/exporter"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/paramdecode"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
	"github.com/fitchmultz/spartan-scraper/internal/research"
)

func (s *Server) handleAIExtractPreviewTool(ctx context.Context, params callParams) (interface{}, error) {
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
}

func (s *Server) handleAITemplateGenerateTool(ctx context.Context, params callParams) (interface{}, error) {
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
}

func (s *Server) handleAITemplateDebugTool(ctx context.Context, params callParams) (interface{}, error) {
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
}

func (s *Server) handleAIRenderProfileGenerateTool(ctx context.Context, params callParams) (interface{}, error) {
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
}

func (s *Server) handleAIRenderProfileDebugTool(ctx context.Context, params callParams) (interface{}, error) {
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
}

func (s *Server) handleAIPipelineJSGenerateTool(ctx context.Context, params callParams) (interface{}, error) {
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
}

func (s *Server) handleAIPipelineJSDebugTool(ctx context.Context, params callParams) (interface{}, error) {
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
}

func (s *Server) handleAIResearchRefineTool(ctx context.Context, params callParams) (interface{}, error) {
	researchResult := paramdecode.Decode[research.Result](params.Arguments, "result")
	result, err := s.aiAuthoring.RefineResearch(ctx, aiauthoring.ResearchRefineRequest{
		Result:       researchResult,
		Instructions: strings.TrimSpace(paramdecode.String(params.Arguments, "instructions")),
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Server) handleAIExportShapeTool(ctx context.Context, params callParams) (interface{}, error) {
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
}

func (s *Server) handleAITransformGenerateTool(ctx context.Context, params callParams) (interface{}, error) {
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
}
