package exporter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

			// The streaming implementation adds a trailing newline, which is acceptable
			// Normalize both for comparison
			normalizedResult := strings.TrimRight(result, "\n")
			normalizedOriginal := strings.TrimRight(original, "\n")

			if normalizedResult != normalizedOriginal {
				t.Errorf("JSONL output content should be unchanged from input\nExpected: %q\nGot: %q", normalizedOriginal, normalizedResult)
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

// TestParseLargeJSONLLine verifies that JSONL lines larger than the default
// 64KB scanner buffer can be parsed successfully. This is a regression test
// for RQ-0024.
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
			// Create a JSONL line larger than 64KB
			largeText := strings.Repeat("x", tt.sizeKB*1024)
			jsonLine := fmt.Sprintf(`{"url":"https://example.com","status":200,"title":"Large Content","text":"%s"}`, largeText)

			// Test parseSingle path (Scrape job)
			t.Run("parseSingle", func(t *testing.T) {
				job := model.Job{Kind: model.KindScrape}
				result, err := Export(job, []byte(jsonLine), "json")
				if err != nil {
					t.Fatalf("Export() failed for %d KB line: %v", tt.sizeKB, err)
				}

				// Verify output is valid JSON
				var scrapeResult ScrapeResult
				if err := json.Unmarshal([]byte(result), &scrapeResult); err != nil {
					t.Fatalf("Output is not valid JSON: %v", err)
				}

				// Verify the text content was preserved
				if scrapeResult.Text != largeText {
					t.Errorf("Text content not preserved: got length %d, want %d", len(scrapeResult.Text), len(largeText))
				}
			})

			// Test parseLines path (Crawl job with multiple large lines)
			t.Run("parseLines", func(t *testing.T) {
				// Create multiple large JSONL lines
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

				// Verify output is valid JSON array
				var crawlResults []CrawlResult
				if err := json.Unmarshal([]byte(result), &crawlResults); err != nil {
					t.Fatalf("Output is not valid JSON: %v", err)
				}

				// Verify all items were parsed
				if len(crawlResults) != 3 {
					t.Errorf("Expected 3 items, got %d", len(crawlResults))
				}

				// Verify each line's text content was preserved
				for i, item := range crawlResults {
					if item.Text != largeText {
						t.Errorf("Item %d: text content not preserved: got length %d, want %d", i, len(item.Text), len(largeText))
					}
				}
			})
		})
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

func TestExportMarkdownHasStableFieldOrder(t *testing.T) {
	// Create a scrape result with multiple fields in random map order
	raw := []byte(`{"url":"https://example.com","status":200,"title":"Test","text":"Content","metadata":{"description":"Desc"},"normalized":{"fields":{"zebra":{"values":["z"]},"apple":{"values":["a"]},"banana":{"values":["b"]}}}}`)
	job := model.Job{Kind: model.KindScrape}

	// Export multiple times and verify the output is identical
	var results []string
	for i := 0; i < 5; i++ {
		result, err := Export(job, raw, "md")
		if err != nil {
			t.Fatalf("Export() failed on iteration %d: %v", i, err)
		}
		results = append(results, result)
	}

	// All exports should produce identical output
	for i := 1; i < len(results); i++ {
		if results[i] != results[0] {
			t.Errorf("Export %d differs from export 0\n%s\nvs\n%s", i, results[0], results[i])
		}
	}

	// Verify fields are alphabetically ordered
	firstResult := results[0]
	appleIdx := strings.Index(firstResult, "**apple**:")
	bananaIdx := strings.Index(firstResult, "**banana**:")
	zebraIdx := strings.Index(firstResult, "**zebra**:")

	if appleIdx == -1 || bananaIdx == -1 || zebraIdx == -1 {
		t.Fatal("One or more field names not found in output")
	}

	// Verify alphabetical order: apple before banana before zebra
	if !(appleIdx < bananaIdx && bananaIdx < zebraIdx) {
		t.Error("Fields are not in alphabetical order")
	}
}

func TestExportCSVHasStableFieldOrder(t *testing.T) {
	// Create a scrape result with multiple fields
	raw := []byte(`{"url":"https://example.com","status":200,"title":"Test","text":"Content","metadata":{"description":"Desc"},"normalized":{"fields":{"zebra":{"values":["z"]},"apple":{"values":["a"]},"banana":{"values":["b"]}}}}`)
	job := model.Job{Kind: model.KindScrape}

	// Export multiple times
	var results []string
	for i := 0; i < 5; i++ {
		result, err := Export(job, raw, "csv")
		if err != nil {
			t.Fatalf("Export() failed on iteration %d: %v", i, err)
		}
		results = append(results, result)
	}

	// All exports should produce identical output
	for i := 1; i < len(results); i++ {
		if results[i] != results[0] {
			t.Errorf("Export %d differs from export 0\n%s\nvs\n%s", i, results[0], results[i])
		}
	}

	// Verify CSV headers are alphabetically ordered
	firstResult := results[0]
	lines := strings.Split(strings.TrimSpace(firstResult), "\n")
	if len(lines) < 2 {
		t.Fatal("Expected at least 2 lines in CSV output (header + data)")
	}

	header := lines[0]
	expectedHeader := "url,status,title,description,field_apple,field_banana,field_zebra"
	if header != expectedHeader {
		t.Errorf("CSV header order incorrect.\nGot: %s\nWant: %s", header, expectedHeader)
	}
}

func TestExportCrawlCSVFieldOrderIsStable(t *testing.T) {
	// Create crawl results with different field sets across items
	raw := []byte(`{"url":"https://example.com/page1","status":200,"title":"Page1","text":"Text1","normalized":{"fields":{"zebra":{"values":["z1"]},"apple":{"values":["a1"]}}}}
{"url":"https://example.com/page2","status":200,"title":"Page2","text":"Text2","normalized":{"fields":{"banana":{"values":["b2"]},"apple":{"values":["a2"]}}}}
{"url":"https://example.com/page3","status":200,"title":"Page3","text":"Text3","normalized":{"fields":{"zebra":{"values":["z3"]},"banana":{"values":["b3"]}}}}`)
	job := model.Job{Kind: model.KindCrawl}

	// Export multiple times
	var results []string
	for i := 0; i < 5; i++ {
		result, err := Export(job, raw, "csv")
		if err != nil {
			t.Fatalf("Export() failed on iteration %d: %v", i, err)
		}
		results = append(results, result)
	}

	// All exports should be identical
	for i := 1; i < len(results); i++ {
		if results[i] != results[0] {
			t.Errorf("Export %d differs from export 0", i)
		}
	}

	// Verify headers are alphabetically ordered
	firstResult := results[0]
	lines := strings.Split(strings.TrimSpace(firstResult), "\n")
	header := lines[0]
	expectedHeader := "url,status,title,field_apple,field_banana,field_zebra"
	if header != expectedHeader {
		t.Errorf("Crawl CSV header order incorrect.\nGot: %s\nWant: %s", header, expectedHeader)
	}
}

func TestExportCrawlMarkdownFieldOrderIsStable(t *testing.T) {
	// Create crawl results with fields in non-alphabetical order
	raw := []byte(`{"url":"https://example.com/page1","status":200,"title":"Page1","text":"Text1","normalized":{"fields":{"zebra":{"values":["z"]},"apple":{"values":["a"]}}}}
{"url":"https://example.com/page2","status":200,"title":"Page2","text":"Text2","normalized":{"fields":{"banana":{"values":["b"]}}}}`)
	job := model.Job{Kind: model.KindCrawl}

	// Export multiple times
	var results []string
	for i := 0; i < 5; i++ {
		result, err := Export(job, raw, "md")
		if err != nil {
			t.Fatalf("Export() failed on iteration %d: %v", i, err)
		}
		results = append(results, result)
	}

	// All exports should be identical
	for i := 1; i < len(results); i++ {
		if results[i] != results[0] {
			t.Errorf("Export %d differs from export 0", i)
		}
	}

	// Verify page1 fields are in alphabetical order
	firstResult := results[0]
	page1Start := strings.Index(firstResult, "## Page1")
	if page1Start == -1 {
		t.Fatal("Page1 section not found")
	}

	// Find the positions of field markers within Page1's section
	page1End := strings.Index(firstResult[page1Start:], "\n## ")
	if page1End == -1 {
		page1End = len(firstResult)
	} else {
		page1End += page1Start
	}
	page1Section := firstResult[page1Start:page1End]

	appleIdx := strings.Index(page1Section, "**apple**:")
	zebraIdx := strings.Index(page1Section, "**zebra**:")

	if appleIdx == -1 || zebraIdx == -1 {
		t.Fatal("Fields not found in Page1 section")
	}

	if !(appleIdx < zebraIdx) {
		t.Error("Page1 fields are not in alphabetical order")
	}
}

// Tests for ExportStream functionality

func TestExportStreamJSONL(t *testing.T) {
	raw := sampleScrapeResultJSONL()
	job := model.Job{Kind: model.KindScrape}
	var buf bytes.Buffer

	err := ExportStream(job, strings.NewReader(raw), "jsonl", &buf)
	if err != nil {
		t.Fatalf("ExportStream() failed: %v", err)
	}

	result := buf.String()
	if result != raw+"\n" {
		t.Errorf("JSONL output mismatch\nExpected: %q\nGot: %q", raw+"\n", result)
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

			// Verify output is valid JSON
			var decoded interface{}
			if err := json.Unmarshal([]byte(result), &decoded); err != nil {
				t.Fatalf("Output is not valid JSON: %v", err)
			}

			// Verify proper indentation
			if !strings.Contains(result, "\n  ") {
				t.Error("Expected 2-space indentation in JSON output")
			}
		})
	}
}

func TestExportStreamMarkdown(t *testing.T) {
	tests := []struct {
		name string
		kind model.Kind
		raw  string
	}{
		{"Scrape job", model.KindScrape, sampleScrapeResultJSONL()},
		{"Crawl job", model.KindCrawl, sampleCrawlResultJSONL(2)},
		{"Research job", model.KindResearch, sampleResearchResultJSONL()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := model.Job{Kind: tt.kind}
			var buf bytes.Buffer

			err := ExportStream(job, strings.NewReader(tt.raw), "md", &buf)
			if err != nil {
				t.Fatalf("ExportStream() failed: %v", err)
			}

			result := buf.String()

			// Verify markdown formatting
			if !strings.Contains(result, "# ") {
				t.Error("Expected markdown headers (starting with #)")
			}
		})
	}
}

func TestExportStreamCSV(t *testing.T) {
	tests := []struct {
		name string
		kind model.Kind
		raw  string
	}{
		{"Scrape job", model.KindScrape, sampleScrapeResultJSONL()},
		{"Crawl job", model.KindCrawl, sampleCrawlResultJSONL(2)},
		{"Research job", model.KindResearch, sampleResearchResultJSONL()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := model.Job{Kind: tt.kind}
			var buf bytes.Buffer

			err := ExportStream(job, strings.NewReader(tt.raw), "csv", &buf)
			if err != nil {
				t.Fatalf("ExportStream() failed: %v", err)
			}

			result := buf.String()

			// Verify CSV format (has headers and data)
			lines := strings.Split(strings.TrimSpace(result), "\n")
			if len(lines) < 1 {
				t.Error("Expected at least header line in CSV output")
			}
		})
	}
}

func TestExportStreamLargeFile(t *testing.T) {
	// Create a large JSONL file (100KB of data)
	lines := make([]string, 100)
	for i := 0; i < 100; i++ {
		largeText := strings.Repeat("x", 1024) // 1KB per line
		lines[i] = fmt.Sprintf(`{"url":"https://example.com/page%d","status":200,"title":"Page %d","text":"%s"}`, i, i, largeText)
	}
	raw := strings.Join(lines, "\n")

	job := model.Job{Kind: model.KindCrawl}
	var buf bytes.Buffer

	// This should not cause memory issues
	err := ExportStream(job, strings.NewReader(raw), "json", &buf)
	if err != nil {
		t.Fatalf("ExportStream() failed for large file: %v", err)
	}

	result := buf.String()

	// Verify all items were exported
	var crawlResults []CrawlResult
	if err := json.Unmarshal([]byte(result), &crawlResults); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	if len(crawlResults) != 100 {
		t.Errorf("Expected 100 items, got %d", len(crawlResults))
	}
}

func TestExportStreamErrorHandling(t *testing.T) {
	t.Run("Reader error", func(t *testing.T) {
		job := model.Job{Kind: model.KindScrape}
		var buf bytes.Buffer

		// Create a reader that always returns an error
		errReader := &errorReader{err: io.ErrClosedPipe}
		err := ExportStream(job, errReader, "json", &buf)
		if err == nil {
			t.Error("Expected error from errorReader, got nil")
		}
	})

	t.Run("Writer error", func(t *testing.T) {
		job := model.Job{Kind: model.KindScrape}
		raw := sampleScrapeResultJSONL()

		// Create a writer that always returns an error
		errWriter := &errorWriter{err: io.ErrClosedPipe}
		err := ExportStream(job, strings.NewReader(raw), "jsonl", errWriter)
		if err == nil {
			t.Error("Expected error from errorWriter, got nil")
		}
	})

	t.Run("Unsupported format", func(t *testing.T) {
		job := model.Job{Kind: model.KindScrape}
		raw := sampleScrapeResultJSONL()
		var buf bytes.Buffer

		err := ExportStream(job, strings.NewReader(raw), "xml", &buf)
		if err == nil {
			t.Error("Expected error for unsupported format, got nil")
		}
		if !strings.Contains(err.Error(), "unsupported format") {
			t.Errorf("Expected 'unsupported format' error, got: %v", err)
		}
	})
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

func TestExportStreamMatchesExport(t *testing.T) {
	tests := []struct {
		name   string
		kind   model.Kind
		raw    string
		format string
	}{
		{"Scrape JSONL", model.KindScrape, sampleScrapeResultJSONL(), "jsonl"},
		{"Scrape JSON", model.KindScrape, sampleScrapeResultJSONL(), "json"},
		{"Scrape Markdown", model.KindScrape, sampleScrapeResultJSONL(), "md"},
		{"Scrape CSV", model.KindScrape, sampleScrapeResultJSONL(), "csv"},
		{"Crawl JSONL", model.KindCrawl, sampleCrawlResultJSONL(3), "jsonl"},
		{"Crawl JSON", model.KindCrawl, sampleCrawlResultJSONL(3), "json"},
		{"Crawl Markdown", model.KindCrawl, sampleCrawlResultJSONL(2), "md"},
		{"Crawl CSV", model.KindCrawl, sampleCrawlResultJSONL(2), "csv"},
		{"Research JSONL", model.KindResearch, sampleResearchResultJSONL(), "jsonl"},
		{"Research JSON", model.KindResearch, sampleResearchResultJSONL(), "json"},
		{"Research Markdown", model.KindResearch, sampleResearchResultJSONL(), "md"},
		{"Research CSV", model.KindResearch, sampleResearchResultJSONL(), "csv"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := model.Job{Kind: tt.kind}

			// Old API (Export)
			resultOld, err := Export(job, []byte(tt.raw), tt.format)
			if err != nil {
				t.Fatalf("Export() failed: %v", err)
			}

			// New API (ExportStream)
			var buf bytes.Buffer
			err = ExportStream(job, strings.NewReader(tt.raw), tt.format, &buf)
			if err != nil {
				t.Fatalf("ExportStream() failed: %v", err)
			}
			resultNew := buf.String()

			// For JSONL, the old API doesn't add a trailing newline but stream does
			// Normalize for comparison
			if tt.format == "jsonl" {
				resultOld = strings.TrimRight(resultOld, "\n")
				resultNew = strings.TrimRight(resultNew, "\n")
			}

			if resultOld != resultNew {
				t.Errorf("Export and ExportStream produced different output\nOld (len=%d): %q\nNew (len=%d): %q",
					len(resultOld), resultOld, len(resultNew), resultNew)
			}
		})
	}
}

// Test helper types for error handling

type errorReader struct {
	err error
}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, r.err
}

type errorWriter struct {
	err error
}

func (w *errorWriter) Write(p []byte) (n int, err error) {
	return 0, w.err
}
