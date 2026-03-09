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
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	var req CrawlRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	if req.URL == "" {
		writeError(w, r, apperrors.Validation("url is required"))
		return
	}
	if err := validate.ValidateJob(validate.JobValidationOpts{
		URL:         req.URL,
		MaxDepth:    req.MaxDepth,
		MaxPages:    req.MaxPages,
		Timeout:     req.TimeoutSeconds,
		AuthProfile: req.AuthProfile,
	}, model.KindCrawl); err != nil {
		writeError(w, r, err)
		return
	}

	spec := jobs.JobSpec{
		Kind:        model.KindCrawl,
		URL:         req.URL,
		MaxDepth:    req.MaxDepth,
		MaxPages:    req.MaxPages,
		Headless:    req.Headless,
		SitemapURL:  req.SitemapURL,
		SitemapOnly: valueOr(req.SitemapOnly, false),
	}
	if err := s.applySingleJobDefaults(&spec, jobRequestOptions{
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
	}); err != nil {
		writeError(w, r, err)
		return
	}

	job, err := s.manager.CreateJob(r.Context(), spec)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if err := s.manager.Enqueue(job); err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, model.SanitizeJob(job))
}
