// Package scheduler provides tests for export schedule types.
package scheduler

import (
	"testing"
)

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()

	if cfg.MaxRetries != 3 {
		t.Errorf("expected MaxRetries=3, got %d", cfg.MaxRetries)
	}

	if cfg.BaseDelayMs != 1000 {
		t.Errorf("expected BaseDelayMs=1000, got %d", cfg.BaseDelayMs)
	}
}

func TestExportRetryConfig_GetMaxRetries(t *testing.T) {
	tests := []struct {
		name     string
		cfg      ExportRetryConfig
		expected int
	}{
		{"default values", ExportRetryConfig{}, 3},
		{"zero values", ExportRetryConfig{MaxRetries: 0}, 3},
		{"negative values", ExportRetryConfig{MaxRetries: -1}, 3},
		{"positive value", ExportRetryConfig{MaxRetries: 5}, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.GetMaxRetries()
			if got != tt.expected {
				t.Errorf("GetMaxRetries() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestIsValidExportFormat(t *testing.T) {
	validFormats := []string{"json", "jsonl", "md", "csv", "xlsx"}
	for _, format := range validFormats {
		if !IsValidExportFormat(format) {
			t.Errorf("IsValidExportFormat(%q) = false, want true", format)
		}
	}

	invalidFormats := []string{"xml", "yaml", "txt", "parquet", "har", "pdf", ""}
	for _, format := range invalidFormats {
		if IsValidExportFormat(format) {
			t.Errorf("IsValidExportFormat(%q) = true, want false", format)
		}
	}
}

func TestIsValidDestinationType(t *testing.T) {
	validDests := []string{"local", "webhook"}
	for _, dest := range validDests {
		if !IsValidDestinationType(dest) {
			t.Errorf("IsValidDestinationType(%q) = false, want true", dest)
		}
	}

	invalidDests := []string{"ftp", "sftp", "email", "s3", "gcs", "azure", ""}
	for _, dest := range invalidDests {
		if IsValidDestinationType(dest) {
			t.Errorf("IsValidDestinationType(%q) = true, want false", dest)
		}
	}
}

func TestIsCloudDestination(t *testing.T) {
	nonCloudDests := []string{"local", "webhook", "ftp", "s3", "gcs", "azure"}
	for _, dest := range nonCloudDests {
		if IsCloudDestination(dest) {
			t.Errorf("IsCloudDestination(%q) = true, want false", dest)
		}
	}
}
