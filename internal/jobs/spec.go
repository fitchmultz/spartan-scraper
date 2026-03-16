// Package jobs provides the creation-time job contract shared by the retained 1.0 entrypoints.
//
// Purpose:
// - Define the canonical in-memory job creation spec used before persistence.
//
// Responsibilities:
// - Hold the shared create-time fields for scrape, crawl, and research jobs.
// - Validate create-time job specs before they are persisted.
// - Provide lightweight constructors for the supported job kinds.
//
// Scope:
// - Creation-time job specification only. Persisted typed specs live in internal/model.
//
// Usage:
// - Built by API, CLI, scheduler, and MCP entrypoints, then passed to Manager.CreateJob.
//
// Invariants/Assumptions:
// - All job kinds share one creation-time contract.
// - AuthProfile is carried so persisted specs can preserve profile-based auth intent.
package jobs

import (
	"fmt"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
	"github.com/fitchmultz/spartan-scraper/internal/validate"
	webhookvalidate "github.com/fitchmultz/spartan-scraper/internal/webhook"
)

// JobSpec defines all parameters for creating any kind of job.
// It unifies the parameter sets for scrape, crawl, and research jobs,
// allowing callers to use a single CreateJob method instead of kind-specific methods.
type JobSpec struct {
	Kind             model.Kind
	URL              string
	Method           string // HTTP method (GET, POST, PUT, DELETE, PATCH, etc.)
	Body             []byte // Request body for POST/PUT/PATCH
	ContentType      string // Content-Type header for request body
	Query            string
	URLs             []string
	MaxDepth         int
	MaxPages         int
	Headless         bool
	UsePlaywright    bool
	AuthProfile      string
	Auth             fetch.AuthOptions
	TimeoutSeconds   int
	Extract          extract.ExtractOptions
	Pipeline         pipeline.Options
	Incremental      bool
	RequestID        string
	SitemapURL       string                        // Optional sitemap.xml URL for URL discovery
	SitemapOnly      bool                          // If true, only crawl URLs from sitemap, not the root URL
	WebhookURL       string                        // Optional webhook URL for job notifications
	WebhookEvents    []string                      // Events to trigger webhook (default: ["completed"])
	WebhookSecret    string                        // Optional HMAC secret for webhook signatures
	IncludePatterns  []string                      // URL path patterns to include (glob syntax, e.g., /blog/**)
	ExcludePatterns  []string                      // URL path patterns to exclude (glob syntax, e.g., /admin/*)
	Screenshot       *fetch.ScreenshotConfig       // Optional screenshot capture configuration
	NetworkIntercept *fetch.NetworkInterceptConfig // Optional network interception configuration
	Device           *fetch.DeviceEmulation        // Optional device emulation for mobile/responsive content
	RespectRobotsTxt bool                          // Respect robots.txt files (default: false)
	SkipDuplicates   bool                          // Skip pages with near-duplicate content (default: false)
	SimHashThreshold int                           // Hamming distance threshold for duplicate detection (default: 3)
	Agentic          *model.ResearchAgenticConfig  // Optional pi-powered bounded follow-up and synthesis for research jobs
}

// Validate checks that the JobSpec has all required fields for its Kind.
// Returns an error if validation fails.
func (s JobSpec) Validate() error {
	switch s.Kind {
	case model.KindScrape:
		if err := validate.ValidateURL(s.URL); err != nil {
			return err
		}
		if err := validate.ValidateTimeout(s.TimeoutSeconds); err != nil {
			return err
		}
	case model.KindCrawl:
		if err := validate.ValidateURL(s.URL); err != nil {
			return err
		}
		if err := validate.ValidateMaxDepth(s.MaxDepth); err != nil {
			return err
		}
		if err := validate.ValidateMaxPages(s.MaxPages); err != nil {
			return err
		}
		if err := validate.ValidateTimeout(s.TimeoutSeconds); err != nil {
			return err
		}
		if s.SitemapOnly && s.SitemapURL == "" {
			return apperrors.Validation("sitemapOnly requires sitemapURL to be set")
		}
		if s.SitemapURL != "" {
			if err := validate.ValidateURL(s.SitemapURL); err != nil {
				return apperrors.Wrap(apperrors.KindValidation, "invalid sitemapURL", err)
			}
		}
	case model.KindResearch:
		if s.Query == "" {
			return apperrors.Validation("query is required for research jobs")
		}
		if err := validate.ValidateURLs(s.URLs); err != nil {
			return err
		}
		if err := validate.ValidateMaxDepth(s.MaxDepth); err != nil {
			return err
		}
		if err := validate.ValidateMaxPages(s.MaxPages); err != nil {
			return err
		}
		if err := validate.ValidateTimeout(s.TimeoutSeconds); err != nil {
			return err
		}
		if err := model.ValidateResearchAgenticConfig(s.Agentic); err != nil {
			return err
		}
	default:
		return apperrors.Validation(fmt.Sprintf("unknown job kind: %s", s.Kind))
	}
	if s.WebhookURL == "" {
		if len(s.WebhookEvents) > 0 || s.WebhookSecret != "" {
			return apperrors.Validation("webhook URL is required")
		}
		return nil
	}
	if err := webhookvalidate.ValidateConfigURL(s.WebhookURL); err != nil {
		return err
	}
	return nil
}

// NewScrapeSpec creates a JobSpec for a scrape job with required URL.
// Common parameters are set to defaults; callers can override them.
func NewScrapeSpec(url string) JobSpec {
	return JobSpec{
		Kind: model.KindScrape,
		URL:  url,
	}
}

// NewCrawlSpec creates a JobSpec for a crawl job with required URL, depth, and pages.
// Common parameters are set to defaults; callers can override them.
func NewCrawlSpec(url string, maxDepth, maxPages int) JobSpec {
	return JobSpec{
		Kind:     model.KindCrawl,
		URL:      url,
		MaxDepth: maxDepth,
		MaxPages: maxPages,
	}
}

// NewResearchSpec creates a JobSpec for a research job with required query and URLs.
// Common parameters are set to defaults; callers can override them.
func NewResearchSpec(query string, urls []string, maxDepth, maxPages int) JobSpec {
	return JobSpec{
		Kind:     model.KindResearch,
		Query:    query,
		URLs:     urls,
		MaxDepth: maxDepth,
		MaxPages: maxPages,
	}
}
