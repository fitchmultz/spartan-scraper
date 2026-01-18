package exporter

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

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
}

type CrawlResult struct {
	URL    string `json:"url"`
	Status int    `json:"status"`
	Title  string `json:"title"`
	Text   string `json:"text"`
}

type ResearchResult struct {
	Query    string `json:"query"`
	Summary  string `json:"summary"`
	Evidence []struct {
		URL     string  `json:"url"`
		Title   string  `json:"title"`
		Snippet string  `json:"snippet"`
		Score   float64 `json:"score"`
	} `json:"evidence"`
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
		return fmt.Sprintf("# Scrape Result\n\n- URL: %s\n- Status: %d\n- Title: %s\n- Description: %s\n\n## Extracted Text\n\n%s\n", item.URL, item.Status, item.Title, item.Metadata.Description, item.Text), nil
	case model.KindCrawl:
		items, err := parseLines[CrawlResult](raw)
		if err != nil {
			return "", err
		}
		var b strings.Builder
		b.WriteString("# Crawl Results\n\n")
		for _, item := range items {
			b.WriteString(fmt.Sprintf("## %s\n\n- URL: %s\n- Status: %d\n\n%s\n\n", safe(item.Title, item.URL), item.URL, item.Status, item.Text))
		}
		return b.String(), nil
	case model.KindResearch:
		item, err := parseSingle[ResearchResult](raw)
		if err != nil {
			return "", err
		}
		var b strings.Builder
		b.WriteString("# Research Report\n\n")
		b.WriteString(fmt.Sprintf("**Query:** %s\n\n", item.Query))
		b.WriteString("## Summary\n\n")
		b.WriteString(item.Summary + "\n\n")
		b.WriteString("## Evidence\n\n")
		for _, ev := range item.Evidence {
			b.WriteString(fmt.Sprintf("- **%s** (%s) — score %.2f\n  \n  %s\n", safe(ev.Title, ev.URL), ev.URL, ev.Score, ev.Snippet))
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
		_ = writer.Write([]string{"url", "status", "title", "description"})
		_ = writer.Write([]string{item.URL, fmt.Sprint(item.Status), item.Title, item.Metadata.Description})
	case model.KindCrawl:
		items, err := parseLines[CrawlResult](raw)
		if err != nil {
			return "", err
		}
		_ = writer.Write([]string{"url", "status", "title"})
		for _, item := range items {
			_ = writer.Write([]string{item.URL, fmt.Sprint(item.Status), item.Title})
		}
	case model.KindResearch:
		item, err := parseSingle[ResearchResult](raw)
		if err != nil {
			return "", err
		}
		_ = writer.Write([]string{"query", "summary"})
		_ = writer.Write([]string{item.Query, item.Summary})
		_ = writer.Write([]string{})
		_ = writer.Write([]string{"url", "title", "score", "snippet"})
		for _, ev := range item.Evidence {
			_ = writer.Write([]string{ev.URL, ev.Title, fmt.Sprintf("%.2f", ev.Score), ev.Snippet})
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
