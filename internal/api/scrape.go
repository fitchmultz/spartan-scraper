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

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/validate"
)

func (s *Server) handleScrape(w http.ResponseWriter, r *http.Request) {
	handleSingleJobSubmission(s, w, r, singleJobSubmission[ScrapeRequest]{
		kind: model.KindScrape,
		validate: func(req ScrapeRequest) error {
			if req.URL == "" {
				return apperrors.Validation("url is required")
			}
			return validate.ValidateJob(validate.JobValidationOpts{
				URL:         req.URL,
				Timeout:     req.TimeoutSeconds,
				AuthProfile: req.AuthProfile,
			}, model.KindScrape)
		},
		buildSpec: func(req ScrapeRequest) jobs.JobSpec {
			spec := jobs.JobSpec{
				Kind:        model.KindScrape,
				URL:         req.URL,
				Method:      req.Method,
				Body:        []byte(req.Body),
				ContentType: req.ContentType,
				Headless:    req.Headless,
			}
			if spec.Method == "" {
				spec.Method = http.MethodGet
			}
			return spec
		},
		requestOptions: func(r *http.Request, req ScrapeRequest) jobRequestOptions {
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
