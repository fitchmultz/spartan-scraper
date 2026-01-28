// Package api provides HTTP handlers for health check endpoints.
// The health handler checks database connectivity, queue status, and browser availability.
package api

import (
	"net/http"

	"github.com/fitchmultz/spartan-scraper/internal/buildinfo"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	res := HealthResponse{
		Status:     "ok",
		Version:    buildinfo.Version,
		Components: make(map[string]ComponentStatus),
	}
	healthy := true

	dbStatus := ComponentStatus{Status: "ok"}
	if err := s.store.Ping(ctx); err != nil {
		dbStatus.Status = "error"
		dbStatus.Message = err.Error()
		healthy = false
	}
	res.Components["database"] = dbStatus

	qStatus := s.manager.Status()
	res.Components["queue"] = ComponentStatus{
		Status: "ok",
		Details: map[string]int{
			"queued": qStatus.QueuedJobs,
			"active": qStatus.ActiveJobs,
		},
	}

	browserStatus := ComponentStatus{Status: "ok"}
	usePlaywright := s.cfg.UsePlaywright
	if err := fetch.CheckBrowserAvailability(usePlaywright); err != nil {
		browserStatus.Status = "error"
		browserStatus.Message = err.Error()
	}
	res.Components["browser"] = browserStatus

	// Ensure Content-Type is set before WriteHeader (headers are committed on WriteHeader/Write).
	w.Header().Set("Content-Type", "application/json")

	if !healthy {
		res.Status = "error"
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	writeJSON(w, res)
}
