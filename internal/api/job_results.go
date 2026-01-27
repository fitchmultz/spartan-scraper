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
	"strings"

	"spartan-scraper/internal/apperrors"
	"spartan-scraper/internal/exporter"
	"spartan-scraper/internal/model"
)

func (s *Server) handleJobResults(w http.ResponseWriter, r *http.Request) {
	id := filepath.Base(strings.TrimSuffix(r.URL.Path, "/results"))
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, "id required")
		return
	}
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	job, err := s.store.Get(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	if job.ResultPath == "" {
		writeError(w, apperrors.NotFound("no results"))
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "jsonl"
	}

	validFormats := map[string]bool{"jsonl": true, "json": true, "md": true, "csv": true}
	if !validFormats[format] {
		writeError(w, apperrors.Validation("invalid format: must be jsonl, json, md, or csv"))
		return
	}

	if format == "jsonl" {
		hasPagination := r.URL.Query().Get("limit") != "" || r.URL.Query().Get("offset") != ""

		if hasPagination {
			limit := exporter.Limit(r)
			offset := exporter.Offset(r)

			f, err := os.Open(job.ResultPath)
			if err != nil {
				writeError(w, err)
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
				writeError(w, err)
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
		writeError(w, err)
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
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.%s"`, job.ID, format))

	if err := exporter.ExportStream(job, f, format, w); err != nil {
		writeError(w, err)
		return
	}
}
