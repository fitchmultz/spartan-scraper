// Package api provides HTTP handlers for job listing and management endpoints.
// Job handlers support listing jobs with filters, retrieving job details,
// canceling jobs, and force-deleting jobs with their artifacts.
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

	var jobsList []model.Job
	var total int
	var status model.Status

	if statusParam != "" {
		status = model.Status(statusParam)
		if !status.IsValid() {
			writeError(w, r, apperrors.Validation(fmt.Sprintf("invalid status: %s (must be queued, running, succeeded, failed, or canceled)", statusParam)))
			return
		}
		opts := store.ListByStatusOptions{Limit: page.Limit, Offset: page.Offset}
		jobsList, err = s.store.ListByStatus(r.Context(), status, opts)
	} else {
		opts := store.ListOptions{Limit: page.Limit, Offset: page.Offset}
		jobsList, err = s.store.ListOpts(r.Context(), opts)
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

	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	writeJSON(w, BuildJobListResponse(jobsList, total, page.Limit, page.Offset))
}

func (s *Server) handleJob(w http.ResponseWriter, r *http.Request) {
	if handlePathSuffix(r.URL.Path, "/results", func() {
		s.handleJobResults(w, r)
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
		writeJSON(w, BuildJobResponse(job))
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
			writeJSON(w, BuildJobResponse(job))
		}
	default:
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
	}
}
