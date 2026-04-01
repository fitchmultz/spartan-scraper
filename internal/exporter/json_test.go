// Package exporter provides exporter functionality for Spartan Scraper.
//
// Purpose:
// - Verify json test behavior for package exporter.
//
// Responsibilities:
// - Define focused Go test coverage, fixtures, and assertions for the package behavior exercised here.
//
// Scope:
// - Automated test coverage only; production behavior stays in non-test package files.
//
// Usage:
// - Run with `go test` for package `exporter` or through `make test-ci`/`make ci`.
//
// Invariants/Assumptions:
// - Tests should remain deterministic and describe the package contract they protect.

package exporter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestExportJSONForScrapeJob(t *testing.T) {
	raw := []byte(sampleScrapeResultJSONL())
	job := model.Job{Kind: model.KindScrape}

	result, err := Export(job, raw, "json")
	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	var scrapeResult ScrapeResult
	if err := json.Unmarshal([]byte(result), &scrapeResult); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	if strings.HasPrefix(strings.TrimSpace(result), "[") {
		t.Error("Expected JSON object, got JSON array")
	}

	if !strings.Contains(result, "\n  ") {
		t.Error("Expected 2-space indentation in JSON output")
	}

	expectedURL := "https://example.com"
	if scrapeResult.URL != expectedURL {
		t.Errorf("Expected URL %q, got %q", expectedURL, scrapeResult.URL)
	}

	expectedStatus := 200
	if scrapeResult.Status != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, scrapeResult.Status)
	}
}

func TestExportJSONForCrawlJob(t *testing.T) {
	raw := []byte(sampleCrawlResultJSONL(3))
	job := model.Job{Kind: model.KindCrawl}

	result, err := Export(job, raw, "json")
	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	var crawlResults []CrawlResult
	if err := json.Unmarshal([]byte(result), &crawlResults); err != nil {
		t.Fatalf("Output is not valid JSON array: %v", err)
	}

	if !strings.HasPrefix(strings.TrimSpace(result), "[") {
		t.Error("Expected JSON array, got JSON object")
	}

	expectedCount := 3
	if len(crawlResults) != expectedCount {
		t.Errorf("Expected %d items, got %d", expectedCount, len(crawlResults))
	}

	if !strings.Contains(result, "\n  ") {
		t.Error("Expected 2-space indentation in JSON output")
	}

	for i, item := range crawlResults {
		expectedURL := "https://example.com/page" + string(rune('1'+i))
		if item.URL != expectedURL {
			t.Errorf("Item %d: expected URL %q, got %q", i, expectedURL, item.URL)
		}
	}
}

func TestExportJSONForResearchJob(t *testing.T) {
	raw := []byte(sampleResearchResultJSONL())
	job := model.Job{Kind: model.KindResearch}

	result, err := Export(job, raw, "json")
	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	var researchResult ResearchResult
	if err := json.Unmarshal([]byte(result), &researchResult); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	if strings.HasPrefix(strings.TrimSpace(result), "[") {
		t.Error("Expected JSON object, got JSON array")
	}

	if !strings.Contains(result, "\n  ") {
		t.Error("Expected 2-space indentation in JSON output")
	}

	expectedQuery := "test query"
	if researchResult.Query != expectedQuery {
		t.Errorf("Expected query %q, got %q", expectedQuery, researchResult.Query)
	}

	if len(researchResult.Evidence) == 0 {
		t.Error("Expected evidence items, got none")
	}
}

func TestExportJSONEmptyInput(t *testing.T) {
	t.Run("Scrape job empty input returns error", func(t *testing.T) {
		job := model.Job{Kind: model.KindScrape}
		raw := []byte("")

		_, err := Export(job, raw, "json")
		if err == nil {
			t.Error("Expected error for empty input, got nil")
		}
		if !strings.Contains(err.Error(), "no content") {
			t.Errorf("Expected 'no content' error, got: %v", err)
		}
	})

	t.Run("Research job empty input returns error", func(t *testing.T) {
		job := model.Job{Kind: model.KindResearch}
		raw := []byte("")

		_, err := Export(job, raw, "json")
		if err == nil {
			t.Error("Expected error for empty input, got nil")
		}
		if !strings.Contains(err.Error(), "no content") {
			t.Errorf("Expected 'no content' error, got: %v", err)
		}
	})

	t.Run("Crawl job empty input returns empty JSON array", func(t *testing.T) {
		job := model.Job{Kind: model.KindCrawl}
		raw := []byte("")

		result, err := Export(job, raw, "json")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if strings.TrimSpace(result) != "[]" {
			t.Errorf("Expected empty JSON array '[]', got: %q", result)
		}
	})
}

func TestExportJSONInvalidJSONInput(t *testing.T) {
	tests := []struct {
		name string
		kind model.Kind
		raw  string
	}{
		{
			name: "Scrape job invalid JSON",
			kind: model.KindScrape,
			raw:  `{invalid json}`,
		},
		{
			name: "Crawl job invalid JSON",
			kind: model.KindCrawl,
			raw:  `{invalid json}\n{"url": "valid"}\ninvalid`,
		},
		{
			name: "Research job invalid JSON",
			kind: model.KindResearch,
			raw:  `not json at all`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := model.Job{Kind: tt.kind}
			_, err := Export(job, []byte(tt.raw), "json")
			if err == nil {
				t.Error("Expected error for invalid JSON input, got nil")
			}
		})
	}
}

func TestExportJSONSingleCrawlResultStillArray(t *testing.T) {
	raw := []byte(sampleCrawlResultJSONL(1))
	job := model.Job{Kind: model.KindCrawl}

	result, err := Export(job, raw, "json")
	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	var crawlResults []CrawlResult
	if err := json.Unmarshal([]byte(result), &crawlResults); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	if len(crawlResults) != 1 {
		t.Errorf("Expected 1 item in array, got %d", len(crawlResults))
	}

	if !strings.HasPrefix(strings.TrimSpace(result), "[") {
		t.Error("Expected JSON array even for single item")
	}
}

func TestExportCrawlResultWithBlankLines(t *testing.T) {
	raw := []byte(sampleCrawlResultJSONL(2) + "\n\n\n" + sampleCrawlResultJSONL(1))
	job := model.Job{Kind: model.KindCrawl}

	result, err := Export(job, raw, "json")
	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	var crawlResults []CrawlResult
	if err := json.Unmarshal([]byte(result), &crawlResults); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	expectedCount := 3
	if len(crawlResults) != expectedCount {
		t.Errorf("Expected %d items (blank lines should be skipped), got %d", expectedCount, len(crawlResults))
	}
}

func TestExportJSONUnknownJobKind(t *testing.T) {
	raw := []byte(`{"test": "data"}`)
	job := model.Job{Kind: model.Kind("unknown")}

	_, err := Export(job, raw, "json")
	if err == nil {
		t.Error("Expected error for unknown job kind, got nil")
	}

	if !strings.Contains(err.Error(), "unknown job kind") {
		t.Errorf("Expected 'unknown job kind' error, got: %v", err)
	}
}

func TestParseLargeJSONLLine(t *testing.T) {
	tests := []struct {
		name        string
		sizeKB      int
		exceeds64KB bool
	}{
		{"Small line (1KB)", 1, false},
		{"Medium line (50KB)", 50, false},
		{"Large line (100KB) - exceeds default buffer", 100, true},
		{"Very large line (500KB)", 500, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			largeText := strings.Repeat("x", tt.sizeKB*1024)
			jsonLine := fmt.Sprintf(`{"url":"https://example.com","status":200,"title":"Large Content","text":"%s"}`, largeText)

			t.Run("parseSingle", func(t *testing.T) {
				job := model.Job{Kind: model.KindScrape}
				result, err := Export(job, []byte(jsonLine), "json")
				if err != nil {
					t.Fatalf("Export() failed for %d KB line: %v", tt.sizeKB, err)
				}

				var scrapeResult ScrapeResult
				if err := json.Unmarshal([]byte(result), &scrapeResult); err != nil {
					t.Fatalf("Output is not valid JSON: %v", err)
				}

				if scrapeResult.Text != largeText {
					t.Errorf("Text content not preserved: got length %d, want %d", len(scrapeResult.Text), len(largeText))
				}
			})

			t.Run("parseLines", func(t *testing.T) {
				jsonlLines := []string{
					fmt.Sprintf(`{"url":"https://example.com/page1","status":200,"title":"Page 1","text":"%s"}`, largeText),
					fmt.Sprintf(`{"url":"https://example.com/page2","status":200,"title":"Page 2","text":"%s"}`, largeText),
					fmt.Sprintf(`{"url":"https://example.com/page3","status":200,"title":"Page 3","text":"%s"}`, largeText),
				}
				raw := []byte(strings.Join(jsonlLines, "\n"))

				job := model.Job{Kind: model.KindCrawl}
				result, err := Export(job, raw, "json")
				if err != nil {
					t.Fatalf("Export() failed for %d KB lines: %v", tt.sizeKB, err)
				}

				var crawlResults []CrawlResult
				if err := json.Unmarshal([]byte(result), &crawlResults); err != nil {
					t.Fatalf("Output is not valid JSON: %v", err)
				}

				if len(crawlResults) != 3 {
					t.Errorf("Expected 3 items, got %d", len(crawlResults))
				}

				for i, item := range crawlResults {
					if item.Text != largeText {
						t.Errorf("Item %d: text content not preserved: got length %d, want %d", i, len(item.Text), len(largeText))
					}
				}
			})
		})
	}
}

func TestExportStreamJSON(t *testing.T) {
	tests := []struct {
		name string
		kind model.Kind
		raw  string
	}{
		{"Scrape job", model.KindScrape, sampleScrapeResultJSONL()},
		{"Crawl job", model.KindCrawl, sampleCrawlResultJSONL(3)},
		{"Research job", model.KindResearch, sampleResearchResultJSONL()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := model.Job{Kind: tt.kind}
			var buf bytes.Buffer

			err := ExportStream(job, strings.NewReader(tt.raw), "json", &buf)
			if err != nil {
				t.Fatalf("ExportStream() failed: %v", err)
			}

			result := buf.String()

			var decoded interface{}
			if err := json.Unmarshal([]byte(result), &decoded); err != nil {
				t.Fatalf("Output is not valid JSON: %v", err)
			}

			if !strings.Contains(result, "\n  ") {
				t.Error("Expected 2-space indentation in JSON output")
			}
		})
	}
}

func TestExportStreamLargeFile(t *testing.T) {
	lines := make([]string, 100)
	for i := 0; i < 100; i++ {
		largeText := strings.Repeat("x", 1024)
		lines[i] = fmt.Sprintf(`{"url":"https://example.com/page%d","status":200,"title":"Page %d","text":"%s"}`, i, i, largeText)
	}
	raw := strings.Join(lines, "\n")

	job := model.Job{Kind: model.KindCrawl}
	var buf bytes.Buffer

	err := ExportStream(job, strings.NewReader(raw), "json", &buf)
	if err != nil {
		t.Fatalf("ExportStream() failed for large file: %v", err)
	}

	result := buf.String()

	var crawlResults []CrawlResult
	if err := json.Unmarshal([]byte(result), &crawlResults); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	if len(crawlResults) != 100 {
		t.Errorf("Expected 100 items, got %d", len(crawlResults))
	}
}

func TestExportStreamEmptyInput(t *testing.T) {
	t.Run("Scrape job empty input returns error", func(t *testing.T) {
		job := model.Job{Kind: model.KindScrape}
		var buf bytes.Buffer

		err := ExportStream(job, strings.NewReader(""), "json", &buf)
		if err == nil {
			t.Error("Expected error for empty input, got nil")
		}
		if !strings.Contains(err.Error(), "no content") {
			t.Errorf("Expected 'no content' error, got: %v", err)
		}
	})

	t.Run("Crawl job empty input returns empty JSON array", func(t *testing.T) {
		job := model.Job{Kind: model.KindCrawl}
		var buf bytes.Buffer

		err := ExportStream(job, strings.NewReader(""), "json", &buf)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		result := buf.String()
		if strings.TrimSpace(result) != "[]" {
			t.Errorf("Expected empty JSON array '[]', got: %q", result)
		}
	})
}

func TestExportStreamMatchesExportJSON(t *testing.T) {
	tests := []struct {
		name string
		kind model.Kind
		raw  string
	}{
		{"Scrape", model.KindScrape, sampleScrapeResultJSONL()},
		{"Crawl", model.KindCrawl, sampleCrawlResultJSONL(3)},
		{"Research", model.KindResearch, sampleResearchResultJSONL()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := model.Job{Kind: tt.kind}

			resultOld, err := Export(job, []byte(tt.raw), "json")
			if err != nil {
				t.Fatalf("Export() failed: %v", err)
			}

			var buf bytes.Buffer
			err = ExportStream(job, strings.NewReader(tt.raw), "json", &buf)
			if err != nil {
				t.Fatalf("ExportStream() failed: %v", err)
			}
			resultNew := buf.String()

			if resultOld != resultNew {
				t.Errorf("Export and ExportStream produced different output\nOld (len=%d): %q\nNew (len=%d): %q",
					len(resultOld), resultOld, len(resultNew), resultNew)
			}
		})
	}
}
