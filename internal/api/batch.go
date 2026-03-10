// Package api provides HTTP handlers for batch job operations.
//
// Purpose:
// - Accept batch submissions and batch-status operations over the API.
//
// Responsibilities:
// - Validate and submit scrape, crawl, and research batches.
// - Return aggregate batch status and optionally paginated jobs.
// - Cancel batches and their constituent jobs.
//
// Scope:
// - API request handling only; persistence and job execution live in internal/store and internal/jobs.
//
// Usage:
// - Registered for /v1/jobs/batch/* routes.
//
// Invariants/Assumptions:
// - All batch requests are JSON and validated before any jobs are created.
// - Research batches create a single research job containing all submitted URLs.
package api

import (
	"fmt"
	"net/http"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/store"
	"github.com/fitchmultz/spartan-scraper/internal/validate"
)

// handleBatchScrape handles POST /v1/jobs/batch/scrape.
func (s *Server) handleBatchScrape(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	var req BatchScrapeRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	if err := validateBatchSize(len(req.Jobs), s.cfg.MaxBatchSize); err != nil {
		writeError(w, r, err)
		return
	}
	if err := validateBatchURLs(req.Jobs); err != nil {
		writeError(w, r, err)
		return
	}

	specs := make([]jobs.JobSpec, len(req.Jobs))
	requestID := contextRequestID(r.Context())
	for i, job := range req.Jobs {
		specs[i] = jobs.JobSpec{
			Kind:        model.KindScrape,
			URL:         job.URL,
			Method:      job.Method,
			Body:        []byte(job.Body),
			ContentType: job.ContentType,
			Headless:    req.Headless,
		}
		if specs[i].Method == "" {
			specs[i].Method = http.MethodGet
		}
		if err := s.applyBatchJobDefaults(&specs[i], jobRequestOptions{
			authURL:        job.URL,
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
			requestID:      requestID,
		}); err != nil {
			writeError(w, r, err)
			return
		}
	}

	resp, err := s.createAndEnqueueBatch(r.Context(), model.KindScrape, specs)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeCreatedJSON(w, resp)
}

// handleBatchCrawl handles POST /v1/jobs/batch/crawl.
func (s *Server) handleBatchCrawl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	var req BatchCrawlRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	if err := validateBatchSize(len(req.Jobs), s.cfg.MaxBatchSize); err != nil {
		writeError(w, r, err)
		return
	}
	if err := validateBatchURLs(req.Jobs); err != nil {
		writeError(w, r, err)
		return
	}
	if err := validate.ValidateMaxDepth(req.MaxDepth); err != nil {
		writeError(w, r, err)
		return
	}
	if err := validate.ValidateMaxPages(req.MaxPages); err != nil {
		writeError(w, r, err)
		return
	}

	specs := make([]jobs.JobSpec, len(req.Jobs))
	requestID := contextRequestID(r.Context())
	for i, job := range req.Jobs {
		specs[i] = jobs.JobSpec{
			Kind:        model.KindCrawl,
			URL:         job.URL,
			MaxDepth:    req.MaxDepth,
			MaxPages:    req.MaxPages,
			Headless:    req.Headless,
			SitemapURL:  req.SitemapURL,
			SitemapOnly: valueOr(req.SitemapOnly, false),
		}
		if err := s.applyBatchJobDefaults(&specs[i], jobRequestOptions{
			authURL:        job.URL,
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
			requestID:      requestID,
		}); err != nil {
			writeError(w, r, err)
			return
		}
	}

	resp, err := s.createAndEnqueueBatch(r.Context(), model.KindCrawl, specs)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeCreatedJSON(w, resp)
}

// handleBatchResearch handles POST /v1/jobs/batch/research.
func (s *Server) handleBatchResearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	var req BatchResearchRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	if err := validateBatchSize(len(req.Jobs), s.cfg.MaxBatchSize); err != nil {
		writeError(w, r, err)
		return
	}
	if err := validateBatchURLs(req.Jobs); err != nil {
		writeError(w, r, err)
		return
	}
	if req.Query == "" {
		writeError(w, r, apperrors.Validation("query is required for research jobs"))
		return
	}
	if err := validate.ValidateMaxDepth(req.MaxDepth); err != nil {
		writeError(w, r, err)
		return
	}
	if err := validate.ValidateMaxPages(req.MaxPages); err != nil {
		writeError(w, r, err)
		return
	}

	researchURLs := make([]string, len(req.Jobs))
	for i, job := range req.Jobs {
		researchURLs[i] = job.URL
	}

	spec := jobs.JobSpec{
		Kind:     model.KindResearch,
		Query:    req.Query,
		URLs:     researchURLs,
		MaxDepth: req.MaxDepth,
		MaxPages: req.MaxPages,
		Headless: req.Headless,
	}
	if err := s.applyBatchJobDefaults(&spec, jobRequestOptions{
		authURL:        researchURLs[0],
		authProfile:    req.AuthProfile,
		auth:           req.Auth,
		extract:        req.Extract,
		pipeline:       req.Pipeline,
		webhook:        req.Webhook,
		screenshot:     req.Screenshot,
		device:         req.Device,
		playwright:     req.Playwright,
		timeoutSeconds: req.TimeoutSeconds,
		requestID:      contextRequestID(r.Context()),
	}); err != nil {
		writeError(w, r, err)
		return
	}

	resp, err := s.createAndEnqueueBatch(r.Context(), model.KindResearch, []jobs.JobSpec{spec})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeCreatedJSON(w, resp)
}

// handleBatchGet handles GET /v1/jobs/batch/{id} and DELETE /v1/jobs/batch/{id}.
func (s *Server) handleBatchGet(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleBatchGetStatus(w, r)
	case http.MethodDelete:
		s.handleBatchCancel(w, r)
	default:
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
	}
}

// handleBatchGetStatus handles GET /v1/jobs/batch/{id}.
func (s *Server) handleBatchGetStatus(w http.ResponseWriter, r *http.Request) {
	batchID, err := requireResourceID(r, "batch", "batch id")
	if err != nil {
		writeError(w, r, err)
		return
	}

	batch, stats, err := s.manager.GetBatchStatus(r.Context(), batchID)
	if err != nil {
		writeError(w, r, err)
		return
	}

	resp := BatchStatusResponse{
		ID:        batch.ID,
		Kind:      string(batch.Kind),
		Status:    string(batch.Status),
		JobCount:  batch.JobCount,
		Stats:     stats,
		CreatedAt: batch.CreatedAt,
		UpdatedAt: batch.UpdatedAt,
	}

	if r.URL.Query().Get("include_jobs") == "true" {
		page, err := parsePageParams(r, 50, 0)
		if err != nil {
			writeError(w, r, err)
			return
		}
		jobsByBatch, err := s.store.ListJobsByBatch(r.Context(), batchID, store.ListOptions{
			Limit:  page.Limit,
			Offset: page.Offset,
		})
		if err != nil {
			writeError(w, r, err)
			return
		}
		resp.Jobs = model.SanitizeJobs(jobsByBatch)
	}

	writeJSON(w, resp)
}

// handleBatchCancel handles DELETE /v1/jobs/batch/{id}.
func (s *Server) handleBatchCancel(w http.ResponseWriter, r *http.Request) {
	batchID, err := requireResourceID(r, "batch", "batch id")
	if err != nil {
		writeError(w, r, err)
		return
	}

	if _, err := s.manager.CancelBatch(r.Context(), batchID); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// validateBatchSize validates that the batch size is within limits.
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
