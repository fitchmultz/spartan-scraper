// Package api provides HTTP handlers for batch job operations.
//
// Purpose:
// - Accept batch submissions and batch-status operations over the API.
//
// Responsibilities:
// - List persisted batches with pagination metadata.
// - Validate and submit scrape, crawl, and research batches.
// - Return aggregate batch status and optionally paginated jobs.
// - Cancel batches and their constituent jobs.
//
// Scope:
// - API request handling only; persistence and job execution live in internal/store and internal/jobs.
//
// Usage:
// - Registered for /v1/jobs/batch and /v1/jobs/batch/* routes.
//
// Invariants/Assumptions:
// - All batch requests are JSON and validated before any jobs are created.
// - Batch list responses expose aggregate summaries only; batch detail loads individual jobs separately.
// - Research batches create a single research job containing all submitted URLs.
package api

import (
	"net/http"
	"strconv"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/store"
	"github.com/fitchmultz/spartan-scraper/internal/submission"
)

// handleBatches handles GET /v1/jobs/batch.
func (s *Server) handleBatches(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	page, err := parsePageParams(r, 100, 0)
	if err != nil {
		writeError(w, r, err)
		return
	}

	batches, stats, total, err := s.manager.ListBatchStatuses(r.Context(), store.ListOptions{
		Limit:  page.Limit,
		Offset: page.Offset,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	writeJSON(w, BuildBatchListResponse(batches, stats, total, page.Limit, page.Offset))
}

// handleBatchScrape handles POST /v1/jobs/batch/scrape.
func (s *Server) handleBatchScrape(w http.ResponseWriter, r *http.Request) {
	handleBatchJobSubmission(s, w, r, batchJobSubmission[BatchScrapeRequest]{
		kind: model.KindScrape,
		buildSpecs: func(r *http.Request, req BatchScrapeRequest) ([]jobs.JobSpec, error) {
			return submission.JobSpecsFromBatchScrapeRequest(s.cfg, s.requestBatchDefaults(r), req)
		},
	})
}

// handleBatchCrawl handles POST /v1/jobs/batch/crawl.
func (s *Server) handleBatchCrawl(w http.ResponseWriter, r *http.Request) {
	handleBatchJobSubmission(s, w, r, batchJobSubmission[BatchCrawlRequest]{
		kind: model.KindCrawl,
		buildSpecs: func(r *http.Request, req BatchCrawlRequest) ([]jobs.JobSpec, error) {
			return submission.JobSpecsFromBatchCrawlRequest(s.cfg, s.requestBatchDefaults(r), req)
		},
	})
}

// handleBatchResearch handles POST /v1/jobs/batch/research.
func (s *Server) handleBatchResearch(w http.ResponseWriter, r *http.Request) {
	handleBatchJobSubmission(s, w, r, batchJobSubmission[BatchResearchRequest]{
		kind: model.KindResearch,
		buildSpecs: func(r *http.Request, req BatchResearchRequest) ([]jobs.JobSpec, error) {
			return submission.JobSpecsFromBatchResearchRequest(s.cfg, s.requestBatchDefaults(r), req)
		},
	})
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

	resp := BuildBatchResponse(batch, stats, nil, batch.JobCount, 0, 0)

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
		resp = BuildBatchResponse(batch, stats, jobsByBatch, batch.JobCount, page.Limit, page.Offset)
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

	batch, stats, err := s.manager.GetBatchStatus(r.Context(), batchID)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, BuildBatchResponse(batch, stats, nil, batch.JobCount, 0, 0))
}
