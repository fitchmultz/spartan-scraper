// Package api provides HTTP handlers for job listing and management endpoints.
//
// Purpose:
// - Expose canonical recent-run inspection and job control routes over HTTP.
//
// Responsibilities:
// - List persisted jobs with pagination and optional status filtering.
// - List recent failed jobs with operator-meaningful failure context.
// - Retrieve single job details and cancel or force-delete jobs.
// - Route all job responses through the shared sanitized observability builders.
//
// Scope:
// - Job transport handling only; persistence and execution stay in internal/store and internal/jobs.
//
// Usage:
// - Registered for /v1/jobs, /v1/jobs/failures, and /v1/jobs/* routes.
//
// Invariants/Assumptions:
// - Job list responses always return paginated envelopes.
// - Failure inspection is a filtered view over persisted failed jobs, not a separate history store.
// - Force delete permanently removes persisted job data and artifacts.
package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

func (s *Server) handleJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	query := r.URL.Query()
	page, err := parsePageParams(r, 100, 0)
	if err != nil {
		writeError(w, r, err)
		return
	}

	statusParam := query.Get("status")
	var (
		jobsList []model.Job
		total    int
		status   model.Status
	)

	if statusParam != "" {
		status = model.Status(statusParam)
		if !status.IsValid() {
			writeError(w, r, apperrors.Validation(fmt.Sprintf("invalid status: %s (must be queued, running, succeeded, failed, or canceled)", statusParam)))
			return
		}
		jobsList, err = s.store.ListByStatus(r.Context(), status, store.ListByStatusOptions{
			Limit:  page.Limit,
			Offset: page.Offset,
		})
	} else {
		jobsList, err = s.store.ListOpts(r.Context(), store.ListOptions{
			Limit:  page.Limit,
			Offset: page.Offset,
		})
	}
	if err != nil {
		writeError(w, r, err)
		return
	}

	total, err = s.store.CountJobs(r.Context(), status)
	if err != nil {
		writeError(w, r, err)
		return
	}

	resp, err := BuildStoreBackedJobListResponse(r.Context(), s.store, jobsList, total, page.Limit, page.Offset)
	if err != nil {
		writeError(w, r, err)
		return
	}

	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	writeJSON(w, resp)
}

func (s *Server) handleJobFailures(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	page, err := parsePageParams(r, 50, 0)
	if err != nil {
		writeError(w, r, err)
		return
	}

	jobsList, err := s.store.ListByStatus(r.Context(), model.StatusFailed, store.ListByStatusOptions{
		Limit:  page.Limit,
		Offset: page.Offset,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	total, err := s.store.CountJobs(r.Context(), model.StatusFailed)
	if err != nil {
		writeError(w, r, err)
		return
	}

	resp, err := BuildStoreBackedJobListResponse(r.Context(), s.store, jobsList, total, page.Limit, page.Offset)
	if err != nil {
		writeError(w, r, err)
		return
	}

	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	writeJSON(w, resp)
}

func (s *Server) handleJob(w http.ResponseWriter, r *http.Request) {
	if handlePathSuffix(r.URL.Path, "/results", func() {
		s.handleJobResults(w, r)
	}) {
		return
	}
	if handlePathSuffix(r.URL.Path, "/exports", func() {
		s.handleJobExportHistory(w, r)
	}) {
		return
	}
	if handlePathSuffix(r.URL.Path, "/export", func() {
		s.handleJobExport(w, r)
	}) {
		return
	}
	if handlePathSuffix(r.URL.Path, "/preview-transform", func() {
		s.handlePreviewTransform(w, r)
	}) {
		return
	}
	id, err := requireJobID(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	switch r.Method {
	case http.MethodGet:
		job, err := s.store.Get(r.Context(), id)
		if err != nil {
			writeError(w, r, err)
			return
		}
		resp, err := BuildStoreBackedJobResponse(r.Context(), s.store, job)
		if err != nil {
			writeError(w, r, err)
			return
		}
		writeJSON(w, resp)
	case http.MethodDelete:
		if r.URL.Query().Get("force") == "true" {
			if err := s.store.DeleteWithArtifacts(r.Context(), id); err != nil {
				writeError(w, r, err)
				return
			}
			writeStatusJSON(w, "deleted")
		} else {
			if err := s.manager.CancelJob(r.Context(), id); err != nil {
				writeError(w, r, err)
				return
			}
			job, err := s.store.Get(r.Context(), id)
			if err != nil {
				writeError(w, r, err)
				return
			}
			resp, err := BuildStoreBackedJobResponse(r.Context(), s.store, job)
			if err != nil {
				writeError(w, r, err)
				return
			}
			writeJSON(w, resp)
		}
	default:
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
	}
}
