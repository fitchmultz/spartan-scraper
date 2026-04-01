// Package api provides HTTP handlers for bounded AI authoring endpoints.
//
// Purpose:
// - Define the stable request and response payloads for bounded AI preview and authoring routes.
//
// Responsibilities:
// - Hold AI authoring request and response structs and the shared request-body size limit.
//
// Scope:
// - Type definitions for `/v1/ai/*` handlers only; handler implementations and helpers live in adjacent files.
//
// Usage:
// - Used by AI authoring HTTP handlers mounted under `/v1/ai/*`.
//
// Invariants/Assumptions:
// - Requests use JSON payloads and bounded body sizes.
// - Response shapes remain stable for generated API clients.
package api

import (
	piai "github.com/fitchmultz/spartan-scraper/internal/ai"
	"github.com/fitchmultz/spartan-scraper/internal/aiauthoring"
	"github.com/fitchmultz/spartan-scraper/internal/exporter"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
	"github.com/fitchmultz/spartan-scraper/internal/research"
)

const maxAIAuthoringRequestBodySize int64 = 8 * 1024 * 1024

// AIExtractPreviewRequest for POST /v1/ai/extract-preview
type AIExtractPreviewRequest struct {
	URL           string                   `json:"url"`
	HTML          string                   `json:"html,omitempty"`
	Mode          extract.AIExtractionMode `json:"mode"`
	Prompt        string                   `json:"prompt,omitempty"`
	Schema        map[string]interface{}   `json:"schema,omitempty"`
	Fields        []string                 `json:"fields,omitempty"`
	Images        []extract.AIImageInput   `json:"images,omitempty"`
	Headless      bool                     `json:"headless,omitempty"`
	UsePlaywright bool                     `json:"playwright,omitempty"`
	Visual        bool                     `json:"visual,omitempty"`
}

// AIExtractPreviewResponse for preview endpoint
type AIExtractPreviewResponse struct {
	Fields            map[string]extract.FieldValue `json:"fields"`
	Confidence        float64                       `json:"confidence"`
	Explanation       string                        `json:"explanation,omitempty"`
	TokensUsed        int                           `json:"tokens_used"`
	RouteID           string                        `json:"route_id,omitempty"`
	Provider          string                        `json:"provider,omitempty"`
	Model             string                        `json:"model,omitempty"`
	Cached            bool                          `json:"cached"`
	VisualContextUsed bool                          `json:"visual_context_used"`
}

// AIExtractTemplateGenerateRequest for POST /v1/ai/template-generate
type AIExtractTemplateGenerateRequest struct {
	URL           string                 `json:"url,omitempty"`
	HTML          string                 `json:"html,omitempty"`
	Description   string                 `json:"description"`
	SampleFields  []string               `json:"sample_fields,omitempty"`
	Images        []extract.AIImageInput `json:"images,omitempty"`
	Headless      bool                   `json:"headless,omitempty"`
	UsePlaywright bool                   `json:"playwright,omitempty"`
	Visual        bool                   `json:"visual,omitempty"`
}

// AIExtractTemplateGenerateResponse for template generation
type AIExtractTemplateGenerateResponse struct {
	Template          extract.Template `json:"template"`
	Explanation       string           `json:"explanation,omitempty"`
	RouteID           string           `json:"route_id,omitempty"`
	Provider          string           `json:"provider,omitempty"`
	Model             string           `json:"model,omitempty"`
	VisualContextUsed bool             `json:"visual_context_used"`
}

type AIExtractTemplateDebugRequest struct {
	URL           string                 `json:"url,omitempty"`
	HTML          string                 `json:"html,omitempty"`
	Template      extract.Template       `json:"template"`
	Instructions  string                 `json:"instructions,omitempty"`
	Images        []extract.AIImageInput `json:"images,omitempty"`
	Headless      bool                   `json:"headless,omitempty"`
	UsePlaywright bool                   `json:"playwright,omitempty"`
	Visual        bool                   `json:"visual,omitempty"`
}

type AIExtractTemplateDebugResponse struct {
	Issues            []string                      `json:"issues,omitempty"`
	ExtractedFields   map[string]extract.FieldValue `json:"extracted_fields,omitempty"`
	Explanation       string                        `json:"explanation,omitempty"`
	SuggestedTemplate *extract.Template             `json:"suggested_template,omitempty"`
	RouteID           string                        `json:"route_id,omitempty"`
	Provider          string                        `json:"provider,omitempty"`
	Model             string                        `json:"model,omitempty"`
	VisualContextUsed bool                          `json:"visual_context_used"`
}

type AIRenderProfileGenerateRequest struct {
	URL           string                 `json:"url"`
	Name          string                 `json:"name,omitempty"`
	HostPatterns  []string               `json:"host_patterns,omitempty"`
	Instructions  string                 `json:"instructions,omitempty"`
	Images        []extract.AIImageInput `json:"images,omitempty"`
	Headless      bool                   `json:"headless,omitempty"`
	UsePlaywright bool                   `json:"playwright,omitempty"`
	Visual        bool                   `json:"visual,omitempty"`
}

type AIRenderProfileGenerateResponse struct {
	Profile           fetch.RenderProfile       `json:"profile"`
	ResolvedGoal      *aiauthoring.ResolvedGoal `json:"resolved_goal,omitempty"`
	Explanation       string                    `json:"explanation,omitempty"`
	RouteID           string                    `json:"route_id,omitempty"`
	Provider          string                    `json:"provider,omitempty"`
	Model             string                    `json:"model,omitempty"`
	VisualContextUsed bool                      `json:"visual_context_used"`
}

type AIRenderProfileDebugRequest struct {
	URL           string                 `json:"url"`
	Profile       fetch.RenderProfile    `json:"profile"`
	Instructions  string                 `json:"instructions,omitempty"`
	Images        []extract.AIImageInput `json:"images,omitempty"`
	Headless      bool                   `json:"headless,omitempty"`
	UsePlaywright bool                   `json:"playwright,omitempty"`
	Visual        bool                   `json:"visual,omitempty"`
}

type AIRenderProfileDebugResponse struct {
	Issues            []string                  `json:"issues,omitempty"`
	ResolvedGoal      *aiauthoring.ResolvedGoal `json:"resolved_goal,omitempty"`
	Explanation       string                    `json:"explanation,omitempty"`
	SuggestedProfile  *fetch.RenderProfile      `json:"suggested_profile,omitempty"`
	RouteID           string                    `json:"route_id,omitempty"`
	Provider          string                    `json:"provider,omitempty"`
	Model             string                    `json:"model,omitempty"`
	VisualContextUsed bool                      `json:"visual_context_used"`
	RecheckStatus     int                       `json:"recheck_status,omitempty"`
	RecheckEngine     string                    `json:"recheck_engine,omitempty"`
	RecheckError      string                    `json:"recheck_error,omitempty"`
}

type AIPipelineJSGenerateRequest struct {
	URL           string                 `json:"url"`
	Name          string                 `json:"name,omitempty"`
	HostPatterns  []string               `json:"host_patterns,omitempty"`
	Instructions  string                 `json:"instructions,omitempty"`
	Images        []extract.AIImageInput `json:"images,omitempty"`
	Headless      bool                   `json:"headless,omitempty"`
	UsePlaywright bool                   `json:"playwright,omitempty"`
	Visual        bool                   `json:"visual,omitempty"`
}

type AIPipelineJSGenerateResponse struct {
	Script            pipeline.JSTargetScript   `json:"script"`
	ResolvedGoal      *aiauthoring.ResolvedGoal `json:"resolved_goal,omitempty"`
	Explanation       string                    `json:"explanation,omitempty"`
	RouteID           string                    `json:"route_id,omitempty"`
	Provider          string                    `json:"provider,omitempty"`
	Model             string                    `json:"model,omitempty"`
	VisualContextUsed bool                      `json:"visual_context_used"`
}

type AIPipelineJSDebugRequest struct {
	URL           string                  `json:"url"`
	Script        pipeline.JSTargetScript `json:"script"`
	Instructions  string                  `json:"instructions,omitempty"`
	Images        []extract.AIImageInput  `json:"images,omitempty"`
	Headless      bool                    `json:"headless,omitempty"`
	UsePlaywright bool                    `json:"playwright,omitempty"`
	Visual        bool                    `json:"visual,omitempty"`
}

type AIPipelineJSDebugResponse struct {
	Issues            []string                  `json:"issues,omitempty"`
	ResolvedGoal      *aiauthoring.ResolvedGoal `json:"resolved_goal,omitempty"`
	Explanation       string                    `json:"explanation,omitempty"`
	SuggestedScript   *pipeline.JSTargetScript  `json:"suggested_script,omitempty"`
	RouteID           string                    `json:"route_id,omitempty"`
	Provider          string                    `json:"provider,omitempty"`
	Model             string                    `json:"model,omitempty"`
	VisualContextUsed bool                      `json:"visual_context_used"`
	RecheckStatus     int                       `json:"recheck_status,omitempty"`
	RecheckEngine     string                    `json:"recheck_engine,omitempty"`
	RecheckError      string                    `json:"recheck_error,omitempty"`
}

type AIResearchRefineRequest struct {
	Result       research.Result `json:"result"`
	Instructions string          `json:"instructions,omitempty"`
}

type AIResearchRefineResponse struct {
	Issues      []string                             `json:"issues,omitempty"`
	InputStats  aiauthoring.ResearchRefineInputStats `json:"inputStats"`
	Refined     piai.ResearchRefinedContent          `json:"refined"`
	Markdown    string                               `json:"markdown"`
	Explanation string                               `json:"explanation,omitempty"`
	RouteID     string                               `json:"route_id,omitempty"`
	Provider    string                               `json:"provider,omitempty"`
	Model       string                               `json:"model,omitempty"`
}

type AIExportShapeRequest struct {
	JobID        string               `json:"job_id"`
	Format       string               `json:"format"`
	CurrentShape exporter.ShapeConfig `json:"currentShape,omitempty"`
	Instructions string               `json:"instructions,omitempty"`
}

type AIExportShapeResponse struct {
	Issues      []string                          `json:"issues,omitempty"`
	InputStats  aiauthoring.ExportShapeInputStats `json:"inputStats"`
	Shape       exporter.ShapeConfig              `json:"shape"`
	Explanation string                            `json:"explanation,omitempty"`
	RouteID     string                            `json:"route_id,omitempty"`
	Provider    string                            `json:"provider,omitempty"`
	Model       string                            `json:"model,omitempty"`
}

type AITransformGenerateRequest struct {
	JobID             string                   `json:"job_id"`
	CurrentTransform  exporter.TransformConfig `json:"currentTransform,omitempty"`
	PreferredLanguage string                   `json:"preferredLanguage,omitempty"`
	Instructions      string                   `json:"instructions,omitempty"`
}

type AITransformGenerateResponse struct {
	Issues      []string                        `json:"issues,omitempty"`
	InputStats  aiauthoring.TransformInputStats `json:"inputStats"`
	Transform   exporter.TransformConfig        `json:"transform"`
	Preview     []any                           `json:"preview,omitempty"`
	Explanation string                          `json:"explanation,omitempty"`
	RouteID     string                          `json:"route_id,omitempty"`
	Provider    string                          `json:"provider,omitempty"`
	Model       string                          `json:"model,omitempty"`
}
