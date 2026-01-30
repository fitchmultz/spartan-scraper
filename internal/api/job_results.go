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

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/exporter"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func (s *Server) handleJobResults(w http.ResponseWriter, r *http.Request) {
	id := extractID(r.URL.Path, "jobs")
	if id == "" {
		writeError(w, r, apperrors.Validation("id required"))
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}
	job, err := s.store.Get(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	switch job.Status {
	case model.StatusQueued:
		writeError(w, r, apperrors.Validation("job is queued and has no results yet"))
		return
	case model.StatusRunning:
		writeError(w, r, apperrors.Validation("job is still running and has no results yet"))
		return
	case model.StatusFailed:
		writeError(w, r, apperrors.Validation("job failed and produced no results"))
		return
	case model.StatusCanceled:
		writeError(w, r, apperrors.Validation("job was canceled and produced no results"))
		return
	case model.StatusSucceeded:
		if job.ResultPath == "" {
			writeError(w, r, apperrors.NotFound("job succeeded but no result path was recorded"))
			return
		}

		info, err := os.Stat(job.ResultPath)
		if err != nil {
			writeError(w, r, apperrors.NotFound("job succeeded but result file is missing"))
			return
		}
		if info.Size() == 0 {
			writeError(w, r, apperrors.NotFound("job succeeded but result file is empty"))
			return
		}
	default:
		if job.ResultPath == "" {
			writeError(w, r, apperrors.NotFound("no results"))
			return
		}
		info, err := os.Stat(job.ResultPath)
		if err != nil {
			writeError(w, r, apperrors.NotFound("no results"))
			return
		}
		if info.Size() == 0 {
			writeError(w, r, apperrors.NotFound("no results"))
			return
		}
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

	if format == "jsonl" {
		hasPagination := r.URL.Query().Get("limit") != "" || r.URL.Query().Get("offset") != ""

		if hasPagination {
			limit, err := parseIntParamStrict(r.URL.Query().Get("limit"), "limit")
			if err != nil {
				writeError(w, r, err)
				return
			}
			if limit == 0 {
				limit = 100
			}
			if limit > 1000 {
				limit = 1000
			}

			offset, err := parseIntParamStrict(r.URL.Query().Get("offset"), "offset")
			if err != nil {
				writeError(w, r, err)
				return
			}

			f, err := os.Open(job.ResultPath)
			if err != nil {
				writeError(w, r, err)
				return
			}
			defer f.Close()

			var items interface{}
			var total int

			switch job.Kind {
			case model.KindCrawl:
				items, total, err = exporter.ExportPaginated[exporter.CrawlResult](f, limit, offset)
			case model.KindScrape:
				items, total, err = exporter.ExportPaginated[exporter.ScrapeResult](f, limit, offset)
			case model.KindResearch:
				items, total, err = exporter.ExportPaginated[exporter.ResearchResult](f, limit, offset)
			default:
				items, total, err = exporter.ExportPaginated[map[string]interface{}](f, limit, offset)
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

	if err := exporter.ExportStream(job, f, format, w); err != nil {
		writeError(w, r, err)
		return
	}
}
