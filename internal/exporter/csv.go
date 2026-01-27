// Package exporter provides CSV export implementation.
//
// CSV export transforms job results into comma-separated values format.
// Functions include:
// - exportCSVStream: Stream export to CSV
// - writeScrapeCSV/writeCrawlCSV/writeResearchCSV: Writer helpers
//
// This file does NOT handle other formats (JSON, JSONL, Markdown).
package exporter

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"spartan-scraper/internal/model"
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
		items, err := parseLinesReader[CrawlResult](r)
		if err != nil {
			return err
		}
		return writeCrawlCSV(items, writer)
	case model.KindResearch:
		item, err := parseSingleReader[ResearchResult](r)
		if err != nil {
			return err
		}
		return writeResearchCSV(item, writer)
	default:
		return errors.New("unknown job kind")
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

// writeCrawlCSV writes multiple crawl results to the CSV writer.
func writeCrawlCSV(items []CrawlResult, writer *csv.Writer) error {
	fieldSet := make(map[string]bool)
	for _, item := range items {
		for k := range item.Normalized.Fields {
			fieldSet[k] = true
		}
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

	for _, item := range items {
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
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	return writer.Error()
}

// writeResearchCSV writes a research result to the CSV writer.
func writeResearchCSV(item ResearchResult, writer *csv.Writer) error {
	if err := writer.Write([]string{"query", "summary", "confidence"}); err != nil {
		return err
	}
	if err := writer.Write([]string{item.Query, item.Summary, fmt.Sprintf("%.2f", item.Confidence)}); err != nil {
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
