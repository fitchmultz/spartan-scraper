// Package submission validates operator-facing job requests and converts them into
// the canonical create-time jobs.JobSpec and back again.
//
// Purpose:
//   - Centralize request decoding, validation, auth resolution, and typed-spec conversion
//     for live jobs and automation surfaces.
//
// Responsibilities:
// - Strictly decode raw scrape/crawl/research request JSON.
// - Convert validated requests into jobs.JobSpec values with shared defaults.
// - Reconstruct operator-facing requests from persisted typed specs.
//
// Scope:
// - Single-job submission flows only.
//
// Usage:
// - Used by API, MCP, schedules, chains, and watches before creating jobs.
//
// Invariants/Assumptions:
// - Unknown JSON fields are rejected.
// - Shared defaults behave consistently across direct and automation-driven submissions.
// - Auth profile resolution can be deferred for persisted automation templates.
package submission

import (
	"bytes"
	"encoding/json"
	"io"
	"reflect"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/auth"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
	"github.com/fitchmultz/spartan-scraper/internal/validate"
	webhookvalidate "github.com/fitchmultz/spartan-scraper/internal/webhook"
)

// Defaults defines the shared runtime defaults used when turning a request into a job spec.
type Defaults struct {
	DefaultTimeoutSeconds int
	DefaultUsePlaywright  bool
	RequestID             string
	ResolveAuth           bool
}

type jobRequestOptions struct {
	authURL          string
	authProfile      string
	auth             *fetch.AuthOptions
	extract          *extract.ExtractOptions
	pipeline         *pipeline.Options
	webhook          *WebhookConfig
	screenshot       *fetch.ScreenshotConfig
	device           *fetch.DeviceEmulation
	networkIntercept *fetch.NetworkInterceptConfig
	incremental      *bool
	playwright       *bool
	timeoutSeconds   int
	requestID        string
}

func ValidateExtractOptions(opts *extract.ExtractOptions) error {
	if opts == nil || opts.AI == nil || !opts.AI.Enabled {
		return nil
	}
	switch opts.AI.Mode {
	case "", extract.AIModeNaturalLanguage:
		return nil
	case extract.AIModeSchemaGuided:
		if len(opts.AI.Schema) == 0 {
			return apperrors.Validation("extract.ai.schema is required when extract.ai.mode is schema_guided")
		}
		return nil
	default:
		return apperrors.Validation("invalid extract.ai.mode: must be natural_language or schema_guided")
	}
}

func ValidateWebhookConfig(cfg *WebhookConfig) error {
	if cfg == nil {
		return nil
	}
	return webhookvalidate.ValidateConfigURL(cfg.URL)
}

func ValidateScrapeRequest(req ScrapeRequest) error {
	if req.URL == "" {
		return apperrors.Validation("url is required")
	}
	if err := ValidateExtractOptions(req.Extract); err != nil {
		return err
	}
	if err := validate.ValidateJob(validate.JobValidationOpts{
		URL:         req.URL,
		Timeout:     req.TimeoutSeconds,
		AuthProfile: req.AuthProfile,
	}, model.KindScrape); err != nil {
		return err
	}
	return ValidateWebhookConfig(req.Webhook)
}

func ValidateCrawlRequest(req CrawlRequest) error {
	if req.URL == "" {
		return apperrors.Validation("url is required")
	}
	if err := ValidateExtractOptions(req.Extract); err != nil {
		return err
	}
	if err := validate.ValidateJob(validate.JobValidationOpts{
		URL:         req.URL,
		MaxDepth:    req.MaxDepth,
		MaxPages:    req.MaxPages,
		Timeout:     req.TimeoutSeconds,
		AuthProfile: req.AuthProfile,
	}, model.KindCrawl); err != nil {
		return err
	}
	return ValidateWebhookConfig(req.Webhook)
}

func ValidateResearchRequest(req ResearchRequest) error {
	if req.Query == "" || len(req.URLs) == 0 {
		return apperrors.Validation("query and urls are required")
	}
	if err := ValidateExtractOptions(req.Extract); err != nil {
		return err
	}
	if err := model.ValidateResearchAgenticConfig(req.Agentic); err != nil {
		return err
	}
	if err := validate.ValidateJob(validate.JobValidationOpts{
		Query:       req.Query,
		URLs:        req.URLs,
		MaxDepth:    req.MaxDepth,
		MaxPages:    req.MaxPages,
		Timeout:     req.TimeoutSeconds,
		AuthProfile: req.AuthProfile,
	}, model.KindResearch); err != nil {
		return err
	}
	return ValidateWebhookConfig(req.Webhook)
}

func JobSpecFromScrapeRequest(cfg config.Config, defaults Defaults, req ScrapeRequest) (jobs.JobSpec, error) {
	if err := ValidateScrapeRequest(req); err != nil {
		return jobs.JobSpec{}, err
	}
	spec := jobs.JobSpec{
		Kind:        model.KindScrape,
		URL:         req.URL,
		Method:      req.Method,
		Body:        []byte(req.Body),
		ContentType: req.ContentType,
		Headless:    req.Headless,
	}
	if spec.Method == "" {
		spec.Method = "GET"
	}
	if err := applyDefaultsWithConfig(cfg, defaults, &spec, jobRequestOptions{
		authURL:          req.URL,
		authProfile:      req.AuthProfile,
		auth:             req.Auth,
		extract:          req.Extract,
		pipeline:         req.Pipeline,
		webhook:          req.Webhook,
		screenshot:       req.Screenshot,
		device:           req.Device,
		networkIntercept: req.NetworkIntercept,
		incremental:      req.Incremental,
		playwright:       req.Playwright,
		timeoutSeconds:   req.TimeoutSeconds,
		requestID:        defaults.RequestID,
	}, defaults.ResolveAuth); err != nil {
		return jobs.JobSpec{}, err
	}
	return spec, nil
}

func JobSpecFromCrawlRequest(cfg config.Config, defaults Defaults, req CrawlRequest) (jobs.JobSpec, error) {
	if err := ValidateCrawlRequest(req); err != nil {
		return jobs.JobSpec{}, err
	}
	spec := jobs.JobSpec{
		Kind:             model.KindCrawl,
		URL:              req.URL,
		MaxDepth:         req.MaxDepth,
		MaxPages:         req.MaxPages,
		Headless:         req.Headless,
		SitemapURL:       req.SitemapURL,
		SitemapOnly:      valueOr(req.SitemapOnly, false),
		IncludePatterns:  req.IncludePatterns,
		ExcludePatterns:  req.ExcludePatterns,
		RespectRobotsTxt: valueOr(req.RespectRobotsTxt, false),
		SkipDuplicates:   valueOr(req.SkipDuplicates, false),
		SimHashThreshold: valueOr(req.SimHashThreshold, 0),
	}
	if err := applyDefaultsWithConfig(cfg, defaults, &spec, jobRequestOptions{
		authURL:          req.URL,
		authProfile:      req.AuthProfile,
		auth:             req.Auth,
		extract:          req.Extract,
		pipeline:         req.Pipeline,
		webhook:          req.Webhook,
		screenshot:       req.Screenshot,
		device:           req.Device,
		networkIntercept: req.NetworkIntercept,
		incremental:      req.Incremental,
		playwright:       req.Playwright,
		timeoutSeconds:   req.TimeoutSeconds,
		requestID:        defaults.RequestID,
	}, defaults.ResolveAuth); err != nil {
		return jobs.JobSpec{}, err
	}
	return spec, nil
}

func JobSpecFromResearchRequest(cfg config.Config, defaults Defaults, req ResearchRequest) (jobs.JobSpec, error) {
	if err := ValidateResearchRequest(req); err != nil {
		return jobs.JobSpec{}, err
	}
	spec := jobs.JobSpec{
		Kind:     model.KindResearch,
		Query:    req.Query,
		URLs:     req.URLs,
		MaxDepth: req.MaxDepth,
		MaxPages: req.MaxPages,
		Headless: req.Headless,
		Agentic:  req.Agentic,
	}
	if err := applyDefaultsWithConfig(cfg, defaults, &spec, jobRequestOptions{
		authURL:          firstURL(req.URLs),
		authProfile:      req.AuthProfile,
		auth:             req.Auth,
		extract:          req.Extract,
		pipeline:         req.Pipeline,
		webhook:          req.Webhook,
		screenshot:       req.Screenshot,
		device:           req.Device,
		networkIntercept: req.NetworkIntercept,
		playwright:       req.Playwright,
		timeoutSeconds:   req.TimeoutSeconds,
		requestID:        defaults.RequestID,
	}, defaults.ResolveAuth); err != nil {
		return jobs.JobSpec{}, err
	}
	return spec, nil
}

// DecodeRequestJSON strictly decodes a raw request payload for the selected kind.
func DecodeRequestJSON(kind model.Kind, data []byte) (any, error) {
	switch kind {
	case model.KindScrape:
		var req ScrapeRequest
		if err := decodeJSONBytes(data, &req); err != nil {
			return nil, err
		}
		return req, nil
	case model.KindCrawl:
		var req CrawlRequest
		if err := decodeJSONBytes(data, &req); err != nil {
			return nil, err
		}
		return req, nil
	case model.KindResearch:
		var req ResearchRequest
		if err := decodeJSONBytes(data, &req); err != nil {
			return nil, err
		}
		return req, nil
	default:
		return nil, apperrors.Validation("kind must be scrape, crawl, or research")
	}
}

// JobSpecFromRawRequest converts a strict raw request payload into a validated jobs.JobSpec.
func JobSpecFromRawRequest(cfg config.Config, defaults Defaults, kind model.Kind, data []byte) (jobs.JobSpec, any, error) {
	decoded, err := DecodeRequestJSON(kind, data)
	if err != nil {
		return jobs.JobSpec{}, nil, err
	}
	switch typed := decoded.(type) {
	case ScrapeRequest:
		spec, err := JobSpecFromScrapeRequest(cfg, defaults, typed)
		return spec, typed, err
	case CrawlRequest:
		spec, err := JobSpecFromCrawlRequest(cfg, defaults, typed)
		return spec, typed, err
	case ResearchRequest:
		spec, err := JobSpecFromResearchRequest(cfg, defaults, typed)
		return spec, typed, err
	default:
		return jobs.JobSpec{}, nil, apperrors.Validation("unsupported job request type")
	}
}

// NormalizeRawRequest re-marshals a decoded operator-facing request into canonical JSON.
func NormalizeRawRequest(kind model.Kind, data []byte) ([]byte, error) {
	decoded, err := DecodeRequestJSON(kind, data)
	if err != nil {
		return nil, err
	}
	normalized, err := json.Marshal(decoded)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to marshal request", err)
	}
	return normalized, nil
}

// RequestFromTypedSpec reconstructs an operator-facing request from a persisted typed spec.
func RequestFromTypedSpec(spec any) (any, error) {
	switch typed := spec.(type) {
	case model.ScrapeSpecV1:
		return ScrapeRequest{
			URL:              typed.URL,
			Method:           typed.Method,
			Body:             string(typed.Body),
			ContentType:      typed.ContentType,
			Headless:         typed.Execution.Headless,
			Playwright:       falseAsNil(typed.Execution.UsePlaywright),
			TimeoutSeconds:   typed.Execution.TimeoutSeconds,
			AuthProfile:      typed.Execution.AuthProfile,
			Auth:             zeroAsNil(typed.Execution.Auth),
			Extract:          zeroAsNil(typed.Execution.Extract),
			Pipeline:         zeroAsNil(typed.Execution.Pipeline),
			Incremental:      falseAsNil(typed.Incremental),
			Webhook:          webhookConfigFromSpec(typed.Execution.Webhook),
			Screenshot:       typed.Execution.Screenshot,
			Device:           typed.Execution.Device,
			NetworkIntercept: typed.Execution.NetworkIntercept,
		}, nil
	case *model.ScrapeSpecV1:
		if typed == nil {
			return nil, apperrors.Validation("unsupported typed request spec")
		}
		return RequestFromTypedSpec(*typed)
	case model.CrawlSpecV1:
		return CrawlRequest{
			URL:              typed.URL,
			MaxDepth:         typed.MaxDepth,
			MaxPages:         typed.MaxPages,
			Headless:         typed.Execution.Headless,
			Playwright:       falseAsNil(typed.Execution.UsePlaywright),
			TimeoutSeconds:   typed.Execution.TimeoutSeconds,
			AuthProfile:      typed.Execution.AuthProfile,
			Auth:             zeroAsNil(typed.Execution.Auth),
			Extract:          zeroAsNil(typed.Execution.Extract),
			Pipeline:         zeroAsNil(typed.Execution.Pipeline),
			Incremental:      falseAsNil(typed.Incremental),
			SitemapURL:       typed.SitemapURL,
			SitemapOnly:      falseAsNil(typed.SitemapOnly),
			IncludePatterns:  typed.IncludePatterns,
			ExcludePatterns:  typed.ExcludePatterns,
			RespectRobotsTxt: falseAsNil(typed.RespectRobotsTxt),
			SkipDuplicates:   falseAsNil(typed.SkipDuplicates),
			SimHashThreshold: intZeroAsNil(typed.SimHashThreshold),
			Webhook:          webhookConfigFromSpec(typed.Execution.Webhook),
			Screenshot:       typed.Execution.Screenshot,
			Device:           typed.Execution.Device,
			NetworkIntercept: typed.Execution.NetworkIntercept,
		}, nil
	case *model.CrawlSpecV1:
		if typed == nil {
			return nil, apperrors.Validation("unsupported typed request spec")
		}
		return RequestFromTypedSpec(*typed)
	case model.ResearchSpecV1:
		return ResearchRequest{
			Query:            typed.Query,
			URLs:             typed.URLs,
			MaxDepth:         typed.MaxDepth,
			MaxPages:         typed.MaxPages,
			Headless:         typed.Execution.Headless,
			Playwright:       falseAsNil(typed.Execution.UsePlaywright),
			TimeoutSeconds:   typed.Execution.TimeoutSeconds,
			AuthProfile:      typed.Execution.AuthProfile,
			Auth:             zeroAsNil(typed.Execution.Auth),
			Extract:          zeroAsNil(typed.Execution.Extract),
			Pipeline:         zeroAsNil(typed.Execution.Pipeline),
			Webhook:          webhookConfigFromSpec(typed.Execution.Webhook),
			Screenshot:       typed.Execution.Screenshot,
			Device:           typed.Execution.Device,
			NetworkIntercept: typed.Execution.NetworkIntercept,
			Agentic:          typed.Agentic,
		}, nil
	case *model.ResearchSpecV1:
		if typed == nil {
			return nil, apperrors.Validation("unsupported typed request spec")
		}
		return RequestFromTypedSpec(*typed)
	default:
		return nil, apperrors.Validation("unsupported typed request spec")
	}
}

func applyDefaultsWithConfig(cfg config.Config, defaults Defaults, spec *jobs.JobSpec, opts jobRequestOptions, resolveAuth bool) error {
	spec.TimeoutSeconds = opts.timeoutSeconds
	if spec.TimeoutSeconds <= 0 {
		spec.TimeoutSeconds = defaults.DefaultTimeoutSeconds
	}
	spec.UsePlaywright = valueOr(opts.playwright, defaults.DefaultUsePlaywright)
	spec.AuthProfile = opts.authProfile
	spec.Extract = valueOr(opts.extract, extract.ExtractOptions{})
	spec.Pipeline = valueOr(opts.pipeline, pipeline.Options{})
	spec.Incremental = valueOr(opts.incremental, false)
	spec.RequestID = opts.requestID
	spec.Screenshot = opts.screenshot
	spec.Device = opts.device
	spec.NetworkIntercept = opts.networkIntercept
	applyWebhookConfig(spec, opts.webhook)

	if !resolveAuth {
		spec.Auth = valueOr(opts.auth, fetch.AuthOptions{})
		spec.Auth.NormalizeTransport()
		if err := spec.Auth.ValidateTransport(); err != nil {
			return err
		}
		return spec.Validate()
	}

	authOptions, err := resolveAuthForRequest(cfg, opts.authURL, opts.authProfile, opts.auth)
	if err != nil {
		return err
	}
	spec.Auth = authOptions
	return spec.Validate()
}

func applyWebhookConfig(spec *jobs.JobSpec, webhook *WebhookConfig) {
	if webhook == nil {
		return
	}
	spec.WebhookURL = webhook.URL
	spec.WebhookEvents = webhook.Events
	spec.WebhookSecret = webhook.Secret
}

func resolveAuthForRequest(cfg config.Config, url string, profile string, override *fetch.AuthOptions) (fetch.AuthOptions, error) {
	input := auth.ResolveInput{
		ProfileName: profile,
		URL:         url,
		Env:         &cfg.AuthOverrides,
	}
	if override != nil {
		input.Headers = toHeaderKVs(override.Headers)
		input.Cookies = toCookies(override.Cookies)
		input.Tokens = tokensFromOverride(*override)
		if login := loginFromOverride(*override); login != nil {
			input.Login = login
		}
	}
	resolved, err := auth.Resolve(cfg.DataDir, input)
	if err != nil {
		return fetch.AuthOptions{}, err
	}
	authOptions := auth.ToFetchOptions(resolved)
	mergeAuthTransportOverrides(&authOptions, override)
	authOptions.NormalizeTransport()
	if err := authOptions.ValidateTransport(); err != nil {
		return fetch.AuthOptions{}, err
	}
	return authOptions, nil
}

func mergeAuthTransportOverrides(dst *fetch.AuthOptions, override *fetch.AuthOptions) {
	if dst == nil || override == nil {
		return
	}
	if override.Proxy != nil {
		dst.Proxy = override.Proxy
	}
	if override.ProxyHints != nil {
		dst.ProxyHints = fetch.NormalizeProxySelectionHints(override.ProxyHints)
	}
	if override.OAuth2 != nil {
		dst.OAuth2 = override.OAuth2
	}
}

func toHeaderKVs(headers map[string]string) []auth.HeaderKV {
	if len(headers) == 0 {
		return nil
	}
	out := make([]auth.HeaderKV, 0, len(headers))
	for key, value := range headers {
		if strings.TrimSpace(key) == "" {
			continue
		}
		out = append(out, auth.HeaderKV{Key: key, Value: value})
	}
	return out
}

func toCookies(cookies []string) []auth.Cookie {
	if len(cookies) == 0 {
		return nil
	}
	out := make([]auth.Cookie, 0, len(cookies))
	for _, raw := range cookies {
		parts := strings.SplitN(strings.TrimSpace(raw), "=", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if name == "" {
			continue
		}
		out = append(out, auth.Cookie{Name: name, Value: value})
	}
	return out
}

func tokensFromOverride(override fetch.AuthOptions) []auth.Token {
	out := []auth.Token{}
	if override.Basic != "" {
		out = append(out, auth.Token{Kind: auth.TokenBasic, Value: override.Basic})
	}
	for key, value := range override.Query {
		out = append(out, auth.Token{Kind: auth.TokenApiKey, Value: value, Query: key})
	}
	return out
}

func loginFromOverride(override fetch.AuthOptions) *auth.LoginFlow {
	if override.LoginURL == "" && override.LoginUserSelector == "" && override.LoginPassSelector == "" && override.LoginSubmitSelector == "" && override.LoginUser == "" && override.LoginPass == "" {
		return nil
	}
	return &auth.LoginFlow{
		URL:            override.LoginURL,
		UserSelector:   override.LoginUserSelector,
		PassSelector:   override.LoginPassSelector,
		SubmitSelector: override.LoginSubmitSelector,
		Username:       override.LoginUser,
		Password:       override.LoginPass,
	}
}

func decodeJSONBytes(data []byte, dst any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return apperrors.Validation("invalid JSON: " + err.Error())
	}
	var extra any
	if err := decoder.Decode(&extra); err != nil {
		if err == io.EOF {
			return nil
		}
		return apperrors.Validation("invalid JSON: " + err.Error())
	}
	return apperrors.Validation("invalid JSON: request body must contain a single JSON value")
}

func webhookConfigFromSpec(spec *model.WebhookSpec) *WebhookConfig {
	if spec == nil || spec.URL == "" {
		return nil
	}
	return &WebhookConfig{URL: spec.URL, Events: spec.Events, Secret: spec.Secret}
}

func zeroAsNil[T any](value T) *T {
	var zero T
	if reflect.DeepEqual(value, zero) {
		return nil
	}
	copyValue := value
	return &copyValue
}

func falseAsNil(value bool) *bool {
	if !value {
		return nil
	}
	copyValue := value
	return &copyValue
}

func intZeroAsNil(value int) *int {
	if value == 0 {
		return nil
	}
	copyValue := value
	return &copyValue
}

func valueOr[T any](value *T, fallback T) T {
	if value != nil {
		return *value
	}
	return fallback
}

func firstURL(urls []string) string {
	if len(urls) == 0 {
		return ""
	}
	return urls[0]
}
