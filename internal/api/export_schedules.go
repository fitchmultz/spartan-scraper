// Package api provides HTTP handlers for automated export schedule management.
//
// Purpose:
// - Manage export schedules and schedule history over HTTP.
//
// Responsibilities:
// - Validate create/update requests for export schedules.
// - Return consistent JSON envelopes for list and history responses.
// - Normalize export defaults before persistence.
//
// Scope:
// - CRUD handlers for export schedules and their history endpoints.
//
// Usage:
// - Mounted under `/v1/export-schedules` and `/v1/export-schedules/{id}/history`.
//
// Invariants/Assumptions:
// - JSON request bodies are decoded through shared strict helpers.
// - Cloud export defaults are applied consistently on create and update.
package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/scheduler"
)

// ExportScheduleRequest represents a request to create/update an export schedule.
type ExportScheduleRequest struct {
	Name    string                       `json:"name"`
	Enabled *bool                        `json:"enabled,omitempty"`
	Filters scheduler.ExportFilters      `json:"filters"`
	Export  scheduler.ExportConfig       `json:"export"`
	Retry   *scheduler.ExportRetryConfig `json:"retry,omitempty"`
}

// ExportScheduleResponse represents an export schedule in API responses.
type ExportScheduleResponse struct {
	ID        string                      `json:"id"`
	Name      string                      `json:"name"`
	Enabled   bool                        `json:"enabled"`
	CreatedAt time.Time                   `json:"created_at"`
	UpdatedAt time.Time                   `json:"updated_at"`
	Filters   scheduler.ExportFilters     `json:"filters"`
	Export    scheduler.ExportConfig      `json:"export"`
	Retry     scheduler.ExportRetryConfig `json:"retry"`
}

func (s *Server) handleExportSchedules(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listExportSchedules(w, r)
	case http.MethodPost:
		s.createExportSchedule(w, r)
	default:
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
	}
}

func (s *Server) handleExportSchedule(w http.ResponseWriter, r *http.Request) {
	id, err := requireResourceID(r, "export-schedules", "export schedule id")
	if err != nil {
		writeError(w, r, err)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.getExportSchedule(w, r, id)
	case http.MethodPut:
		s.updateExportSchedule(w, r, id)
	case http.MethodDelete:
		s.deleteExportSchedule(w, r, id)
	default:
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
	}
}

// handleExportScheduleDetail handles requests to /v1/export-schedules/{id}/history
func (s *Server) handleExportScheduleDetail(w http.ResponseWriter, r *http.Request) {
	if handlePathSuffix(r.URL.Path, "/history", func() {
		s.handleExportScheduleHistory(w, r)
	}) {
		return
	}

	s.handleExportSchedule(w, r)
}

func (s *Server) handleExportScheduleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	id, err := requireResourceID(r, "export-schedules", "export schedule id")
	if err != nil {
		writeError(w, r, err)
		return
	}

	page, err := parsePageParams(r, 50, 1000)
	if err != nil {
		writeError(w, r, err)
		return
	}

	historyStore := scheduler.NewExportHistoryStore(s.cfg.DataDir)
	records, total, err := historyStore.GetBySchedule(id, page.Limit, page.Offset)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, BuildExportOutcomeListResponse(records, total, page.Limit, page.Offset))
}

func (s *Server) listExportSchedules(w http.ResponseWriter, r *http.Request) {
	store := scheduler.NewExportStorage(s.cfg.DataDir)
	schedules, err := store.List()
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeCollectionJSON(w, "schedules", mapSlice(schedules, toExportScheduleResponse))
}

func (s *Server) createExportSchedule(w http.ResponseWriter, r *http.Request) {
	var req ExportScheduleRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}

	// Validate request
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, r, apperrors.Validation("name is required"))
		return
	}

	// Build export schedule
	schedule := normalizeExportScheduleRequest(scheduler.ExportSchedule{
		Name:    req.Name,
		Enabled: true, // default
		Filters: req.Filters,
		Export:  req.Export,
	})

	if req.Enabled != nil {
		schedule.Enabled = *req.Enabled
	}

	if req.Retry != nil {
		schedule.Retry = *req.Retry
	}

	// Create storage and add schedule
	store := scheduler.NewExportStorage(s.cfg.DataDir)
	created, err := store.Add(schedule)
	if err != nil {
		writeError(w, r, err)
		return
	}

	if s.exportScheduleRuntime != nil {
		s.exportScheduleRuntime.AddSchedule(created)
	}

	writeCreatedJSON(w, toExportScheduleResponse(*created))
}

func (s *Server) getExportSchedule(w http.ResponseWriter, r *http.Request, id string) {
	store := scheduler.NewExportStorage(s.cfg.DataDir)
	schedule, err := store.Get(id)
	if err != nil {
		if scheduler.IsNotFoundError(err) {
			writeError(w, r, apperrors.NotFound("export schedule not found"))
			return
		}
		writeError(w, r, err)
		return
	}

	writeJSON(w, toExportScheduleResponse(*schedule))
}

func (s *Server) updateExportSchedule(w http.ResponseWriter, r *http.Request, id string) {
	store := scheduler.NewExportStorage(s.cfg.DataDir)

	// Get existing schedule
	existing, err := store.Get(id)
	if err != nil {
		if scheduler.IsNotFoundError(err) {
			writeError(w, r, apperrors.NotFound("export schedule not found"))
			return
		}
		writeError(w, r, err)
		return
	}

	var req ExportScheduleRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}

	// Update fields
	if strings.TrimSpace(req.Name) != "" {
		existing.Name = req.Name
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}
	existing.Filters = req.Filters
	existing.Export = req.Export
	if req.Retry != nil {
		existing.Retry = *req.Retry
	}
	*existing = normalizeExportScheduleRequest(*existing)

	// Update schedule
	updated, err := store.Update(*existing)
	if err != nil {
		writeError(w, r, err)
		return
	}

	if s.exportScheduleRuntime != nil {
		s.exportScheduleRuntime.UpdateSchedule(updated)
	}

	writeJSON(w, toExportScheduleResponse(*updated))
}

func (s *Server) deleteExportSchedule(w http.ResponseWriter, r *http.Request, id string) {
	store := scheduler.NewExportStorage(s.cfg.DataDir)
	if err := store.Delete(id); err != nil {
		if scheduler.IsNotFoundError(err) {
			writeError(w, r, apperrors.NotFound("export schedule not found"))
			return
		}
		writeError(w, r, err)
		return
	}

	if s.exportScheduleRuntime != nil {
		s.exportScheduleRuntime.RemoveSchedule(id)
	}

	writeNoContent(w)
}

func toExportScheduleResponse(schedule scheduler.ExportSchedule) ExportScheduleResponse {
	return ExportScheduleResponse{
		ID:        schedule.ID,
		Name:      schedule.Name,
		Enabled:   schedule.Enabled,
		CreatedAt: schedule.CreatedAt,
		UpdatedAt: schedule.UpdatedAt,
		Filters:   schedule.Filters,
		Export:    schedule.Export,
		Retry:     schedule.Retry,
	}
}

func normalizeExportScheduleRequest(schedule scheduler.ExportSchedule) scheduler.ExportSchedule {
	return scheduler.NormalizeExportSchedule(schedule)
}
