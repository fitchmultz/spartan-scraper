// Package api provides HTTP handlers for health check endpoints.
// The health handler checks database connectivity, queue status, and browser availability.
package api

import (
	"context"
	"net/http"

	"github.com/fitchmultz/spartan-scraper/internal/buildinfo"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
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

	res.Components["ai"] = s.aiHealthStatus(ctx)
	if res.Components["ai"].Status == "error" {
		healthy = false
	}

	if !healthy {
		res.Status = "error"
		writeJSONStatus(w, http.StatusServiceUnavailable, res)
	} else {
		writeJSONStatus(w, http.StatusOK, res)
	}
}

func (s *Server) aiHealthStatus(ctx context.Context) ComponentStatus {
	if !extract.IsAIEnabled(s.cfg.AI) {
		return ComponentStatus{
			Status: "disabled",
			Details: map[string]interface{}{
				"enabled": false,
			},
		}
	}

	if s.aiExtractor == nil {
		return ComponentStatus{
			Status:  "error",
			Message: "AI extractor failed to initialize",
			Details: map[string]interface{}{
				"enabled": true,
				"mode":    s.cfg.AI.Mode,
			},
		}
	}

	status := ComponentStatus{
		Status: "ok",
		Details: map[string]interface{}{
			"enabled": true,
			"mode":    s.cfg.AI.Mode,
		},
	}
	if err := s.aiExtractor.HealthCheck(ctx); err != nil {
		status.Status = "error"
		status.Message = err.Error()
	}
	return status
}
