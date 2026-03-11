// Package model defines the canonical typed job specification contract.
//
// Purpose:
// - Provide versioned persisted job specs for scrape, crawl, and research jobs.
//
// Responsibilities:
// - Define the shared execution settings used by all job kinds.
// - Define typed V1 specs for scrape, crawl, and research jobs.
// - Decode persisted spec JSON into typed in-memory structures.
//
// Scope:
// - Stable job contract types only.
//
// Usage:
// - Used by API, scheduler, MCP, store, and jobs runtime code as the source of truth.
//
// Invariants/Assumptions:
// - Persisted jobs always store a kind, spec version, and spec JSON payload.
// - Version 1 is the only supported persisted spec version in Balanced 1.0.
package model

import (
	"encoding/json"
	"fmt"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/auth"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

const (
	// JobSpecVersion1 is the canonical persisted job spec version for Balanced 1.0.
	JobSpecVersion1 = 1
)

// WebhookSpec defines webhook delivery configuration embedded in stable job specs.
type WebhookSpec struct {
	URL    string   `json:"url,omitempty"`
	Events []string `json:"events,omitempty"`
	Secret string   `json:"secret,omitempty"`
}

// ExecutionSpec defines the shared execution settings reused across all job kinds.
type ExecutionSpec struct {
	RequestID        string                        `json:"requestId,omitempty"`
	Headless         bool                          `json:"headless"`
	UsePlaywright    bool                          `json:"playwright"`
	TimeoutSeconds   int                           `json:"timeoutSeconds"`
	AuthProfile      string                        `json:"authProfile,omitempty"`
	Auth             fetch.AuthOptions             `json:"auth,omitempty"`
	Extract          extract.ExtractOptions        `json:"extract,omitempty"`
	Pipeline         pipeline.Options              `json:"pipeline,omitempty"`
	Screenshot       *fetch.ScreenshotConfig       `json:"screenshot,omitempty"`
	NetworkIntercept *fetch.NetworkInterceptConfig `json:"networkIntercept,omitempty"`
	Device           *fetch.DeviceEmulation        `json:"device,omitempty"`
	Webhook          *WebhookSpec                  `json:"webhook,omitempty"`
}

// ScrapeSpecV1 is the persisted scrape job contract.
type ScrapeSpecV1 struct {
	Version     int           `json:"version"`
	URL         string        `json:"url"`
	Method      string        `json:"method,omitempty"`
	Body        []byte        `json:"body,omitempty"`
	ContentType string        `json:"contentType,omitempty"`
	Incremental bool          `json:"incremental,omitempty"`
	Execution   ExecutionSpec `json:"execution"`
}

// CrawlSpecV1 is the persisted crawl job contract.
type CrawlSpecV1 struct {
	Version           int           `json:"version"`
	URL               string        `json:"url"`
	MaxDepth          int           `json:"maxDepth"`
	MaxPages          int           `json:"maxPages"`
	Incremental       bool          `json:"incremental,omitempty"`
	SitemapURL        string        `json:"sitemapURL,omitempty"`
	SitemapOnly       bool          `json:"sitemapOnly,omitempty"`
	IncludePatterns   []string      `json:"includePatterns,omitempty"`
	ExcludePatterns   []string      `json:"excludePatterns,omitempty"`
	RespectRobotsTxt  bool          `json:"respectRobotsTxt,omitempty"`
	SkipDuplicates    bool          `json:"skipDuplicates,omitempty"`
	SimHashThreshold  int           `json:"simHashThreshold,omitempty"`
	CrossJobDedup     bool          `json:"crossJobDedup,omitempty"`
	CrossJobThreshold int           `json:"crossJobDedupThreshold,omitempty"`
	Execution         ExecutionSpec `json:"execution"`
}

// ResearchSpecV1 is the persisted research job contract.
type ResearchSpecV1 struct {
	Version   int           `json:"version"`
	Query     string        `json:"query"`
	URLs      []string      `json:"urls"`
	MaxDepth  int           `json:"maxDepth"`
	MaxPages  int           `json:"maxPages"`
	Execution ExecutionSpec `json:"execution"`
}

// DecodeJobSpec decodes persisted spec JSON into the supported typed spec for the kind/version pair.
func DecodeJobSpec(kind Kind, version int, raw []byte) (any, error) {
	switch {
	case kind == KindScrape && version == JobSpecVersion1:
		var spec ScrapeSpecV1
		if err := json.Unmarshal(raw, &spec); err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to decode scrape spec", err)
		}
		return spec, nil
	case kind == KindCrawl && version == JobSpecVersion1:
		var spec CrawlSpecV1
		if err := json.Unmarshal(raw, &spec); err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to decode crawl spec", err)
		}
		return spec, nil
	case kind == KindResearch && version == JobSpecVersion1:
		var spec ResearchSpecV1
		if err := json.Unmarshal(raw, &spec); err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to decode research spec", err)
		}
		return spec, nil
	default:
		return nil, apperrors.Validation(fmt.Sprintf("unsupported job spec version %d for kind %s", version, kind))
	}
}

// MarshalJobSpec marshals a typed spec into persisted JSON.
func MarshalJobSpec(spec any) ([]byte, error) {
	raw, err := json.Marshal(spec)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to marshal job spec", err)
	}
	return raw, nil
}

// ExtractWebhookSpec extracts webhook config from a typed persisted spec.
func ExtractWebhookSpec(spec any) *WebhookSpec {
	extract := func(webhook *WebhookSpec) *WebhookSpec {
		if webhook == nil || webhook.URL == "" {
			return nil
		}
		return webhook
	}
	switch typed := spec.(type) {
	case ScrapeSpecV1:
		return extract(typed.Execution.Webhook)
	case *ScrapeSpecV1:
		return extract(typed.Execution.Webhook)
	case CrawlSpecV1:
		return extract(typed.Execution.Webhook)
	case *CrawlSpecV1:
		return extract(typed.Execution.Webhook)
	case ResearchSpecV1:
		return extract(typed.Execution.Webhook)
	case *ResearchSpecV1:
		return extract(typed.Execution.Webhook)
	default:
		return nil
	}
}

// AuthOverridesFromExecution returns auth resolve input from a shared execution spec.
func AuthOverridesFromExecution(exec ExecutionSpec) auth.ResolveInput {
	input := auth.ResolveInput{
		ProfileName: exec.AuthProfile,
		Headers:     make([]auth.HeaderKV, 0, len(exec.Auth.Headers)),
		Cookies:     make([]auth.Cookie, 0, len(exec.Auth.Cookies)),
	}
	for key, value := range exec.Auth.Headers {
		input.Headers = append(input.Headers, auth.HeaderKV{Key: key, Value: value})
	}
	for _, raw := range exec.Auth.Cookies {
		if raw == "" {
			continue
		}
		input.Cookies = append(input.Cookies, auth.Cookie{Name: raw, Value: raw})
	}
	if exec.Auth.Basic != "" {
		input.Tokens = append(input.Tokens, auth.Token{Kind: auth.TokenBasic, Value: exec.Auth.Basic})
	}
	if exec.Auth.LoginURL != "" {
		input.Login = &auth.LoginFlow{
			URL:            exec.Auth.LoginURL,
			UserSelector:   exec.Auth.LoginUserSelector,
			PassSelector:   exec.Auth.LoginPassSelector,
			SubmitSelector: exec.Auth.LoginSubmitSelector,
			Username:       exec.Auth.LoginUser,
			Password:       exec.Auth.LoginPass,
			AutoDetect:     exec.Auth.LoginAutoDetect,
		}
	}
	return input
}
