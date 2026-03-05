// Package api provides HTTP handlers for webhook delivery history endpoints.
// These endpoints allow querying webhook delivery records for debugging and monitoring.
package api

import (
	"net/http"
	"strconv"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/webhook"
)

// handleWebhookDeliveries handles GET /v1/webhooks/deliveries
// Query params: job_id, limit, offset
func (s *Server) handleWebhookDeliveries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	// Get query parameters
	jobID := r.URL.Query().Get("job_id")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 100
	if limitStr != "" {
		parsed, err := strconv.Atoi(limitStr)
		if err != nil || parsed < 1 || parsed > 1000 {
			writeError(w, r, apperrors.Validation("limit must be between 1 and 1000"))
			return
		}
		limit = parsed
	}

	offset := 0
	if offsetStr != "" {
		parsed, err := strconv.Atoi(offsetStr)
		if err != nil || parsed < 0 {
			writeError(w, r, apperrors.Validation("offset must be non-negative"))
			return
		}
		offset = parsed
	}

	// Check if webhook store is available
	if s.webhookDispatcher == nil || s.webhookDispatcher.Store() == nil {
		writeJSON(w, map[string]any{
			"deliveries": []any{},
			"total":      0,
		})
		return
	}

	store := s.webhookDispatcher.Store()
	ctx := r.Context()

	// Get total count
	total, err := store.CountRecords(ctx, jobID)
	if err != nil {
		writeError(w, r, apperrors.Internal("failed to count delivery records"))
		return
	}

	// Get delivery records
	records, err := store.ListRecords(ctx, jobID, limit, offset)
	if err != nil {
		writeError(w, r, apperrors.Internal("failed to list delivery records"))
		return
	}

	// Convert to response format
	deliveries := make([]any, len(records))
	for i, r := range records {
		deliveries[i] = deliveryRecordToResponse(r)
	}

	writeJSON(w, map[string]any{
		"deliveries": deliveries,
		"total":      total,
	})
}

// handleWebhookDeliveryDetail handles GET /v1/webhooks/deliveries/{id}
func (s *Server) handleWebhookDeliveryDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	id := extractID(r.URL.Path, "deliveries")
	if id == "" {
		writeError(w, r, apperrors.Validation("delivery id required"))
		return
	}

	// Check if webhook store is available
	if s.webhookDispatcher == nil || s.webhookDispatcher.Store() == nil {
		writeError(w, r, apperrors.NotFound("webhook delivery not found"))
		return
	}

	store := s.webhookDispatcher.Store()
	ctx := r.Context()

	// Get delivery record
	record, found, err := store.GetRecord(ctx, id)
	if err != nil {
		writeError(w, r, apperrors.Internal("failed to get delivery record"))
		return
	}
	if !found {
		writeError(w, r, apperrors.NotFound("webhook delivery not found"))
		return
	}

	writeJSON(w, deliveryRecordToResponse(record))
}

// deliveryRecordToResponse converts a DeliveryRecord to a response map.
func deliveryRecordToResponse(r *webhook.DeliveryRecord) map[string]any {
	result := map[string]any{
		"id":        r.ID,
		"eventId":   r.EventID,
		"eventType": r.EventType,
		"jobId":     r.JobID,
		"url":       r.URL,
		"status":    r.Status,
		"attempts":  r.Attempts,
		"createdAt": r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		"updatedAt": r.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if r.LastError != "" {
		result["lastError"] = r.LastError
	}

	if r.DeliveredAt != nil {
		formatted := r.DeliveredAt.Format("2006-01-02T15:04:05Z07:00")
		result["deliveredAt"] = formatted
	}

	if r.ResponseCode != 0 {
		result["responseCode"] = r.ResponseCode
	}

	return result
}
