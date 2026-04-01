// Package scheduler provides tests for export schedule validation.
package scheduler

import (
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/exporter"
)

func TestValidateExportSchedule(t *testing.T) {
	tests := []struct {
		name      string
		schedule  ExportSchedule
		wantError bool
		errorKind apperrors.Kind
	}{
		{
			name: "valid schedule",
			schedule: ExportSchedule{
				Name: "Test Schedule",
				Filters: ExportFilters{
					JobKinds: []string{"crawl"},
				},
				Export: ExportConfig{
					Format:          "jsonl",
					DestinationType: "local",
					LocalPath:       "/tmp/exports/{job_id}.jsonl",
				},
			},
			wantError: false,
		},
		{
			name: "valid schedule with transform",
			schedule: ExportSchedule{
				Name: "Projected Schedule",
				Filters: ExportFilters{
					JobKinds: []string{"scrape"},
				},
				Export: ExportConfig{
					Format:          "csv",
					DestinationType: "local",
					LocalPath:       "/tmp/exports/{job_id}.csv",
					Transform: exporter.TransformConfig{
						Expression: "{title: title, url: url}",
						Language:   "jmespath",
					},
				},
			},
			wantError: false,
		},
		{
			name: "missing name",
			schedule: ExportSchedule{
				Name: "",
				Filters: ExportFilters{
					JobKinds: []string{"crawl"},
				},
				Export: ExportConfig{
					Format:          "jsonl",
					DestinationType: "local",
					LocalPath:       "/tmp/exports/{job_id}.jsonl",
				},
			},
			wantError: true,
			errorKind: apperrors.KindValidation,
		},
		{
			name: "whitespace name",
			schedule: ExportSchedule{
				Name: "   ",
				Filters: ExportFilters{
					JobKinds: []string{"crawl"},
				},
				Export: ExportConfig{
					Format:          "jsonl",
					DestinationType: "local",
					LocalPath:       "/tmp/exports/{job_id}.jsonl",
				},
			},
			wantError: true,
			errorKind: apperrors.KindValidation,
		},
		{
			name: "no filters",
			schedule: ExportSchedule{
				Name:    "Test Schedule",
				Filters: ExportFilters{},
				Export: ExportConfig{
					Format:          "jsonl",
					DestinationType: "local",
					LocalPath:       "/tmp/exports/{job_id}.jsonl",
				},
			},
			wantError: true,
			errorKind: apperrors.KindValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExportSchedule(tt.schedule)
			if tt.wantError {
				if err == nil {
					t.Errorf("ValidateExportSchedule() error = nil, want error")
					return
				}
				if tt.errorKind != "" {
					if !apperrors.IsKind(err, tt.errorKind) {
						t.Errorf("ValidateExportSchedule() error kind = %v, want %v", apperrors.KindOf(err), tt.errorKind)
					}
				}
			} else {
				if err != nil {
					t.Errorf("ValidateExportSchedule() error = %v, want nil", err)
				}
			}
		})
	}
}

func TestValidateExportFilters(t *testing.T) {
	tests := []struct {
		name      string
		filters   ExportFilters
		wantError bool
	}{
		{
			name:      "no filters",
			filters:   ExportFilters{},
			wantError: true,
		},
		{
			name: "valid job kinds",
			filters: ExportFilters{
				JobKinds: []string{"scrape", "crawl"},
			},
			wantError: false,
		},
		{
			name: "invalid job kind",
			filters: ExportFilters{
				JobKinds: []string{"invalid"},
			},
			wantError: true,
		},
		{
			name: "valid job status",
			filters: ExportFilters{
				JobStatus: []string{"completed", "failed"},
			},
			wantError: false,
		},
		{
			name: "invalid job status",
			filters: ExportFilters{
				JobStatus: []string{"invalid"},
			},
			wantError: true,
		},
		{
			name: "has results only",
			filters: ExportFilters{
				HasResults: true,
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExportFilters(tt.filters)
			if tt.wantError {
				if err == nil {
					t.Errorf("ValidateExportFilters() error = nil, want error")
				}
			} else {
				if err != nil {
					t.Errorf("ValidateExportFilters() error = %v, want nil", err)
				}
			}
		})
	}
}

func TestValidateExportConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    ExportConfig
		wantError bool
	}{
		{
			name: "valid local config",
			config: ExportConfig{
				Format:          "jsonl",
				DestinationType: "local",
				LocalPath:       "/tmp/exports/{job_id}.jsonl",
			},
			wantError: false,
		},
		{
			name: "missing format",
			config: ExportConfig{
				DestinationType: "local",
				LocalPath:       "/tmp/exports/{job_id}.jsonl",
			},
			wantError: true,
		},
		{
			name: "invalid format",
			config: ExportConfig{
				Format:          "xml",
				DestinationType: "local",
				LocalPath:       "/tmp/exports/{job_id}.jsonl",
			},
			wantError: true,
		},
		{
			name: "missing destination",
			config: ExportConfig{
				Format: "jsonl",
			},
			wantError: true,
		},
		{
			name: "invalid destination",
			config: ExportConfig{
				Format:          "jsonl",
				DestinationType: "ftp",
			},
			wantError: true,
		},
		{
			name: "local without path uses shared default",
			config: ExportConfig{
				Format:          "jsonl",
				DestinationType: "local",
			},
			wantError: false,
		},
		{
			name: "webhook without url",
			config: ExportConfig{
				Format:          "jsonl",
				DestinationType: "webhook",
			},
			wantError: true,
		},
		{
			name: "valid webhook",
			config: ExportConfig{
				Format:          "jsonl",
				DestinationType: "webhook",
				WebhookURL:      "https://example.com/webhook",
			},
			wantError: false,
		},
		{
			name: "unsupported destination",
			config: ExportConfig{
				Format:          "jsonl",
				DestinationType: "s3",
			},
			wantError: true,
		},
		{
			name: "valid transform config",
			config: ExportConfig{
				Format:          "csv",
				DestinationType: "local",
				LocalPath:       "/tmp/exports/{job_id}.csv",
				Transform: exporter.TransformConfig{
					Expression: "{title: title, url: url}",
					Language:   "jmespath",
				},
			},
			wantError: false,
		},
		{
			name: "invalid transform config",
			config: ExportConfig{
				Format:          "csv",
				DestinationType: "local",
				LocalPath:       "/tmp/exports/{job_id}.csv",
				Transform: exporter.TransformConfig{
					Expression: "[",
					Language:   "jmespath",
				},
			},
			wantError: true,
		},
		{
			name: "shape and transform cannot be combined",
			config: ExportConfig{
				Format:          "md",
				DestinationType: "local",
				LocalPath:       "/tmp/exports/{job_id}.md",
				Shape: exporter.ShapeConfig{
					TopLevelFields: []string{"url"},
				},
				Transform: exporter.TransformConfig{
					Expression: "{title: title}",
					Language:   "jmespath",
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExportConfig(tt.config)
			if tt.wantError {
				if err == nil {
					t.Errorf("ValidateExportConfig() error = nil, want error")
				}
			} else {
				if err != nil {
					t.Errorf("ValidateExportConfig() error = %v, want nil", err)
				}
			}
		})
	}
}

func TestValidateExportConfig_WebhookURLValidation(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantError bool
	}{
		{"valid https", "https://example.com/webhook", false},
		{"valid http", "http://example.com/webhook", false},
		{"invalid scheme", "ftp://example.com/webhook", true},
		{"missing host", "https:///webhook", true},
		{"invalid url", "://invalid", true},
		{"empty url", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExportConfig(ExportConfig{
				Format:          "json",
				DestinationType: "webhook",
				WebhookURL:      tt.url,
			})
			if tt.wantError {
				if err == nil {
					t.Fatalf("ValidateExportConfig() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ValidateExportConfig() error = %v, want nil", err)
			}
		})
	}
}
