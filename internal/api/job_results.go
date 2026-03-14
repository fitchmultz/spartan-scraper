// Package api provides HTTP handlers for job result retrieval endpoints.
// Job result handlers support retrieving persisted raw results for inspection
// and pagination. Shaped/transformed direct exports are handled separately by
// POST /v1/jobs/{id}/export.
package api

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/exporter"
	"github.com/fitchmultz/spartan-scraper/internal/model"
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
	if format != "jsonl" && format != "json" {
		writeError(w, r, apperrors.Validation("job results format must be jsonl or json; use POST /v1/jobs/{id}/export for direct exports"))
		return
	}

	if format == "jsonl" {
		page, hasPagination, err := parseOptionalResultsPageParams(r)
		if err != nil {
			writeError(w, r, err)
			return
		}
		if hasPagination {
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
		} else {
			w.Header().Set("Content-Type", exporter.ResultExportContentType("jsonl"))
		}
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, exporter.ResultExportFilename(job, exporter.ResultExportConfig{Format: "jsonl"})))
		http.ServeFile(w, r, job.ResultPath)
		return
	}

	f, err := s.openJobResultFile(job, resultMessages.MissingPath)
	if err != nil {
		writeError(w, r, err)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", exporter.ResultExportContentType("json"))
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, exporter.ResultExportFilename(job, exporter.ResultExportConfig{Format: "json"})))
	if err := exporter.ExportStream(job, f, "json", w); err != nil {
		writeError(w, r, err)
		return
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

func parseOptionalResultsPageParams(r *http.Request) (pageParams, bool, error) {
	hasPagination := r.URL.Query().Get("limit") != "" || r.URL.Query().Get("offset") != ""
	if !hasPagination {
		return pageParams{}, false, nil
	}
	page, err := parsePageParams(r, 100, 1000)
	if err != nil {
		return pageParams{}, false, err
	}
	return page, true, nil
}
