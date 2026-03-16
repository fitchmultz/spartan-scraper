// Package api provides HTTP handlers for scrape job endpoints.
//
// Purpose:
// - Accept single-page scrape submissions over the API.
//
// Responsibilities:
// - Validate scrape requests.
// - Build a scrape JobSpec with shared request defaults.
// - Create and enqueue the job, then return the sanitized record.
//
// Scope:
// - Scrape request handling only; job execution lives in internal/jobs.
//
// Usage:
// - Registered for POST /v1/scrape.
//
// Invariants/Assumptions:
// - Requests must be JSON and include a URL.
// - Method defaults to GET when omitted.
package api

import (
	"net/http"

	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/submission"
)

func (s *Server) handleScrape(w http.ResponseWriter, r *http.Request) {
	handleSingleJobSubmission(s, w, r, singleJobSubmission[ScrapeRequest]{
		buildSpec: func(r *http.Request, req ScrapeRequest) (jobs.JobSpec, error) {
			return submission.JobSpecFromScrapeRequest(s.cfg, s.requestSubmissionDefaults(r), req)
		},
	})
}
