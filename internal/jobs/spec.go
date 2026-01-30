// Package jobs provides a unified job specification for creating scrape, crawl, and research jobs.
// It defines JobSpec which consolidates all job parameters into a single structure,
// eliminating duplication across scheduler, MCP, API, and CLI entry points.
package jobs

import (
	"fmt"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
	"github.com/fitchmultz/spartan-scraper/internal/validate"
)

// JobSpec defines all parameters for creating any kind of job.
// It unifies the parameter sets for scrape, crawl, and research jobs,
// allowing callers to use a single CreateJob method instead of kind-specific methods.
type JobSpec struct {
	Kind           model.Kind
	URL            string
	Query          string
	URLs           []string
	MaxDepth       int
	MaxPages       int
	Headless       bool
	UsePlaywright  bool
	Auth           fetch.AuthOptions
	TimeoutSeconds int
	Extract        extract.ExtractOptions
	Pipeline       pipeline.Options
	Incremental    bool
	RequestID      string
	SitemapURL     string   // Optional sitemap.xml URL for URL discovery
	SitemapOnly    bool     // If true, only crawl URLs from sitemap, not the root URL
	WebhookURL     string   // Optional webhook URL for job notifications
	WebhookEvents  []string // Events to trigger webhook (default: ["completed"])
	WebhookSecret  string   // Optional HMAC secret for webhook signatures
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
	default:
		return apperrors.Validation(fmt.Sprintf("unknown job kind: %s", s.Kind))
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
