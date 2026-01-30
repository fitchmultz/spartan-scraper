// Package api provides HTTP handlers for batch job operations.
//
// This file is responsible for:
// - Creating batches of scrape, crawl, and research jobs
// - Retrieving batch status with aggregated statistics
// - Canceling batches and their constituent jobs
//
// This file does NOT handle:
// - Individual job operations (see jobs.go)
// - Batch persistence (see store package)
//
// Invariants:
// - Batch size is validated against MaxBatchSize (default 100)
// - All URLs in a batch are validated before any jobs are created
// - Batch responses include sanitized job data
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/store"
	"github.com/fitchmultz/spartan-scraper/internal/validate"
)

// handleBatchScrape handles POST /v1/jobs/batch/scrape
func (s *Server) handleBatchScrape(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	var req BatchScrapeRequest
	if err := decodeBatchRequest(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}

	// Validate batch size
	if err := validateBatchSize(len(req.Jobs), s.cfg.MaxBatchSize); err != nil {
		writeError(w, r, err)
		return
	}

	// Validate all URLs first
	for i, job := range req.Jobs {
		if err := validate.ValidateURL(job.URL); err != nil {
			writeError(w, r, apperrors.Validation(fmt.Sprintf("invalid URL at index %d: %v", i, err)))
			return
		}
	}

	// Build job specs
	specs := make([]jobs.JobSpec, len(req.Jobs))
	for i, job := range req.Jobs {
		specs[i] = buildScrapeJobSpec(job, req, s.cfg)
	}

	// Create batch
	batchID := jobs.GenerateBatchID()
	createdJobs, err := s.manager.CreateBatchJobs(r.Context(), model.KindScrape, specs, batchID)
	if err != nil {
		writeError(w, r, err)
		return
	}

	// Enqueue all jobs
	if err := s.manager.EnqueueBatch(createdJobs); err != nil {
		writeError(w, r, err)
		return
	}

	// Return response
	resp := BatchResponse{
		ID:        batchID,
		Kind:      string(model.KindScrape),
		Status:    string(model.BatchStatusPending),
		JobCount:  len(createdJobs),
		Jobs:      model.SanitizeJobs(createdJobs),
		CreatedAt: createdJobs[0].CreatedAt,
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, resp)
}

// handleBatchCrawl handles POST /v1/jobs/batch/crawl
func (s *Server) handleBatchCrawl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	var req BatchCrawlRequest
	if err := decodeBatchRequest(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}

	// Validate batch size
	if err := validateBatchSize(len(req.Jobs), s.cfg.MaxBatchSize); err != nil {
		writeError(w, r, err)
		return
	}

	// Validate all URLs first
	for i, job := range req.Jobs {
		if err := validate.ValidateURL(job.URL); err != nil {
			writeError(w, r, apperrors.Validation(fmt.Sprintf("invalid URL at index %d: %v", i, err)))
			return
		}
	}

	// Validate crawl-specific parameters
	if err := validate.ValidateMaxDepth(req.MaxDepth); err != nil {
		writeError(w, r, err)
		return
	}
	if err := validate.ValidateMaxPages(req.MaxPages); err != nil {
		writeError(w, r, err)
		return
	}

	// Build job specs
	specs := make([]jobs.JobSpec, len(req.Jobs))
	for i, job := range req.Jobs {
		specs[i] = buildCrawlJobSpec(job, req, s.cfg)
	}

	// Create batch
	batchID := jobs.GenerateBatchID()
	createdJobs, err := s.manager.CreateBatchJobs(r.Context(), model.KindCrawl, specs, batchID)
	if err != nil {
		writeError(w, r, err)
		return
	}

	// Enqueue all jobs
	if err := s.manager.EnqueueBatch(createdJobs); err != nil {
		writeError(w, r, err)
		return
	}

	// Return response
	resp := BatchResponse{
		ID:        batchID,
		Kind:      string(model.KindCrawl),
		Status:    string(model.BatchStatusPending),
		JobCount:  len(createdJobs),
		Jobs:      model.SanitizeJobs(createdJobs),
		CreatedAt: createdJobs[0].CreatedAt,
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, resp)
}

// handleBatchResearch handles POST /v1/jobs/batch/research
func (s *Server) handleBatchResearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	var req BatchResearchRequest
	if err := decodeBatchRequest(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}

	// Validate batch size
	if err := validateBatchSize(len(req.Jobs), s.cfg.MaxBatchSize); err != nil {
		writeError(w, r, err)
		return
	}

	// Validate all URLs first
	for i, job := range req.Jobs {
		if err := validate.ValidateURL(job.URL); err != nil {
			writeError(w, r, apperrors.Validation(fmt.Sprintf("invalid URL at index %d: %v", i, err)))
			return
		}
	}

	// Validate research-specific parameters
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

	// Build job specs
	specs := make([]jobs.JobSpec, len(req.Jobs))
	for i, job := range req.Jobs {
		specs[i] = buildResearchJobSpec(job, req, s.cfg)
	}

	// Create batch
	batchID := jobs.GenerateBatchID()
	createdJobs, err := s.manager.CreateBatchJobs(r.Context(), model.KindResearch, specs, batchID)
	if err != nil {
		writeError(w, r, err)
		return
	}

	// Enqueue all jobs
	if err := s.manager.EnqueueBatch(createdJobs); err != nil {
		writeError(w, r, err)
		return
	}

	// Return response
	resp := BatchResponse{
		ID:        batchID,
		Kind:      string(model.KindResearch),
		Status:    string(model.BatchStatusPending),
		JobCount:  len(createdJobs),
		Jobs:      model.SanitizeJobs(createdJobs),
		CreatedAt: createdJobs[0].CreatedAt,
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, resp)
}

// handleBatchGet handles GET /v1/jobs/batch/{id} and DELETE /v1/jobs/batch/{id}
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

// handleBatchGetStatus handles GET /v1/jobs/batch/{id}
func (s *Server) handleBatchGetStatus(w http.ResponseWriter, r *http.Request) {
	batchID := extractID(r.URL.Path, "batch")
	if batchID == "" {
		writeError(w, r, apperrors.Validation("batch ID required"))
		return
	}

	// Get batch status with stats
	batch, stats, err := s.manager.GetBatchStatus(r.Context(), batchID)
	if err != nil {
		writeError(w, r, err)
		return
	}

	// Check if jobs should be included
	includeJobs := r.URL.Query().Get("include_jobs") == "true"

	resp := BatchStatusResponse{
		ID:        batch.ID,
		Kind:      string(batch.Kind),
		Status:    string(batch.Status),
		JobCount:  batch.JobCount,
		Stats:     stats,
		CreatedAt: batch.CreatedAt,
		UpdatedAt: batch.UpdatedAt,
	}

	if includeJobs {
		opts := store.ListOptions{
			Limit:  parseIntParam(r.URL.Query().Get("limit"), 50),
			Offset: parseIntParam(r.URL.Query().Get("offset"), 0),
		}
		jobs, err := s.store.ListJobsByBatch(r.Context(), batchID, opts)
		if err != nil {
			writeError(w, r, err)
			return
		}
		resp.Jobs = model.SanitizeJobs(jobs)
	}

	writeJSON(w, resp)
}

// handleBatchCancel handles DELETE /v1/jobs/batch/{id}
func (s *Server) handleBatchCancel(w http.ResponseWriter, r *http.Request) {
	batchID := extractID(r.URL.Path, "batch")
	if batchID == "" {
		writeError(w, r, apperrors.Validation("batch ID required"))
		return
	}

	// Cancel all jobs in the batch
	_, err := s.manager.CancelBatch(r.Context(), batchID)
	if err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// decodeBatchRequest decodes a JSON request body into the provided struct.
func decodeBatchRequest(w http.ResponseWriter, r *http.Request, v interface{}) error {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		return apperrors.UnsupportedMediaType("content-type must be application/json")
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		return apperrors.Validation(err.Error())
	}
	return nil
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

// buildScrapeJobSpec builds a JobSpec from a BatchJobRequest and BatchScrapeRequest.
func buildScrapeJobSpec(job BatchJobRequest, req BatchScrapeRequest, cfg config.Config) jobs.JobSpec {
	spec := jobs.JobSpec{
		Kind:           model.KindScrape,
		URL:            job.URL,
		Method:         job.Method,
		Body:           []byte(job.Body),
		ContentType:    job.ContentType,
		Headless:       req.Headless,
		TimeoutSeconds: req.TimeoutSeconds,
	}

	if req.Playwright != nil {
		spec.UsePlaywright = *req.Playwright
	} else {
		spec.UsePlaywright = cfg.UsePlaywright
	}

	if req.Auth != nil {
		spec.Auth = *req.Auth
	}
	if req.Extract != nil {
		spec.Extract = *req.Extract
	}
	if req.Pipeline != nil {
		spec.Pipeline = *req.Pipeline
	}
	if req.Incremental != nil {
		spec.Incremental = *req.Incremental
	}
	if req.Webhook != nil {
		spec.WebhookURL = req.Webhook.URL
		spec.WebhookEvents = req.Webhook.Events
		spec.WebhookSecret = req.Webhook.Secret
	}
	if req.Screenshot != nil {
		spec.Screenshot = req.Screenshot
	}
	if req.Device != nil {
		spec.Device = req.Device
	}

	return spec
}

// buildCrawlJobSpec builds a JobSpec from a BatchJobRequest and BatchCrawlRequest.
func buildCrawlJobSpec(job BatchJobRequest, req BatchCrawlRequest, cfg config.Config) jobs.JobSpec {
	spec := jobs.JobSpec{
		Kind:           model.KindCrawl,
		URL:            job.URL,
		MaxDepth:       req.MaxDepth,
		MaxPages:       req.MaxPages,
		Headless:       req.Headless,
		TimeoutSeconds: req.TimeoutSeconds,
		SitemapURL:     req.SitemapURL,
	}

	if req.Playwright != nil {
		spec.UsePlaywright = *req.Playwright
	} else {
		spec.UsePlaywright = cfg.UsePlaywright
	}
	if req.SitemapOnly != nil {
		spec.SitemapOnly = *req.SitemapOnly
	}
	if req.Incremental != nil {
		spec.Incremental = *req.Incremental
	}
	if req.Auth != nil {
		spec.Auth = *req.Auth
	}
	if req.Extract != nil {
		spec.Extract = *req.Extract
	}
	if req.Pipeline != nil {
		spec.Pipeline = *req.Pipeline
	}
	if req.Webhook != nil {
		spec.WebhookURL = req.Webhook.URL
		spec.WebhookEvents = req.Webhook.Events
		spec.WebhookSecret = req.Webhook.Secret
	}
	if req.Screenshot != nil {
		spec.Screenshot = req.Screenshot
	}
	if req.Device != nil {
		spec.Device = req.Device
	}

	return spec
}

// buildResearchJobSpec builds a JobSpec from a BatchJobRequest and BatchResearchRequest.
func buildResearchJobSpec(_ BatchJobRequest, req BatchResearchRequest, cfg config.Config) jobs.JobSpec {
	// For research jobs, we use the URLs from the batch jobs as the research URLs
	researchURLs := make([]string, 0, len(req.Jobs))
	for _, j := range req.Jobs {
		researchURLs = append(researchURLs, j.URL)
	}

	spec := jobs.JobSpec{
		Kind:           model.KindResearch,
		Query:          req.Query,
		URLs:           researchURLs,
		MaxDepth:       req.MaxDepth,
		MaxPages:       req.MaxPages,
		Headless:       req.Headless,
		TimeoutSeconds: req.TimeoutSeconds,
	}

	if req.Playwright != nil {
		spec.UsePlaywright = *req.Playwright
	} else {
		spec.UsePlaywright = cfg.UsePlaywright
	}
	if req.Auth != nil {
		spec.Auth = *req.Auth
	}
	if req.Extract != nil {
		spec.Extract = *req.Extract
	}
	if req.Pipeline != nil {
		spec.Pipeline = *req.Pipeline
	}
	if req.Webhook != nil {
		spec.WebhookURL = req.Webhook.URL
		spec.WebhookEvents = req.Webhook.Events
		spec.WebhookSecret = req.Webhook.Secret
	}
	if req.Screenshot != nil {
		spec.Screenshot = req.Screenshot
	}
	if req.Device != nil {
		spec.Device = req.Device
	}

	return spec
}
