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

	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/submission"
)

func (s *Server) handleCrawl(w http.ResponseWriter, r *http.Request) {
	handleSingleJobSubmission(s, w, r, singleJobSubmission[CrawlRequest]{
		buildSpec: func(r *http.Request, req CrawlRequest) (jobs.JobSpec, error) {
			return submission.JobSpecFromCrawlRequest(s.cfg, s.requestSubmissionDefaults(r), req)
		},
	})
}
