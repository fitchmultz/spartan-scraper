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
func exportMarkdownStream(job model.Job, r io.Reader, w io.Writer, shape ShapeConfig) error {
	switch job.Kind {
	case model.KindScrape:
		item, err := parseSingleReader[ScrapeResult](r)
		if err != nil {
			return err
		}
		if HasMeaningfulShape(shape) {
			return writeScrapeMarkdownShaped(item, w, shape)
		}
		return writeScrapeMarkdown(item, w)
	case model.KindCrawl:
		if HasMeaningfulShape(shape) {
			fmt.Fprintf(w, "# %s\n\n", safe(firstNonEmpty(shapeMarkdownTitle(shape), "Crawl Export"), "Crawl Export"))
			return scanReader[CrawlResult](r, func(item CrawlResult) error {
				return writeCrawlItemMarkdownShaped(item, w, shape)
			})
		}
		fmt.Fprint(w, "# Crawl Results\n\n")
		return scanReader[CrawlResult](r, func(item CrawlResult) error {
			return writeCrawlItemMarkdown(item, w)
		})
	case model.KindResearch:
		item, err := parseSingleReader[ResearchResult](r)
		if err != nil {
			return err
		}
		if HasMeaningfulShape(shape) {
			return writeResearchMarkdownShaped(item, w, shape)
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

func writeScrapeMarkdownShaped(item ScrapeResult, w io.Writer, shape ShapeConfig) error {
	title := firstNonEmpty(shapeMarkdownTitle(shape), scrapeShapeValue(item, "title", shape), "Scrape Export")
	fmt.Fprintf(w, "# %s\n\n", safe(title, "Scrape Export"))
	summaryFields := selectFields(shape.SummaryFields, nonEmptyFields([]string{"title", "description", "url", "status"}, func(key string) string {
		return scrapeShapeValue(item, key, shape)
	}))
	writeMarkdownSummaryFields(w, shape, summaryFields, func(key string) string {
		return scrapeShapeValue(item, key, shape)
	})
	normalizedFields := selectFields(shape.NormalizedFields, scrapeNormalizedFieldRefs(item))
	if len(normalizedFields) > 0 {
		fmt.Fprint(w, "## Extracted Fields\n\n")
		for _, key := range normalizedFields {
			fmt.Fprintf(w, "- **%s**: %s\n", labelForShapeField(shape, key), safe(scrapeShapeValue(item, key, shape), shapeEmptyValue(shape)))
		}
		fmt.Fprint(w, "\n")
	}
	if text := scrapeShapeValue(item, "text", shape); strings.TrimSpace(text) != "" {
		fmt.Fprint(w, "## Text Content\n\n"+text+"\n")
	}
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

func writeCrawlItemMarkdownShaped(item CrawlResult, w io.Writer, shape ShapeConfig) error {
	title := firstNonEmpty(crawlShapeValue(item, "title", shape), item.URL)
	fmt.Fprintf(w, "## %s\n\n", safe(title, item.URL))
	summaryFields := selectFields(shape.SummaryFields, nonEmptyFields([]string{"title", "url", "status"}, func(key string) string {
		return crawlShapeValue(item, key, shape)
	}))
	writeMarkdownSummaryFields(w, shape, summaryFields, func(key string) string {
		return crawlShapeValue(item, key, shape)
	})
	normalizedFields := selectFields(shape.NormalizedFields, []string{})
	if len(normalizedFields) > 0 {
		fmt.Fprint(w, "### Extracted Fields\n\n")
		for _, key := range normalizedFields {
			fmt.Fprintf(w, "- **%s**: %s\n", labelForShapeField(shape, key), safe(crawlShapeValue(item, key, shape), shapeEmptyValue(shape)))
		}
		fmt.Fprint(w, "\n")
	}
	return nil
}

// writeResearchMarkdown writes a research result to the writer in Markdown format.
func writeResearchMarkdown(item ResearchResult, w io.Writer) error {
	fmt.Fprint(w, "# Research Report\n\n")
	fmt.Fprintf(w, "**Query:** %s\n", item.Query)
	fmt.Fprintf(w, "**Confidence:** %.2f\n\n", item.Confidence)
	fmt.Fprint(w, "## Summary\n\n"+item.Summary+"\n")
	if item.Agentic != nil {
		fmt.Fprint(w, "\n## Agentic Research\n\n")
		fmt.Fprintf(w, "**Status:** %s\n", safe(item.Agentic.Status, "unknown"))
		if item.Agentic.Confidence > 0 {
			fmt.Fprintf(w, "**Confidence:** %.2f\n", item.Agentic.Confidence)
		}
		if item.Agentic.Provider != "" || item.Agentic.Model != "" {
			fmt.Fprintf(w, "**Route:** %s/%s\n", safe(item.Agentic.Provider, "unknown"), safe(item.Agentic.Model, "unknown"))
		}
		if item.Agentic.Summary != "" {
			fmt.Fprint(w, "\n"+item.Agentic.Summary+"\n")
		}
		for _, section := range []struct {
			Title string
			Items []string
		}{
			{Title: "Focus Areas", Items: item.Agentic.FocusAreas},
			{Title: "Key Findings", Items: item.Agentic.KeyFindings},
			{Title: "Open Questions", Items: item.Agentic.OpenQuestions},
			{Title: "Recommended Next Steps", Items: item.Agentic.RecommendedNextSteps},
			{Title: "Follow-Up URLs", Items: item.Agentic.FollowUpUrls},
		} {
			if len(section.Items) == 0 {
				continue
			}
			fmt.Fprintf(w, "\n### %s\n\n", section.Title)
			for _, entry := range section.Items {
				fmt.Fprintf(w, "- %s\n", entry)
			}
		}
		if item.Agentic.Error != "" {
			fmt.Fprintf(w, "\n**Error:** %s\n", item.Agentic.Error)
		}
		fmt.Fprint(w, "\n")
	}
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

func writeResearchMarkdownShaped(item ResearchResult, w io.Writer, shape ShapeConfig) error {
	title := firstNonEmpty(shapeMarkdownTitle(shape), item.Query, "Research Export")
	fmt.Fprintf(w, "# %s\n\n", safe(title, "Research Export"))
	summaryFields := selectFields(shape.SummaryFields, nonEmptyFields([]string{"query", "summary", "confidence", "agentic.status", "agentic.summary"}, func(key string) string {
		return researchShapeValue(item, key, shape)
	}))
	writeMarkdownSummaryFields(w, shape, summaryFields, func(key string) string {
		return researchShapeValue(item, key, shape)
	})
	evidenceFields := selectFields(shape.EvidenceFields, []string{"evidence.url", "evidence.title", "evidence.score", "evidence.confidence", "evidence.snippet"})
	if len(item.Evidence) > 0 {
		fmt.Fprint(w, "## Evidence\n\n")
		for _, ev := range item.Evidence {
			primary := firstNonEmpty(researchEvidenceShapeValue(ev, "evidence.title", shape), researchEvidenceShapeValue(ev, "evidence.url", shape), "Evidence")
			fmt.Fprintf(w, "### %s\n\n", safe(primary, "Evidence"))
			for _, key := range evidenceFields {
				fmt.Fprintf(w, "- **%s**: %s\n", labelForShapeField(shape, key), safe(researchEvidenceShapeValue(ev, key, shape), shapeEmptyValue(shape)))
			}
			fmt.Fprint(w, "\n")
		}
	}
	return nil
}

func writeMarkdownSummaryFields(w io.Writer, shape ShapeConfig, fields []string, resolve func(string) string) {
	if len(fields) == 0 {
		return
	}
	fmt.Fprint(w, "## Summary\n\n")
	for _, field := range fields {
		fmt.Fprintf(w, "- **%s**: %s\n", labelForShapeField(shape, field), safe(resolve(field), shapeEmptyValue(shape)))
	}
	fmt.Fprint(w, "\n")
}
