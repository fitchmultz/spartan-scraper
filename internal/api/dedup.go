// Package api provides deduplication API endpoints for the Spartan Scraper.
//
// This file implements REST endpoints for querying content fingerprints
// and finding duplicate content across jobs.
//
// Endpoints:
// - GET /v1/dedup/duplicates - Find duplicate content by simhash
// - GET /v1/dedup/history - Get content history for a URL
// - GET /v1/dedup/stats - Get deduplication statistics
// - DELETE /v1/dedup/job/{jobId} - Remove all dedup entries for a job
//
// These endpoints enable clients to:
// - Detect duplicate content before crawling
// - Analyze content history across jobs
// - Monitor deduplication statistics
// - Clean up dedup data for deleted jobs
package api

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

// handleDedupRoutes routes dedup-related requests to the appropriate handler.
func (s *Server) handleDedupRoutes(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case path == "/v1/dedup/duplicates":
		s.handleDedupDuplicates(w, r)
	case path == "/v1/dedup/history":
		s.handleDedupHistory(w, r)
	case path == "/v1/dedup/stats":
		s.handleDedupStats(w, r)
	case len(path) > len("/v1/dedup/job/") && path[:len("/v1/dedup/job/")] == "/v1/dedup/job/":
		s.handleDedupJobDelete(w, r)
	default:
		writeError(w, r, apperrors.NotFound("endpoint not found"))
	}
}

// handleDedupDuplicates handles GET /v1/dedup/duplicates
// Query params:
//   - simhash (required): The simhash value to find duplicates for
//   - threshold (optional): Hamming distance threshold (default: 3)
//
// Returns: []dedup.DuplicateMatch
func (s *Server) handleDedupDuplicates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	if s.store == nil {
		writeError(w, r, apperrors.Internal("store not initialized"))
		return
	}

	// Parse simhash parameter
	simhashStr := r.URL.Query().Get("simhash")
	if simhashStr == "" {
		writeError(w, r, apperrors.Validation("simhash parameter is required"))
		return
	}

	simhashVal, err := strconv.ParseUint(simhashStr, 10, 64)
	if err != nil {
		writeError(w, r, apperrors.Validation("invalid simhash value: must be a non-negative integer"))
		return
	}

	// Parse threshold parameter (optional, default: 3)
	threshold := 3
	thresholdStr := r.URL.Query().Get("threshold")
	if thresholdStr != "" {
		parsedThreshold, err := strconv.Atoi(thresholdStr)
		if err != nil || parsedThreshold < 0 {
			writeError(w, r, apperrors.Validation("invalid threshold value: must be a non-negative integer"))
			return
		}
		threshold = parsedThreshold
	}

	// Get content index from store
	contentIndex := s.store.GetContentIndex()
	if contentIndex == nil {
		writeError(w, r, apperrors.Internal("content index not initialized"))
		return
	}

	matches, err := contentIndex.FindDuplicates(r.Context(), simhashVal, threshold)
	if err != nil {
		writeError(w, r, apperrors.Wrap(apperrors.KindInternal, "failed to find duplicates", err))
		return
	}

	writeJSON(w, matches)
}

// handleDedupHistory handles GET /v1/dedup/history
// Query params:
//   - url (required): The URL to get content history for
//
// Returns: []dedup.ContentEntry
func (s *Server) handleDedupHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	if s.store == nil {
		writeError(w, r, apperrors.Internal("store not initialized"))
		return
	}

	// Parse URL parameter
	url := r.URL.Query().Get("url")
	if url == "" {
		writeError(w, r, apperrors.Validation("url parameter is required"))
		return
	}

	// Get content index from store
	contentIndex := s.store.GetContentIndex()
	if contentIndex == nil {
		writeError(w, r, apperrors.Internal("content index not initialized"))
		return
	}

	entries, err := contentIndex.GetContentHistory(r.Context(), url)
	if err != nil {
		writeError(w, r, apperrors.Wrap(apperrors.KindInternal, "failed to get content history", err))
		return
	}

	writeJSON(w, entries)
}

// handleDedupStats handles GET /v1/dedup/stats
// Returns: dedup.Stats
func (s *Server) handleDedupStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	if s.store == nil {
		writeError(w, r, apperrors.Internal("store not initialized"))
		return
	}

	// Get content index from store
	contentIndex := s.store.GetContentIndex()
	if contentIndex == nil {
		writeError(w, r, apperrors.Internal("content index not initialized"))
		return
	}

	stats, err := contentIndex.Stats(r.Context())
	if err != nil {
		writeError(w, r, apperrors.Wrap(apperrors.KindInternal, "failed to get dedup stats", err))
		return
	}

	writeJSON(w, stats)
}

// handleDedupJobDelete handles DELETE /v1/dedup/job/{jobId}
// Removes all dedup entries for a specific job.
// Returns: 204 No Content on success
func (s *Server) handleDedupJobDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	if s.store == nil {
		writeError(w, r, apperrors.Internal("store not initialized"))
		return
	}

	// Extract job ID from path
	jobID := extractID(r.URL.Path, "job")
	if jobID == "" {
		writeError(w, r, apperrors.Validation("job ID is required"))
		return
	}

	// Validate job ID format
	if err := validateJobID(jobID); err != nil {
		writeError(w, r, err)
		return
	}

	// Get content index from store
	contentIndex := s.store.GetContentIndex()
	if contentIndex == nil {
		writeError(w, r, apperrors.Internal("content index not initialized"))
		return
	}

	deleted, err := contentIndex.DeleteJobEntries(r.Context(), jobID)
	if err != nil {
		writeError(w, r, apperrors.Wrap(apperrors.KindInternal, "failed to delete job entries", err))
		return
	}

	// Return 204 No Content on success
	w.WriteHeader(http.StatusNoContent)
	slog.Info("deleted dedup entries for job", "jobID", jobID, "count", deleted)
}
