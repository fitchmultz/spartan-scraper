// Package exporter provides tests for cloud storage configuration functionality.
//
// These tests verify cloud export configuration validation and extraction
// from job parameters, including content type detection for various formats.
package exporter

import (
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
