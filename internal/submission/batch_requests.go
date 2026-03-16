// Package submission validates operator-facing batch requests and converts them into
// canonical create-time jobs.JobSpec values.
//
// Purpose:
//   - Keep batch submission validation, defaults, auth resolution, and request-to-spec
//     conversion in the same canonical layer as single-job submission flows.
//
// Responsibilities:
// - Validate batch size, per-item URLs, and shared batch execution options.
// - Reuse single-job request conversion so scrape/crawl/research batch behavior stays aligned.
// - Return the exact job specs a batch submission will persist.
//
// Scope:
// - Operator-facing batch submission flows only.
//
// Usage:
// - Used by REST batch handlers and direct CLI batch execution.
//
// Invariants/Assumptions:
// - Batch item-specific fields are limited to URL/body/method metadata.
// - Shared defaults should match equivalent single-job submissions.
package submission

import (
	"fmt"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/validate"
)

// BatchDefaults defines the shared runtime defaults used when turning a batch request into job specs.
type BatchDefaults struct {
	Defaults
	MaxBatchSize int
}

// JobSpecsFromBatchScrapeRequest converts a batch scrape request into validated scrape job specs.
func JobSpecsFromBatchScrapeRequest(cfg config.Config, defaults BatchDefaults, req BatchScrapeRequest) ([]jobs.JobSpec, error) {
	if err := validateBatchScrapeRequest(req, defaults.MaxBatchSize); err != nil {
		return nil, err
	}
	requests := makeBatchScrapeRequests(req)
	specs := make([]jobs.JobSpec, len(requests))
	for i, jobReq := range requests {
		spec, err := JobSpecFromScrapeRequest(cfg, defaults.Defaults, jobReq)
		if err != nil {
			return nil, err
		}
		specs[i] = spec
	}
	return specs, nil
}

// JobSpecsFromBatchCrawlRequest converts a batch crawl request into validated crawl job specs.
func JobSpecsFromBatchCrawlRequest(cfg config.Config, defaults BatchDefaults, req BatchCrawlRequest) ([]jobs.JobSpec, error) {
	if err := validateBatchCrawlRequest(req, defaults.MaxBatchSize); err != nil {
		return nil, err
	}
	requests := makeBatchCrawlRequests(req)
	specs := make([]jobs.JobSpec, len(requests))
	for i, jobReq := range requests {
		spec, err := JobSpecFromCrawlRequest(cfg, defaults.Defaults, jobReq)
		if err != nil {
			return nil, err
		}
		specs[i] = spec
	}
	return specs, nil
}

// JobSpecsFromBatchResearchRequest converts a batch research request into a validated research job spec slice.
func JobSpecsFromBatchResearchRequest(cfg config.Config, defaults BatchDefaults, req BatchResearchRequest) ([]jobs.JobSpec, error) {
	if err := validateBatchResearchRequest(req, defaults.MaxBatchSize); err != nil {
		return nil, err
	}
	spec, err := JobSpecFromResearchRequest(cfg, defaults.Defaults, makeBatchResearchRequest(req))
	if err != nil {
		return nil, err
	}
	return []jobs.JobSpec{spec}, nil
}

func validateBatchScrapeRequest(req BatchScrapeRequest, maxBatchSize int) error {
	if err := validateBatchJobs(req.Jobs, maxBatchSize); err != nil {
		return err
	}
	return ValidateScrapeRequest(makeBatchScrapeRequest(req, req.Jobs[0]))
}

func validateBatchCrawlRequest(req BatchCrawlRequest, maxBatchSize int) error {
	if err := validateBatchJobs(req.Jobs, maxBatchSize); err != nil {
		return err
	}
	return ValidateCrawlRequest(makeBatchCrawlRequest(req, req.Jobs[0]))
}

func validateBatchResearchRequest(req BatchResearchRequest, maxBatchSize int) error {
	if err := validateBatchJobs(req.Jobs, maxBatchSize); err != nil {
		return err
	}
	return ValidateResearchRequest(makeBatchResearchRequest(req))
}

func validateBatchJobs(items []BatchJobRequest, maxSize int) error {
	if err := validateBatchSize(len(items), maxSize); err != nil {
		return err
	}
	return validateBatchURLs(items)
}

func validateBatchSize(size, maxSize int) error {
	if size == 0 {
		return apperrors.Validation("batch must contain at least one job")
	}
	if maxSize <= 0 {
		maxSize = jobs.DefaultMaxBatchSize
	}
	if size > maxSize {
		return apperrors.Validation(fmt.Sprintf("batch size %d exceeds maximum of %d", size, maxSize))
	}
	return nil
}

func validateBatchURLs(items []BatchJobRequest) error {
	for i, job := range items {
		if err := validate.ValidateURL(job.URL); err != nil {
			return apperrors.Validation(fmt.Sprintf("invalid URL at index %d: %v", i, err))
		}
	}
	return nil
}

func makeBatchScrapeRequests(req BatchScrapeRequest) []ScrapeRequest {
	requests := make([]ScrapeRequest, len(req.Jobs))
	for i, job := range req.Jobs {
		requests[i] = makeBatchScrapeRequest(req, job)
	}
	return requests
}

func makeBatchScrapeRequest(req BatchScrapeRequest, job BatchJobRequest) ScrapeRequest {
	return ScrapeRequest{
		URL:              job.URL,
		Method:           job.Method,
		Body:             job.Body,
		ContentType:      job.ContentType,
		Headless:         req.Headless,
		Playwright:       req.Playwright,
		TimeoutSeconds:   req.TimeoutSeconds,
		AuthProfile:      req.AuthProfile,
		Auth:             req.Auth,
		Extract:          req.Extract,
		Pipeline:         req.Pipeline,
		Incremental:      req.Incremental,
		Webhook:          req.Webhook,
		Screenshot:       req.Screenshot,
		Device:           req.Device,
		NetworkIntercept: req.NetworkIntercept,
	}
}

func makeBatchCrawlRequests(req BatchCrawlRequest) []CrawlRequest {
	requests := make([]CrawlRequest, len(req.Jobs))
	for i, job := range req.Jobs {
		requests[i] = makeBatchCrawlRequest(req, job)
	}
	return requests
}

func makeBatchCrawlRequest(req BatchCrawlRequest, job BatchJobRequest) CrawlRequest {
	return CrawlRequest{
		URL:              job.URL,
		MaxDepth:         req.MaxDepth,
		MaxPages:         req.MaxPages,
		Headless:         req.Headless,
		Playwright:       req.Playwright,
		TimeoutSeconds:   req.TimeoutSeconds,
		AuthProfile:      req.AuthProfile,
		Auth:             req.Auth,
		Extract:          req.Extract,
		Pipeline:         req.Pipeline,
		Incremental:      req.Incremental,
		SitemapURL:       req.SitemapURL,
		SitemapOnly:      req.SitemapOnly,
		IncludePatterns:  req.IncludePatterns,
		ExcludePatterns:  req.ExcludePatterns,
		RespectRobotsTxt: req.RespectRobotsTxt,
		SkipDuplicates:   req.SkipDuplicates,
		SimHashThreshold: req.SimHashThreshold,
		Webhook:          req.Webhook,
		Screenshot:       req.Screenshot,
		Device:           req.Device,
		NetworkIntercept: req.NetworkIntercept,
	}
}

func makeBatchResearchRequest(req BatchResearchRequest) ResearchRequest {
	urls := make([]string, len(req.Jobs))
	for i, job := range req.Jobs {
		urls[i] = job.URL
	}
	return ResearchRequest{
		Query:            req.Query,
		URLs:             urls,
		MaxDepth:         req.MaxDepth,
		MaxPages:         req.MaxPages,
		Headless:         req.Headless,
		Playwright:       req.Playwright,
		TimeoutSeconds:   req.TimeoutSeconds,
		AuthProfile:      req.AuthProfile,
		Auth:             req.Auth,
		Extract:          req.Extract,
		Pipeline:         req.Pipeline,
		Webhook:          req.Webhook,
		Screenshot:       req.Screenshot,
		Device:           req.Device,
		NetworkIntercept: req.NetworkIntercept,
		Agentic:          req.Agentic,
	}
}
