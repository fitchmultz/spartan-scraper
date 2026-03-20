// Package api handles direct export inspection and delivery workflows.
//
// Purpose:
// - Run direct exports, persist canonical export outcomes, and expose export inspection endpoints.
//
// Responsibilities:
// - Validate direct-export requests and render export artifacts.
// - Persist export outcome history for API-triggered exports.
// - Return operator-facing export inspection envelopes instead of raw transport bodies.
// - Dispatch export-completed webhooks with the shared multipart contract when configured.
//
// Scope:
// - Direct export API handlers only; raw result inspection stays in job_results.go.
//
// Usage:
// - Mounted under POST /v1/jobs/{id}/export, GET /v1/jobs/{id}/exports, and GET /v1/exports/{id}.
//
// Invariants/Assumptions:
// - Direct export failures that happen after request validation should still produce a persisted export outcome.
// - Inline artifact content is returned only on the immediate POST response.
package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/exporter"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/scheduler"
	"github.com/fitchmultz/spartan-scraper/internal/webhook"
)

func (s *Server) handleJobExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	id, err := requireResourceID(r, "jobs", "job id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	job, err := s.store.Get(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	var req exporter.ResultExportConfig
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	req = exporter.NormalizeResultExportConfig(req)
	if err := exporter.ValidateResultExportConfig(req); err != nil {
		writeError(w, r, err)
		return
	}

	webhookCfg, shouldDispatchWebhook := s.exportCompletedWebhookConfig(job)
	destination := "api response"
	if shouldDispatchWebhook && webhookCfg != nil {
		destination = webhookCfg.URL
	}

	historyStore := scheduler.NewExportHistoryStore(s.cfg.DataDir)
	record, err := historyStore.CreateRecord(scheduler.CreateRecordInput{
		JobID:       job.ID,
		Trigger:     exporter.OutcomeTriggerAPI,
		Destination: destination,
		Request:     req,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	if err := s.requireJobResultFile(job, jobResultFileMessages{
		MissingPath: "job has no results",
		MissingFile: "job result file is missing",
		EmptyFile:   "job result file is empty",
	}); err != nil {
		s.writeFailedExportOutcome(w, r, historyStore, record.ID, err)
		return
	}

	raw, err := os.ReadFile(job.ResultPath)
	if err != nil {
		s.writeFailedExportOutcome(w, r, historyStore, record.ID, err)
		return
	}

	rendered, err := exporter.RenderResultExport(job, raw, req)
	if err != nil {
		s.writeFailedExportOutcome(w, r, historyStore, record.ID, err)
		return
	}

	if err := historyStore.MarkSuccess(record.ID, rendered); err != nil {
		writeError(w, r, err)
		return
	}
	stored, err := historyStore.GetByID(record.ID)
	if err != nil {
		writeError(w, r, err)
		return
	}

	if shouldDispatchWebhook && webhookCfg != nil {
		s.dispatchExportCompletedWebhook(job, webhookCfg, rendered)
	}

	writeJSON(w, ExportOutcomeResponse{Export: BuildExportInspection(*stored, rendered.Content)})
}

func (s *Server) handleJobExportHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	id, err := requireResourceID(r, "jobs", "job id")
	if err != nil {
		writeError(w, r, err)
		return
	}

	page, err := parsePageParams(r, 50, 1000)
	if err != nil {
		writeError(w, r, err)
		return
	}

	records, total, err := scheduler.NewExportHistoryStore(s.cfg.DataDir).GetByJob(id, page.Limit, page.Offset)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, BuildExportOutcomeListResponse(records, total, page.Limit, page.Offset))
}

func (s *Server) handleExportOutcome(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	id, err := requireResourceID(r, "exports", "export id")
	if err != nil {
		writeError(w, r, err)
		return
	}

	record, err := scheduler.NewExportHistoryStore(s.cfg.DataDir).GetByID(id)
	if err != nil {
		writeError(w, r, apperrors.NotFound("export outcome not found"))
		return
	}

	writeJSON(w, ExportOutcomeResponse{Export: BuildExportInspection(*record, nil)})
}

func (s *Server) writeFailedExportOutcome(w http.ResponseWriter, r *http.Request, historyStore *scheduler.ExportHistoryStore, recordID string, exportErr error) {
	if err := historyStore.MarkFailed(recordID, exportErr); err != nil {
		writeError(w, r, err)
		return
	}
	record, err := historyStore.GetByID(recordID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSONStatus(w, http.StatusOK, ExportOutcomeResponse{Export: BuildExportInspection(*record, nil)})
}

func (s *Server) exportCompletedWebhookConfig(job model.Job) (*model.WebhookSpec, bool) {
	if s.webhookDispatcher == nil {
		return nil, false
	}
	webhookCfg := job.ExtractWebhookConfig()
	if webhookCfg == nil || !webhook.ShouldSendEvent(webhook.EventExportCompleted, "", webhookCfg.Events) {
		return nil, false
	}
	return webhookCfg, true
}

func (s *Server) dispatchExportCompletedWebhook(job model.Job, webhookCfg *model.WebhookSpec, rendered exporter.RenderedResultExport) {
	payload := webhook.Payload{
		EventID:      fmt.Sprintf("%s-export-%s", job.ID, rendered.Format),
		EventType:    webhook.EventExportCompleted,
		Timestamp:    time.Now(),
		JobID:        job.ID,
		JobKind:      string(job.Kind),
		Status:       string(job.Status),
		ResultURL:    fmt.Sprintf("/v1/jobs/%s/results", job.ID),
		ExportFormat: rendered.Format,
		Filename:     rendered.Filename,
		ContentType:  rendered.ContentType,
		RecordCount:  rendered.RecordCount,
		ExportSize:   rendered.Size,
	}
	s.webhookDispatcher.DispatchExport(context.Background(), webhookCfg.URL, payload, rendered.Content, webhookCfg.Secret)
}
