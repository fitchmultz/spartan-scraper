// Package exporter provides functionality for exporting job results to various formats.
// It supports JSON, JSONL, Markdown, and CSV output formats with both buffered
// and streaming interfaces for memory-efficient processing of large results.
package exporter

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
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

// Export exports job results to the specified format and returns the output as a string.
// For large result files, consider using ExportStream instead to avoid loading the entire
// output into memory.
func Export(job model.Job, raw []byte, format string) (string, error) {
	var buf strings.Builder
	if err := ExportStream(job, bytes.NewReader(raw), format, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ExportStream exports job results to the specified format, writing the output directly
// to the provided writer. This is more memory-efficient for large result files as it
// streams the input and processes it incrementally where possible.
func ExportStream(job model.Job, r io.Reader, format string, w io.Writer) error {
	switch format {
	case "json":
		return exportJSONStream(job, r, w)
	case "jsonl":
		return exportJSONLStream(r, w)
	case "md":
		return exportMarkdownStream(job, r, w)
	case "csv":
		return exportCSVStream(job, r, w)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func exportJSON(job model.Job, raw []byte) (string, error) {
	switch job.Kind {
	case model.KindScrape:
		item, err := parseSingle[ScrapeResult](raw)
		if err != nil {
			return "", err
		}
		data, err := json.MarshalIndent(item, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil
	case model.KindCrawl:
		items, err := parseLines[CrawlResult](raw)
		if err != nil {
			return "", err
		}
		data, err := json.MarshalIndent(items, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil
	case model.KindResearch:
		item, err := parseSingle[ResearchResult](raw)
		if err != nil {
			return "", err
		}
		data, err := json.MarshalIndent(item, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil
	default:
		return "", errors.New("unknown job kind")
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
		fieldKeys := make([]string, 0, len(item.Normalized.Fields))
		for k := range item.Normalized.Fields {
			fieldKeys = append(fieldKeys, k)
		}
		sort.Strings(fieldKeys)
		for _, k := range fieldKeys {
			v := item.Normalized.Fields[k]
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
				fieldKeys := make([]string, 0, len(item.Normalized.Fields))
				for k := range item.Normalized.Fields {
					fieldKeys = append(fieldKeys, k)
				}
				sort.Strings(fieldKeys)
				for _, k := range fieldKeys {
					v := item.Normalized.Fields[k]
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
		}
		sort.Strings(fieldNames)
		for _, k := range fieldNames {
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
		sort.Strings(fieldNames)

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
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
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
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
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

// Streaming export functions

func exportJSONLStream(r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func exportJSONStream(job model.Job, r io.Reader, w io.Writer) error {
	// For JSON output, we need to parse the input first since we need to
	// re-encode it as proper JSON (not JSONL). This still benefits from
	// streaming the input parsing, but the final output is buffered.
	var data []byte
	var err error

	switch job.Kind {
	case model.KindScrape:
		item, err := parseSingleReader[ScrapeResult](r)
		if err != nil {
			return err
		}
		data, err = json.MarshalIndent(item, "", "  ")
		if err != nil {
			return err
		}
	case model.KindCrawl:
		items, err := parseLinesReader[CrawlResult](r)
		if err != nil {
			return err
		}
		data, err = json.MarshalIndent(items, "", "  ")
		if err != nil {
			return err
		}
	case model.KindResearch:
		item, err := parseSingleReader[ResearchResult](r)
		if err != nil {
			return err
		}
		data, err = json.MarshalIndent(item, "", "  ")
		if err != nil {
			return err
		}
	default:
		return errors.New("unknown job kind")
	}

	_, err = w.Write(data)
	return err
}

func exportMarkdownStream(job model.Job, r io.Reader, w io.Writer) error {
	switch job.Kind {
	case model.KindScrape:
		item, err := parseSingleReader[ScrapeResult](r)
		if err != nil {
			return err
		}
		return writeScrapeMarkdown(item, w)
	case model.KindCrawl:
		items, err := parseLinesReader[CrawlResult](r)
		if err != nil {
			return err
		}
		return writeCrawlMarkdown(items, w)
	case model.KindResearch:
		item, err := parseSingleReader[ResearchResult](r)
		if err != nil {
			return err
		}
		return writeResearchMarkdown(item, w)
	default:
		return errors.New("unknown job kind")
	}
}

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

// Helper functions for streaming that use io.Reader instead of []byte

func parseSingleReader[T any](r io.Reader) (T, error) {
	var out T
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
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

func parseLinesReader[T any](r io.Reader) ([]T, error) {
	items := make([]T, 0)
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
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

// Writer-based output functions for markdown

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

func writeCrawlMarkdown(items []CrawlResult, w io.Writer) error {
	fmt.Fprint(w, "# Crawl Results\n\n")
	for _, item := range items {
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
	}
	return nil
}

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

// Writer-based output functions for CSV

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
