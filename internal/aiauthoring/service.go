package aiauthoring

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/andybalholm/cascadia"
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/webhook"
)

type Service struct {
	cfg           config.Config
	aiExtractor   *extract.AIExtractor
	allowInternal bool
}

type PreviewRequest struct {
	URL           string
	HTML          string
	Mode          extract.AIExtractionMode
	Prompt        string
	Schema        map[string]interface{}
	Fields        []string
	Headless      bool
	UsePlaywright bool
}

type PreviewResult struct {
	Fields      map[string]extract.FieldValue `json:"fields"`
	Confidence  float64                       `json:"confidence"`
	Explanation string                        `json:"explanation,omitempty"`
	TokensUsed  int                           `json:"tokens_used"`
	RouteID     string                        `json:"route_id,omitempty"`
	Provider    string                        `json:"provider,omitempty"`
	Model       string                        `json:"model,omitempty"`
	Cached      bool                          `json:"cached"`
}

type TemplateRequest struct {
	URL           string
	HTML          string
	Description   string
	SampleFields  []string
	Headless      bool
	UsePlaywright bool
}

type TemplateResult struct {
	Template    extract.Template `json:"template"`
	Explanation string           `json:"explanation,omitempty"`
	RouteID     string           `json:"route_id,omitempty"`
	Provider    string           `json:"provider,omitempty"`
	Model       string           `json:"model,omitempty"`
}

func NewService(cfg config.Config, aiExtractor *extract.AIExtractor, allowInternal bool) *Service {
	return &Service{cfg: cfg, aiExtractor: aiExtractor, allowInternal: allowInternal}
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

	requestTimeoutSecs := s.cfg.AI.RequestTimeoutSecs
	if requestTimeoutSecs <= 0 {
		requestTimeoutSecs = config.DefaultPIRequestTimeoutSecs
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(requestTimeoutSecs)*time.Second)
	defer cancel()

	html := req.HTML
	if strings.TrimSpace(html) == "" {
		fetched, err := s.FetchHTML(ctx, req.URL, req.Headless, req.UsePlaywright)
		if err != nil {
			return PreviewResult{}, err
		}
		html = fetched.HTML
	}

	aiResult, err := s.aiExtractor.Extract(ctx, extract.AIExtractRequest{
		HTML:            html,
		URL:             req.URL,
		Mode:            req.Mode,
		Prompt:          req.Prompt,
		SchemaExample:   req.Schema,
		Fields:          req.Fields,
		MaxContentChars: extract.DefaultMaxContentChars,
	})
	if err != nil {
		return PreviewResult{}, apperrors.Wrap(apperrors.KindInternal, "AI extraction failed", err)
	}

	return PreviewResult{
		Fields:      aiResult.Fields,
		Confidence:  aiResult.Confidence,
		Explanation: aiResult.Explanation,
		TokensUsed:  aiResult.TokensUsed,
		RouteID:     aiResult.RouteID,
		Provider:    aiResult.Provider,
		Model:       aiResult.Model,
		Cached:      aiResult.Cached,
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

	requestTimeoutSecs := s.cfg.AI.RequestTimeoutSecs
	if requestTimeoutSecs <= 0 {
		requestTimeoutSecs = config.DefaultPIRequestTimeoutSecs
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(requestTimeoutSecs)*time.Second)
	defer cancel()

	html := req.HTML
	if strings.TrimSpace(html) == "" {
		fetched, err := s.FetchHTML(ctx, req.URL, req.Headless, req.UsePlaywright)
		if err != nil {
			return TemplateResult{}, err
		}
		html = fetched.HTML
	}

	aiReq := extract.AITemplateGenerateRequest{
		HTML:         html,
		URL:          req.URL,
		Description:  req.Description,
		SampleFields: append([]string(nil), req.SampleFields...),
	}

	for attempt := 0; attempt < 2; attempt++ {
		aiResult, err := s.aiExtractor.GenerateTemplate(ctx, aiReq)
		if err != nil {
			return TemplateResult{}, apperrors.Wrap(apperrors.KindInternal, "AI template generation failed", err)
		}

		validationErrors := validateGeneratedTemplate(html, aiResult.Template)
		if len(validationErrors) == 0 {
			return TemplateResult{
				Template:    aiResult.Template,
				Explanation: aiResult.Explanation,
				RouteID:     aiResult.RouteID,
				Provider:    aiResult.Provider,
				Model:       aiResult.Model,
			}, nil
		}

		if attempt == 1 {
			return TemplateResult{}, apperrors.Validation(strings.Join(validationErrors, "; "))
		}
		aiReq.Feedback = "The previous template did not validate against the fetched HTML. Fix these issues: " + strings.Join(validationErrors, "; ")
	}

	return TemplateResult{}, apperrors.Internal("AI template generation failed")
}

func (s *Service) requireAIExtractor() error {
	if s == nil || s.aiExtractor == nil {
		return apperrors.Validation("AI extraction is not configured. Enable the pi bridge with PI_ENABLED and build tools/pi-bridge.")
	}
	return nil
}

func (s *Service) FetchHTML(ctx context.Context, pageURL string, headless bool, usePlaywright bool) (fetch.Result, error) {
	if err := webhook.ValidateURL(pageURL, s.allowInternal); err != nil {
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

func validateHTTPURL(raw string) error {
	parsedURL, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return apperrors.Validation("invalid URL format")
	}
	return nil
}

func validateGeneratedTemplate(html string, template extract.Template) []string {
	if strings.TrimSpace(template.Name) == "" {
		return []string{"name is required"}
	}
	if len(template.Selectors) == 0 {
		return []string{"at least one selector is required"}
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
			validationErrors = append(validationErrors, fmt.Sprintf("selector %s is empty", rule.Name))
			continue
		}
		if _, err := cascadia.ParseGroup(rule.Selector); err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("selector %s is invalid: %s", rule.Name, err.Error()))
			continue
		}
		if doc.Find(rule.Selector).Length() == 0 {
			validationErrors = append(validationErrors, fmt.Sprintf("selector %s matched no elements", rule.Name))
		}
	}

	return validationErrors
}
