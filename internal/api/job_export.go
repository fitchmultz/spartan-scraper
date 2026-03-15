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

	webhookCfg, shouldDispatchWebhook := s.exportCompletedWebhookConfig(job)
	if shouldDispatchWebhook {
		raw, err := os.ReadFile(job.ResultPath)
		if err != nil {
			writeError(w, r, err)
			return
		}
		rendered, err := exporter.RenderResultExport(job, raw, req)
		if err != nil {
			writeError(w, r, err)
			return
		}
		w.Header().Set("Content-Type", rendered.ContentType)
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, rendered.Filename))
		if _, err := w.Write(rendered.Content); err != nil {
			return
		}
		s.dispatchExportCompletedWebhook(job, webhookCfg, rendered)
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
