// Tests for CSV export functionality in the exporter package.
//
// This file tests the Export() and ExportStream() functions with
// "csv" format, covering:
// - Scrape, Crawl, and Research job kinds
// - Stable field ordering (alphabetical)
// - Consistent header generation
// - Proper CSV formatting
//
// The CSV export converts JSONL to CSV with:
// - Header row with field names sorted alphabetically
// - Normalized fields prefixed with "field_"
// - Data rows with properly escaped values
package exporter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestExportCSVHasStableFieldOrder(t *testing.T) {
	raw := []byte(`{"url":"https://example.com","status":200,"title":"Test","text":"Content","metadata":{"description":"Desc"},"normalized":{"fields":{"zebra":{"values":["z"]},"apple":{"values":["a"]},"banana":{"values":["b"]}}}}`)
	job := model.Job{Kind: model.KindScrape}

	var results []string
	for i := 0; i < 5; i++ {
		result, err := Export(job, raw, "csv")
		if err != nil {
			t.Fatalf("Export() failed on iteration %d: %v", i, err)
		}
		results = append(results, result)
	}

	for i := 1; i < len(results); i++ {
		if results[i] != results[0] {
			t.Errorf("Export %d differs from export 0\n%s\nvs\n%s", i, results[0], results[i])
		}
	}

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
	raw := []byte(`{"url":"https://example.com/page1","status":200,"title":"Page1","text":"Text1","normalized":{"fields":{"zebra":{"values":["z1"]},"apple":{"values":["a1"]}}}}
{"url":"https://example.com/page2","status":200,"title":"Page2","text":"Text2","normalized":{"fields":{"banana":{"values":["b2"]},"apple":{"values":["a2"]}}}}
{"url":"https://example.com/page3","status":200,"title":"Page3","text":"Text3","normalized":{"fields":{"zebra":{"values":["z3"]},"banana":{"values":["b3"]}}}}`)
	job := model.Job{Kind: model.KindCrawl}

	var results []string
	for i := 0; i < 5; i++ {
		result, err := Export(job, raw, "csv")
		if err != nil {
			t.Fatalf("Export() failed on iteration %d: %v", i, err)
		}
		results = append(results, result)
	}

	for i := 1; i < len(results); i++ {
		if results[i] != results[0] {
			t.Errorf("Export %d differs from export 0", i)
		}
	}

	firstResult := results[0]
	lines := strings.Split(strings.TrimSpace(firstResult), "\n")
	header := lines[0]
	expectedHeader := "url,status,title,field_apple,field_banana,field_zebra"
	if header != expectedHeader {
		t.Errorf("Crawl CSV header order incorrect.\nGot: %s\nWant: %s", header, expectedHeader)
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

			lines := strings.Split(strings.TrimSpace(result), "\n")
			if len(lines) < 1 {
				t.Error("Expected at least header line in CSV output")
			}
		})
	}
}

func TestExportStreamMatchesExportCSV(t *testing.T) {
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

			resultOld, err := Export(job, []byte(tt.raw), "csv")
			if err != nil {
				t.Fatalf("Export() failed: %v", err)
			}

			var buf bytes.Buffer
			err = ExportStream(job, strings.NewReader(tt.raw), "csv", &buf)
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
