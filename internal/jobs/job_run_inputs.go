// Package jobs provides typed execution inputs decoded from persisted jobs.
//
// Purpose:
// - Convert stored typed job specs into typed runtime inputs before execution.
//
// Responsibilities:
// - Apply stable defaults for persisted typed specs.
// - Validate decoded inputs before dispatching to scrape, crawl, or research.
// - Keep job execution logic free from repeated type assertions.
//
// Scope:
// - Runtime decoding for persisted jobs in Manager.run only.
//
// Usage:
// - Called by job_run.go before invoking kind-specific execution paths.
//
// Invariants/Assumptions:
// - Required fields are validated via JobSpec after decoding.
// - Scheduler-created jobs and direct jobs resolve shared defaults identically.
// - Decode failures return classified validation errors instead of panicking.
package jobs

import (
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

type executionConfig struct {
	RequestID      string
	Headless       bool
	UsePlaywright  bool
	Auth           fetch.AuthOptions
	Extract        extract.ExtractOptions
	Pipeline       pipeline.Options
	TimeoutSeconds int
	Screenshot     *fetch.ScreenshotConfig
}

type scrapeExecutionInput struct {
	Config      executionConfig
	URL         string
	Method      string
	Body        []byte
	ContentType string
	Incremental bool
}

type crawlExecutionInput struct {
	Config                 executionConfig
	URL                    string
	MaxDepth               int
	MaxPages               int
	Incremental            bool
	SitemapURL             string
	SitemapOnly            bool
	IncludePatterns        []string
	ExcludePatterns        []string
	RespectRobotsTxt       bool
	SkipDuplicates         bool
	SimHashThreshold       int
	CrossJobDedup          bool
	CrossJobDedupThreshold int
}

type researchExecutionInput struct {
	Config   executionConfig
	Query    string
	URLs     []string
	MaxDepth int
	MaxPages int
}

func decodeExecutionConfig(spec model.ExecutionSpec, requestID string, manager *Manager) executionConfig {
	timeoutSeconds := spec.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = int(manager.requestTimeout.Seconds())
	}
	return executionConfig{
		RequestID:      requestID,
		Headless:       spec.Headless,
		UsePlaywright:  spec.UsePlaywright,
		Auth:           spec.Auth,
		Extract:        spec.Extract,
		Pipeline:       spec.Pipeline,
		TimeoutSeconds: timeoutSeconds,
		Screenshot:     spec.Screenshot,
	}
}

func decodeScrapeExecutionInput(job model.Job, manager *Manager) (scrapeExecutionInput, error) {
	spec, ok := job.Spec.(model.ScrapeSpecV1)
	if !ok {
		return scrapeExecutionInput{}, apperrors.Validation("persisted scrape job spec is invalid")
	}
	input := scrapeExecutionInput{
		Config:      decodeExecutionConfig(spec.Execution, spec.Execution.RequestID, manager),
		URL:         spec.URL,
		Method:      spec.Method,
		Body:        spec.Body,
		ContentType: spec.ContentType,
		Incremental: spec.Incremental,
	}
	if input.Method == "" {
		input.Method = "GET"
	}
	createSpec := JobSpec{
		Kind:           model.KindScrape,
		URL:            input.URL,
		Method:         input.Method,
		Body:           input.Body,
		ContentType:    input.ContentType,
		Headless:       input.Config.Headless,
		UsePlaywright:  input.Config.UsePlaywright,
		Auth:           input.Config.Auth,
		TimeoutSeconds: input.Config.TimeoutSeconds,
		Extract:        input.Config.Extract,
		Pipeline:       input.Config.Pipeline,
		Incremental:    input.Incremental,
		RequestID:      input.Config.RequestID,
		Screenshot:     input.Config.Screenshot,
	}
	if err := createSpec.Validate(); err != nil {
		return scrapeExecutionInput{}, apperrors.Wrap(apperrors.KindValidation, "invalid scrape job parameters", err)
	}
	return input, nil
}

func decodeCrawlExecutionInput(job model.Job, manager *Manager) (crawlExecutionInput, error) {
	spec, ok := job.Spec.(model.CrawlSpecV1)
	if !ok {
		return crawlExecutionInput{}, apperrors.Validation("persisted crawl job spec is invalid")
	}
	input := crawlExecutionInput{
		Config:                 decodeExecutionConfig(spec.Execution, spec.Execution.RequestID, manager),
		URL:                    spec.URL,
		MaxDepth:               spec.MaxDepth,
		MaxPages:               spec.MaxPages,
		Incremental:            spec.Incremental,
		SitemapURL:             spec.SitemapURL,
		SitemapOnly:            spec.SitemapOnly,
		IncludePatterns:        spec.IncludePatterns,
		ExcludePatterns:        spec.ExcludePatterns,
		RespectRobotsTxt:       spec.RespectRobotsTxt,
		SkipDuplicates:         spec.SkipDuplicates,
		SimHashThreshold:       spec.SimHashThreshold,
		CrossJobDedup:          spec.CrossJobDedup,
		CrossJobDedupThreshold: spec.CrossJobThreshold,
	}
	createSpec := JobSpec{
		Kind:             model.KindCrawl,
		URL:              input.URL,
		MaxDepth:         input.MaxDepth,
		MaxPages:         input.MaxPages,
		Headless:         input.Config.Headless,
		UsePlaywright:    input.Config.UsePlaywright,
		Auth:             input.Config.Auth,
		TimeoutSeconds:   input.Config.TimeoutSeconds,
		Extract:          input.Config.Extract,
		Pipeline:         input.Config.Pipeline,
		Incremental:      input.Incremental,
		RequestID:        input.Config.RequestID,
		SitemapURL:       input.SitemapURL,
		SitemapOnly:      input.SitemapOnly,
		IncludePatterns:  input.IncludePatterns,
		ExcludePatterns:  input.ExcludePatterns,
		Screenshot:       input.Config.Screenshot,
		RespectRobotsTxt: input.RespectRobotsTxt,
		SkipDuplicates:   input.SkipDuplicates,
		SimHashThreshold: input.SimHashThreshold,
	}
	if err := createSpec.Validate(); err != nil {
		return crawlExecutionInput{}, apperrors.Wrap(apperrors.KindValidation, "invalid crawl job parameters", err)
	}
	return input, nil
}

func decodeResearchExecutionInput(job model.Job, manager *Manager) (researchExecutionInput, error) {
	spec, ok := job.Spec.(model.ResearchSpecV1)
	if !ok {
		return researchExecutionInput{}, apperrors.Validation("persisted research job spec is invalid")
	}
	input := researchExecutionInput{
		Config:   decodeExecutionConfig(spec.Execution, spec.Execution.RequestID, manager),
		Query:    spec.Query,
		URLs:     spec.URLs,
		MaxDepth: spec.MaxDepth,
		MaxPages: spec.MaxPages,
	}
	createSpec := JobSpec{
		Kind:           model.KindResearch,
		Query:          input.Query,
		URLs:           input.URLs,
		MaxDepth:       input.MaxDepth,
		MaxPages:       input.MaxPages,
		Headless:       input.Config.Headless,
		UsePlaywright:  input.Config.UsePlaywright,
		Auth:           input.Config.Auth,
		TimeoutSeconds: input.Config.TimeoutSeconds,
		Extract:        input.Config.Extract,
		Pipeline:       input.Config.Pipeline,
		RequestID:      input.Config.RequestID,
		Screenshot:     input.Config.Screenshot,
	}
	if err := createSpec.Validate(); err != nil {
		return researchExecutionInput{}, apperrors.Wrap(apperrors.KindValidation, "invalid research job parameters", err)
	}
	return input, nil
}
