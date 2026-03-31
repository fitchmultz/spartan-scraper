// Package mcp implements export-oriented MCP tool handlers.
//
// Purpose:
// - Keep export execution, history inspection, and schedule management in one focused domain.
//
// Responsibilities:
// - Validate export requests before rendering outcomes.
// - Persist and inspect export history records.
// - Manage automated export schedules through the shared scheduler stores.
//
// Scope:
// - MCP export and schedule handlers only.
//
// Usage:
// - Registered through exportToolRegistry in tool_registry.go.
//
// Invariants/Assumptions:
// - Export outcomes always record success or failure in history.
// - Schedule CRUD uses scheduler storage as the source of truth.
// - Missing jobs, results, and schedules return classified not-found errors.
package mcp

import (
	"context"
	"os"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/api"
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/exporter"
	"github.com/fitchmultz/spartan-scraper/internal/paramdecode"
	"github.com/fitchmultz/spartan-scraper/internal/scheduler"
)

func (s *Server) handleJobExportTool(ctx context.Context, params callParams) (interface{}, error) {
	id := strings.TrimSpace(paramdecode.String(params.Arguments, "id"))
	if id == "" {
		return nil, apperrors.Validation("id is required")
	}
	exportConfig := exporter.NormalizeResultExportConfig(exporter.ResultExportConfig{
		Format:    strings.TrimSpace(paramdecode.String(params.Arguments, "format")),
		Shape:     paramdecode.Decode[exporter.ShapeConfig](params.Arguments, "shape"),
		Transform: paramdecode.Decode[exporter.TransformConfig](params.Arguments, "transform"),
	})
	if err := exporter.ValidateResultExportConfig(exportConfig); err != nil {
		return nil, err
	}
	job, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindNotFound, "job not found", err)
	}
	if job.ResultPath == "" {
		return nil, apperrors.NotFound("job has no results")
	}

	historyStore := scheduler.NewExportHistoryStore(s.cfg.DataDir)
	record, err := historyStore.CreateRecord(scheduler.CreateRecordInput{
		JobID:       job.ID,
		Trigger:     exporter.OutcomeTriggerMCP,
		Destination: "mcp response",
		Request:     exportConfig,
	})
	if err != nil {
		return nil, err
	}

	rawBytes, err := os.ReadFile(job.ResultPath)
	if err != nil {
		if markErr := historyStore.MarkFailed(record.ID, apperrors.Wrap(apperrors.KindInternal, "failed to read result file", err)); markErr != nil {
			return nil, markErr
		}
		failed, getErr := historyStore.GetByID(record.ID)
		if getErr != nil {
			return nil, getErr
		}
		return api.ExportOutcomeResponse{Export: api.BuildExportInspection(*failed, nil)}, nil
	}

	rendered, err := exporter.RenderResultExport(job, rawBytes, exportConfig)
	if err != nil {
		if markErr := historyStore.MarkFailed(record.ID, err); markErr != nil {
			return nil, markErr
		}
		failed, getErr := historyStore.GetByID(record.ID)
		if getErr != nil {
			return nil, getErr
		}
		return api.ExportOutcomeResponse{Export: api.BuildExportInspection(*failed, nil)}, nil
	}

	if err := historyStore.MarkSuccess(record.ID, rendered); err != nil {
		return nil, err
	}
	stored, err := historyStore.GetByID(record.ID)
	if err != nil {
		return nil, err
	}
	return api.ExportOutcomeResponse{Export: api.BuildExportInspection(*stored, rendered.Content)}, nil
}

func (s *Server) handleJobExportHistoryTool(_ context.Context, params callParams) (interface{}, error) {
	id := strings.TrimSpace(paramdecode.String(params.Arguments, "id"))
	if id == "" {
		return nil, apperrors.Validation("id is required")
	}
	limit := paramdecode.PositiveInt(params.Arguments, "limit", 50)
	offset := paramdecode.PositiveInt(params.Arguments, "offset", 0)
	records, total, err := scheduler.NewExportHistoryStore(s.cfg.DataDir).GetByJob(id, limit, offset)
	if err != nil {
		return nil, err
	}
	return api.BuildExportOutcomeListResponse(records, total, limit, offset), nil
}

func (s *Server) handleExportOutcomeGetTool(_ context.Context, params callParams) (interface{}, error) {
	id := strings.TrimSpace(paramdecode.String(params.Arguments, "id"))
	if id == "" {
		return nil, apperrors.Validation("id is required")
	}
	record, err := scheduler.NewExportHistoryStore(s.cfg.DataDir).GetByID(id)
	if err != nil {
		return nil, apperrors.NotFound("export outcome not found")
	}
	return api.ExportOutcomeResponse{Export: api.BuildExportInspection(*record, nil)}, nil
}

func (s *Server) handleExportScheduleListTool(_ context.Context, _ callParams) (interface{}, error) {
	schedules, err := scheduler.NewExportStorage(s.cfg.DataDir).List()
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"schedules": schedules}, nil
}

func (s *Server) handleExportScheduleGetTool(_ context.Context, params callParams) (interface{}, error) {
	id := strings.TrimSpace(paramdecode.String(params.Arguments, "id"))
	if id == "" {
		return nil, apperrors.Validation("id is required")
	}
	schedule, err := scheduler.NewExportStorage(s.cfg.DataDir).Get(id)
	if err != nil {
		if scheduler.IsNotFoundError(err) {
			return nil, apperrors.NotFound("export schedule not found")
		}
		return nil, err
	}
	return schedule, nil
}

func (s *Server) handleExportScheduleCreateTool(_ context.Context, params callParams) (interface{}, error) {
	schedule := scheduler.ExportSchedule{
		Name:    strings.TrimSpace(paramdecode.String(params.Arguments, "name")),
		Enabled: paramdecode.BoolDefault(params.Arguments, "enabled", true),
		Filters: paramdecode.Decode[scheduler.ExportFilters](params.Arguments, "filters"),
		Export:  paramdecode.Decode[scheduler.ExportConfig](params.Arguments, "export"),
		Retry:   paramdecode.Decode[scheduler.ExportRetryConfig](params.Arguments, "retry"),
	}
	created, err := scheduler.NewExportStorage(s.cfg.DataDir).Add(schedule)
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (s *Server) handleExportScheduleUpdateTool(_ context.Context, params callParams) (interface{}, error) {
	id := strings.TrimSpace(paramdecode.String(params.Arguments, "id"))
	if id == "" {
		return nil, apperrors.Validation("id is required")
	}
	store := scheduler.NewExportStorage(s.cfg.DataDir)
	existing, err := store.Get(id)
	if err != nil {
		if scheduler.IsNotFoundError(err) {
			return nil, apperrors.NotFound("export schedule not found")
		}
		return nil, err
	}
	existing.Name = strings.TrimSpace(paramdecode.String(params.Arguments, "name"))
	existing.Enabled = paramdecode.BoolDefault(params.Arguments, "enabled", existing.Enabled)
	existing.Filters = paramdecode.Decode[scheduler.ExportFilters](params.Arguments, "filters")
	existing.Export = paramdecode.Decode[scheduler.ExportConfig](params.Arguments, "export")
	if _, ok := params.Arguments["retry"]; ok {
		existing.Retry = paramdecode.Decode[scheduler.ExportRetryConfig](params.Arguments, "retry")
	}
	updated, err := store.Update(*existing)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *Server) handleExportScheduleDeleteTool(_ context.Context, params callParams) (interface{}, error) {
	id := strings.TrimSpace(paramdecode.String(params.Arguments, "id"))
	if id == "" {
		return nil, apperrors.Validation("id is required")
	}
	if err := scheduler.NewExportStorage(s.cfg.DataDir).Delete(id); err != nil {
		if scheduler.IsNotFoundError(err) {
			return nil, apperrors.NotFound("export schedule not found")
		}
		return nil, err
	}
	return map[string]interface{}{"deleted": true, "id": id}, nil
}

func (s *Server) handleExportScheduleHistoryTool(_ context.Context, params callParams) (interface{}, error) {
	id := strings.TrimSpace(paramdecode.String(params.Arguments, "id"))
	if id == "" {
		return nil, apperrors.Validation("id is required")
	}
	limit := paramdecode.PositiveInt(params.Arguments, "limit", 50)
	offset := paramdecode.PositiveInt(params.Arguments, "offset", 0)
	records, total, err := scheduler.NewExportHistoryStore(s.cfg.DataDir).GetBySchedule(id, limit, offset)
	if err != nil {
		return nil, err
	}
	return api.BuildExportOutcomeListResponse(records, total, limit, offset), nil
}
