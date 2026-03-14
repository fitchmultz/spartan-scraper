// Tests for Markdown export functionality in the exporter package.
//
// This file tests the Export() and ExportStream() functions with
// "md" format, covering:
// - Scrape, Crawl, and Research job kinds
// - Stable field ordering (alphabetical)
// - Consistent output across multiple exports
// - Proper markdown headers and formatting
//
// The Markdown export converts JSONL to formatted Markdown with:
// - Fields sorted alphabetically for consistency
// - Normalized fields prefixed with "field_"
// - H1 headers for scrape/research, H2 for crawl pages
package exporter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestExportMarkdownHasStableFieldOrder(t *testing.T) {
	raw := []byte(`{"url":"https://example.com","status":200,"title":"Test","text":"Content","metadata":{"description":"Desc"},"normalized":{"fields":{"zebra":{"values":["z"]},"apple":{"values":["a"]},"banana":{"values":["b"]}}}}`)
	job := model.Job{Kind: model.KindScrape}

	var results []string
	for i := 0; i < 5; i++ {
		result, err := Export(job, raw, "md")
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
	appleIdx := strings.Index(firstResult, "**apple**:")
	bananaIdx := strings.Index(firstResult, "**banana**:")
	zebraIdx := strings.Index(firstResult, "**zebra**:")

	if appleIdx == -1 || bananaIdx == -1 || zebraIdx == -1 {
		t.Fatal("One or more field names not found in output")
	}

	if !(appleIdx < bananaIdx && bananaIdx < zebraIdx) {
		t.Error("Fields are not in alphabetical order")
	}
}

func TestExportCrawlMarkdownFieldOrderIsStable(t *testing.T) {
	raw := []byte(`{"url":"https://example.com/page1","status":200,"title":"Page1","text":"Text1","normalized":{"fields":{"zebra":{"values":["z"]},"apple":{"values":["a"]}}}}
{"url":"https://example.com/page2","status":200,"title":"Page2","text":"Text2","normalized":{"fields":{"banana":{"values":["b"]}}}}`)
	job := model.Job{Kind: model.KindCrawl}

	var results []string
	for i := 0; i < 5; i++ {
		result, err := Export(job, raw, "md")
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
	page1Start := strings.Index(firstResult, "## Page1")
	if page1Start == -1 {
		t.Fatal("Page1 section not found")
	}

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

func TestExportResearchMarkdownIncludesAgenticSection(t *testing.T) {
	raw := []byte(`{"query":"pricing","summary":"Deterministic summary","confidence":0.73,"agentic":{"status":"completed","summary":"Agentic summary","focusAreas":["pricing model"],"followUpUrls":["https://example.com/support"]},"evidence":[],"clusters":[],"citations":[]}`)
	job := model.Job{Kind: model.KindResearch}

	result, err := Export(job, raw, "md")
	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	for _, want := range []string{
		"## Agentic Research",
		"**Status:** completed",
		"Agentic summary",
		"### Focus Areas",
		"- pricing model",
		"### Follow-Up URLs",
		"- https://example.com/support",
	} {
		if !strings.Contains(result, want) {
			t.Fatalf("expected markdown to contain %q\n%s", want, result)
		}
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

			if !strings.Contains(result, "# ") {
				t.Error("Expected markdown headers (starting with #)")
			}
		})
	}
}

func TestExportStreamMatchesExportMarkdown(t *testing.T) {
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

			resultOld, err := Export(job, []byte(tt.raw), "md")
			if err != nil {
				t.Fatalf("Export() failed: %v", err)
			}

			var buf bytes.Buffer
			err = ExportStream(job, strings.NewReader(tt.raw), "md", &buf)
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
