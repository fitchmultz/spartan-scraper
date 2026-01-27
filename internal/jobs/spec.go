// Package jobs provides a unified job specification for creating scrape, crawl, and research jobs.
// It defines JobSpec which consolidates all job parameters into a single structure,
// eliminating duplication across scheduler, MCP, API, and CLI entry points.
package jobs

import (
	"fmt"

	"spartan-scraper/internal/apperrors"
	"spartan-scraper/internal/extract"
	"spartan-scraper/internal/fetch"
	"spartan-scraper/internal/model"
	"spartan-scraper/internal/pipeline"
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
}

// Validate checks that the JobSpec has all required fields for its Kind.
// Returns an error if validation fails.
func (s JobSpec) Validate() error {
	switch s.Kind {
	case model.KindScrape:
		if s.URL == "" {
			return apperrors.Validation("url is required for scrape jobs")
		}
	case model.KindCrawl:
		if s.URL == "" {
			return apperrors.Validation("url is required for crawl jobs")
		}
	case model.KindResearch:
		if s.Query == "" {
			return apperrors.Validation("query is required for research jobs")
		}
		if len(s.URLs) == 0 {
			return apperrors.Validation("urls is required for research jobs")
		}
	default:
		return apperrors.Validation(fmt.Sprintf("unknown job kind: %s", s.Kind))
	}
	if s.TimeoutSeconds <= 0 {
		return apperrors.Validation("timeout must be > 0")
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
