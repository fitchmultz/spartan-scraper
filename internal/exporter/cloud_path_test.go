// Package exporter provides tests for cloud storage path template functionality.
//
// These tests verify path template rendering and normalization for cloud
// storage exports, including timestamp formatting and variable substitution.
package exporter

import (
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestRenderPathTemplate(t *testing.T) {
	job := model.Job{
		ID:   "job-abc123",
		Kind: model.KindScrape,
	}

	tests := []struct {
		name     string
		template string
		format   string
		want     []string // Substrings that should be present
	}{
		{
			name:     "default template",
			template: "",
			format:   "jsonl",
			want:     []string{"scrape/", ".jsonl"},
		},
		{
			name:     "with job_id",
			template: "exports/{job_id}.{format}",
			format:   "csv",
			want:     []string{"exports/job-abc123.csv"},
		},
		{
			name:     "with kind",
			template: "{kind}/data.{format}",
			format:   "json",
			want:     []string{"scrape/data.json"},
		},
		{
			name:     "fixed path",
			template: "fixed/path.jsonl",
			format:   "jsonl",
			want:     []string{"fixed/path.jsonl"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderPathTemplate(tt.template, job, tt.format)
			for _, want := range tt.want {
				assert.Contains(t, got, want)
			}
		})
	}
}

func TestRenderPathTemplate_TimestampFormat(t *testing.T) {
	job := model.Job{
		ID:   "job-test",
		Kind: model.KindCrawl,
	}

	// Timestamp should be in format 20060102_150405
	template := "{timestamp}"
	result := RenderPathTemplate(template, job, "jsonl")

	// Parse the result to verify it's a valid timestamp
	_, err := time.Parse("20060102_150405", result)
	assert.NoError(t, err, "timestamp should be parseable in expected format")
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/path/to/file", "path/to/file"},
		{"path/to/file", "path/to/file"},
		{"///path/to/file", "path/to/file"},
		{"/", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizePath(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
