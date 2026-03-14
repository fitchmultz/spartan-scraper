// Package api provides HTTP handlers for job result retrieval endpoints.
// Job result handlers support retrieving results in various formats (JSON, CSV, XML)
// with pagination and content-type negotiation based on file extensions.
package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/exporter"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/webhook"
)

func (s *Server) handleJobResults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
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
	resultMessages := jobResultFileMessages{
		MissingPath: "no results",
		MissingFile: "no results",
		EmptyFile:   "no results",
	}
	if job.Status == model.StatusSucceeded {
		resultMessages = jobResultFileMessages{
			MissingPath: "job succeeded but no result path was recorded",
			MissingFile: "job succeeded but result file is missing",
			EmptyFile:   "job succeeded but result file is empty",
		}
	}
	if err := s.requireJobResultFile(job, resultMessages); err != nil {
		writeError(w, r, err)
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "jsonl"
	}

	validFormats := map[string]bool{"jsonl": true, "json": true, "md": true, "csv": true, "xlsx": true}
	if !validFormats[format] {
		writeError(w, r, apperrors.Validation("invalid format: must be jsonl, json, md, csv, or xlsx"))
		return
	}

	transform := exporter.TransformConfig{
		Expression: r.URL.Query().Get("transform_expression"),
		Language:   r.URL.Query().Get("transform_language"),
	}
	if err := exporter.ValidateTransformConfig(transform); err != nil {
		writeError(w, r, err)
		return
	}

	if format == "jsonl" {
		hasPagination := r.URL.Query().Get("limit") != "" || r.URL.Query().Get("offset") != ""

		if exporter.HasMeaningfulTransform(transform) {
			results, err := s.loadAllJobResults(job)
			if err != nil {
				writeError(w, r, err)
				return
			}
			transformed, err := exporter.ApplyTransformConfig(results, transform)
			if err != nil {
				writeError(w, r, apperrors.Wrap(apperrors.KindValidation, "transformation failed", err))
				return
			}
			if hasPagination {
				page, err := parsePageParams(r, 100, 1000)
				if err != nil {
					writeError(w, r, err)
					return
				}
				total := len(transformed)
				if page.Offset >= total {
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("X-Total-Count", strconv.Itoa(total))
					writeJSON(w, []any{})
					return
				}
				end := page.Offset + page.Limit
				if end > total {
					end = total
				}
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Total-Count", strconv.Itoa(total))
				writeJSON(w, transformed[page.Offset:end])
				return
			}
			w.Header().Set("Content-Type", "application/x-ndjson")
			w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.jsonl"`, job.ID))
			if err := exporter.ExportTransformedResults("jsonl", transformed, w); err != nil {
				writeError(w, r, err)
				return
			}
			return
		}

		if hasPagination {
			page, err := parsePageParams(r, 100, 1000)
			if err != nil {
				writeError(w, r, err)
				return
			}

			f, err := s.openJobResultFile(job, resultMessages.MissingPath)
			if err != nil {
				writeError(w, r, err)
				return
			}
			defer f.Close()

			var items interface{}
			var total int

			switch job.Kind {
			case model.KindCrawl:
				items, total, err = exporter.ExportPaginated[exporter.CrawlResult](f, page.Limit, page.Offset)
			case model.KindScrape:
				items, total, err = exporter.ExportPaginated[exporter.ScrapeResult](f, page.Limit, page.Offset)
			case model.KindResearch:
				items, total, err = exporter.ExportPaginated[exporter.ResearchResult](f, page.Limit, page.Offset)
			default:
				items, total, err = exporter.ExportPaginated[map[string]interface{}](f, page.Limit, page.Offset)
			}

			if err != nil {
				writeError(w, r, err)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Total-Count", strconv.Itoa(total))
			writeJSON(w, items)
			return
		}

		ext := filepath.Ext(job.ResultPath)
		if ct := contentTypeForExtension(ext); ct != "" {
			w.Header().Set("Content-Type", ct)
		}
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.jsonl"`, job.ID))
		http.ServeFile(w, r, job.ResultPath)
		return
	}

	f, err := os.Open(job.ResultPath)
	if err != nil {
		writeError(w, r, err)
		return
	}
	defer f.Close()

	switch format {
	case "json":
		w.Header().Set("Content-Type", "application/json")
	case "md":
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	case "csv":
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	case "xlsx":
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.%s"`, job.ID, format))

	if err := exporter.ExportStreamWithShapeAndTransform(job, f, format, exporter.ShapeConfig{}, transform, w); err != nil {
		writeError(w, r, err)
		return
	}

	// Dispatch export.completed webhook event
	if s.webhookDispatcher != nil {
		webhookCfg := job.ExtractWebhookConfig()
		if webhookCfg != nil && webhook.ShouldSendEvent(webhook.EventExportCompleted, "", webhookCfg.Events) {
			payload := webhook.Payload{
				EventID:      fmt.Sprintf("%s-export-%s", job.ID, format),
				EventType:    webhook.EventExportCompleted,
				Timestamp:    time.Now(),
				JobID:        job.ID,
				JobKind:      string(job.Kind),
				Status:       string(job.Status),
				ExportFormat: format,
				ExportPath:   job.ResultPath,
			}
			s.webhookDispatcher.Dispatch(r.Context(), webhookCfg.URL, payload, webhookCfg.Secret)
		}
	}
}

// loadAllJobResults loads all results from a job file.
func (s *Server) loadAllJobResults(job model.Job) ([]any, error) {
	file, err := s.openJobResultFile(job, "job has no results")
	if err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) && job.ResultPath == "" {
			return []any{}, nil
		}
		return nil, err
	}
	defer file.Close()
	return decodeJobResultItems(file, 0)
}
