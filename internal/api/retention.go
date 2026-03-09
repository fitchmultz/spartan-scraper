// Package api provides HTTP handlers for retention management endpoints.
//
// This file is responsible for:
// - Providing retention status information
// - Running manual cleanup operations with dry-run support
//
// This file does NOT handle:
// - Scheduled cleanup execution (scheduler handles this)
// - Policy evaluation logic (retention package handles this)
//
// Invariants:
// - Cleanup defaults to dry-run mode for safety
// - Force flag can override disabled retention
// - All errors use apperrors package for consistent handling
package api

import (
	"net/http"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/retention"
)

// handleRetention routes requests to /v1/retention/status and /v1/retention/cleanup
func (s *Server) handleRetention(w http.ResponseWriter, r *http.Request) {
	// Extract the subpath after /v1/retention/
	path := r.URL.Path
	switch {
	case path == "/v1/retention/status":
		s.handleRetentionStatus(w, r)
	case path == "/v1/retention/cleanup":
		s.handleRetentionCleanup(w, r)
	default:
		writeError(w, r, apperrors.NotFound("endpoint not found"))
	}
}

// handleRetentionStatus handles GET /v1/retention/status
func (s *Server) handleRetentionStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	engine := retention.NewEngine(s.store, s.cfg)
	status, err := engine.GetStatus(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}

	resp := RetentionStatusResponse{
		Enabled:          status.Enabled,
		JobRetentionDays: status.JobRetentionDays,
		CrawlStateDays:   status.CrawlStateDays,
		MaxJobs:          status.MaxJobs,
		MaxStorageGB:     status.MaxStorageGB,
		TotalJobs:        status.TotalJobs,
		JobsEligible:     status.JobsEligible,
		StorageUsedMB:    status.StorageUsedMB,
	}

	writeJSON(w, resp)
}

// handleRetentionCleanup handles POST /v1/retention/cleanup
func (s *Server) handleRetentionCleanup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	var req RetentionCleanupRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}

	// Build cleanup options
	opts := retention.CleanupOptions{
		DryRun: req.DryRun,
		Force:  req.Force,
	}

	if req.OlderThan != nil && *req.OlderThan > 0 {
		cutoff := time.Now().AddDate(0, 0, -*req.OlderThan)
		opts.OlderThan = &cutoff
	}

	if req.Kind != "" {
		k := model.Kind(req.Kind)
		if k != model.KindScrape && k != model.KindCrawl && k != model.KindResearch {
			writeError(w, r, apperrors.Validation("invalid kind: must be scrape, crawl, or research"))
			return
		}
		opts.Kind = &k
	}

	engine := retention.NewEngine(s.store, s.cfg)
	result, err := engine.RunCleanup(r.Context(), opts)
	if err != nil {
		writeError(w, r, err)
		return
	}

	// Convert errors to strings for JSON response
	var errorStrings []string
	for _, e := range result.Errors {
		errorStrings = append(errorStrings, e.Error())
	}

	resp := RetentionCleanupResponse{
		JobsDeleted:        result.JobsDeleted,
		JobsAttempted:      result.JobsAttempted,
		CrawlStatesDeleted: result.CrawlStatesDeleted,
		SpaceReclaimedMB:   result.SpaceReclaimedMB,
		DurationMs:         result.Duration.Milliseconds(),
		FailedJobIDs:       result.FailedJobIDs,
		Errors:             errorStrings,
		DryRun:             req.DryRun,
	}

	writeJSON(w, resp)
}
