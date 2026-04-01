// Package exporter provides exporter functionality for Spartan Scraper.
//
// Purpose:
// - Verify xlsx test behavior for package exporter.
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
	"io"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/xuri/excelize/v2"
)

func TestExportStreamXLSX_LargeDataset(t *testing.T) {
	// Generate a dataset that is reasonably large
	count := 1000
	raw := sampleCrawlResultJSONL(count)
	job := model.Job{Kind: model.KindCrawl}

	t.Run("Seekable reader", func(t *testing.T) {
		var buf bytes.Buffer
		err := ExportStream(job, strings.NewReader(raw), "xlsx", &buf)
		if err != nil {
			t.Fatalf("ExportStream() failed: %v", err)
		}

		// Verify it's a valid XLSX by opening it
		f, err := excelize.OpenReader(&buf)
		if err != nil {
			t.Fatalf("Failed to open XLSX: %v", err)
		}
		defer f.Close()

		// Check sheet exists
		sheets := f.GetSheetList()
		if len(sheets) == 0 {
			t.Error("Expected at least one sheet in XLSX")
		}
	})

	t.Run("Non-seekable reader", func(t *testing.T) {
		// Use a pipe to simulate a non-seekable reader
		pr, pw := io.Pipe()
		go func() {
			defer pw.Close()
			pw.Write([]byte(raw))
		}()

		var buf bytes.Buffer
		err := ExportStream(job, pr, "xlsx", &buf)
		if err != nil {
			t.Fatalf("ExportStream() failed: %v", err)
		}

		// Verify it's a valid XLSX
		f, err := excelize.OpenReader(&buf)
		if err != nil {
			t.Fatalf("Failed to open XLSX: %v", err)
		}
		defer f.Close()
	})
}

func TestExportXLSXHasStableFieldOrder(t *testing.T) {
	raw := []byte(`{"url":"https://example.com","status":200,"title":"Test","text":"Content","metadata":{"description":"Desc"},"normalized":{"fields":{"zebra":{"values":["z"]},"apple":{"values":["a"]},"banana":{"values":["b"]}}}}`)
	job := model.Job{Kind: model.KindScrape}

	var results [][]string
	for i := 0; i < 5; i++ {
		var buf bytes.Buffer
		err := ExportStream(job, bytes.NewReader(raw), "xlsx", &buf)
		if err != nil {
			t.Fatalf("ExportStream() failed on iteration %d: %v", i, err)
		}

		// Open the XLSX and read headers
		f, err := excelize.OpenReader(&buf)
		if err != nil {
			t.Fatalf("Failed to open XLSX: %v", err)
		}

		rows, err := f.GetRows("Results")
		f.Close()
		if err != nil {
			t.Fatalf("Failed to get rows: %v", err)
		}

		if len(rows) > 0 {
			results = append(results, rows[0])
		}
	}

	for i := 1; i < len(results); i++ {
		if !slicesEqual(results[i], results[0]) {
			t.Errorf("Export %d differs from export 0\n%v\nvs\n%v", i, results[0], results[i])
		}
	}

	firstResult := results[0]
	expectedHeader := []string{"url", "status", "title", "description", "field_apple", "field_banana", "field_zebra"}
	if !slicesEqual(firstResult, expectedHeader) {
		t.Errorf("XLSX header order incorrect.\nGot: %v\nWant: %v", firstResult, expectedHeader)
	}
}

func TestExportCrawlXLSXFieldOrderIsStable(t *testing.T) {
	raw := []byte(`{"url":"https://example.com/page1","status":200,"title":"Page1","text":"Text1","normalized":{"fields":{"zebra":{"values":["z1"]},"apple":{"values":["a1"]}}}}
{"url":"https://example.com/page2","status":200,"title":"Page2","text":"Text2","normalized":{"fields":{"banana":{"values":["b2"]},"apple":{"values":["a2"]}}}}
{"url":"https://example.com/page3","status":200,"title":"Page3","text":"Text3","normalized":{"fields":{"zebra":{"values":["z3"]},"banana":{"values":["b3"]}}}}`)
	job := model.Job{Kind: model.KindCrawl}

	var results [][]string
	for i := 0; i < 5; i++ {
		var buf bytes.Buffer
		err := ExportStream(job, bytes.NewReader(raw), "xlsx", &buf)
		if err != nil {
			t.Fatalf("ExportStream() failed on iteration %d: %v", i, err)
		}

		f, err := excelize.OpenReader(&buf)
		if err != nil {
			t.Fatalf("Failed to open XLSX: %v", err)
		}

		rows, err := f.GetRows("Results")
		f.Close()
		if err != nil {
			t.Fatalf("Failed to get rows: %v", err)
		}

		if len(rows) > 0 {
			results = append(results, rows[0])
		}
	}

	for i := 1; i < len(results); i++ {
		if !slicesEqual(results[i], results[0]) {
			t.Errorf("Export %d differs from export 0", i)
		}
	}

	firstResult := results[0]
	expectedHeader := []string{"url", "status", "title", "field_apple", "field_banana", "field_zebra"}
	if !slicesEqual(firstResult, expectedHeader) {
		t.Errorf("Crawl XLSX header order incorrect.\nGot: %v\nWant: %v", firstResult, expectedHeader)
	}
}

func TestExportStreamXLSX(t *testing.T) {
	tests := []struct {
		name        string
		kind        model.Kind
		raw         string
		expectSheet string
	}{
		{"Scrape job", model.KindScrape, sampleScrapeResultJSONL(), "Results"},
		{"Crawl job", model.KindCrawl, sampleCrawlResultJSONL(2), "Results"},
		{"Research job", model.KindResearch, sampleResearchResultJSONL(), "Summary"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := model.Job{Kind: tt.kind}
			var buf bytes.Buffer

			err := ExportStream(job, strings.NewReader(tt.raw), "xlsx", &buf)
			if err != nil {
				t.Fatalf("ExportStream() failed: %v", err)
			}

			// Verify it's a valid XLSX
			f, err := excelize.OpenReader(&buf)
			if err != nil {
				t.Fatalf("Failed to open XLSX: %v", err)
			}
			defer f.Close()

			// Check expected sheet exists
			sheets := f.GetSheetList()
			found := false
			for _, sheet := range sheets {
				if sheet == tt.expectSheet {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected sheet %q not found. Sheets: %v", tt.expectSheet, sheets)
			}

			// Check sheet has data
			rows, err := f.GetRows(tt.expectSheet)
			if err != nil {
				t.Errorf("Failed to get rows from %s: %v", tt.expectSheet, err)
			}
			if len(rows) < 1 {
				t.Error("Expected at least header row in XLSX output")
			}
		})
	}
}

func TestExportStreamMatchesExportXLSX(t *testing.T) {
	tests := []struct {
		name string
		kind model.Kind
		raw  string
	}{
		{"Scrape", model.KindScrape, sampleScrapeResultJSONL()},
		{"Crawl", model.KindCrawl, sampleCrawlResultJSONL(2)},
		{"Research", model.KindResearch, sampleResearchResultJSONL()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := model.Job{Kind: tt.kind}

			// Export using Export (via string)
			resultOld, err := Export(job, []byte(tt.raw), "xlsx")
			if err != nil {
				t.Fatalf("Export() failed: %v", err)
			}

			// Export using ExportStream
			var buf bytes.Buffer
			err = ExportStream(job, strings.NewReader(tt.raw), "xlsx", &buf)
			if err != nil {
				t.Fatalf("ExportStream() failed: %v", err)
			}
			resultNew := buf.String()

			if resultOld != resultNew {
				t.Errorf("Export and ExportStream produced different output\nOld (len=%d) vs New (len=%d)",
					len(resultOld), len(resultNew))
			}
		})
	}
}

func TestExportResearchXLSXMultiSheet(t *testing.T) {
	raw := sampleResearchResultJSONL()
	job := model.Job{Kind: model.KindResearch}

	var buf bytes.Buffer
	err := ExportStream(job, strings.NewReader(raw), "xlsx", &buf)
	if err != nil {
		t.Fatalf("ExportStream() failed: %v", err)
	}

	f, err := excelize.OpenReader(&buf)
	if err != nil {
		t.Fatalf("Failed to open XLSX: %v", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) != 2 {
		t.Errorf("Expected 2 sheets for research job, got %d: %v", len(sheets), sheets)
	}

	// Check for Summary sheet
	hasSummary := false
	hasEvidence := false
	for _, sheet := range sheets {
		if sheet == "Summary" {
			hasSummary = true
		}
		if sheet == "Evidence" {
			hasEvidence = true
		}
	}
	if !hasSummary {
		t.Error("Expected 'Summary' sheet not found")
	}
	if !hasEvidence {
		t.Error("Expected 'Evidence' sheet not found")
	}

	// Verify Summary sheet content
	rows, err := f.GetRows("Summary")
	if err != nil {
		t.Fatalf("Failed to get Summary rows: %v", err)
	}
	if len(rows) < 2 {
		t.Errorf("Expected at least 2 rows in Summary sheet, got %d", len(rows))
	}

	// Check headers
	expectedHeaders := []string{"query", "summary", "confidence", "agentic_status", "agentic_summary"}
	if len(rows) > 0 && !slicesEqual(rows[0], expectedHeaders) {
		t.Errorf("Summary headers incorrect.\nGot: %v\nWant: %v", rows[0], expectedHeaders)
	}

	// Verify Evidence sheet content
	evRows, err := f.GetRows("Evidence")
	if err != nil {
		t.Fatalf("Failed to get Evidence rows: %v", err)
	}
	if len(evRows) < 2 {
		t.Errorf("Expected at least 2 rows in Evidence sheet, got %d", len(evRows))
	}

	// Check evidence headers
	expectedEvHeaders := []string{"url", "title", "score", "confidence", "cluster_id", "citation_url", "snippet"}
	if len(evRows) > 0 && !slicesEqual(evRows[0], expectedEvHeaders) {
		t.Errorf("Evidence headers incorrect.\nGot: %v\nWant: %v", evRows[0], expectedEvHeaders)
	}
}

// slicesEqual compares two string slices for equality
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
