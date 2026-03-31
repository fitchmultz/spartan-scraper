// Package mcp implements observability and diagnostics MCP tool handlers.
//
// Purpose:
// - Keep read-only health, diagnostics, webhook-inspection, and proxy-status tools out of the main dispatch path.
//
// Responsibilities:
// - Validate observability tool inputs.
// - Return transport-safe diagnostic and inspection payloads.
// - Reuse shared store-loading helpers for webhook inspection.
//
// Scope:
// - MCP observability handlers only; runtime systems live in their own packages.
//
// Usage:
// - Registered through the domain tool registries in tool_registry.go.
//
// Invariants/Assumptions:
// - Health and diagnostics remain available in setup mode.
// - Webhook delivery listing caps limit at 1000.
// - Delivery errors are wrapped as internal MCP-safe failures.
package mcp

import (
	"context"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/api"
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/paramdecode"
	"github.com/fitchmultz/spartan-scraper/internal/webhook"
)

func (s *Server) handleHealthStatusTool(ctx context.Context, _ callParams) (interface{}, error) {
	return s.buildHealthStatus(ctx), nil
}

func (s *Server) handleDiagnosticCheckTool(ctx context.Context, params callParams) (interface{}, error) {
	component := api.NormalizeDiagnosticTarget(strings.TrimSpace(paramdecode.String(params.Arguments, "component")))
	if component == "" {
		return nil, apperrors.Validation("component is required and must be one of browser, ai, or proxy_pool")
	}
	return s.runDiagnostic(ctx, component), nil
}

func (s *Server) handleWebhookDeliveryListTool(ctx context.Context, params callParams) (interface{}, error) {
	jobID := strings.TrimSpace(paramdecode.String(params.Arguments, "jobId"))
	if jobID == "" {
		jobID = strings.TrimSpace(paramdecode.String(params.Arguments, "job_id"))
	}
	limit := paramdecode.PositiveInt(params.Arguments, "limit", 100)
	if limit > 1000 {
		limit = 1000
	}
	offset := paramdecode.PositiveInt(params.Arguments, "offset", 0)
	deliveryStore, err := loadWebhookDeliveryStore(s.cfg.DataDir)
	if err != nil {
		return nil, err
	}
	records, err := deliveryStore.ListRecords(ctx, jobID, limit, offset)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to list webhook deliveries", err)
	}
	total, err := deliveryStore.CountRecords(ctx, jobID)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to count webhook deliveries", err)
	}
	return map[string]interface{}{
		"deliveries": webhook.ToInspectableDeliveries(records),
		"total":      total,
		"limit":      limit,
		"offset":     offset,
	}, nil
}

func (s *Server) handleWebhookDeliveryGetTool(ctx context.Context, params callParams) (interface{}, error) {
	id := strings.TrimSpace(paramdecode.String(params.Arguments, "id"))
	if id == "" {
		return nil, apperrors.Validation("id is required")
	}
	deliveryStore, err := loadWebhookDeliveryStore(s.cfg.DataDir)
	if err != nil {
		return nil, err
	}
	record, found, err := deliveryStore.GetRecord(ctx, id)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to get webhook delivery", err)
	}
	if !found {
		return nil, apperrors.NotFound("webhook delivery not found")
	}
	return webhook.ToInspectableDelivery(record), nil
}

func (s *Server) handleProxyPoolStatusTool(_ context.Context, _ callParams) (interface{}, error) {
	return api.BuildProxyPoolStatusResponse(s.manager.GetProxyPool()), nil
}
