// Package exporter provides tests for cloud storage export functionality.
//
// These tests use the in-memory blob driver (mem://) to avoid requiring
// real cloud credentials.
package exporter

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/memblob"
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

func TestContentTypeForFormat(t *testing.T) {
	tests := []struct {
		format string
		want   string
	}{
		{"json", "application/json"},
		{"jsonl", "application/json"},
		{"md", "text/markdown"},
		{"csv", "text/csv"},
		{"xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		{"parquet", "application/octet-stream"},
		{"har", "application/json"},
		{"unknown", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			got := contentTypeForFormat(tt.format)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolveBucketURL(t *testing.T) {
	tests := []struct {
		name    string
		cfg     CloudExportConfig
		want    string
		wantErr bool
	}{
		{
			name: "s3 without region",
			cfg: CloudExportConfig{
				Provider: "s3",
				Bucket:   "my-bucket",
			},
			want: "s3://my-bucket",
		},
		{
			name: "s3 with region",
			cfg: CloudExportConfig{
				Provider: "s3",
				Bucket:   "my-bucket",
				Region:   "us-west-2",
			},
			want: "s3://my-bucket?region=us-west-2",
		},
		{
			name: "gcs",
			cfg: CloudExportConfig{
				Provider: "gcs",
				Bucket:   "my-bucket",
			},
			want: "gs://my-bucket",
		},
		{
			name: "azure",
			cfg: CloudExportConfig{
				Provider: "azure",
				Bucket:   "my-container",
			},
			want: "azblob://my-container",
		},
		{
			name: "unsupported provider",
			cfg: CloudExportConfig{
				Provider: "unknown",
				Bucket:   "my-bucket",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveBucketURL(tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
				assert.True(t, apperrors.IsKind(err, apperrors.KindValidation))
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsCloudFormat(t *testing.T) {
	tests := []struct {
		format string
		want   bool
	}{
		{"s3", true},
		{"gcs", true},
		{"azure", true},
		{"json", false},
		{"jsonl", false},
		{"csv", false},
		{"postgres", false},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			got := IsCloudFormat(tt.format)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsValidCloudProvider(t *testing.T) {
	tests := []struct {
		provider string
		want     bool
	}{
		{"s3", true},
		{"gcs", true},
		{"azure", true},
		{"S3", false}, // case sensitive
		{"unknown", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			got := IsValidCloudProvider(tt.provider)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCloudProviderDisplayName(t *testing.T) {
	tests := []struct {
		provider string
		want     string
	}{
		{"s3", "AWS S3"},
		{"gcs", "Google Cloud Storage"},
		{"azure", "Azure Blob Storage"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			got := CloudProviderDisplayName(tt.provider)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestListCloudProviders(t *testing.T) {
	providers := ListCloudProviders()
	assert.Equal(t, []string{"s3", "gcs", "azure"}, providers)
}

func TestValidateCloudConfig(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		cfg     CloudExportConfig
		wantErr bool
		errKind apperrors.Kind
	}{
		{
			name:    "not a cloud format, no provider",
			format:  "json",
			cfg:     CloudExportConfig{},
			wantErr: false,
		},
		{
			name:   "s3 format missing bucket",
			format: "s3",
			cfg: CloudExportConfig{
				Provider: "s3",
			},
			wantErr: true,
			errKind: apperrors.KindValidation,
		},
		{
			name:   "valid s3 config",
			format: "s3",
			cfg: CloudExportConfig{
				Provider: "s3",
				Bucket:   "my-bucket",
			},
			wantErr: false,
		},
		{
			name:   "unsupported provider",
			format: "json",
			cfg: CloudExportConfig{
				Provider: "unknown",
				Bucket:   "my-bucket",
			},
			wantErr: true,
			errKind: apperrors.KindValidation,
		},
		{
			name:   "invalid content format",
			format: "json",
			cfg: CloudExportConfig{
				Provider:      "s3",
				Bucket:        "my-bucket",
				ContentFormat: "invalid",
			},
			wantErr: true,
			errKind: apperrors.KindValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCloudConfig(tt.format, tt.cfg)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errKind != "" {
					assert.True(t, apperrors.IsKind(err, tt.errKind))
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

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

func TestExtractCloudConfigFromParams(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]interface{}
		want   *CloudExportConfig
	}{
		{
			name:   "nil params",
			params: nil,
			want:   nil,
		},
		{
			name:   "empty params",
			params: map[string]interface{}{},
			want:   nil,
		},
		{
			name: "full config",
			params: map[string]interface{}{
				"cloudProvider":     "s3",
				"cloudBucket":       "my-bucket",
				"cloudPath":         "exports/{job_id}.jsonl",
				"cloudRegion":       "us-west-2",
				"cloudStorageClass": "STANDARD_IA",
				"cloudFormat":       "jsonl",
				"cloudContentType":  "application/json",
			},
			want: &CloudExportConfig{
				Provider:      "s3",
				Bucket:        "my-bucket",
				Path:          "exports/{job_id}.jsonl",
				Region:        "us-west-2",
				StorageClass:  "STANDARD_IA",
				ContentFormat: "jsonl",
				ContentType:   "application/json",
			},
		},
		{
			name: "partial config",
			params: map[string]interface{}{
				"cloudProvider": "gcs",
				"cloudBucket":   "my-bucket",
			},
			want: &CloudExportConfig{
				Provider: "gcs",
				Bucket:   "my-bucket",
			},
		},
		{
			name: "no provider",
			params: map[string]interface{}{
				"cloudBucket": "my-bucket",
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractCloudConfigFromParams(tt.params)
			assert.Equal(t, tt.want, got)
		})
	}
}

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
