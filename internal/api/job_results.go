// Package api provides HTTP handlers for job result retrieval endpoints.
// Job result handlers support retrieving results in various formats (JSON, CSV, XML)
// with pagination and content-type negotiation based on file extensions.
package api

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/exporter"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/webhook"
	"github.com/xuri/excelize/v2"
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

		// Validate result path to prevent path traversal attacks
		if err := model.ValidateResultPath(job.ID, job.ResultPath, s.store.DataDir()); err != nil {
			writeError(w, r, err)
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

		// Validate result path to prevent path traversal attacks
		if err := model.ValidateResultPath(job.ID, job.ResultPath, s.store.DataDir()); err != nil {
			writeError(w, r, err)
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

	// Extract transform parameters
	transformExpression := r.URL.Query().Get("transform_expression")
	transformLanguage := r.URL.Query().Get("transform_language")

	// Validate transform parameters if provided
	if transformExpression != "" {
		if transformLanguage != "jmespath" && transformLanguage != "jsonata" {
			writeError(w, r, apperrors.Validation("transform_language must be 'jmespath' or 'jsonata'"))
			return
		}
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

	// Apply transformation if requested
	if transformExpression != "" {
		if err := s.exportWithTransform(job, format, transformExpression, transformLanguage, w); err != nil {
			writeError(w, r, err)
			return
		}
	} else {
		if err := exporter.ExportStream(job, f, format, w); err != nil {
			writeError(w, r, err)
			return
		}
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

// exportWithTransform applies a transformation expression to job results and exports them.
// It reads all results, applies the transformation item-by-item, and streams the output.
func (s *Server) exportWithTransform(
	job model.Job,
	format string,
	expression string,
	language string,
	w http.ResponseWriter,
) error {
	// Read all results
	results, err := s.loadAllJobResults(job)
	if err != nil {
		return err
	}

	// Apply transformation
	transformedResults, err := ApplyTransformation(results, expression, language)
	if err != nil {
		return apperrors.Wrap(apperrors.KindValidation, "transformation failed", err)
	}

	// Export based on format
	switch format {
	case "json":
		return exportTransformedJSON(transformedResults, w)
	case "csv":
		return exportTransformedCSV(transformedResults, w)
	case "md":
		return exportTransformedMarkdown(transformedResults, job, w)
	case "xlsx":
		return exportTransformedXLSX(transformedResults, w)
	default:
		return apperrors.Validation("unsupported format for transform export: " + format)
	}
}

// loadAllJobResults loads all results from a job file.
func (s *Server) loadAllJobResults(job model.Job) ([]any, error) {
	if job.ResultPath == "" {
		return []any{}, nil
	}

	// Validate result path to prevent path traversal attacks
	if err := model.ValidateResultPath(job.ID, job.ResultPath, s.store.DataDir()); err != nil {
		return nil, err
	}

	file, err := os.Open(job.ResultPath)
	if err != nil {
		return nil, apperrors.Wrap(
			apperrors.KindInternal,
			"failed to open job results file",
			err,
		)
	}
	defer file.Close()

	results := make([]any, 0)
	scanner := bufio.NewScanner(file)
	// Set max line size to 10MB to handle large JSON objects
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var item any
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			return nil, apperrors.Wrap(
				apperrors.KindInternal,
				"failed to parse job result",
				err,
			)
		}
		results = append(results, item)
	}

	if err := scanner.Err(); err != nil {
		return nil, apperrors.Wrap(
			apperrors.KindInternal,
			"error reading job results file",
			err,
		)
	}

	return results, nil
}

// exportTransformedJSON exports transformed results as JSON.
func exportTransformedJSON(results []any, w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(results)
}

// exportTransformedCSV exports transformed results as CSV.
func exportTransformedCSV(results []any, w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	return writeGenericCSVResults(results, w)
}

// exportTransformedMarkdown exports transformed results as Markdown.
func exportTransformedMarkdown(results []any, _ model.Job, w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	return writeGenericMarkdownResults(results, w)
}

// exportTransformedXLSX exports transformed results as XLSX.
func exportTransformedXLSX(results []any, w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	return writeGenericXLSXResults(results, w)
}

// writeGenericCSVResults writes generic CSV from transformed results.
func writeGenericCSVResults(results []any, w io.Writer) error {
	if len(results) == 0 {
		return nil
	}

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Collect all unique keys from all results
	allKeys := make(map[string]bool)
	for _, r := range results {
		if m, ok := r.(map[string]any); ok {
			for k := range m {
				allKeys[k] = true
			}
		}
	}

	// Sort keys for consistent output
	keys := make([]string, 0, len(allKeys))
	for k := range allKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if len(keys) == 0 {
		// Fallback: write as JSON if not a map
		for _, r := range results {
			jsonBytes, _ := json.Marshal(r)
			writer.Write([]string{string(jsonBytes)})
		}
		return writer.Error()
	}

	// Write headers
	if err := writer.Write(keys); err != nil {
		return err
	}

	// Write rows
	for _, r := range results {
		row := make([]string, len(keys))
		if m, ok := r.(map[string]any); ok {
			for i, k := range keys {
				if v, exists := m[k]; exists {
					switch val := v.(type) {
					case string:
						row[i] = val
					case nil:
						row[i] = ""
					default:
						row[i] = fmt.Sprint(val)
					}
				}
			}
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return writer.Error()
}

// writeGenericMarkdownResults writes generic Markdown from transformed results.
func writeGenericMarkdownResults(results []any, w io.Writer) error {
	fmt.Fprint(w, "# Transformed Results\n\n")

	for i, r := range results {
		fmt.Fprintf(w, "## Item %d\n\n", i+1)

		switch val := r.(type) {
		case map[string]any:
			// Get sorted keys
			keys := make([]string, 0, len(val))
			for k := range val {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, k := range keys {
				v := val[k]
				switch innerVal := v.(type) {
				case string:
					fmt.Fprintf(w, "- **%s**: %s\n", k, innerVal)
				case nil:
					fmt.Fprintf(w, "- **%s**: (null)\n", k)
				default:
					fmt.Fprintf(w, "- **%s**: %v\n", k, innerVal)
				}
			}
		default:
			jsonBytes, _ := json.MarshalIndent(r, "", "  ")
			fmt.Fprintf(w, "```json\n%s\n```\n", string(jsonBytes))
		}

		fmt.Fprint(w, "\n")
	}

	return nil
}

// writeGenericXLSXResults writes generic XLSX from transformed results.
func writeGenericXLSXResults(results []any, w io.Writer) error {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Results"
	f.SetSheetName("Sheet1", sheetName)

	if len(results) == 0 {
		return f.Write(w)
	}

	// Collect all unique keys from all results
	allKeys := make(map[string]bool)
	for _, r := range results {
		if m, ok := r.(map[string]any); ok {
			for k := range m {
				allKeys[k] = true
			}
		}
	}

	// Sort keys for consistent output
	keys := make([]string, 0, len(allKeys))
	for k := range allKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if len(keys) == 0 {
		// Fallback: write as JSON strings
		f.SetCellValue(sheetName, "A1", "Data")
		for i, r := range results {
			jsonBytes, _ := json.Marshal(r)
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", i+2), string(jsonBytes))
		}
		return f.Write(w)
	}

	// Write headers
	for i, k := range keys {
		cell := fmt.Sprintf("%s1", string(rune('A'+i)))
		f.SetCellValue(sheetName, cell, k)
	}

	// Apply header style
	style, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#D9D9D9"},
			Pattern: 1,
		},
	})
	if err != nil {
		return err
	}
	f.SetRowStyle(sheetName, 1, 1, style)

	// Write data rows
	for rowIdx, r := range results {
		if m, ok := r.(map[string]any); ok {
			for colIdx, k := range keys {
				cell := fmt.Sprintf("%s%d", string(rune('A'+colIdx)), rowIdx+2)
				if v, exists := m[k]; exists {
					f.SetCellValue(sheetName, cell, v)
				}
			}
		}
	}

	// Auto-size columns
	for i, k := range keys {
		col := string(rune('A' + i))
		if i >= 26 {
			col = string(rune('A'+i/26-1)) + string(rune('A'+i%26))
		}
		width := float64(len(k)) * 1.5
		if width < 10 {
			width = 10
		}
		if width > 50 {
			width = 50
		}
		f.SetColWidth(sheetName, col, col, width)
	}

	return f.Write(w)
}
