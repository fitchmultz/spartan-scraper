// Tests for PDF export functionality in the exporter package.
//
// This file tests the ExportStream() function with "pdf" format, covering:
// - Scrape, Crawl, and Research job kinds
// - PDF magic bytes validation (%PDF)
// - Non-empty output validation
// - Error handling for invalid input
//
// The PDF export converts JSONL to PDF documents using Chromedp's PrintToPDF.
package exporter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestExportPDFStream(t *testing.T) {
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

			err := ExportStream(job, strings.NewReader(tt.raw), "pdf", &buf)
			if err != nil {
				t.Fatalf("ExportStream() failed: %v", err)
			}

			result := buf.Bytes()

			// Verify PDF magic bytes
			if len(result) < 4 {
				t.Fatal("PDF output is too short")
			}
			if string(result[:4]) != "%PDF" {
				t.Errorf("PDF output does not start with %%PDF magic bytes, got: %q", string(result[:4]))
			}

			// Verify non-empty output
			if len(result) < 100 {
				t.Error("PDF output seems too small to be valid")
			}
		})
	}
}

func TestExportPDFStreamEmptyCrawl(t *testing.T) {
	// Test with empty crawl results - should generate a valid PDF with 0 pages
	job := model.Job{Kind: model.KindCrawl}
	var buf bytes.Buffer

	err := ExportStream(job, strings.NewReader(""), "pdf", &buf)
	if err != nil {
		t.Fatalf("ExportStream() failed for empty crawl: %v", err)
	}

	result := buf.Bytes()

	// Verify PDF magic bytes
	if len(result) < 4 {
		t.Fatal("PDF output is too short")
	}
	if string(result[:4]) != "%PDF" {
		t.Errorf("PDF output does not start with %%PDF magic bytes, got: %q", string(result[:4]))
	}
}

func TestExportPDFStreamInvalidJSON(t *testing.T) {
	// Test with invalid JSON input
	job := model.Job{Kind: model.KindScrape}
	var buf bytes.Buffer

	err := ExportStream(job, strings.NewReader("invalid json"), "pdf", &buf)
	if err == nil {
		t.Error("Expected error for invalid JSON input, got nil")
	}
}

func TestExportPDFStreamUnknownJobKind(t *testing.T) {
	// Test with unknown job kind
	job := model.Job{Kind: model.Kind("unknown")}
	var buf bytes.Buffer

	err := ExportStream(job, strings.NewReader(sampleScrapeResultJSONL()), "pdf", &buf)
	if err == nil {
		t.Error("Expected error for unknown job kind, got nil")
	}
}

func TestEscapeHTML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"<script>", "&lt;script&gt;"},
		{"test & test", "test &amp; test"},
		{`"quoted"`, "&quot;quoted&quot;"},
		{"<b>bold</b>", "&lt;b&gt;bold&lt;/b&gt;"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := escapeHTML(tt.input)
			if result != tt.expected {
				t.Errorf("escapeHTML(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDefaultPDFConfig(t *testing.T) {
	cfg := DefaultPDFConfig()

	if cfg.PageSize != "Letter" {
		t.Errorf("Expected PageSize to be 'Letter', got %q", cfg.PageSize)
	}
	if cfg.Orientation != "portrait" {
		t.Errorf("Expected Orientation to be 'portrait', got %q", cfg.Orientation)
	}
	if cfg.MarginTop != 0.5 {
		t.Errorf("Expected MarginTop to be 0.5, got %f", cfg.MarginTop)
	}
	if cfg.MarginBottom != 0.5 {
		t.Errorf("Expected MarginBottom to be 0.5, got %f", cfg.MarginBottom)
	}
	if cfg.MarginLeft != 0.5 {
		t.Errorf("Expected MarginLeft to be 0.5, got %f", cfg.MarginLeft)
	}
	if cfg.MarginRight != 0.5 {
		t.Errorf("Expected MarginRight to be 0.5, got %f", cfg.MarginRight)
	}
	if !cfg.PrintBackground {
		t.Error("Expected PrintBackground to be true")
	}
}

func TestBuildScrapeHTML(t *testing.T) {
	item := ScrapeResult{
		URL:    "https://example.com",
		Status: 200,
		Title:  "Test Page",
		Text:   "Test content",
		Metadata: struct {
			Description string `json:"description"`
		}{
			Description: "Test description",
		},
	}

	html := buildScrapeHTML(item)

	// Check that HTML contains key elements
	if !strings.Contains(html, "Test Page") {
		t.Error("HTML should contain title")
	}
	if !strings.Contains(html, "https://example.com") {
		t.Error("HTML should contain URL")
	}
	if !strings.Contains(html, "Test content") {
		t.Error("HTML should contain text content")
	}
}

func TestBuildCrawlHTML(t *testing.T) {
	items := []CrawlResult{
		{
			URL:    "https://example.com/page1",
			Status: 200,
			Title:  "Page 1",
			Text:   "Content 1",
		},
		{
			URL:    "https://example.com/page2",
			Status: 200,
			Title:  "Page 2",
			Text:   "Content 2",
		},
	}

	html := buildCrawlHTML(items)

	// Check that HTML contains key elements
	if !strings.Contains(html, "Crawl Results") {
		t.Error("HTML should contain 'Crawl Results' heading")
	}
	if !strings.Contains(html, "Page 1") {
		t.Error("HTML should contain first page title")
	}
	if !strings.Contains(html, "Page 2") {
		t.Error("HTML should contain second page title")
	}
	if !strings.Contains(html, "Total Pages:") {
		t.Error("HTML should contain total pages count")
	}
}

func TestBuildResearchHTML(t *testing.T) {
	item := ResearchResult{
		Query:      "test query",
		Summary:    "Test summary",
		Confidence: 0.95,
		Evidence: []struct {
			URL         string  `json:"url"`
			Title       string  `json:"title"`
			Snippet     string  `json:"snippet"`
			Score       float64 `json:"score"`
			SimHash     uint64  `json:"simhash"`
			ClusterID   string  `json:"clusterId"`
			Confidence  float64 `json:"confidence"`
			CitationURL string  `json:"citationUrl"`
		}{
			{
				URL:         "https://example.com/evidence1",
				Title:       "Evidence 1",
				Snippet:     "Test snippet",
				Score:       0.9,
				SimHash:     1234567890,
				ClusterID:   "cluster1",
				Confidence:  0.9,
				CitationURL: "https://example.com/cite1",
			},
		},
		Clusters: []struct {
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
		}{
			{
				ID:         "cluster1",
				Label:      "Test Cluster",
				Confidence: 0.9,
			},
		},
		Citations: []struct {
			URL       string `json:"url"`
			Anchor    string `json:"anchor"`
			Canonical string `json:"canonical"`
		}{
			{
				URL:       "https://example.com/cite1",
				Anchor:    "section1",
				Canonical: "https://example.com/canonical1",
			},
		},
	}

	html := buildResearchHTML(item)

	// Check that HTML contains key elements
	if !strings.Contains(html, "Research Report") {
		t.Error("HTML should contain 'Research Report' heading")
	}
	if !strings.Contains(html, "test query") {
		t.Error("HTML should contain query")
	}
	if !strings.Contains(html, "Test summary") {
		t.Error("HTML should contain summary")
	}
	if !strings.Contains(html, "Evidence Clusters") {
		t.Error("HTML should contain clusters section")
	}
	if !strings.Contains(html, "Citations") {
		t.Error("HTML should contain citations section")
	}
	if !strings.Contains(html, "Evidence") {
		t.Error("HTML should contain evidence section")
	}
}

func TestFormatDescriptionHTML(t *testing.T) {
	// Test with non-empty description
	result := formatDescriptionHTML("Test description")
	if !strings.Contains(result, "Test description") {
		t.Error("formatDescriptionHTML should include the description")
	}
	if !strings.Contains(result, "Description:") {
		t.Error("formatDescriptionHTML should include 'Description:' label")
	}

	// Test with empty description
	result = formatDescriptionHTML("")
	if result != "" {
		t.Errorf("formatDescriptionHTML('') should return empty string, got: %q", result)
	}
}
