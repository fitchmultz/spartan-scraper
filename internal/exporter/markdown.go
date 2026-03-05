// Package exporter provides Markdown export implementation.
//
// Markdown export transforms job results into human-readable Markdown format.
// Functions include:
// - exportMarkdownStream: Stream export to Markdown
// - writeScrapeMarkdown/writeCrawlMarkdown/writeResearchMarkdown: Writer helpers
//
// This file does NOT handle other formats (JSON, JSONL, CSV).
package exporter

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

// exportMarkdownStream exports job results to Markdown format with streaming.
func exportMarkdownStream(job model.Job, r io.Reader, w io.Writer) error {
	switch job.Kind {
	case model.KindScrape:
		item, err := parseSingleReader[ScrapeResult](r)
		if err != nil {
			return err
		}
		return writeScrapeMarkdown(item, w)
	case model.KindCrawl:
		fmt.Fprint(w, "# Crawl Results\n\n")
		return scanReader[CrawlResult](r, func(item CrawlResult) error {
			return writeCrawlItemMarkdown(item, w)
		})
	case model.KindResearch:
		item, err := parseSingleReader[ResearchResult](r)
		if err != nil {
			return err
		}
		return writeResearchMarkdown(item, w)
	default:
		return apperrors.Internal("unknown job kind")
	}
}

// writeScrapeMarkdown writes a single scrape result to the writer in Markdown format.
func writeScrapeMarkdown(item ScrapeResult, w io.Writer) error {
	title := item.Title
	desc := item.Metadata.Description
	text := item.Text
	if item.Normalized.Title != "" {
		title = item.Normalized.Title
	}
	if item.Normalized.Description != "" {
		desc = item.Normalized.Description
	}
	if item.Normalized.Text != "" {
		text = item.Normalized.Text
	}

	fmt.Fprintf(w, "# %s\n\n", safe(title, "Scrape Result"))
	fmt.Fprintf(w, "- **URL**: %s\n", item.URL)
	fmt.Fprintf(w, "- **Status**: %d\n", item.Status)
	if desc != "" {
		fmt.Fprintf(w, "- **Description**: %s\n", desc)
	}
	fmt.Fprint(w, "\n## Extracted Fields\n")
	fieldKeys := make([]string, 0, len(item.Normalized.Fields))
	for k := range item.Normalized.Fields {
		fieldKeys = append(fieldKeys, k)
	}
	sort.Strings(fieldKeys)
	for _, k := range fieldKeys {
		v := item.Normalized.Fields[k]
		fmt.Fprintf(w, "- **%s**: %s\n", k, strings.Join(v.Values, ", "))
	}
	fmt.Fprint(w, "\n## Text Content\n"+text+"\n")
	return nil
}

// writeCrawlItemMarkdown writes a single crawl result to the writer in Markdown format.
func writeCrawlItemMarkdown(item CrawlResult, w io.Writer) error {
	title := item.Title
	if item.Normalized.Title != "" {
		title = item.Normalized.Title
	}
	fmt.Fprintf(w, "## %s\n\n- URL: %s\n- Status: %d\n", safe(title, item.URL), item.URL, item.Status)
	if len(item.Normalized.Fields) > 0 {
		fmt.Fprint(w, "\n### Fields\n")
		fieldKeys := make([]string, 0, len(item.Normalized.Fields))
		for k := range item.Normalized.Fields {
			fieldKeys = append(fieldKeys, k)
		}
		sort.Strings(fieldKeys)
		for _, k := range fieldKeys {
			v := item.Normalized.Fields[k]
			fmt.Fprintf(w, "- **%s**: %s\n", k, strings.Join(v.Values, ", "))
		}
	}
	fmt.Fprint(w, "\n")
	return nil
}

// writeResearchMarkdown writes a research result to the writer in Markdown format.
func writeResearchMarkdown(item ResearchResult, w io.Writer) error {
	fmt.Fprint(w, "# Research Report\n\n")
	fmt.Fprintf(w, "**Query:** %s\n", item.Query)
	fmt.Fprintf(w, "**Confidence:** %.2f\n\n", item.Confidence)
	fmt.Fprint(w, "## Summary\n\n"+item.Summary+"\n")
	if len(item.Clusters) > 0 {
		fmt.Fprint(w, "## Evidence Clusters\n\n")
		for _, cluster := range item.Clusters {
			fmt.Fprintf(w, "- **%s** (confidence %.2f, %d items)\n", safe(cluster.Label, cluster.ID), cluster.Confidence, len(cluster.Evidence))
		}
		fmt.Fprint(w, "\n")
	}
	if len(item.Citations) > 0 {
		fmt.Fprint(w, "## Citations\n\n")
		for _, citation := range item.Citations {
			target := citation.Canonical
			if citation.Anchor != "" {
				target = citation.Canonical + "#" + citation.Anchor
			}
			fmt.Fprintf(w, "- %s\n", target)
		}
		fmt.Fprint(w, "\n")
	}
	fmt.Fprint(w, "## Evidence\n\n")
	for _, ev := range item.Evidence {
		fmt.Fprintf(w, "- **%s** (%s) — score %.2f, confidence %.2f\n  \n  %s\n", safe(ev.Title, ev.URL), ev.URL, ev.Score, ev.Confidence, ev.Snippet)
	}
	return nil
}
