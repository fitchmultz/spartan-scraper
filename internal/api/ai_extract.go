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
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/andybalholm/cascadia"
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/webhook"
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
	Provider    string                        `json:"provider,omitempty"`
	Model       string                        `json:"model,omitempty"`
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

	if req.HTML == "" {
		parsedURL, err := url.Parse(req.URL)
		if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
			writeError(w, r, apperrors.Validation("invalid URL format"))
			return
		}
	}

	// Perform AI extraction
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(s.cfg.AI.RequestTimeoutSecs)*time.Second)
	defer cancel()

	html := req.HTML
	if html == "" {
		fetched, err := s.fetchHTMLForAI(ctx, req.URL, req.Headless, req.UsePlaywright)
		if err != nil {
			writeError(w, r, err)
			return
		}
		html = fetched.HTML
	}

	aiReq := extract.AIExtractRequest{
		HTML:            html,
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
		Provider:    aiResult.Provider,
		Model:       aiResult.Model,
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

	parsedURL, err := url.Parse(req.URL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		writeError(w, r, apperrors.Validation("invalid URL format"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(s.cfg.AI.RequestTimeoutSecs)*time.Second)
	defer cancel()

	result, fetchErr := s.fetchHTMLForAI(ctx, req.URL, req.Headless, s.cfg.UsePlaywright)
	if fetchErr != nil {
		writeError(w, r, fetchErr)
		return
	}

	aiReq := extract.AITemplateGenerateRequest{
		HTML:         result.HTML,
		URL:          req.URL,
		Description:  req.Description,
		SampleFields: req.SampleFields,
	}

	var aiResult extract.AITemplateGenerateResult
	for attempt := 0; attempt < 2; attempt++ {
		aiResult, err = s.aiExtractor.GenerateTemplate(ctx, aiReq)
		if err != nil {
			writeError(w, r, apperrors.Wrap(apperrors.KindInternal, "AI template generation failed", err))
			return
		}

		validationErrors := validateGeneratedTemplate(result.HTML, aiResult.Template)
		if len(validationErrors) == 0 {
			writeJSON(w, AIExtractTemplateGenerateResponse{
				Template:    aiResult.Template,
				Explanation: aiResult.Explanation,
			})
			return
		}

		if attempt == 1 {
			writeError(w, r, apperrors.Validation(strings.Join(validationErrors, "; ")))
			return
		}

		aiReq.Feedback = "The previous template did not validate against the fetched HTML. Fix these issues: " + strings.Join(validationErrors, "; ")
	}
}

func (s *Server) requireAIExtractor() error {
	if s.aiExtractor == nil {
		return apperrors.Validation("AI extraction is not configured. Enable the pi bridge with PI_ENABLED and build tools/pi-bridge.")
	}
	return nil
}

func (s *Server) fetchHTMLForAI(ctx context.Context, pageURL string, headless bool, usePlaywright bool) (fetch.Result, error) {
	allowInternal := !s.cfg.APIAuthEnabled && isLocalhost(s.cfg.BindAddr)
	if err := webhook.ValidateURL(pageURL, allowInternal); err != nil {
		return fetch.Result{}, err
	}

	fetcher := fetch.NewFetcher(s.cfg.DataDir)
	result, err := fetcher.Fetch(ctx, fetch.Request{
		URL:           pageURL,
		Method:        http.MethodGet,
		Timeout:       time.Duration(s.cfg.RequestTimeoutSecs) * time.Second,
		UserAgent:     s.cfg.UserAgent,
		Headless:      headless,
		UsePlaywright: usePlaywright,
		DataDir:       s.cfg.DataDir,
	})
	if err != nil {
		return fetch.Result{}, apperrors.Wrap(apperrors.KindInternal, "failed to fetch page", err)
	}
	return result, nil
}

func validateGeneratedTemplate(html string, template extract.Template) []string {
	_, err := templateFromRequest(template.Name, CreateTemplateRequest{
		Name:      template.Name,
		Selectors: template.Selectors,
		JSONLD:    template.JSONLD,
		Regex:     template.Regex,
		Normalize: template.Normalize,
	})
	if err != nil {
		return []string{apperrors.SafeMessage(err)}
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return []string{"generated template could not be validated because the fetched HTML was not parseable"}
	}

	validationErrors := make([]string, 0)
	for _, rule := range template.Selectors {
		if strings.TrimSpace(rule.Name) == "" {
			validationErrors = append(validationErrors, "selector rule is missing a field name")
			continue
		}
		if strings.TrimSpace(rule.Selector) == "" {
			validationErrors = append(validationErrors, "selector "+rule.Name+" is empty")
			continue
		}
		if _, err := cascadia.ParseGroup(rule.Selector); err != nil {
			validationErrors = append(validationErrors, "selector "+rule.Name+" is invalid: "+err.Error())
			continue
		}
		if doc.Find(rule.Selector).Length() == 0 {
			validationErrors = append(validationErrors, "selector "+rule.Name+" matched no elements")
		}
	}

	return validationErrors
}
