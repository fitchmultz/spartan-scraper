// Package api provides HTTP handlers for job listing and management endpoints.
// Job handlers support listing jobs with filters, retrieving job details,
// canceling jobs, and force-deleting jobs with their artifacts.
package api

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"spartan-scraper/internal/apperrors"
	"spartan-scraper/internal/model"
	"spartan-scraper/internal/store"
)

func (s *Server) handleJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	query := r.URL.Query()
	limit := parseIntParam(query.Get("limit"), 100)
	offset := parseIntParam(query.Get("offset"), 0)
	statusParam := query.Get("status")

	var jobsList []model.Job
	var err error

	if statusParam != "" {
		status := model.Status(statusParam)
		if !status.IsValid() {
			writeError(w, apperrors.Validation(fmt.Sprintf("invalid status: %s (must be queued, running, succeeded, failed, or canceled)", statusParam)))
			return
		}
		opts := store.ListByStatusOptions{Limit: limit, Offset: offset}
		jobsList, err = s.store.ListByStatus(r.Context(), status, opts)
	} else {
		opts := store.ListOptions{Limit: limit, Offset: offset}
		jobsList, err = s.store.ListOpts(r.Context(), opts)
	}

	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, map[string]interface{}{"jobs": jobsList})
}

func (s *Server) handleJob(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if strings.HasSuffix(path, "/results") {
		s.handleJobResults(w, r)
		return
	}
	id := filepath.Base(path)
	if id == "" || id == "jobs" {
		writeJSONError(w, http.StatusBadRequest, "id required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		job, err := s.store.Get(r.Context(), id)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, job)
	case http.MethodDelete:
		if r.URL.Query().Get("force") == "true" {
			if err := s.store.DeleteWithArtifacts(r.Context(), id); err != nil {
				writeError(w, err)
				return
			}
			writeJSON(w, map[string]string{"status": "deleted"})
		} else {
			if err := s.manager.CancelJob(r.Context(), id); err != nil {
				writeError(w, err)
				return
			}
			writeJSON(w, map[string]string{"status": "canceled"})
		}
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
