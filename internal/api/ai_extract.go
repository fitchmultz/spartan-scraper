// Package api provides HTTP handlers for bounded AI authoring endpoints.
//
// Purpose:
// - Expose prompt-heavy AI preview and authoring routes without creating jobs.
//
// Responsibilities:
// - Validate authoring requests and shared AI configuration.
// - Enforce strict JSON request parsing and bounded request sizes.
// - Adapt authoring results into stable API responses.
//
// Scope:
// - AI authoring request handlers only.
//
// Usage:
// - Mounted under `/v1/ai/*`.
//
// Invariants/Assumptions:
// - Requests must use `application/json`.
// - AI handlers require the shared `aiauthoring.Service`.
package api

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"

	piai "github.com/fitchmultz/spartan-scraper/internal/ai"
	"github.com/fitchmultz/spartan-scraper/internal/aiauthoring"
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/exporter"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
	"github.com/fitchmultz/spartan-scraper/internal/research"
)

const maxAIAuthoringRequestBodySize int64 = 8 * 1024 * 1024

// AIExtractPreviewRequest for POST /v1/ai/extract-preview
type AIExtractPreviewRequest struct {
	URL           string                   `json:"url"`
	HTML          string                   `json:"html,omitempty"` // Optional: provide HTML directly
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
	Profile           fetch.RenderProfile `json:"profile"`
	Explanation       string              `json:"explanation,omitempty"`
	RouteID           string              `json:"route_id,omitempty"`
	Provider          string              `json:"provider,omitempty"`
	Model             string              `json:"model,omitempty"`
	VisualContextUsed bool                `json:"visual_context_used"`
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
	Issues            []string             `json:"issues,omitempty"`
	Explanation       string               `json:"explanation,omitempty"`
	SuggestedProfile  *fetch.RenderProfile `json:"suggested_profile,omitempty"`
	RouteID           string               `json:"route_id,omitempty"`
	Provider          string               `json:"provider,omitempty"`
	Model             string               `json:"model,omitempty"`
	VisualContextUsed bool                 `json:"visual_context_used"`
	RecheckStatus     int                  `json:"recheck_status,omitempty"`
	RecheckEngine     string               `json:"recheck_engine,omitempty"`
	RecheckError      string               `json:"recheck_error,omitempty"`
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
	Script            pipeline.JSTargetScript `json:"script"`
	Explanation       string                  `json:"explanation,omitempty"`
	RouteID           string                  `json:"route_id,omitempty"`
	Provider          string                  `json:"provider,omitempty"`
	Model             string                  `json:"model,omitempty"`
	VisualContextUsed bool                    `json:"visual_context_used"`
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
	Issues            []string                 `json:"issues,omitempty"`
	Explanation       string                   `json:"explanation,omitempty"`
	SuggestedScript   *pipeline.JSTargetScript `json:"suggested_script,omitempty"`
	RouteID           string                   `json:"route_id,omitempty"`
	Provider          string                   `json:"provider,omitempty"`
	Model             string                   `json:"model,omitempty"`
	VisualContextUsed bool                     `json:"visual_context_used"`
	RecheckStatus     int                      `json:"recheck_status,omitempty"`
	RecheckEngine     string                   `json:"recheck_engine,omitempty"`
	RecheckError      string                   `json:"recheck_error,omitempty"`
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

func (s *Server) handleAIExtractPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	var req AIExtractPreviewRequest
	if err := decodeJSONBodyWithLimit(w, r, &req, maxAIAuthoringRequestBodySize); err != nil {
		writeError(w, r, err)
		return
	}

	result, err := s.aiAuthoringService().Preview(r.Context(), aiauthoring.PreviewRequest{
		URL:           req.URL,
		HTML:          req.HTML,
		Mode:          req.Mode,
		Prompt:        req.Prompt,
		Schema:        req.Schema,
		Fields:        req.Fields,
		Images:        req.Images,
		Headless:      req.Headless,
		UsePlaywright: req.UsePlaywright,
		Visual:        req.Visual,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	resp := AIExtractPreviewResponse{
		Fields:            result.Fields,
		Confidence:        result.Confidence,
		Explanation:       result.Explanation,
		TokensUsed:        result.TokensUsed,
		RouteID:           result.RouteID,
		Provider:          result.Provider,
		Model:             result.Model,
		Cached:            result.Cached,
		VisualContextUsed: result.VisualContextUsed,
	}

	setAIResponseHeaders(w, result.RouteID, result.Provider, result.Model)
	logAIRequestCompletion("extract_preview", req.URL, result.RouteID, result.Provider, result.Model, result.Cached)
	writeJSON(w, resp)
}

func (s *Server) handleAITemplateGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	var req AIExtractTemplateGenerateRequest
	if err := decodeJSONBodyWithLimit(w, r, &req, maxAIAuthoringRequestBodySize); err != nil {
		writeError(w, r, err)
		return
	}

	result, err := s.aiAuthoringService().GenerateTemplate(r.Context(), aiauthoring.TemplateRequest{
		URL:           req.URL,
		HTML:          req.HTML,
		Description:   req.Description,
		SampleFields:  req.SampleFields,
		Images:        req.Images,
		Headless:      req.Headless,
		UsePlaywright: req.UsePlaywright,
		Visual:        req.Visual,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	resp := AIExtractTemplateGenerateResponse{
		Template:          result.Template,
		Explanation:       result.Explanation,
		RouteID:           result.RouteID,
		Provider:          result.Provider,
		Model:             result.Model,
		VisualContextUsed: result.VisualContextUsed,
	}
	setAIResponseHeaders(w, result.RouteID, result.Provider, result.Model)
	logAIRequestCompletion("template_generate", req.URL, result.RouteID, result.Provider, result.Model, false)
	writeJSON(w, resp)
}

func (s *Server) handleAITemplateDebug(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	var req AIExtractTemplateDebugRequest
	if err := decodeJSONBodyWithLimit(w, r, &req, maxAIAuthoringRequestBodySize); err != nil {
		writeError(w, r, err)
		return
	}

	result, err := s.aiAuthoringService().DebugTemplate(r.Context(), aiauthoring.TemplateDebugRequest{
		URL:           req.URL,
		HTML:          req.HTML,
		Template:      req.Template,
		Instructions:  req.Instructions,
		Images:        req.Images,
		Headless:      req.Headless,
		UsePlaywright: req.UsePlaywright,
		Visual:        req.Visual,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	resp := AIExtractTemplateDebugResponse{
		Issues:            result.Issues,
		ExtractedFields:   result.ExtractedFields,
		Explanation:       result.Explanation,
		SuggestedTemplate: result.SuggestedTemplate,
		RouteID:           result.RouteID,
		Provider:          result.Provider,
		Model:             result.Model,
		VisualContextUsed: result.VisualContextUsed,
	}
	setAIResponseHeaders(w, result.RouteID, result.Provider, result.Model)
	logAIRequestCompletion("template_debug", req.URL, result.RouteID, result.Provider, result.Model, false)
	writeJSON(w, resp)
}

func (s *Server) handleAIRenderProfileGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	var req AIRenderProfileGenerateRequest
	if err := decodeJSONBodyWithLimit(w, r, &req, maxAIAuthoringRequestBodySize); err != nil {
		writeError(w, r, err)
		return
	}

	result, err := s.aiAuthoringService().GenerateRenderProfile(r.Context(), aiauthoring.RenderProfileRequest{
		URL:           req.URL,
		Name:          req.Name,
		HostPatterns:  req.HostPatterns,
		Instructions:  req.Instructions,
		Images:        req.Images,
		Headless:      req.Headless,
		UsePlaywright: req.UsePlaywright,
		Visual:        req.Visual,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	resp := AIRenderProfileGenerateResponse{
		Profile:           result.Profile,
		Explanation:       result.Explanation,
		RouteID:           result.RouteID,
		Provider:          result.Provider,
		Model:             result.Model,
		VisualContextUsed: result.VisualContextUsed,
	}
	setAIResponseHeaders(w, result.RouteID, result.Provider, result.Model)
	logAIRequestCompletion("render_profile_generate", req.URL, result.RouteID, result.Provider, result.Model, false)
	writeJSON(w, resp)
}

func (s *Server) handleAIPipelineJSGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	var req AIPipelineJSGenerateRequest
	if err := decodeJSONBodyWithLimit(w, r, &req, maxAIAuthoringRequestBodySize); err != nil {
		writeError(w, r, err)
		return
	}

	result, err := s.aiAuthoringService().GeneratePipelineJS(r.Context(), aiauthoring.PipelineJSRequest{
		URL:           req.URL,
		Name:          req.Name,
		HostPatterns:  req.HostPatterns,
		Instructions:  req.Instructions,
		Images:        req.Images,
		Headless:      req.Headless,
		UsePlaywright: req.UsePlaywright,
		Visual:        req.Visual,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	resp := AIPipelineJSGenerateResponse{
		Script:            result.Script,
		Explanation:       result.Explanation,
		RouteID:           result.RouteID,
		Provider:          result.Provider,
		Model:             result.Model,
		VisualContextUsed: result.VisualContextUsed,
	}
	setAIResponseHeaders(w, result.RouteID, result.Provider, result.Model)
	logAIRequestCompletion("pipeline_js_generate", req.URL, result.RouteID, result.Provider, result.Model, false)
	writeJSON(w, resp)
}

func (s *Server) handleAIRenderProfileDebug(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	var req AIRenderProfileDebugRequest
	if err := decodeJSONBodyWithLimit(w, r, &req, maxAIAuthoringRequestBodySize); err != nil {
		writeError(w, r, err)
		return
	}

	result, err := s.aiAuthoringService().DebugRenderProfile(r.Context(), aiauthoring.RenderProfileDebugRequest{
		URL:           req.URL,
		Profile:       req.Profile,
		Instructions:  req.Instructions,
		Images:        req.Images,
		Headless:      req.Headless,
		UsePlaywright: req.UsePlaywright,
		Visual:        req.Visual,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	resp := AIRenderProfileDebugResponse{
		Issues:            result.Issues,
		Explanation:       result.Explanation,
		SuggestedProfile:  result.SuggestedProfile,
		RouteID:           result.RouteID,
		Provider:          result.Provider,
		Model:             result.Model,
		VisualContextUsed: result.VisualContextUsed,
		RecheckStatus:     result.RecheckStatus,
		RecheckEngine:     result.RecheckEngine,
		RecheckError:      result.RecheckError,
	}
	setAIResponseHeaders(w, result.RouteID, result.Provider, result.Model)
	logAIRequestCompletion("render_profile_debug", req.URL, result.RouteID, result.Provider, result.Model, false)
	writeJSON(w, resp)
}

func (s *Server) handleAIPipelineJSDebug(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	var req AIPipelineJSDebugRequest
	if err := decodeJSONBodyWithLimit(w, r, &req, maxAIAuthoringRequestBodySize); err != nil {
		writeError(w, r, err)
		return
	}

	result, err := s.aiAuthoringService().DebugPipelineJS(r.Context(), aiauthoring.PipelineJSDebugRequest{
		URL:           req.URL,
		Script:        req.Script,
		Instructions:  req.Instructions,
		Images:        req.Images,
		Headless:      req.Headless,
		UsePlaywright: req.UsePlaywright,
		Visual:        req.Visual,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	resp := AIPipelineJSDebugResponse{
		Issues:            result.Issues,
		Explanation:       result.Explanation,
		SuggestedScript:   result.SuggestedScript,
		RouteID:           result.RouteID,
		Provider:          result.Provider,
		Model:             result.Model,
		VisualContextUsed: result.VisualContextUsed,
		RecheckStatus:     result.RecheckStatus,
		RecheckEngine:     result.RecheckEngine,
		RecheckError:      result.RecheckError,
	}
	setAIResponseHeaders(w, result.RouteID, result.Provider, result.Model)
	logAIRequestCompletion("pipeline_js_debug", req.URL, result.RouteID, result.Provider, result.Model, false)
	writeJSON(w, resp)
}

func (s *Server) handleAIResearchRefine(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	var req AIResearchRefineRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}

	result, err := s.aiAuthoringService().RefineResearch(r.Context(), aiauthoring.ResearchRefineRequest{
		Result:       req.Result,
		Instructions: req.Instructions,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	resp := AIResearchRefineResponse{
		Issues:      result.Issues,
		InputStats:  result.InputStats,
		Refined:     result.Refined,
		Markdown:    result.Markdown,
		Explanation: result.Explanation,
		RouteID:     result.RouteID,
		Provider:    result.Provider,
		Model:       result.Model,
	}
	setAIResponseHeaders(w, result.RouteID, result.Provider, result.Model)
	logAIRequestCompletion("research_refine", "", result.RouteID, result.Provider, result.Model, false)
	writeJSON(w, resp)
}

func (s *Server) handleAIExportShape(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	var req AIExportShapeRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	job, rawResult, err := s.loadAIJobResult(r.Context(), strings.TrimSpace(req.JobID))
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := s.aiAuthoringService().GenerateExportShape(r.Context(), aiauthoring.ExportShapeRequest{
		JobKind:      job.Kind,
		Format:       strings.TrimSpace(req.Format),
		RawResult:    rawResult,
		CurrentShape: req.CurrentShape,
		Instructions: req.Instructions,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	resp := AIExportShapeResponse{
		Issues:      result.Issues,
		InputStats:  result.InputStats,
		Shape:       result.Shape,
		Explanation: result.Explanation,
		RouteID:     result.RouteID,
		Provider:    result.Provider,
		Model:       result.Model,
	}
	setAIResponseHeaders(w, result.RouteID, result.Provider, result.Model)
	logAIRequestCompletion("export_shape", "", result.RouteID, result.Provider, result.Model, false)
	writeJSON(w, resp)
}

func (s *Server) handleAITransformGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	var req AITransformGenerateRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	job, rawResult, err := s.loadAIJobResult(r.Context(), strings.TrimSpace(req.JobID))
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := s.aiAuthoringService().GenerateTransform(r.Context(), aiauthoring.TransformRequest{
		JobKind:           job.Kind,
		RawResult:         rawResult,
		CurrentTransform:  req.CurrentTransform,
		PreferredLanguage: strings.TrimSpace(req.PreferredLanguage),
		Instructions:      req.Instructions,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	resp := AITransformGenerateResponse{
		Issues:      result.Issues,
		InputStats:  result.InputStats,
		Transform:   result.Transform,
		Preview:     result.Preview,
		Explanation: result.Explanation,
		RouteID:     result.RouteID,
		Provider:    result.Provider,
		Model:       result.Model,
	}
	setAIResponseHeaders(w, result.RouteID, result.Provider, result.Model)
	logAIRequestCompletion("transform_generate", "", result.RouteID, result.Provider, result.Model, false)
	writeJSON(w, resp)
}

func (s *Server) loadAIJobResult(ctx context.Context, jobID string) (*model.Job, []byte, error) {
	if jobID == "" {
		return nil, nil, apperrors.Validation("job_id is required")
	}
	job, err := s.store.Get(ctx, jobID)
	if err != nil {
		return nil, nil, apperrors.Wrap(apperrors.KindNotFound, "job not found", err)
	}
	if strings.TrimSpace(job.ResultPath) == "" {
		return nil, nil, apperrors.NotFound("job has no result file")
	}
	data, err := os.ReadFile(job.ResultPath)
	if err != nil {
		return nil, nil, apperrors.Wrap(apperrors.KindInternal, "read job result file", err)
	}
	return &job, data, nil
}

func (s *Server) aiAuthoringService() *aiauthoring.Service {
	if s.aiAuthoring != nil {
		return s.aiAuthoring
	}
	return aiauthoring.NewService(s.cfg, s.aiExtractor, !s.cfg.APIAuthEnabled && isLocalhost(s.cfg.BindAddr))
}

func (s *Server) fetchHTMLForAI(ctx context.Context, pageURL string, headless bool, usePlaywright bool) (fetch.Result, error) {
	return s.aiAuthoringService().FetchHTML(ctx, pageURL, headless, usePlaywright)
}

func setAIResponseHeaders(w http.ResponseWriter, routeID string, provider string, model string) {
	if strings.TrimSpace(routeID) != "" {
		w.Header().Set("X-Spartan-AI-Route", routeID)
	}
	if strings.TrimSpace(provider) != "" {
		w.Header().Set("X-Spartan-AI-Provider", provider)
	}
	if strings.TrimSpace(model) != "" {
		w.Header().Set("X-Spartan-AI-Model", model)
	}
}

func logAIRequestCompletion(operation string, requestURL string, routeID string, provider string, model string, cached bool) {
	slog.Info("AI request completed",
		"operation", operation,
		"url", apperrors.SanitizeURL(requestURL),
		"route_id", routeID,
		"provider", provider,
		"model", model,
		"cached", cached,
	)
}
