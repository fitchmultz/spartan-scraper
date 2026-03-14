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
func exportCSVStream(job model.Job, r io.Reader, w io.Writer, shape ShapeConfig) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	switch job.Kind {
	case model.KindScrape:
		item, err := parseSingleReader[ScrapeResult](r)
		if err != nil {
			return err
		}
		if HasMeaningfulShape(shape) {
			return writeScrapeCSVShaped(item, writer, shape)
		}
		return writeScrapeCSV(item, writer)
	case model.KindCrawl:
		rs, cleanup, err := ensureSeekable(r)
		if err != nil {
			return err
		}
		defer cleanup()
		if HasMeaningfulShape(shape) {
			return writeCrawlCSVStreamShaped(rs, writer, shape)
		}
		return writeCrawlCSVStream(rs, writer)
	case model.KindResearch:
		item, err := parseSingleReader[ResearchResult](r)
		if err != nil {
			return err
		}
		if HasMeaningfulShape(shape) {
			return writeResearchCSVShaped(item, writer, shape)
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

func writeScrapeCSVShaped(item ScrapeResult, writer *csv.Writer, shape ShapeConfig) error {
	topLevelFields := selectFields(shape.TopLevelFields, []string{"url", "status", "title", "description"})
	normalizedFields := selectFields(shape.NormalizedFields, scrapeNormalizedFieldRefs(item))
	headers := append(shapeFieldHeaders(shape, topLevelFields), shapeFieldHeaders(shape, normalizedFields)...)
	if err := writer.Write(headers); err != nil {
		return err
	}

	row := make([]string, 0, len(headers))
	for _, key := range topLevelFields {
		row = append(row, scrapeShapeValue(item, key, shape))
	}
	for _, key := range normalizedFields {
		row = append(row, scrapeShapeValue(item, key, shape))
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

func writeCrawlCSVStreamShaped(rs io.ReadSeeker, writer *csv.Writer, shape ShapeConfig) error {
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
	availableNormalized := make([]string, 0, len(fieldSet))
	for k := range fieldSet {
		availableNormalized = append(availableNormalized, "field."+k)
	}
	sort.Strings(availableNormalized)

	topLevelFields := selectFields(shape.TopLevelFields, []string{"url", "status", "title"})
	normalizedFields := selectFields(shape.NormalizedFields, availableNormalized)
	headers := append(shapeFieldHeaders(shape, topLevelFields), shapeFieldHeaders(shape, normalizedFields)...)
	if err := writer.Write(headers); err != nil {
		return err
	}
	if _, err := rs.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if err := scanReader[CrawlResult](rs, func(item CrawlResult) error {
		row := make([]string, 0, len(headers))
		for _, key := range topLevelFields {
			row = append(row, crawlShapeValue(item, key, shape))
		}
		for _, key := range normalizedFields {
			row = append(row, crawlShapeValue(item, key, shape))
		}
		return writer.Write(row)
	}); err != nil {
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

func writeResearchCSVShaped(item ResearchResult, writer *csv.Writer, shape ShapeConfig) error {
	topLevelFields := selectFields(shape.TopLevelFields, []string{"query", "summary", "confidence", "agentic.status", "agentic.summary"})
	evidenceFields := selectFields(shape.EvidenceFields, []string{"evidence.url", "evidence.title", "evidence.score", "evidence.confidence", "evidence.clusterId", "evidence.citationUrl", "evidence.snippet"})
	if err := writer.Write(shapeFieldHeaders(shape, topLevelFields)); err != nil {
		return err
	}
	summaryRow := make([]string, 0, len(topLevelFields))
	for _, key := range topLevelFields {
		summaryRow = append(summaryRow, researchShapeValue(item, key, shape))
	}
	if err := writer.Write(summaryRow); err != nil {
		return err
	}
	if err := writer.Write([]string{}); err != nil {
		return err
	}
	if err := writer.Write(shapeFieldHeaders(shape, evidenceFields)); err != nil {
		return err
	}
	for _, ev := range item.Evidence {
		row := make([]string, 0, len(evidenceFields))
		for _, key := range evidenceFields {
			row = append(row, researchEvidenceShapeValue(ev, key, shape))
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	return writer.Error()
}

func shapeFieldHeaders(shape ShapeConfig, fields []string) []string {
	headers := make([]string, 0, len(fields))
	for _, field := range fields {
		headers = append(headers, labelForShapeField(shape, field))
	}
	return headers
}

func scrapeNormalizedFieldRefs(item ScrapeResult) []string {
	fieldNames := make([]string, 0, len(item.Normalized.Fields))
	for k := range item.Normalized.Fields {
		fieldNames = append(fieldNames, "field."+k)
	}
	sort.Strings(fieldNames)
	return fieldNames
}

func scrapeShapeValue(item ScrapeResult, key string, shape ShapeConfig) string {
	title := item.Title
	description := item.Metadata.Description
	text := item.Text
	if item.Normalized.Title != "" {
		title = item.Normalized.Title
	}
	if item.Normalized.Description != "" {
		description = item.Normalized.Description
	}
	if item.Normalized.Text != "" {
		text = item.Normalized.Text
	}
	switch key {
	case "url":
		return item.URL
	case "status":
		return formatInt(item.Status)
	case "title":
		return title
	case "description":
		return description
	case "text":
		return text
	default:
		if strings.HasPrefix(key, "field.") {
			fieldName := strings.TrimPrefix(key, "field.")
			if value, ok := item.Normalized.Fields[fieldName]; ok {
				return joinValues(value.Values, shape)
			}
		}
		return shapeEmptyValue(shape)
	}
}

func crawlShapeValue(item CrawlResult, key string, shape ShapeConfig) string {
	title := item.Title
	if item.Normalized.Title != "" {
		title = item.Normalized.Title
	}
	switch key {
	case "url":
		return item.URL
	case "status":
		return formatInt(item.Status)
	case "title":
		return title
	case "text":
		if item.Normalized.Text != "" {
			return item.Normalized.Text
		}
		return item.Text
	default:
		if strings.HasPrefix(key, "field.") {
			fieldName := strings.TrimPrefix(key, "field.")
			if value, ok := item.Normalized.Fields[fieldName]; ok {
				return joinValues(value.Values, shape)
			}
		}
		return shapeEmptyValue(shape)
	}
}

func researchShapeValue(item ResearchResult, key string, shape ShapeConfig) string {
	switch key {
	case "query":
		return item.Query
	case "summary":
		return item.Summary
	case "confidence":
		return formatFloat(item.Confidence)
	case "agentic.status":
		if item.Agentic != nil {
			return item.Agentic.Status
		}
	case "agentic.summary":
		if item.Agentic != nil {
			return item.Agentic.Summary
		}
	case "agentic.objective":
		if item.Agentic != nil {
			return item.Agentic.Objective
		}
	case "agentic.error":
		if item.Agentic != nil {
			return item.Agentic.Error
		}
	case "agentic.instructions":
		if item.Agentic != nil {
			return item.Agentic.Instructions
		}
	default:
		if strings.HasPrefix(key, "field.") {
			return shapeEmptyValue(shape)
		}
	}
	return shapeEmptyValue(shape)
}

func researchEvidenceShapeValue(item struct {
	URL         string  `json:"url"`
	Title       string  `json:"title"`
	Snippet     string  `json:"snippet"`
	Score       float64 `json:"score"`
	SimHash     uint64  `json:"simhash"`
	ClusterID   string  `json:"clusterId"`
	Confidence  float64 `json:"confidence"`
	CitationURL string  `json:"citationUrl"`
}, key string, shape ShapeConfig) string {
	switch key {
	case "evidence.url":
		return item.URL
	case "evidence.title":
		return item.Title
	case "evidence.score":
		return formatFloat(item.Score)
	case "evidence.confidence":
		return formatFloat(item.Confidence)
	case "evidence.clusterId":
		return item.ClusterID
	case "evidence.citationUrl":
		return item.CitationURL
	case "evidence.snippet":
		return item.Snippet
	default:
		return shapeEmptyValue(shape)
	}
}
