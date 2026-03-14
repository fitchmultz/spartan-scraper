package exporter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/xuri/excelize/v2"
)

func TestExportWithShapeScrapeCSVSelectsConfiguredFields(t *testing.T) {
	job := model.Job{Kind: model.KindScrape}
	raw := []byte(`{"url":"https://example.com","status":200,"title":"Example","text":"Body","metadata":{"description":"Desc"},"normalized":{"title":"Example","fields":{"price":{"values":["$10"]},"plan":{"values":["Pro"]}}}}`)
	shape := ShapeConfig{
		TopLevelFields:   []string{"url", "title"},
		NormalizedFields: []string{"field.price"},
		FieldLabels:      map[string]string{"field.price": "Price"},
	}

	result, err := ExportWithShape(job, raw, "csv", shape)
	if err != nil {
		t.Fatalf("ExportWithShape() failed: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(result), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 CSV lines, got %d", len(lines))
	}
	if lines[0] != "Url,Title,Price" {
		t.Fatalf("unexpected shaped header: %s", lines[0])
	}
	if lines[1] != "https://example.com,Example,$10" {
		t.Fatalf("unexpected shaped row: %s", lines[1])
	}
}

func TestExportWithShapeScrapeMarkdownUsesSummaryFieldsAndTitle(t *testing.T) {
	job := model.Job{Kind: model.KindScrape}
	raw := []byte(`{"url":"https://example.com","status":200,"title":"Example","text":"Body","metadata":{"description":"Desc"},"normalized":{"title":"Example","fields":{"price":{"values":["$10"]}}}}`)
	shape := ShapeConfig{
		SummaryFields: []string{"title", "field.price", "url"},
		Formatting: ExportFormattingHints{
			MarkdownTitle: "Pricing Export",
		},
		FieldLabels: map[string]string{"field.price": "Price"},
	}

	result, err := ExportWithShape(job, raw, "md", shape)
	if err != nil {
		t.Fatalf("ExportWithShape() failed: %v", err)
	}
	for _, want := range []string{"# Pricing Export", "## Summary", "**Price**: $10", "**Url**: https://example.com"} {
		if !strings.Contains(result, want) {
			t.Fatalf("expected markdown to contain %q\n%s", want, result)
		}
	}
}

func TestExportStreamWithShapeResearchXLSXUsesConfiguredHeaders(t *testing.T) {
	job := model.Job{Kind: model.KindResearch}
	raw := strings.NewReader(`{"query":"pricing","summary":"Summary","confidence":0.73,"agentic":{"status":"completed","summary":"Agentic summary"},"evidence":[{"url":"https://example.com/pricing","title":"Pricing","snippet":"Contact sales","score":0.9,"confidence":0.8,"clusterId":"cluster-1","citationUrl":"https://example.com/pricing"}],"clusters":[],"citations":[]}`)
	shape := ShapeConfig{
		TopLevelFields: []string{"query", "agentic.summary"},
		EvidenceFields: []string{"evidence.url", "evidence.title"},
		FieldLabels: map[string]string{
			"agentic.summary": "Agentic Summary",
			"evidence.url":    "Source URL",
		},
	}

	var buf bytes.Buffer
	if err := ExportStreamWithShape(job, raw, "xlsx", shape, &buf); err != nil {
		t.Fatalf("ExportStreamWithShape() failed: %v", err)
	}
	f, err := excelize.OpenReader(&buf)
	if err != nil {
		t.Fatalf("OpenReader() failed: %v", err)
	}
	defer f.Close()
	rows, err := f.GetRows("Summary")
	if err != nil {
		t.Fatalf("GetRows(Summary) failed: %v", err)
	}
	if len(rows) < 2 {
		t.Fatalf("expected summary rows, got %d", len(rows))
	}
	if rows[0][0] != "Query" || rows[0][1] != "Agentic Summary" {
		t.Fatalf("unexpected summary headers: %#v", rows[0])
	}
	evidenceRows, err := f.GetRows("Evidence")
	if err != nil {
		t.Fatalf("GetRows(Evidence) failed: %v", err)
	}
	if len(evidenceRows) < 2 || evidenceRows[0][0] != "Source URL" || evidenceRows[0][1] != "Title" {
		t.Fatalf("unexpected evidence headers: %#v", evidenceRows)
	}
}
