package exporter

import (
	"encoding/json"
	"strings"
	"testing"

	"spartan-scraper/internal/model"
)

func TestExportJSONForScrapeJob(t *testing.T) {
	raw := []byte(sampleScrapeResultJSONL())
	job := model.Job{Kind: model.KindScrape}

	result, err := Export(job, raw, "json")
	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	// Verify output is valid JSON (not JSONL)
	var scrapeResult ScrapeResult
	if err := json.Unmarshal([]byte(result), &scrapeResult); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	// Verify it's a single object, not an array
	if strings.HasPrefix(strings.TrimSpace(result), "[") {
		t.Error("Expected JSON object, got JSON array")
	}

	// Verify proper indentation (2-space)
	if !strings.Contains(result, "\n  ") {
		t.Error("Expected 2-space indentation in JSON output")
	}

	// Verify content matches
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

	// Verify output is valid JSON array
	var crawlResults []CrawlResult
	if err := json.Unmarshal([]byte(result), &crawlResults); err != nil {
		t.Fatalf("Output is not valid JSON array: %v", err)
	}

	// Verify it's an array, not a single object
	if !strings.HasPrefix(strings.TrimSpace(result), "[") {
		t.Error("Expected JSON array, got JSON object")
	}

	// Verify array length matches input lines
	expectedCount := 3
	if len(crawlResults) != expectedCount {
		t.Errorf("Expected %d items, got %d", expectedCount, len(crawlResults))
	}

	// Verify proper indentation
	if !strings.Contains(result, "\n  ") {
		t.Error("Expected 2-space indentation in JSON output")
	}

	// Verify content
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

	// Verify output is valid JSON (not JSONL)
	var researchResult ResearchResult
	if err := json.Unmarshal([]byte(result), &researchResult); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	// Verify it's a single object, not an array
	if strings.HasPrefix(strings.TrimSpace(result), "[") {
		t.Error("Expected JSON object, got JSON array")
	}

	// Verify proper indentation
	if !strings.Contains(result, "\n  ") {
		t.Error("Expected 2-space indentation in JSON output")
	}

	// Verify content
	expectedQuery := "test query"
	if researchResult.Query != expectedQuery {
		t.Errorf("Expected query %q, got %q", expectedQuery, researchResult.Query)
	}

	if len(researchResult.Evidence) == 0 {
		t.Error("Expected evidence items, got none")
	}
}

func TestExportJSONLRemainsUnchanged(t *testing.T) {
	tests := []struct {
		name string
		kind model.Kind
		raw  []byte
	}{
		{
			name: "Scrape job JSONL",
			kind: model.KindScrape,
			raw:  []byte(sampleScrapeResultJSONL()),
		},
		{
			name: "Crawl job JSONL",
			kind: model.KindCrawl,
			raw:  []byte(sampleCrawlResultJSONL(2)),
		},
		{
			name: "Research job JSONL",
			kind: model.KindResearch,
			raw:  []byte(sampleResearchResultJSONL()),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := model.Job{Kind: tt.kind}
			original := string(tt.raw)

			result, err := Export(job, tt.raw, "jsonl")
			if err != nil {
				t.Fatalf("Export() failed: %v", err)
			}

			// Verify output is byte-for-byte identical to input
			if result != original {
				t.Errorf("JSONL output should be unchanged from input\nExpected: %q\nGot: %q", original, result)
			}

			// Verify it's still JSONL format (newline-delimited)
			lines := strings.Split(strings.TrimSpace(result), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" {
					var v interface{}
					if err := json.Unmarshal([]byte(line), &v); err != nil {
						t.Errorf("Line is not valid JSON: %q", line)
					}
				}
			}
		})
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

		// Should return empty JSON array
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

	// Verify it's still an array with one element
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
	// Test that blank lines in JSONL input are handled correctly
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

	// Should have 3 items (2 + 1, blank lines skipped)
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

func TestExportUnsupportedFormat(t *testing.T) {
	raw := []byte(sampleScrapeResultJSONL())
	job := model.Job{Kind: model.KindScrape}

	_, err := Export(job, raw, "xml")
	if err == nil {
		t.Error("Expected error for unsupported format, got nil")
	}

	if !strings.Contains(err.Error(), "unsupported format") {
		t.Errorf("Expected 'unsupported format' error, got: %v", err)
	}
}

// Test helper functions

func sampleScrapeResultJSONL() string {
	return `{"url":"https://example.com","status":200,"title":"Example Page","text":"Content here","metadata":{"description":"A test page"},"normalized":{"title":"Example","description":"Test description","text":"Normalized text","fields":{}}}`
}

func sampleCrawlResultJSONL(count int) string {
	var lines []string
	for i := 0; i < count; i++ {
		lines = append(lines, `{"url":"https://example.com/page`+string(rune('1'+i))+`","status":200,"title":"Page `+string(rune('1'+i))+`","text":"Content `+string(rune('1'+i))+`","normalized":{"title":"Page `+string(rune('1'+i))+`","text":"Text `+string(rune('1'+i))+`","fields":{}}}`)
	}
	return strings.Join(lines, "\n")
}

func sampleResearchResultJSONL() string {
	return `{"query":"test query","summary":"Test summary","confidence":0.95,"evidence":[{"url":"https://example.com/evidence1","title":"Evidence 1","snippet":"Test snippet","score":0.9,"simhash":1234567890,"clusterId":"cluster1","confidence":0.9,"citationUrl":"https://example.com/cite1"}],"clusters":[{"id":"cluster1","label":"Test Cluster","confidence":0.9,"evidence":[]}],"citations":[{"url":"https://example.com/cite1","anchor":"section1","canonical":"https://example.com/canonical1"}]}`
}
