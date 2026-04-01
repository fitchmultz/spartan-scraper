// Package api provides HTTP handlers for bounded AI authoring endpoints.
//
// Purpose:
// - Handle AI extract preview and template authoring requests.
//
// Responsibilities:
// - Validate bounded preview/template requests, invoke the shared AI authoring service,
// - and adapt service results into stable API responses.
//
// Scope:
// - Preview, template-generate, and template-debug handlers only.
//
// Usage:
// - Mounted under `/v1/ai/*` by the API server.
//
// Invariants/Assumptions:
// - These handlers only accept POST JSON requests.
package api

import (
	"net/http"

	"github.com/fitchmultz/spartan-scraper/internal/aiauthoring"
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

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
