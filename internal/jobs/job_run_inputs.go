// Package jobs provides typed execution inputs decoded from persisted jobs.
//
// Purpose:
// - Convert stored job parameter maps into typed runtime inputs before execution.
//
// Responsibilities:
// - Apply stable defaults for persisted job params.
// - Validate decoded inputs before dispatching to scrape, crawl, or research.
// - Keep job execution logic free from repeated map lookups.
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
	"github.com/fitchmultz/spartan-scraper/internal/paramdecode"
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

func decodeExecutionConfig(job model.Job, manager *Manager) executionConfig {
	return executionConfig{
		RequestID:      getJobRequestID(job),
		Headless:       paramdecode.Bool(job.Params, "headless"),
		UsePlaywright:  paramdecode.BoolDefault(job.Params, "playwright", manager.usePlaywright),
		Auth:           paramdecode.Decode[fetch.AuthOptions](job.Params, "auth"),
		Extract:        paramdecode.Decode[extract.ExtractOptions](job.Params, "extract"),
		Pipeline:       paramdecode.Decode[pipeline.Options](job.Params, "pipeline"),
		TimeoutSeconds: paramdecode.PositiveInt(job.Params, "timeout", int(manager.requestTimeout.Seconds())),
		Screenshot:     paramdecode.DecodePtr[fetch.ScreenshotConfig](job.Params, "screenshot"),
	}
}

func decodeScrapeExecutionInput(job model.Job, manager *Manager) (scrapeExecutionInput, error) {
	input := scrapeExecutionInput{
		Config:      decodeExecutionConfig(job, manager),
		URL:         paramdecode.String(job.Params, "url"),
		Method:      paramdecode.String(job.Params, "method"),
		Body:        paramdecode.Bytes(job.Params, "body"),
		ContentType: paramdecode.String(job.Params, "contentType"),
		Incremental: paramdecode.Bool(job.Params, "incremental"),
	}
	if input.Method == "" {
		input.Method = "GET"
	}
	spec := JobSpec{
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
	if err := spec.Validate(); err != nil {
		return scrapeExecutionInput{}, apperrors.Wrap(apperrors.KindValidation, "invalid scrape job parameters", err)
	}
	return input, nil
}

func decodeCrawlExecutionInput(job model.Job, manager *Manager) (crawlExecutionInput, error) {
	input := crawlExecutionInput{
		Config:                 decodeExecutionConfig(job, manager),
		URL:                    paramdecode.String(job.Params, "url"),
		MaxDepth:               paramdecode.PositiveInt(job.Params, "maxDepth", 2),
		MaxPages:               paramdecode.PositiveInt(job.Params, "maxPages", 200),
		Incremental:            paramdecode.Bool(job.Params, "incremental"),
		SitemapURL:             paramdecode.String(job.Params, "sitemapURL"),
		SitemapOnly:            paramdecode.Bool(job.Params, "sitemapOnly"),
		IncludePatterns:        paramdecode.StringSlice(job.Params, "includePatterns"),
		ExcludePatterns:        paramdecode.StringSlice(job.Params, "excludePatterns"),
		RespectRobotsTxt:       paramdecode.Bool(job.Params, "respectRobotsTxt"),
		SkipDuplicates:         paramdecode.Bool(job.Params, "skipDuplicates"),
		SimHashThreshold:       paramdecode.PositiveInt(job.Params, "simHashThreshold", 3),
		CrossJobDedup:          paramdecode.Bool(job.Params, "crossJobDedup"),
		CrossJobDedupThreshold: paramdecode.PositiveInt(job.Params, "crossJobDedupThreshold", 3),
	}
	spec := JobSpec{
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
	if err := spec.Validate(); err != nil {
		return crawlExecutionInput{}, apperrors.Wrap(apperrors.KindValidation, "invalid crawl job parameters", err)
	}
	return input, nil
}

func decodeResearchExecutionInput(job model.Job, manager *Manager) (researchExecutionInput, error) {
	input := researchExecutionInput{
		Config:   decodeExecutionConfig(job, manager),
		Query:    paramdecode.String(job.Params, "query"),
		URLs:     paramdecode.StringSlice(job.Params, "urls"),
		MaxDepth: paramdecode.PositiveInt(job.Params, "maxDepth", 2),
		MaxPages: paramdecode.PositiveInt(job.Params, "maxPages", 200),
	}
	spec := JobSpec{
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
	if err := spec.Validate(); err != nil {
		return researchExecutionInput{}, apperrors.Wrap(apperrors.KindValidation, "invalid research job parameters", err)
	}
	return input, nil
}
