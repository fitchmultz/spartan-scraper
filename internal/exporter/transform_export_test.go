package exporter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestExportWithShapeAndTransformJSONLProjectsResults(t *testing.T) {
	job := model.Job{Kind: model.KindCrawl}
	raw := []byte(strings.Join([]string{
		`{"url":"https://example.com/a","title":"A","status":200}`,
		`{"url":"https://example.com/b","title":"B","status":200}`,
	}, "\n"))

	result, err := ExportWithShapeAndTransform(job, raw, "jsonl", ShapeConfig{}, TransformConfig{
		Expression: "{title: title, url: url}",
		Language:   "jmespath",
	})
	if err != nil {
		t.Fatalf("ExportWithShapeAndTransform() failed: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(result), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 transformed lines, got %d", len(lines))
	}
	if strings.Contains(result, "status") {
		t.Fatalf("expected transformed jsonl to omit status: %s", result)
	}
}

func TestExportStreamWithShapeAndTransformRejectsShapeAndTransform(t *testing.T) {
	job := model.Job{Kind: model.KindScrape}
	raw := strings.NewReader(`{"url":"https://example.com","title":"Example","normalized":{"fields":{}}}`)
	var buf bytes.Buffer
	err := ExportStreamWithShapeAndTransform(job, raw, "csv", ShapeConfig{
		TopLevelFields: []string{"url"},
	}, TransformConfig{
		Expression: "{url: url}",
		Language:   "jmespath",
	}, &buf)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "cannot be combined") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExportStreamWithShapeAndTransformMarkdownUsesGenericRenderer(t *testing.T) {
	job := model.Job{Kind: model.KindScrape}
	raw := strings.NewReader(`{"url":"https://example.com","title":"Example","status":200,"normalized":{"fields":{}}}`)
	var buf bytes.Buffer
	if err := ExportStreamWithShapeAndTransform(job, raw, "md", ShapeConfig{}, TransformConfig{
		Expression: "{title: title, url: url}",
		Language:   "jmespath",
	}, &buf); err != nil {
		t.Fatalf("ExportStreamWithShapeAndTransform() failed: %v", err)
	}
	result := buf.String()
	for _, want := range []string{"# Transformed Results", "**title**: Example", "**url**: https://example.com"} {
		if !strings.Contains(strings.ToLower(result), strings.ToLower(want)) {
			t.Fatalf("expected markdown to contain %q\n%s", want, result)
		}
	}
}
