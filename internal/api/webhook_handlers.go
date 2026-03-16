// Package api provides HTTP handlers for webhook delivery history endpoints.
// These endpoints allow querying webhook delivery records for debugging and monitoring.
package api

import (
	"net/http"

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

	jobID := r.URL.Query().Get("job_id")
	page, err := parsePageParams(r, 100, 1000)
	if err != nil {
		writeError(w, r, err)
		return
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
	records, err := store.ListRecords(ctx, jobID, page.Limit, page.Offset)
	if err != nil {
		writeError(w, r, apperrors.Internal("failed to list delivery records"))
		return
	}

	writeJSON(w, map[string]any{
		"deliveries": webhook.ToInspectableDeliveries(records),
		"total":      total,
	})
}

// handleWebhookDeliveryDetail handles GET /v1/webhooks/deliveries/{id}
func (s *Server) handleWebhookDeliveryDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	id, err := requireResourceID(r, "deliveries", "delivery id")
	if err != nil {
		writeError(w, r, err)
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

	writeJSON(w, webhook.ToInspectableDelivery(record))
}
