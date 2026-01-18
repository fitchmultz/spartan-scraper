package exporter

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"spartan-scraper/internal/extract"
	"spartan-scraper/internal/model"
)

type ScrapeResult struct {
	URL      string `json:"url"`
	Status   int    `json:"status"`
	Title    string `json:"title"`
	Text     string `json:"text"`
	Metadata struct {
		Description string `json:"description"`
	} `json:"metadata"`
	Normalized extract.NormalizedDocument `json:"normalized"`
}

type CrawlResult struct {
	URL        string                     `json:"url"`
	Status     int                        `json:"status"`
	Title      string                     `json:"title"`
	Text       string                     `json:"text"`
	Normalized extract.NormalizedDocument `json:"normalized"`
}

type ResearchResult struct {
	Query      string  `json:"query"`
	Summary    string  `json:"summary"`
	Confidence float64 `json:"confidence"`
	Evidence   []struct {
		URL         string  `json:"url"`
		Title       string  `json:"title"`
		Snippet     string  `json:"snippet"`
		Score       float64 `json:"score"`
		SimHash     uint64  `json:"simhash"`
		ClusterID   string  `json:"clusterId"`
		Confidence  float64 `json:"confidence"`
		CitationURL string  `json:"citationUrl"`
	} `json:"evidence"`
	Clusters []struct {
		ID         string  `json:"id"`
		Label      string  `json:"label"`
		Confidence float64 `json:"confidence"`
		Evidence   []struct {
			URL         string  `json:"url"`
			Title       string  `json:"title"`
			Snippet     string  `json:"snippet"`
			Score       float64 `json:"score"`
			SimHash     uint64  `json:"simhash"`
			ClusterID   string  `json:"clusterId"`
			Confidence  float64 `json:"confidence"`
			CitationURL string  `json:"citationUrl"`
		} `json:"evidence"`
	} `json:"clusters"`
	Citations []struct {
		URL       string `json:"url"`
		Anchor    string `json:"anchor"`
		Canonical string `json:"canonical"`
	} `json:"citations"`
}

func Export(job model.Job, raw []byte, format string) (string, error) {
	switch format {
	case "json", "jsonl":
		return string(raw), nil
	case "md":
		return exportMarkdown(job, raw)
	case "csv":
		return exportCSV(job, raw)
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

func exportMarkdown(job model.Job, raw []byte) (string, error) {
	switch job.Kind {
	case model.KindScrape:
		item, err := parseSingle[ScrapeResult](raw)
		if err != nil {
			return "", err
		}
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

		var b strings.Builder
		b.WriteString(fmt.Sprintf("# %s\n\n", safe(title, "Scrape Result")))
		b.WriteString(fmt.Sprintf("- **URL**: %s\n", item.URL))
		b.WriteString(fmt.Sprintf("- **Status**: %d\n", item.Status))
		if desc != "" {
			b.WriteString(fmt.Sprintf("- **Description**: %s\n", desc))
		}
		b.WriteString("\n## Extracted Fields\n\n")
		for k, v := range item.Normalized.Fields {
			b.WriteString(fmt.Sprintf("- **%s**: %s\n", k, strings.Join(v.Values, ", ")))
		}
		b.WriteString("\n## Text Content\n\n")
		b.WriteString(text + "\n")
		return b.String(), nil
	case model.KindCrawl:
		items, err := parseLines[CrawlResult](raw)
		if err != nil {
			return "", err
		}
		var b strings.Builder
		b.WriteString("# Crawl Results\n\n")
		for _, item := range items {
			title := item.Title
			if item.Normalized.Title != "" {
				title = item.Normalized.Title
			}
			b.WriteString(fmt.Sprintf("## %s\n\n- URL: %s\n- Status: %d\n", safe(title, item.URL), item.URL, item.Status))
			if len(item.Normalized.Fields) > 0 {
				b.WriteString("\n### Fields\n")
				for k, v := range item.Normalized.Fields {
					b.WriteString(fmt.Sprintf("- **%s**: %s\n", k, strings.Join(v.Values, ", ")))
				}
			}
			b.WriteString("\n")
		}
		return b.String(), nil
	case model.KindResearch:
		item, err := parseSingle[ResearchResult](raw)
		if err != nil {
			return "", err
		}
		var b strings.Builder
		b.WriteString("# Research Report\n\n")
		b.WriteString(fmt.Sprintf("**Query:** %s\n", item.Query))
		b.WriteString(fmt.Sprintf("**Confidence:** %.2f\n\n", item.Confidence))
		b.WriteString("## Summary\n\n")
		b.WriteString(item.Summary + "\n\n")
		if len(item.Clusters) > 0 {
			b.WriteString("## Evidence Clusters\n\n")
			for _, cluster := range item.Clusters {
				b.WriteString(fmt.Sprintf("- **%s** (confidence %.2f, %d items)\n", safe(cluster.Label, cluster.ID), cluster.Confidence, len(cluster.Evidence)))
			}
			b.WriteString("\n")
		}
		if len(item.Citations) > 0 {
			b.WriteString("## Citations\n\n")
			for _, citation := range item.Citations {
				target := citation.Canonical
				if citation.Anchor != "" {
					target = citation.Canonical + "#" + citation.Anchor
				}
				b.WriteString(fmt.Sprintf("- %s\n", target))
			}
			b.WriteString("\n")
		}
		b.WriteString("## Evidence\n\n")
		for _, ev := range item.Evidence {
			b.WriteString(fmt.Sprintf("- **%s** (%s) — score %.2f, confidence %.2f\n  \n  %s\n", safe(ev.Title, ev.URL), ev.URL, ev.Score, ev.Confidence, ev.Snippet))
		}
		return b.String(), nil
	default:
		return "", errors.New("unknown job kind")
	}
}

func exportCSV(job model.Job, raw []byte) (string, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	switch job.Kind {
	case model.KindScrape:
		item, err := parseSingle[ScrapeResult](raw)
		if err != nil {
			return "", err
		}
		headers := []string{"url", "status", "title", "description"}
		fieldNames := make([]string, 0, len(item.Normalized.Fields))
		for k := range item.Normalized.Fields {
			fieldNames = append(fieldNames, k)
			headers = append(headers, "field_"+k)
		}
		_ = writer.Write(headers)

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
		_ = writer.Write(row)
	case model.KindCrawl:
		items, err := parseLines[CrawlResult](raw)
		if err != nil {
			return "", err
		}
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

		headers := []string{"url", "status", "title"}
		for _, k := range fieldNames {
			headers = append(headers, "field_"+k)
		}
		_ = writer.Write(headers)

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
			_ = writer.Write(row)
		}
	case model.KindResearch:
		item, err := parseSingle[ResearchResult](raw)
		if err != nil {
			return "", err
		}
		_ = writer.Write([]string{"query", "summary", "confidence"})
		_ = writer.Write([]string{item.Query, item.Summary, fmt.Sprintf("%.2f", item.Confidence)})
		_ = writer.Write([]string{})
		_ = writer.Write([]string{"url", "title", "score", "confidence", "cluster_id", "citation_url", "snippet"})
		for _, ev := range item.Evidence {
			_ = writer.Write([]string{
				ev.URL,
				ev.Title,
				fmt.Sprintf("%.2f", ev.Score),
				fmt.Sprintf("%.2f", ev.Confidence),
				ev.ClusterID,
				ev.CitationURL,
				ev.Snippet,
			})
		}
	default:
		return "", errors.New("unknown job kind")
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func parseSingle[T any](raw []byte) (T, error) {
	var out T
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if err := json.Unmarshal([]byte(line), &out); err != nil {
			return out, err
		}
		return out, nil
	}
	return out, errors.New("no content")
}

func parseLines[T any](raw []byte) ([]T, error) {
	items := make([]T, 0)
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var item T
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func safe(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
