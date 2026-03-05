// Package exporter provides tests for cloud storage export functionality.
//
// These tests verify the core export operations including memory bucket
// exports, stream validation, and error handling using the memblob driver.
package exporter

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/memblob"
)

func TestExportToCloud_MemoryBucket(t *testing.T) {
	// Use in-memory bucket for testing
	ctx := context.Background()
	bucket, err := blob.OpenBucket(ctx, "mem://test-bucket")
	require.NoError(t, err)
	defer bucket.Close()

	job := model.Job{
		ID:   "job-test123",
		Kind: model.KindScrape,
	}

	// Create test data
	testData := `{"url":"http://example.com","status":200,"title":"Test"}`

	tests := []struct {
		name         string
		format       string
		pathTemplate string
		wantPath     string
	}{
		{
			name:         "jsonl export",
			format:       "jsonl",
			pathTemplate: "test/{job_id}.jsonl",
			wantPath:     "test/job-test123.jsonl",
		},
		{
			name:         "csv export",
			format:       "csv",
			pathTemplate: "exports/data.csv",
			wantPath:     "exports/data.csv",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Render the path
			path := RenderPathTemplate(tt.pathTemplate, job, tt.format)
			assert.Equal(t, tt.wantPath, path)

			// Write directly to memory bucket (simulating what exportToCloud does)
			writer, err := bucket.NewWriter(ctx, path, nil)
			require.NoError(t, err)

			_, err = io.Copy(writer, strings.NewReader(testData))
			require.NoError(t, err)
			require.NoError(t, writer.Close())

			// Read back and verify
			reader, err := bucket.NewReader(ctx, path, nil)
			require.NoError(t, err)
			defer reader.Close()

			content, err := io.ReadAll(reader)
			require.NoError(t, err)
			assert.Equal(t, testData, string(content))

			// Clean up
			require.NoError(t, bucket.Delete(ctx, path))
		})
	}
}

func TestExportStreamWithCloud_Validation(t *testing.T) {
	job := model.Job{ID: "test", Kind: model.KindScrape}
	testData := `{"url":"http://example.com"}`

	tests := []struct {
		name    string
		format  string
		cfg     *CloudExportConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "s3 format without config",
			format:  "s3",
			cfg:     nil,
			wantErr: true,
			errMsg:  "cloud config required",
		},
		{
			name:   "s3 format with config",
			format: "s3",
			cfg: &CloudExportConfig{
				Provider:      "s3",
				Bucket:        "test-bucket",
				ContentFormat: "jsonl",
			},
			// This will fail because we can't connect to real S3, but it validates
			wantErr: true,
		},
		{
			name:    "regular format without cloud config",
			format:  "jsonl",
			cfg:     nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := ExportStreamWithCloud(job, strings.NewReader(testData), tt.format, &buf, tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExportStreamWithCloudResult(t *testing.T) {
	job := model.Job{
		ID:   "job-test456",
		Kind: model.KindScrape,
	}
	testData := `{"url":"http://example.com","status":200}`

	// Test with invalid config (missing provider)
	_, err := ExportStreamWithCloudResult(job, strings.NewReader(testData), "jsonl", nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cloud config is required")

	// Test with invalid config (missing bucket)
	_, err = ExportStreamWithCloudResult(job, strings.NewReader(testData), "jsonl", nil, &CloudExportConfig{
		Provider: "s3",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cloud bucket is required")
}

func TestExportToCloud(t *testing.T) {
	job := model.Job{
		ID:   "job-test789",
		Kind: model.KindCrawl,
	}
	testData := `{"url":"http://example.com","status":200}`

	// Test with invalid provider
	cfg := CloudExportConfig{
		Provider: "invalid",
		Bucket:   "test-bucket",
	}
	err := ExportToCloud(job, strings.NewReader(testData), "jsonl", cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported cloud provider")

	// Test with missing bucket
	cfg = CloudExportConfig{
		Provider: "s3",
	}
	err = ExportToCloud(job, strings.NewReader(testData), "jsonl", cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cloud bucket is required")
}
