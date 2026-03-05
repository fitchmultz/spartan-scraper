// Package exporter provides tests for cloud storage result types and utilities.
//
// These tests verify CloudExportResult methods, error handling, and string
// replacement utilities used in path template rendering.
package exporter

import (
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/stretchr/testify/assert"
)

func TestCloudExportResult_ToCloudURL(t *testing.T) {
	tests := []struct {
		name   string
		result CloudExportResult
		want   string
	}{
		{
			name: "s3",
			result: CloudExportResult{
				Provider: "s3",
				Bucket:   "my-bucket",
				Path:     "path/to/file.jsonl",
			},
			want: "s3://my-bucket/path/to/file.jsonl",
		},
		{
			name: "gcs",
			result: CloudExportResult{
				Provider: "gcs",
				Bucket:   "my-bucket",
				Path:     "exports/data.csv",
			},
			want: "gs://my-bucket/exports/data.csv",
		},
		{
			name: "azure",
			result: CloudExportResult{
				Provider: "azure",
				Bucket:   "my-container",
				Path:     "results/output.json",
			},
			want: "azblob://my-container/results/output.json",
		},
		{
			name: "unknown",
			result: CloudExportResult{
				Provider: "custom",
				Bucket:   "my-bucket",
				Path:     "file.txt",
			},
			want: "custom://my-bucket/file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.ToCloudURL()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCloudExportResult_String(t *testing.T) {
	result := CloudExportResult{
		Provider:    "s3",
		Bucket:      "my-bucket",
		Path:        "exports/data.jsonl",
		ContentType: "application/json",
		Size:        1024,
	}

	got := result.String()
	assert.Contains(t, got, "s3://my-bucket/exports/data.jsonl")
	assert.Contains(t, got, "application/json")
	assert.Contains(t, got, "1024 bytes")
}

func TestCloudExportError(t *testing.T) {
	originalErr := apperrors.Internal("connection refused")
	err := &CloudExportError{
		Provider: "s3",
		Bucket:   "my-bucket",
		Path:     "exports/data.jsonl",
		Op:       "upload",
		Err:      originalErr,
	}

	// Test Error() string
	got := err.Error()
	assert.Contains(t, got, "s3")
	assert.Contains(t, got, "my-bucket")
	assert.Contains(t, got, "upload")
	assert.Contains(t, got, "connection refused")

	// Test Unwrap
	assert.Equal(t, originalErr, err.Unwrap())
}

func TestReplaceAll(t *testing.T) {
	tests := []struct {
		s    string
		old  string
		new  string
		want string
	}{
		{"hello world", "world", "universe", "hello universe"},
		{"foo bar foo", "foo", "baz", "baz bar baz"},
		{"no match", "xyz", "abc", "no match"},
		{"", "a", "b", ""},
		{"aaa", "a", "b", "bbb"},
		{"{job_id}/{format}", "{job_id}", "abc123", "abc123/{format}"},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			got := replaceAll(tt.s, tt.old, tt.new)
			assert.Equal(t, tt.want, got)
		})
	}
}
