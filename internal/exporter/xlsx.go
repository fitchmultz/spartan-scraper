// Package exporter provides XLSX export implementation.
//
// XLSX export transforms job results into Excel spreadsheet format with:
// - Header formatting (bold, background color)
// - Auto-sized columns based on content
// - Multi-sheet support for research jobs
//
// This file does NOT handle other formats (JSON, JSONL, Markdown, CSV).
package exporter

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/xuri/excelize/v2"
)

// exportXLSXStream exports job results to XLSX format with streaming.
func exportXLSXStream(job model.Job, r io.Reader, w io.Writer) error {
	f := excelize.NewFile()
	defer f.Close()

	switch job.Kind {
	case model.KindScrape:
		item, err := parseSingleReader[ScrapeResult](r)
		if err != nil {
			return err
		}
		return writeScrapeXLSX(item, f, w)
	case model.KindCrawl:
		rs, cleanup, err := ensureSeekable(r)
		if err != nil {
			return err
		}
		defer cleanup()
		return writeCrawlXLSXStream(rs, f, w)
	case model.KindResearch:
		item, err := parseSingleReader[ResearchResult](r)
		if err != nil {
			return err
		}
		return writeResearchXLSX(item, f, w)
	default:
		return apperrors.Internal("unknown job kind")
	}
}

// createHeaderStyle creates a style for header cells with bold text and gray background.
func createHeaderStyle(f *excelize.File) (int, error) {
	return f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#D9D9D9"},
			Pattern: 1,
		},
	})
}

// setColumnWidths auto-sizes columns based on content length.
func setColumnWidths(f *excelize.File, sheet string, headers []string, rows [][]string) {
	for i, header := range headers {
		maxLen := len(header)
		for _, row := range rows {
			if i < len(row) && len(row[i]) > maxLen {
				maxLen = len(row[i])
			}
		}
		// Convert column index to letter (A, B, C, ...)
		col := string(rune('A' + i))
		if i >= 26 {
			// Handle columns beyond Z (AA, AB, etc.)
			col = string(rune('A'+i/26-1)) + string(rune('A'+i%26))
		}
		// Set width with a minimum and maximum, accounting for padding
		width := float64(maxLen) * 1.2
		if width < 10 {
			width = 10
		}
		if width > 50 {
			width = 50
		}
		f.SetColWidth(sheet, col, col, width)
	}
}

// writeScrapeXLSX writes a single scrape result to the XLSX file.
func writeScrapeXLSX(item ScrapeResult, f *excelize.File, w io.Writer) error {
	sheetName := "Results"
	f.SetSheetName("Sheet1", sheetName)

	headers := []string{"url", "status", "title", "description"}
	fieldNames := make([]string, 0, len(item.Normalized.Fields))
	for k := range item.Normalized.Fields {
		fieldNames = append(fieldNames, k)
	}
	sort.Strings(fieldNames)
	for _, k := range fieldNames {
		headers = append(headers, "field_"+k)
	}

	// Write headers
	for i, h := range headers {
		cell := fmt.Sprintf("%s1", string(rune('A'+i)))
		f.SetCellValue(sheetName, cell, h)
	}

	// Apply header style
	style, err := createHeaderStyle(f)
	if err != nil {
		return err
	}
	f.SetRowStyle(sheetName, 1, 1, style)

	// Prepare data row
	title := item.Title
	desc := item.Metadata.Description
	if item.Normalized.Title != "" {
		title = item.Normalized.Title
	}
	if item.Normalized.Description != "" {
		desc = item.Normalized.Description
	}

	row := []string{item.URL, fmt.Sprint(item.Status), title, desc}
	for _, k := range fieldNames {
		val := ""
		if v, ok := item.Normalized.Fields[k]; ok {
			val = strings.Join(v.Values, "; ")
		}
		row = append(row, val)
	}

	// Write data row
	for i, val := range row {
		cell := fmt.Sprintf("%s2", string(rune('A'+i)))
		f.SetCellValue(sheetName, cell, val)
	}

	// Set column widths
	setColumnWidths(f, sheetName, headers, [][]string{row})

	return f.Write(w)
}

// writeCrawlXLSXStream writes multiple crawl results to the XLSX file using two-pass streaming.
func writeCrawlXLSXStream(rs io.ReadSeeker, f *excelize.File, w io.Writer) error {
	sheetName := "Results"
	f.SetSheetName("Sheet1", sheetName)

	// Pass 1: Collect unique field keys and all rows
	fieldSet := make(map[string]bool)
	var allRows []CrawlResult

	err := scanReader[CrawlResult](rs, func(item CrawlResult) error {
		for k := range item.Normalized.Fields {
			fieldSet[k] = true
		}
		allRows = append(allRows, item)
		return nil
	})
	if err != nil {
		return err
	}

	fieldNames := make([]string, 0, len(fieldSet))
	for k := range fieldSet {
		fieldNames = append(fieldNames, k)
	}
	sort.Strings(fieldNames)

	headers := []string{"url", "status", "title"}
	for _, k := range fieldNames {
		headers = append(headers, "field_"+k)
	}

	// Write headers
	for i, h := range headers {
		cell := fmt.Sprintf("%s1", string(rune('A'+i)))
		f.SetCellValue(sheetName, cell, h)
	}

	// Apply header style
	style, err := createHeaderStyle(f)
	if err != nil {
		return err
	}
	f.SetRowStyle(sheetName, 1, 1, style)

	// Write data rows
	var dataRows [][]string
	for rowIdx, item := range allRows {
		title := item.Title
		if item.Normalized.Title != "" {
			title = item.Normalized.Title
		}
		row := []string{item.URL, fmt.Sprint(item.Status), title}
		for _, k := range fieldNames {
			val := ""
			if v, ok := item.Normalized.Fields[k]; ok {
				val = strings.Join(v.Values, "; ")
			}
			row = append(row, val)
		}
		dataRows = append(dataRows, row)

		// Write row to sheet
		for i, val := range row {
			cell := fmt.Sprintf("%s%d", string(rune('A'+i)), rowIdx+2)
			f.SetCellValue(sheetName, cell, val)
		}
	}

	// Set column widths
	setColumnWidths(f, sheetName, headers, dataRows)

	return f.Write(w)
}

// writeResearchXLSX writes a research result to the XLSX file with multiple sheets.
func writeResearchXLSX(item ResearchResult, f *excelize.File, w io.Writer) error {
	// Summary sheet - rename the default sheet
	summarySheet := "Summary"
	f.SetSheetName("Sheet1", summarySheet)

	agenticStatus := ""
	agenticSummary := ""
	if item.Agentic != nil {
		agenticStatus = item.Agentic.Status
		agenticSummary = item.Agentic.Summary
	}

	summaryHeaders := []string{"query", "summary", "confidence", "agentic_status", "agentic_summary"}
	for i, h := range summaryHeaders {
		cell := fmt.Sprintf("%s1", string(rune('A'+i)))
		f.SetCellValue(summarySheet, cell, h)
	}

	style, err := createHeaderStyle(f)
	if err != nil {
		return err
	}
	f.SetRowStyle(summarySheet, 1, 1, style)

	f.SetCellValue(summarySheet, "A2", item.Query)
	f.SetCellValue(summarySheet, "B2", item.Summary)
	f.SetCellValue(summarySheet, "C2", fmt.Sprintf("%.2f", item.Confidence))
	f.SetCellValue(summarySheet, "D2", agenticStatus)
	f.SetCellValue(summarySheet, "E2", agenticSummary)

	setColumnWidths(f, summarySheet, summaryHeaders, [][]string{{item.Query, item.Summary, fmt.Sprintf("%.2f", item.Confidence), agenticStatus, agenticSummary}})

	// Evidence sheet
	evidenceSheet := "Evidence"
	f.NewSheet(evidenceSheet)

	evidenceHeaders := []string{"url", "title", "score", "confidence", "cluster_id", "citation_url", "snippet"}
	for i, h := range evidenceHeaders {
		cell := fmt.Sprintf("%s1", string(rune('A'+i)))
		f.SetCellValue(evidenceSheet, cell, h)
	}
	f.SetRowStyle(evidenceSheet, 1, 1, style)

	var evidenceRows [][]string
	for rowIdx, ev := range item.Evidence {
		row := []string{
			ev.URL,
			ev.Title,
			fmt.Sprintf("%.2f", ev.Score),
			fmt.Sprintf("%.2f", ev.Confidence),
			ev.ClusterID,
			ev.CitationURL,
			ev.Snippet,
		}
		evidenceRows = append(evidenceRows, row)

		for i, val := range row {
			cell := fmt.Sprintf("%s%d", string(rune('A'+i)), rowIdx+2)
			f.SetCellValue(evidenceSheet, cell, val)
		}
	}

	setColumnWidths(f, evidenceSheet, evidenceHeaders, evidenceRows)

	// Set active sheet to Summary
	idx, _ := f.GetSheetIndex(summarySheet)
	f.SetActiveSheet(idx)

	return f.Write(w)
}
