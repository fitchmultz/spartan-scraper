// Package api provides HTTP handlers for automated export schedule management.
//
// This file implements CRUD operations for export schedules that trigger
// automatic exports when jobs complete matching specified filter criteria.
//
// Endpoints:
// - GET    /v1/export-schedules      - List all export schedules
// - POST   /v1/export-schedules      - Create a new export schedule
// - GET    /v1/export-schedules/{id} - Get a specific export schedule
// - PUT    /v1/export-schedules/{id} - Update an export schedule
// - DELETE /v1/export-schedules/{id} - Delete an export schedule
// - GET    /v1/export-schedules/{id}/history - Get export history for a schedule
//
// This file does NOT handle:
// - Export execution (scheduler/export_trigger.go handles that)
// - Export validation (scheduler/export_validation.go handles that)
// - History persistence (scheduler/export_history.go handles that)
package api

import (
	"encoding/json"
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

	writeJSON(w, map[string]interface{}{
		"records": response,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
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

	writeJSON(w, map[string]interface{}{"schedules": response})
}

func (s *Server) createExportSchedule(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeError(w, r, apperrors.UnsupportedMediaType("content-type must be application/json"))
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	var req ExportScheduleRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, r, apperrors.Validation("invalid json: "+err.Error()))
		return
	}

	// Validate request
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, r, apperrors.Validation("name is required"))
		return
	}

	// Build export schedule
	schedule := scheduler.ExportSchedule{
		Name:    req.Name,
		Enabled: true, // default
		Filters: req.Filters,
		Export:  req.Export,
	}

	if req.Enabled != nil {
		schedule.Enabled = *req.Enabled
	}

	if req.Retry != nil {
		schedule.Retry = *req.Retry
	}

	// Set defaults for cloud config if provided
	if schedule.Export.CloudConfig != nil {
		if schedule.Export.CloudConfig.Path == "" {
			schedule.Export.CloudConfig.Path = "exports/{kind}/{job_id}.{format}"
		}
		if schedule.Export.CloudConfig.ContentFormat == "" {
			schedule.Export.CloudConfig.ContentFormat = schedule.Export.Format
		}
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

	writeJSON(w, toExportScheduleResponse(*created))
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
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeError(w, r, apperrors.UnsupportedMediaType("content-type must be application/json"))
		return
	}

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

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	var req ExportScheduleRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, r, apperrors.Validation("invalid json: "+err.Error()))
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

	w.WriteHeader(http.StatusNoContent)
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
