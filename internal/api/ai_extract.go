// Package api provides HTTP handlers for AI-powered extraction endpoints.
//
// Purpose:
// - Expose AI-assisted extraction preview and template-generation routes.
//
// Responsibilities:
// - Validate extraction requests and shared AI configuration.
// - Enforce strict JSON request parsing and bounded request sizes.
// - Adapt extractor results into stable API responses.
//
// Scope:
// - AI extraction request handlers only.
//
// Usage:
// - Mounted under `/v1/extract/ai-preview` and `/v1/extract/ai-template-generate`.
//
// Invariants/Assumptions:
// - Requests must use `application/json`.
// - AI handlers require `Server.aiExtractor` to be configured.
package api

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/aiauthoring"
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
)

// AIExtractPreviewRequest for POST /v1/extract/ai-preview
type AIExtractPreviewRequest struct {
	URL           string                   `json:"url"`
	HTML          string                   `json:"html,omitempty"` // Optional: provide HTML directly
	Mode          extract.AIExtractionMode `json:"mode"`
	Prompt        string                   `json:"prompt,omitempty"`
	Schema        map[string]interface{}   `json:"schema,omitempty"`
	Fields        []string                 `json:"fields,omitempty"`
	Headless      bool                     `json:"headless,omitempty"`
	UsePlaywright bool                     `json:"playwright,omitempty"`
}

// AIExtractPreviewResponse for preview endpoint
type AIExtractPreviewResponse struct {
	Fields      map[string]extract.FieldValue `json:"fields"`
	Confidence  float64                       `json:"confidence"`
	Explanation string                        `json:"explanation,omitempty"`
	TokensUsed  int                           `json:"tokens_used"`
	RouteID     string                        `json:"route_id,omitempty"`
	Provider    string                        `json:"provider,omitempty"`
	Model       string                        `json:"model,omitempty"`
	Cached      bool                          `json:"cached"`
}

// AIExtractTemplateGenerateRequest for POST /v1/extract/ai-template-generate
type AIExtractTemplateGenerateRequest struct {
	URL           string   `json:"url,omitempty"`
	HTML          string   `json:"html,omitempty"`
	Description   string   `json:"description"`
	SampleFields  []string `json:"sample_fields,omitempty"`
	Headless      bool     `json:"headless,omitempty"`
	UsePlaywright bool     `json:"playwright,omitempty"`
}

// AIExtractTemplateGenerateResponse for template generation
type AIExtractTemplateGenerateResponse struct {
	Template    extract.Template `json:"template"`
	Explanation string           `json:"explanation,omitempty"`
	RouteID     string           `json:"route_id,omitempty"`
	Provider    string           `json:"provider,omitempty"`
	Model       string           `json:"model,omitempty"`
}

func (s *Server) handleAIExtractPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	var req AIExtractPreviewRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
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
		Headless:      req.Headless,
		UsePlaywright: req.UsePlaywright,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	resp := AIExtractPreviewResponse{
		Fields:      result.Fields,
		Confidence:  result.Confidence,
		Explanation: result.Explanation,
		TokensUsed:  result.TokensUsed,
		RouteID:     result.RouteID,
		Provider:    result.Provider,
		Model:       result.Model,
		Cached:      result.Cached,
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
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}

	result, err := s.aiAuthoringService().GenerateTemplate(r.Context(), aiauthoring.TemplateRequest{
		URL:           req.URL,
		HTML:          req.HTML,
		Description:   req.Description,
		SampleFields:  req.SampleFields,
		Headless:      req.Headless,
		UsePlaywright: req.UsePlaywright,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	resp := AIExtractTemplateGenerateResponse{
		Template:    result.Template,
		Explanation: result.Explanation,
		RouteID:     result.RouteID,
		Provider:    result.Provider,
		Model:       result.Model,
	}
	setAIResponseHeaders(w, result.RouteID, result.Provider, result.Model)
	logAIRequestCompletion("template_generate", req.URL, result.RouteID, result.Provider, result.Model, false)
	writeJSON(w, resp)
}

func (s *Server) aiAuthoringService() *aiauthoring.Service {
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
