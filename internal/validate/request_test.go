// Package validate provides tests for job request validation across all job types.
// Tests cover URL validation, timeout bounds, auth profile name validation, and job-specific rules for scrape, crawl, and research jobs.
// Does NOT test individual field validators (see validate_test.go) or non-job validation logic.
package validate

import (
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestValidateJob_Scrape(t *testing.T) {
	tests := []struct {
		name        string
		opts        JobValidationOpts
		wantErr     bool
		errContains string
	}{
		{
			name: "valid request with all fields",
			opts: JobValidationOpts{
				URL:         "https://example.com",
				Timeout:     30,
				AuthProfile: "test-profile",
			},
			wantErr: false,
		},
		{
			name: "valid request with zero timeout",
			opts: JobValidationOpts{
				URL:         "https://example.com",
				Timeout:     0,
				AuthProfile: "test-profile",
			},
			wantErr: false,
		},
		{
			name: "invalid URL - wrong scheme",
			opts: JobValidationOpts{
				URL:         "ftp://example.com",
				Timeout:     30,
				AuthProfile: "test-profile",
			},
			wantErr:     true,
			errContains: "invalid url: must be http or https and have a host",
		},
		{
			name: "invalid URL - missing host",
			opts: JobValidationOpts{
				URL:         "https://",
				Timeout:     30,
				AuthProfile: "test-profile",
			},
			wantErr:     true,
			errContains: "invalid url: must have a host",
		},
		{
			name: "empty URL",
			opts: JobValidationOpts{
				URL:         "",
				Timeout:     30,
				AuthProfile: "test-profile",
			},
			wantErr:     true,
			errContains: "url is required",
		},
		{
			name: "invalid timeout - too low",
			opts: JobValidationOpts{
				URL:         "https://example.com",
				Timeout:     4,
				AuthProfile: "test-profile",
			},
			wantErr:     true,
			errContains: "timeoutSeconds must be between 5 and 300",
		},
		{
			name: "invalid timeout - too high",
			opts: JobValidationOpts{
				URL:         "https://example.com",
				Timeout:     301,
				AuthProfile: "test-profile",
			},
			wantErr:     true,
			errContains: "timeoutSeconds must be between 5 and 300",
		},
		{
			name: "invalid authProfile - invalid chars",
			opts: JobValidationOpts{
				URL:         "https://example.com",
				Timeout:     30,
				AuthProfile: "test profile",
			},
			wantErr:     true,
			errContains: "invalid authProfile: only alphanumeric, hyphens, and underscores allowed",
		},
		{
			name: "empty authProfile is valid",
			opts: JobValidationOpts{
				URL:         "https://example.com",
				Timeout:     30,
				AuthProfile: "",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateJob(tt.opts, model.KindScrape)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateJob() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errContains != "" {
				if !containsString(err.Error(), tt.errContains) {
					t.Errorf("Error = %v, want containing %v", err.Error(), tt.errContains)
				}
			}
		})
	}
}

func TestValidateJob_Crawl(t *testing.T) {
	tests := []struct {
		name        string
		opts        JobValidationOpts
		wantErr     bool
		errContains string
	}{
		{
			name: "valid request with all fields",
			opts: JobValidationOpts{
				URL:         "https://example.com",
				MaxDepth:    3,
				MaxPages:    100,
				Timeout:     30,
				AuthProfile: "test-profile",
			},
			wantErr: false,
		},
		{
			name: "valid request with zero maxDepth/maxPages",
			opts: JobValidationOpts{
				URL:         "https://example.com",
				MaxDepth:    0,
				MaxPages:    0,
				Timeout:     30,
				AuthProfile: "test-profile",
			},
			wantErr: false,
		},
		{
			name: "invalid URL - wrong scheme",
			opts: JobValidationOpts{
				URL:         "ftp://example.com",
				MaxDepth:    3,
				MaxPages:    100,
				Timeout:     30,
				AuthProfile: "test-profile",
			},
			wantErr:     true,
			errContains: "invalid url: must be http or https and have a host",
		},
		{
			name: "invalid maxDepth - too low",
			opts: JobValidationOpts{
				URL:         "https://example.com",
				MaxDepth:    -1,
				MaxPages:    100,
				Timeout:     30,
				AuthProfile: "test-profile",
			},
			wantErr:     true,
			errContains: "maxDepth must be between 1 and 10",
		},
		{
			name: "invalid maxDepth - too high",
			opts: JobValidationOpts{
				URL:         "https://example.com",
				MaxDepth:    11,
				MaxPages:    100,
				Timeout:     30,
				AuthProfile: "test-profile",
			},
			wantErr:     true,
			errContains: "maxDepth must be between 1 and 10",
		},
		{
			name: "invalid maxPages - too low",
			opts: JobValidationOpts{
				URL:         "https://example.com",
				MaxDepth:    3,
				MaxPages:    -1,
				Timeout:     30,
				AuthProfile: "test-profile",
			},
			wantErr:     true,
			errContains: "maxPages must be between 1 and 10000",
		},
		{
			name: "invalid maxPages - too high",
			opts: JobValidationOpts{
				URL:         "https://example.com",
				MaxDepth:    3,
				MaxPages:    10001,
				Timeout:     30,
				AuthProfile: "test-profile",
			},
			wantErr:     true,
			errContains: "maxPages must be between 1 and 10000",
		},
		{
			name: "invalid timeout",
			opts: JobValidationOpts{
				URL:         "https://example.com",
				MaxDepth:    3,
				MaxPages:    100,
				Timeout:     301,
				AuthProfile: "test-profile",
			},
			wantErr:     true,
			errContains: "timeoutSeconds must be between 5 and 300",
		},
		{
			name: "invalid authProfile name",
			opts: JobValidationOpts{
				URL:         "https://example.com",
				MaxDepth:    3,
				MaxPages:    100,
				Timeout:     30,
				AuthProfile: "test@profile",
			},
			wantErr:     true,
			errContains: "invalid authProfile: only alphanumeric, hyphens, and underscores allowed",
		},
		{
			name: "all invalid fields",
			opts: JobValidationOpts{
				URL:         "invalid",
				MaxDepth:    0,
				MaxPages:    0,
				Timeout:     4,
				AuthProfile: "test@profile",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateJob(tt.opts, model.KindCrawl)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateJob() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errContains != "" {
				if !containsString(err.Error(), tt.errContains) {
					t.Errorf("Error = %v, want containing %v", err.Error(), tt.errContains)
				}
			}
		})
	}
}

func TestValidateJob_Research(t *testing.T) {
	tests := []struct {
		name        string
		opts        JobValidationOpts
		wantErr     bool
		errContains string
	}{
		{
			name: "valid request with multiple URLs",
			opts: JobValidationOpts{
				Query:       "test query",
				URLs:        []string{"https://example.com", "https://example.org"},
				MaxDepth:    3,
				MaxPages:    100,
				Timeout:     30,
				AuthProfile: "test-profile",
			},
			wantErr: false,
		},
		{
			name: "empty query",
			opts: JobValidationOpts{
				Query:       "",
				URLs:        []string{"https://example.com"},
				MaxDepth:    3,
				MaxPages:    100,
				Timeout:     30,
				AuthProfile: "test-profile",
			},
			wantErr:     true,
			errContains: "query is required",
		},
		{
			name: "invalid URL in list",
			opts: JobValidationOpts{
				Query:       "test query",
				URLs:        []string{"https://example.com", "ftp://example.org"},
				MaxDepth:    3,
				MaxPages:    100,
				Timeout:     30,
				AuthProfile: "test-profile",
			},
			wantErr:     true,
			errContains: "invalid url at index 1",
		},
		{
			name: "empty URLs list",
			opts: JobValidationOpts{
				Query:       "test query",
				URLs:        []string{},
				MaxDepth:    3,
				MaxPages:    100,
				Timeout:     30,
				AuthProfile: "test-profile",
			},
			wantErr:     true,
			errContains: "urls list is empty",
		},
		{
			name: "invalid maxDepth",
			opts: JobValidationOpts{
				Query:       "test query",
				URLs:        []string{"https://example.com"},
				MaxDepth:    11,
				MaxPages:    100,
				Timeout:     30,
				AuthProfile: "test-profile",
			},
			wantErr:     true,
			errContains: "maxDepth must be between 1 and 10",
		},
		{
			name: "invalid maxPages",
			opts: JobValidationOpts{
				Query:       "test query",
				URLs:        []string{"https://example.com"},
				MaxDepth:    3,
				MaxPages:    10001,
				Timeout:     30,
				AuthProfile: "test-profile",
			},
			wantErr:     true,
			errContains: "maxPages must be between 1 and 10000",
		},
		{
			name: "invalid timeout",
			opts: JobValidationOpts{
				Query:       "test query",
				URLs:        []string{"https://example.com"},
				MaxDepth:    3,
				MaxPages:    100,
				Timeout:     301,
				AuthProfile: "test-profile",
			},
			wantErr:     true,
			errContains: "timeoutSeconds must be between 5 and 300",
		},
		{
			name: "invalid authProfile name",
			opts: JobValidationOpts{
				Query:       "test query",
				URLs:        []string{"https://example.com"},
				MaxDepth:    3,
				MaxPages:    100,
				Timeout:     30,
				AuthProfile: "test@profile",
			},
			wantErr:     true,
			errContains: "invalid authProfile: only alphanumeric, hyphens, and underscores allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateJob(tt.opts, model.KindResearch)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateJob() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errContains != "" {
				if !containsString(err.Error(), tt.errContains) {
					t.Errorf("Error = %v, want containing %v", err.Error(), tt.errContains)
				}
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
