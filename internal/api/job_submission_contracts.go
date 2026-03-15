package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"reflect"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/scheduler"
	"github.com/fitchmultz/spartan-scraper/internal/validate"
)

type JobSubmissionDefaults struct {
	DefaultTimeoutSeconds int
	DefaultUsePlaywright  bool
	RequestID             string
	ResolveAuth           bool
}

func validateExtractOptions(opts *extract.ExtractOptions) error {
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

func validateScrapeRequest(req ScrapeRequest) error {
	if req.URL == "" {
		return apperrors.Validation("url is required")
	}
	if err := validateExtractOptions(req.Extract); err != nil {
		return err
	}
	return validate.ValidateJob(validate.JobValidationOpts{
		URL:         req.URL,
		Timeout:     req.TimeoutSeconds,
		AuthProfile: req.AuthProfile,
	}, model.KindScrape)
}

func scrapeJobSpecFromRequest(req ScrapeRequest) jobs.JobSpec {
	spec := jobs.JobSpec{
		Kind:        model.KindScrape,
		URL:         req.URL,
		Method:      req.Method,
		Body:        []byte(req.Body),
		ContentType: req.ContentType,
		Headless:    req.Headless,
	}
	if spec.Method == "" {
		spec.Method = http.MethodGet
	}
	return spec
}

func scrapeJobRequestOptions(requestID string, req ScrapeRequest) jobRequestOptions {
	return jobRequestOptions{
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
		requestID:        requestID,
	}
}

func validateCrawlRequest(req CrawlRequest) error {
	if req.URL == "" {
		return apperrors.Validation("url is required")
	}
	if err := validateExtractOptions(req.Extract); err != nil {
		return err
	}
	return validate.ValidateJob(validate.JobValidationOpts{
		URL:         req.URL,
		MaxDepth:    req.MaxDepth,
		MaxPages:    req.MaxPages,
		Timeout:     req.TimeoutSeconds,
		AuthProfile: req.AuthProfile,
	}, model.KindCrawl)
}

func crawlJobSpecFromRequest(req CrawlRequest) jobs.JobSpec {
	return jobs.JobSpec{
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
}

func crawlJobRequestOptions(requestID string, req CrawlRequest) jobRequestOptions {
	return jobRequestOptions{
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
		requestID:        requestID,
	}
}

func validateResearchRequest(req ResearchRequest) error {
	if req.Query == "" || len(req.URLs) == 0 {
		return apperrors.Validation("query and urls are required")
	}
	if err := validateExtractOptions(req.Extract); err != nil {
		return err
	}
	if err := model.ValidateResearchAgenticConfig(req.Agentic); err != nil {
		return err
	}
	return validate.ValidateJob(validate.JobValidationOpts{
		Query:       req.Query,
		URLs:        req.URLs,
		MaxDepth:    req.MaxDepth,
		MaxPages:    req.MaxPages,
		Timeout:     req.TimeoutSeconds,
		AuthProfile: req.AuthProfile,
	}, model.KindResearch)
}

func researchJobSpecFromRequest(req ResearchRequest) jobs.JobSpec {
	return jobs.JobSpec{
		Kind:     model.KindResearch,
		Query:    req.Query,
		URLs:     req.URLs,
		MaxDepth: req.MaxDepth,
		MaxPages: req.MaxPages,
		Headless: req.Headless,
		Agentic:  req.Agentic,
	}
}

func researchJobRequestOptions(requestID string, req ResearchRequest) jobRequestOptions {
	return jobRequestOptions{
		authURL:          req.URLs[0],
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
		requestID:        requestID,
	}
}

func webhookConfigFromSpec(spec *model.WebhookSpec) *WebhookConfig {
	if spec == nil || spec.URL == "" {
		return nil
	}
	return &WebhookConfig{
		URL:    spec.URL,
		Events: spec.Events,
		Secret: spec.Secret,
	}
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

func unsupportedScheduleSpecError(kind model.Kind) error {
	return apperrors.Validation("unsupported typed schedule spec for kind " + string(kind))
}

func requestFromSchedule(schedule scheduler.Schedule) (any, error) {
	switch typed := schedule.Spec.(type) {
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
	default:
		return nil, unsupportedScheduleSpecError(schedule.Kind)
	}
}

func JobSpecFromScrapeRequest(cfg config.Config, defaults JobSubmissionDefaults, req ScrapeRequest) (jobs.JobSpec, error) {
	if err := validateScrapeRequest(req); err != nil {
		return jobs.JobSpec{}, err
	}
	spec := scrapeJobSpecFromRequest(req)
	if err := applyJobDefaultsWithConfig(cfg, defaults.DefaultTimeoutSeconds, defaults.DefaultUsePlaywright, &spec, scrapeJobRequestOptions(defaults.RequestID, req), defaults.ResolveAuth); err != nil {
		return jobs.JobSpec{}, err
	}
	return spec, nil
}

func JobSpecFromCrawlRequest(cfg config.Config, defaults JobSubmissionDefaults, req CrawlRequest) (jobs.JobSpec, error) {
	if err := validateCrawlRequest(req); err != nil {
		return jobs.JobSpec{}, err
	}
	spec := crawlJobSpecFromRequest(req)
	if err := applyJobDefaultsWithConfig(cfg, defaults.DefaultTimeoutSeconds, defaults.DefaultUsePlaywright, &spec, crawlJobRequestOptions(defaults.RequestID, req), defaults.ResolveAuth); err != nil {
		return jobs.JobSpec{}, err
	}
	return spec, nil
}

func JobSpecFromResearchRequest(cfg config.Config, defaults JobSubmissionDefaults, req ResearchRequest) (jobs.JobSpec, error) {
	if err := validateResearchRequest(req); err != nil {
		return jobs.JobSpec{}, err
	}
	spec := researchJobSpecFromRequest(req)
	if err := applyJobDefaultsWithConfig(cfg, defaults.DefaultTimeoutSeconds, defaults.DefaultUsePlaywright, &spec, researchJobRequestOptions(defaults.RequestID, req), defaults.ResolveAuth); err != nil {
		return jobs.JobSpec{}, err
	}
	return spec, nil
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

func convertScheduleRequestToTypedSpec(s *Server, kind model.Kind, rawRequest []byte) (jobs.JobSpec, int, any, error) {
	switch kind {
	case model.KindScrape:
		var req ScrapeRequest
		if err := decodeJSONBytes(rawRequest, &req); err != nil {
			return jobs.JobSpec{}, 0, nil, err
		}
		spec, err := JobSpecFromScrapeRequest(s.cfg, JobSubmissionDefaults{
			DefaultTimeoutSeconds: s.manager.DefaultTimeoutSeconds(),
			DefaultUsePlaywright:  s.manager.DefaultUsePlaywright(),
			ResolveAuth:           false,
		}, req)
		if err != nil {
			return jobs.JobSpec{}, 0, nil, err
		}
		version, typedSpec, err := jobs.TypedSpecFromJobSpec(spec)
		return spec, version, typedSpec, err
	case model.KindCrawl:
		var req CrawlRequest
		if err := decodeJSONBytes(rawRequest, &req); err != nil {
			return jobs.JobSpec{}, 0, nil, err
		}
		spec, err := JobSpecFromCrawlRequest(s.cfg, JobSubmissionDefaults{
			DefaultTimeoutSeconds: s.manager.DefaultTimeoutSeconds(),
			DefaultUsePlaywright:  s.manager.DefaultUsePlaywright(),
			ResolveAuth:           false,
		}, req)
		if err != nil {
			return jobs.JobSpec{}, 0, nil, err
		}
		version, typedSpec, err := jobs.TypedSpecFromJobSpec(spec)
		return spec, version, typedSpec, err
	case model.KindResearch:
		var req ResearchRequest
		if err := decodeJSONBytes(rawRequest, &req); err != nil {
			return jobs.JobSpec{}, 0, nil, err
		}
		spec, err := JobSpecFromResearchRequest(s.cfg, JobSubmissionDefaults{
			DefaultTimeoutSeconds: s.manager.DefaultTimeoutSeconds(),
			DefaultUsePlaywright:  s.manager.DefaultUsePlaywright(),
			ResolveAuth:           false,
		}, req)
		if err != nil {
			return jobs.JobSpec{}, 0, nil, err
		}
		version, typedSpec, err := jobs.TypedSpecFromJobSpec(spec)
		return spec, version, typedSpec, err
	default:
		return jobs.JobSpec{}, 0, nil, apperrors.Validation("kind must be scrape, crawl, or research")
	}
}
