package api

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/exporter"
	"github.com/fitchmultz/spartan-scraper/internal/model"
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
	if err := s.requireJobResultFile(job, jobResultFileMessages{
		MissingPath: "job has no results",
		MissingFile: "job result file is missing",
		EmptyFile:   "job result file is empty",
	}); err != nil {
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

	f, err := os.Open(job.ResultPath)
	if err != nil {
		writeError(w, r, err)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", exporter.ResultExportContentType(req.Format))
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, exporter.ResultExportFilename(job, req)))
	if err := exporter.ExportResultStream(job, f, req, w); err != nil {
		writeError(w, r, err)
		return
	}

	s.dispatchExportCompletedWebhook(r, job, req)
}

func (s *Server) dispatchExportCompletedWebhook(r *http.Request, job model.Job, exportConfig exporter.ResultExportConfig) {
	if s.webhookDispatcher == nil {
		return
	}
	webhookCfg := job.ExtractWebhookConfig()
	if webhookCfg == nil || !webhook.ShouldSendEvent(webhook.EventExportCompleted, "", webhookCfg.Events) {
		return
	}
	payload := webhook.Payload{
		EventID:      fmt.Sprintf("%s-export-%s", job.ID, exportConfig.Format),
		EventType:    webhook.EventExportCompleted,
		Timestamp:    time.Now(),
		JobID:        job.ID,
		JobKind:      string(job.Kind),
		Status:       string(job.Status),
		ExportFormat: exportConfig.Format,
		ExportPath:   job.ResultPath,
	}
	s.webhookDispatcher.Dispatch(r.Context(), webhookCfg.URL, payload, webhookCfg.Secret)
}
