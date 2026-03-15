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

	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func (s *Server) handleScrape(w http.ResponseWriter, r *http.Request) {
	handleSingleJobSubmission(s, w, r, singleJobSubmission[ScrapeRequest]{
		kind:      model.KindScrape,
		validate:  validateScrapeRequest,
		buildSpec: scrapeJobSpecFromRequest,
		requestOptions: func(r *http.Request, req ScrapeRequest) jobRequestOptions {
			return scrapeJobRequestOptions(contextRequestID(r.Context()), req)
		},
	})
}
