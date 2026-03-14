// Package exporter provides CSV export implementation.
//
// CSV export transforms job results into comma-separated values format.
// Functions include:
// - exportCSVStream: Stream export to CSV
// - writeScrapeCSV/writeCrawlCSVStream/writeResearchCSV: Writer helpers
//
// This file does NOT handle other formats (JSON, JSONL, Markdown).
package exporter

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

// exportCSVStream exports job results to CSV format with streaming.
func exportCSVStream(job model.Job, r io.Reader, w io.Writer) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	switch job.Kind {
	case model.KindScrape:
		item, err := parseSingleReader[ScrapeResult](r)
		if err != nil {
			return err
		}
		return writeScrapeCSV(item, writer)
	case model.KindCrawl:
		rs, cleanup, err := ensureSeekable(r)
		if err != nil {
			return err
		}
		defer cleanup()
		return writeCrawlCSVStream(rs, writer)
	case model.KindResearch:
		item, err := parseSingleReader[ResearchResult](r)
		if err != nil {
			return err
		}
		return writeResearchCSV(item, writer)
	default:
		return apperrors.Internal("unknown job kind")
	}
}

// writeScrapeCSV writes a single scrape result to the CSV writer.
func writeScrapeCSV(item ScrapeResult, writer *csv.Writer) error {
	headers := []string{"url", "status", "title", "description"}
	fieldNames := make([]string, 0, len(item.Normalized.Fields))
	for k := range item.Normalized.Fields {
		fieldNames = append(fieldNames, k)
	}
	sort.Strings(fieldNames)
	for _, k := range fieldNames {
		headers = append(headers, "field_"+k)
	}
	if err := writer.Write(headers); err != nil {
		return err
	}

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
	if err := writer.Write(row); err != nil {
		return err
	}
	return writer.Error()
}

// writeCrawlCSVStream writes multiple crawl results to the CSV writer using two-pass streaming.
func writeCrawlCSVStream(rs io.ReadSeeker, writer *csv.Writer) error {
	// Pass 1: Collect unique field keys
	fieldSet := make(map[string]bool)
	err := scanReader[CrawlResult](rs, func(item CrawlResult) error {
		for k := range item.Normalized.Fields {
			fieldSet[k] = true
		}
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
	if err := writer.Write(headers); err != nil {
		return err
	}

	// Reset for Pass 2
	if _, err := rs.Seek(0, io.SeekStart); err != nil {
		return err
	}

	// Pass 2: Write rows
	err = scanReader[CrawlResult](rs, func(item CrawlResult) error {
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
		return writer.Write(row)
	})
	if err != nil {
		return err
	}

	return writer.Error()
}

// writeResearchCSV writes a research result to the CSV writer.
func writeResearchCSV(item ResearchResult, writer *csv.Writer) error {
	agenticStatus := ""
	agenticSummary := ""
	if item.Agentic != nil {
		agenticStatus = item.Agentic.Status
		agenticSummary = item.Agentic.Summary
	}
	if err := writer.Write([]string{"query", "summary", "confidence", "agentic_status", "agentic_summary"}); err != nil {
		return err
	}
	if err := writer.Write([]string{item.Query, item.Summary, fmt.Sprintf("%.2f", item.Confidence), agenticStatus, agenticSummary}); err != nil {
		return err
	}
	if err := writer.Write([]string{}); err != nil {
		return err
	}
	if err := writer.Write([]string{"url", "title", "score", "confidence", "cluster_id", "citation_url", "snippet"}); err != nil {
		return err
	}
	for _, ev := range item.Evidence {
		if err := writer.Write([]string{
			ev.URL,
			ev.Title,
			fmt.Sprintf("%.2f", ev.Score),
			fmt.Sprintf("%.2f", ev.Confidence),
			ev.ClusterID,
			ev.CitationURL,
			ev.Snippet,
		}); err != nil {
			return err
		}
	}
	return writer.Error()
}
