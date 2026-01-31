// Package exporter provides integration tests for cloud storage functionality.
//
// These tests verify end-to-end cloud export workflows using the memblob
// driver for in-memory bucket testing without requiring real cloud credentials.
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

func TestExportStreamWithDatabaseAndCloud(t *testing.T) {
	job := model.Job{
		ID:   "job-test",
		Kind: model.KindScrape,
	}
	testData := `{"url":"http://example.com"}`

	// Test with no special config - should use regular export
	var buf bytes.Buffer
	err := ExportStreamWithDatabaseAndCloud(job, strings.NewReader(testData), "jsonl", &buf, nil, nil)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "http://example.com")

	// Test with cloud format but no config - should error
	err = ExportStreamWithDatabaseAndCloud(job, strings.NewReader(testData), "s3", &buf, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cloud config required")
}

// Integration test with memory bucket
func TestCloudExport_Integration(t *testing.T) {
	ctx := context.Background()

	// Open memory bucket
	bucket, err := blob.OpenBucket(ctx, "mem://integration-test")
	require.NoError(t, err)
	defer bucket.Close()

	job := model.Job{
		ID:   "integration-job",
		Kind: model.KindScrape,
	}

	// Create test data that looks like a scrape result
	testData := `{"url":"http://example.com","status":200,"title":"Test Page","text":"Hello World"}`

	// Test path rendering
	path := RenderPathTemplate("{kind}/{job_id}.{format}", job, "jsonl")
	assert.Equal(t, "scrape/integration-job.jsonl", path)

	// Write to bucket
	writer, err := bucket.NewWriter(ctx, path, &blob.WriterOptions{
		ContentType: "application/json",
	})
	require.NoError(t, err)

	_, err = io.Copy(writer, strings.NewReader(testData))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	// Read back
	reader, err := bucket.NewReader(ctx, path, nil)
	require.NoError(t, err)

	content, err := io.ReadAll(reader)
	require.NoError(t, err)
	reader.Close()

	assert.Equal(t, testData, string(content))

	// Verify attributes
	attrs, err := bucket.Attributes(ctx, path)
	require.NoError(t, err)
	assert.Equal(t, "application/json", attrs.ContentType)
	assert.Equal(t, int64(len(testData)), attrs.Size)

	// Clean up
	require.NoError(t, bucket.Delete(ctx, path))
}
