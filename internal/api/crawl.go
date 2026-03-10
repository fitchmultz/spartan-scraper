// Package api provides HTTP handlers for crawl job endpoints.
//
// Purpose:
// - Accept website crawl submissions over the API.
//
// Responsibilities:
// - Validate crawl requests.
// - Build a crawl JobSpec with shared request defaults.
// - Create and enqueue the job, then return the sanitized record.
//
// Scope:
// - Crawl request handling only; job execution lives in internal/jobs.
//
// Usage:
// - Registered for POST /v1/crawl.
//
// Invariants/Assumptions:
// - Requests must be JSON and include a URL.
// - Optional sitemap-only mode defaults to false.
package api

import (
	"net/http"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/validate"
)

func (s *Server) handleCrawl(w http.ResponseWriter, r *http.Request) {
	handleSingleJobSubmission(s, w, r, singleJobSubmission[CrawlRequest]{
		kind: model.KindCrawl,
		validate: func(req CrawlRequest) error {
			if req.URL == "" {
				return apperrors.Validation("url is required")
			}
			return validate.ValidateJob(validate.JobValidationOpts{
				URL:         req.URL,
				MaxDepth:    req.MaxDepth,
				MaxPages:    req.MaxPages,
				Timeout:     req.TimeoutSeconds,
				AuthProfile: req.AuthProfile,
			}, model.KindCrawl)
		},
		buildSpec: func(req CrawlRequest) jobs.JobSpec {
			return jobs.JobSpec{
				Kind:        model.KindCrawl,
				URL:         req.URL,
				MaxDepth:    req.MaxDepth,
				MaxPages:    req.MaxPages,
				Headless:    req.Headless,
				SitemapURL:  req.SitemapURL,
				SitemapOnly: valueOr(req.SitemapOnly, false),
			}
		},
		requestOptions: func(r *http.Request, req CrawlRequest) jobRequestOptions {
			return jobRequestOptions{
				authURL:        req.URL,
				authProfile:    req.AuthProfile,
				auth:           req.Auth,
				extract:        req.Extract,
				pipeline:       req.Pipeline,
				webhook:        req.Webhook,
				screenshot:     req.Screenshot,
				device:         req.Device,
				incremental:    req.Incremental,
				playwright:     req.Playwright,
				timeoutSeconds: req.TimeoutSeconds,
				requestID:      contextRequestID(r.Context()),
			}
		},
	})
}
