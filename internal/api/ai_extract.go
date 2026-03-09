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
	"net/http"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
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
	Cached      bool                          `json:"cached"`
}

// AIExtractTemplateGenerateRequest for POST /v1/extract/ai-template-generate
type AIExtractTemplateGenerateRequest struct {
	URL          string   `json:"url"`
	Description  string   `json:"description"`
	SampleFields []string `json:"sample_fields,omitempty"`
	Headless     bool     `json:"headless,omitempty"`
}

// AIExtractTemplateGenerateResponse for template generation
type AIExtractTemplateGenerateResponse struct {
	Template    extract.Template `json:"template"`
	Explanation string           `json:"explanation,omitempty"`
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
	if err := s.requireAIExtractor(); err != nil {
		writeError(w, r, err)
		return
	}

	// Validate request
	if req.URL == "" && req.HTML == "" {
		writeError(w, r, apperrors.Validation("url or html is required"))
		return
	}

	// Default mode
	if req.Mode == "" {
		req.Mode = extract.AIModeNaturalLanguage
	}

	// Note: For security reasons, we don't automatically fetch arbitrary URLs
	// Clients should fetch the HTML themselves or use the job system
	if req.HTML == "" {
		writeError(w, r, apperrors.Validation("html content is required. Fetch the URL client-side and provide the HTML, or use the job system for server-side fetching."))
		return
	}

	// Perform AI extraction
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(s.cfg.AI.TimeoutSecs)*time.Second)
	defer cancel()

	aiReq := extract.AIExtractRequest{
		HTML:            req.HTML,
		URL:             req.URL,
		Mode:            req.Mode,
		Prompt:          req.Prompt,
		SchemaExample:   req.Schema,
		Fields:          req.Fields,
		MaxContentChars: extract.DefaultMaxContentChars,
	}

	aiResult, err := s.aiExtractor.Extract(ctx, aiReq)
	if err != nil {
		writeError(w, r, apperrors.Wrap(apperrors.KindInternal, "AI extraction failed", err))
		return
	}

	resp := AIExtractPreviewResponse{
		Fields:      aiResult.Fields,
		Confidence:  aiResult.Confidence,
		Explanation: aiResult.Explanation,
		TokensUsed:  aiResult.TokensUsed,
		Cached:      aiResult.Cached,
	}

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
	if err := s.requireAIExtractor(); err != nil {
		writeError(w, r, err)
		return
	}

	// Validate request
	if req.URL == "" {
		writeError(w, r, apperrors.Validation("url is required"))
		return
	}
	if req.Description == "" {
		writeError(w, r, apperrors.Validation("description is required"))
		return
	}

	// Note: For security reasons, we don't automatically fetch arbitrary URLs
	// Return an error directing clients to use the job system
	writeError(w, r, apperrors.Validation("AI template generation requires HTML content. Use the job system to fetch and analyze URLs, or fetch client-side and use the template-preview endpoint."))
}

func (s *Server) requireAIExtractor() error {
	if s.aiExtractor == nil {
		return apperrors.Validation("AI extraction is not configured. Set AI_PROVIDER and AI_API_KEY environment variables.")
	}
	return nil
}
