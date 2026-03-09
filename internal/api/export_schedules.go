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
	"strconv"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/exporter"
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

// ExportHistoryResponse represents an export history record in API responses.
type ExportHistoryResponse struct {
	ID           string     `json:"id"`
	ScheduleID   string     `json:"schedule_id"`
	JobID        string     `json:"job_id"`
	Status       string     `json:"status"`
	Destination  string     `json:"destination"`
	ExportedAt   time.Time  `json:"exported_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
	RetryCount   int        `json:"retry_count"`
	ExportSize   int64      `json:"export_size,omitempty"`
	RecordCount  int        `json:"record_count,omitempty"`
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
	id := extractID(r.URL.Path, "export-schedules")
	if id == "" {
		writeError(w, r, apperrors.Validation("id required"))
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
	// Check if this is a history request
	if strings.HasSuffix(r.URL.Path, "/history") {
		s.handleExportScheduleHistory(w, r)
		return
	}

	// Otherwise, treat as schedule ID request
	s.handleExportSchedule(w, r)
}

func (s *Server) handleExportScheduleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	id := extractID(r.URL.Path, "export-schedules")
	if id == "" {
		writeError(w, r, apperrors.Validation("id required"))
		return
	}

	// Parse query parameters
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	// Create history store
	historyStore := scheduler.NewExportHistoryStore(s.cfg.DataDir)

	// Get history
	records, total, err := historyStore.GetBySchedule(id, limit, offset)
	if err != nil {
		writeError(w, r, err)
		return
	}

	// Convert to response format
	response := make([]ExportHistoryResponse, len(records))
	for i, record := range records {
		response[i] = ExportHistoryResponse{
			ID:           record.ID,
			ScheduleID:   record.ScheduleID,
			JobID:        record.JobID,
			Status:       record.Status,
			Destination:  record.Destination,
			ExportedAt:   record.ExportedAt,
			CompletedAt:  record.CompletedAt,
			ErrorMessage: record.ErrorMessage,
			RetryCount:   record.RetryCount,
			ExportSize:   record.ExportSize,
			RecordCount:  record.RecordCount,
		}
	}

	writeRecordsPageJSON(w, response, total, limit, offset)
}

func (s *Server) listExportSchedules(w http.ResponseWriter, r *http.Request) {
	store := scheduler.NewExportStorage(s.cfg.DataDir)
	schedules, err := store.List()
	if err != nil {
		writeError(w, r, err)
		return
	}

	response := make([]ExportScheduleResponse, len(schedules))
	for i, sched := range schedules {
		response[i] = toExportScheduleResponse(sched)
	}

	writeCollectionJSON(w, "schedules", response)
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

	// Notify export trigger if running
	if s.manager != nil {
		// The export trigger will be notified via reload
	}

	writeCreatedJSON(w, toExportScheduleResponse(*created))
}

func (s *Server) getExportSchedule(w http.ResponseWriter, r *http.Request, id string) {
	store := scheduler.NewExportStorage(s.cfg.DataDir)
	schedule, err := store.Get(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
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
		if strings.Contains(err.Error(), "not found") {
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

	writeJSON(w, toExportScheduleResponse(*updated))
}

func (s *Server) deleteExportSchedule(w http.ResponseWriter, r *http.Request, id string) {
	store := scheduler.NewExportStorage(s.cfg.DataDir)
	if err := store.Delete(id); err != nil {
		writeError(w, r, err)
		return
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

// CloudExportConfigResponse wraps exporter.CloudExportConfig for API responses.
type CloudExportConfigResponse struct {
	Provider      string `json:"provider,omitempty"`
	Bucket        string `json:"bucket,omitempty"`
	Path          string `json:"path,omitempty"`
	Region        string `json:"region,omitempty"`
	StorageClass  string `json:"storageClass,omitempty"`
	ContentFormat string `json:"contentFormat,omitempty"`
	ContentType   string `json:"contentType,omitempty"`
}

// ToCloudExportConfigResponse converts exporter.CloudExportConfig to response format.
func ToCloudExportConfigResponse(cfg *exporter.CloudExportConfig) *CloudExportConfigResponse {
	if cfg == nil {
		return nil
	}
	return &CloudExportConfigResponse{
		Provider:      cfg.Provider,
		Bucket:        cfg.Bucket,
		Path:          cfg.Path,
		Region:        cfg.Region,
		StorageClass:  cfg.StorageClass,
		ContentFormat: cfg.ContentFormat,
		ContentType:   cfg.ContentType,
	}
}

func normalizeExportScheduleRequest(schedule scheduler.ExportSchedule) scheduler.ExportSchedule {
	if schedule.Export.CloudConfig == nil {
		return schedule
	}
	if schedule.Export.CloudConfig.Path == "" {
		schedule.Export.CloudConfig.Path = "exports/{kind}/{job_id}.{format}"
	}
	if schedule.Export.CloudConfig.ContentFormat == "" {
		schedule.Export.CloudConfig.ContentFormat = schedule.Export.Format
	}
	return schedule
}
