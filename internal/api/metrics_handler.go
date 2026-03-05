// Package api provides HTTP handlers for metrics endpoints.
// The metrics handler exposes performance and rate limiting data for monitoring.
package api

import (
	"net/http"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

// handleMetrics returns the current performance metrics
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	snapshot := s.metricsCollector.GetSnapshot()
	writeJSON(w, snapshot)
}
