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

	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func (s *Server) handleCrawl(w http.ResponseWriter, r *http.Request) {
	handleSingleJobSubmission(s, w, r, singleJobSubmission[CrawlRequest]{
		kind:      model.KindCrawl,
		validate:  validateCrawlRequest,
		buildSpec: crawlJobSpecFromRequest,
		requestOptions: func(r *http.Request, req CrawlRequest) jobRequestOptions {
			return crawlJobRequestOptions(contextRequestID(r.Context()), req)
		},
	})
}
