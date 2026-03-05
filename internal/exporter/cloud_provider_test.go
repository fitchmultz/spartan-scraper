// Package exporter provides tests for cloud storage provider functionality.
//
// These tests verify provider validation, display names, and bucket URL
// resolution for AWS S3, Google Cloud Storage, and Azure Blob Storage.
package exporter

import (
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
