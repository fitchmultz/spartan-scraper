// Package aiauthoring provides aiauthoring functionality for Spartan Scraper.
//
// Purpose:
// - Implement service support for package aiauthoring.
//
// Responsibilities:
// - Define the file-local types, functions, and helpers that belong to this package concern.
//
// Scope:
// - Package-internal behavior owned by this file; broader orchestration stays in adjacent package files.
//
// Usage:
// - Used by other files in package `aiauthoring` and any exported callers that depend on this package.
//
// Invariants/Assumptions:
// - This file should preserve the package contract and rely on surrounding package configuration as the source of truth.

package aiauthoring

import (
	"context"
	"strings"
	"time"

	piai "github.com/fitchmultz/spartan-scraper/internal/ai"
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
)

type AutomationClient interface {
	GenerateRenderProfile(ctx context.Context, req piai.GenerateRenderProfileRequest) (piai.GenerateRenderProfileResult, error)
	GeneratePipelineJS(ctx context.Context, req piai.GeneratePipelineJSRequest) (piai.GeneratePipelineJSResult, error)
	GenerateResearchRefinement(ctx context.Context, req piai.ResearchRefineRequest) (piai.ResearchRefineResult, error)
	GenerateExportShape(ctx context.Context, req piai.ExportShapeRequest) (piai.ExportShapeResult, error)
	GenerateTransform(ctx context.Context, req piai.GenerateTransformRequest) (piai.GenerateTransformResult, error)
}

type Service struct {
	cfg              config.Config
	aiExtractor      *extract.AIExtractor
	automationClient AutomationClient
	allowInternal    bool
}

type PreviewRequest struct {
	URL           string
	HTML          string
	Mode          extract.AIExtractionMode
	Prompt        string
	Schema        map[string]interface{}
	Fields        []string
	Images        []extract.AIImageInput
	Headless      bool
	UsePlaywright bool
	Visual        bool
}

type PreviewResult struct {
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

type TemplateRequest struct {
	URL           string
	HTML          string
	Description   string
	SampleFields  []string
	Images        []extract.AIImageInput
	Headless      bool
	UsePlaywright bool
	Visual        bool
}

type TemplateResult struct {
	Template          extract.Template `json:"template"`
	Explanation       string           `json:"explanation,omitempty"`
	RouteID           string           `json:"route_id,omitempty"`
	Provider          string           `json:"provider,omitempty"`
	Model             string           `json:"model,omitempty"`
	VisualContextUsed bool             `json:"visual_context_used"`
}

type TemplateDebugRequest struct {
	URL           string
	HTML          string
	Template      extract.Template
	Instructions  string
	Images        []extract.AIImageInput
	Headless      bool
	UsePlaywright bool
	Visual        bool
}

type TemplateDebugResult struct {
	Issues            []string                      `json:"issues,omitempty"`
	ExtractedFields   map[string]extract.FieldValue `json:"extracted_fields,omitempty"`
	Explanation       string                        `json:"explanation,omitempty"`
	SuggestedTemplate *extract.Template             `json:"suggested_template,omitempty"`
	RouteID           string                        `json:"route_id,omitempty"`
	Provider          string                        `json:"provider,omitempty"`
	Model             string                        `json:"model,omitempty"`
	VisualContextUsed bool                          `json:"visual_context_used"`
}

func NewService(cfg config.Config, aiExtractor *extract.AIExtractor, allowInternal bool) *Service {
	var automationClient AutomationClient
	if cfg.AI.Enabled {
		automationClient = piai.NewClient(cfg.AI)
	}
	return NewServiceWithAutomationClient(cfg, aiExtractor, automationClient, allowInternal)
}

func NewServiceWithAutomationClient(cfg config.Config, aiExtractor *extract.AIExtractor, automationClient AutomationClient, allowInternal bool) *Service {
	return &Service{
		cfg:              cfg,
		aiExtractor:      aiExtractor,
		automationClient: automationClient,
		allowInternal:    allowInternal,
	}
}

func (s *Service) Preview(ctx context.Context, req PreviewRequest) (PreviewResult, error) {
	if err := s.requireAIExtractor(); err != nil {
		return PreviewResult{}, err
	}
	if strings.TrimSpace(req.URL) == "" && strings.TrimSpace(req.HTML) == "" {
		return PreviewResult{}, apperrors.Validation("url or html is required")
	}
	if req.Mode == "" {
		req.Mode = extract.AIModeNaturalLanguage
	}
	if strings.TrimSpace(req.HTML) == "" {
		if err := validateHTTPURL(req.URL); err != nil {
			return PreviewResult{}, err
		}
	}
	images, err := normalizeDirectAIImages(req.Images)
	if err != nil {
		return PreviewResult{}, err
	}

	ctx, cancel := s.withRequestTimeout(ctx)
	defer cancel()

	page, err := s.resolvePageContext(ctx, req.URL, req.HTML, images, req.Headless, req.UsePlaywright, req.Visual)
	if err != nil {
		return PreviewResult{}, err
	}

	aiResult, err := s.aiExtractor.Extract(ctx, extract.AIExtractRequest{
		HTML:            page.HTML,
		URL:             page.URL,
		Mode:            req.Mode,
		Prompt:          req.Prompt,
		SchemaExample:   req.Schema,
		Fields:          req.Fields,
		Images:          page.Images,
		MaxContentChars: extract.DefaultMaxContentChars,
	})
	if err != nil {
		return PreviewResult{}, apperrors.Wrap(apperrors.KindInternal, "AI extraction failed", err)
	}

	return PreviewResult{
		Fields:            aiResult.Fields,
		Confidence:        aiResult.Confidence,
		Explanation:       aiResult.Explanation,
		TokensUsed:        aiResult.TokensUsed,
		RouteID:           aiResult.RouteID,
		Provider:          aiResult.Provider,
		Model:             aiResult.Model,
		Cached:            aiResult.Cached,
		VisualContextUsed: page.VisualContextUsed,
	}, nil
}

func (s *Service) GenerateTemplate(ctx context.Context, req TemplateRequest) (TemplateResult, error) {
	if err := s.requireAIExtractor(); err != nil {
		return TemplateResult{}, err
	}
	if strings.TrimSpace(req.URL) == "" && strings.TrimSpace(req.HTML) == "" {
		return TemplateResult{}, apperrors.Validation("url or html is required")
	}
	if strings.TrimSpace(req.Description) == "" {
		return TemplateResult{}, apperrors.Validation("description is required")
	}
	if strings.TrimSpace(req.HTML) == "" {
		if err := validateHTTPURL(req.URL); err != nil {
			return TemplateResult{}, err
		}
	}
	images, err := normalizeDirectAIImages(req.Images)
	if err != nil {
		return TemplateResult{}, err
	}

	ctx, cancel := s.withRequestTimeout(ctx)
	defer cancel()

	page, err := s.resolvePageContext(ctx, req.URL, req.HTML, images, req.Headless, req.UsePlaywright, req.Visual)
	if err != nil {
		return TemplateResult{}, err
	}

	aiResult, err := s.generateTemplateWithValidation(ctx, page, extract.AITemplateGenerateRequest{
		HTML:         page.HTML,
		URL:          page.URL,
		Description:  req.Description,
		SampleFields: append([]string(nil), req.SampleFields...),
		Images:       page.Images,
	})
	if err != nil {
		return TemplateResult{}, err
	}

	return TemplateResult{
		Template:          aiResult.Template,
		Explanation:       aiResult.Explanation,
		RouteID:           aiResult.RouteID,
		Provider:          aiResult.Provider,
		Model:             aiResult.Model,
		VisualContextUsed: page.VisualContextUsed,
	}, nil
}

func (s *Service) DebugTemplate(ctx context.Context, req TemplateDebugRequest) (TemplateDebugResult, error) {
	if err := s.requireAIExtractor(); err != nil {
		return TemplateDebugResult{}, err
	}
	if strings.TrimSpace(req.URL) == "" && strings.TrimSpace(req.HTML) == "" {
		return TemplateDebugResult{}, apperrors.Validation("url or html is required")
	}
	if strings.TrimSpace(req.Template.Name) == "" {
		return TemplateDebugResult{}, apperrors.Validation("template.name is required")
	}
	if strings.TrimSpace(req.HTML) == "" {
		if err := validateHTTPURL(req.URL); err != nil {
			return TemplateDebugResult{}, err
		}
	}
	images, err := normalizeDirectAIImages(req.Images)
	if err != nil {
		return TemplateDebugResult{}, err
	}

	ctx, cancel := s.withRequestTimeout(ctx)
	defer cancel()

	page, err := s.resolvePageContext(ctx, req.URL, req.HTML, images, req.Headless, req.UsePlaywright, req.Visual)
	if err != nil {
		return TemplateDebugResult{}, err
	}

	diagnostics := analyzeTemplate(page.URL, page.HTML, req.Template)
	result := TemplateDebugResult{
		Issues:            diagnostics.Issues,
		ExtractedFields:   diagnostics.ExtractedFields,
		VisualContextUsed: page.VisualContextUsed,
	}
	if len(diagnostics.Issues) == 0 && strings.TrimSpace(req.Instructions) == "" {
		result.Explanation = "No local template issues detected."
		return result, nil
	}

	feedback := buildTemplateDebugFeedback(req.Template, diagnostics, req.Instructions)
	sampleFields := diagnostics.FieldNames()
	if len(sampleFields) == 0 {
		sampleFields = templateFieldNames(req.Template)
	}

	aiResult, err := s.generateTemplateWithValidation(ctx, page, extract.AITemplateGenerateRequest{
		HTML:         page.HTML,
		URL:          page.URL,
		Description:  buildTemplateDebugDescription(req.Template, req.Instructions),
		SampleFields: sampleFields,
		Feedback:     feedback,
		Images:       page.Images,
	})
	if err != nil {
		return TemplateDebugResult{}, err
	}

	result.SuggestedTemplate = &aiResult.Template
	result.Explanation = aiResult.Explanation
	result.RouteID = aiResult.RouteID
	result.Provider = aiResult.Provider
	result.Model = aiResult.Model
	return result, nil
}

func (s *Service) FetchHTML(ctx context.Context, pageURL string, headless bool, usePlaywright bool) (fetch.Result, error) {
	return s.fetchPage(ctx, pageURL, headless, usePlaywright, false)
}

func (s *Service) requireAIExtractor() error {
	if s == nil || s.aiExtractor == nil {
		return apperrors.Validation("AI extraction is not configured. Enable the pi bridge with PI_ENABLED and build tools/pi-bridge.")
	}
	return nil
}

func (s *Service) requireAutomationClient() error {
	if s == nil || s.automationClient == nil {
		return apperrors.Validation("AI authoring is not configured. Enable the pi bridge with PI_ENABLED and build tools/pi-bridge.")
	}
	return nil
}

func (s *Service) withRequestTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	requestTimeoutSecs := s.cfg.AI.RequestTimeoutSecs
	if requestTimeoutSecs <= 0 {
		requestTimeoutSecs = config.DefaultPIRequestTimeoutSecs
	}
	return context.WithTimeout(ctx, time.Duration(requestTimeoutSecs)*time.Second)
}

func (s *Service) generateTemplateWithValidation(ctx context.Context, page pageContext, aiReq extract.AITemplateGenerateRequest) (extract.AITemplateGenerateResult, error) {
	for attempt := 0; attempt < 2; attempt++ {
		aiResult, err := s.aiExtractor.GenerateTemplate(ctx, aiReq)
		if err != nil {
			return extract.AITemplateGenerateResult{}, apperrors.Wrap(apperrors.KindInternal, "AI template generation failed", err)
		}

		diagnostics := analyzeTemplate(page.URL, page.HTML, aiResult.Template)
		if len(diagnostics.Issues) == 0 {
			return aiResult, nil
		}
		if attempt == 1 {
			return extract.AITemplateGenerateResult{}, apperrors.Validation(strings.Join(diagnostics.Issues, "; "))
		}

		feedbackParts := []string{}
		if strings.TrimSpace(aiReq.Feedback) != "" {
			feedbackParts = append(feedbackParts, strings.TrimSpace(aiReq.Feedback))
		}
		feedbackParts = append(feedbackParts, "The previous template did not validate against the current page. Fix these issues: "+strings.Join(diagnostics.Issues, "; "))
		aiReq.Feedback = strings.Join(feedbackParts, "\n\n")
	}

	return extract.AITemplateGenerateResult{}, apperrors.Internal("AI template generation failed")
}
