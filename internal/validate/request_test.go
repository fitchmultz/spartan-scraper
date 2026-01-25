package validate

import (
	"testing"
)

func TestScrapeRequestValidator(t *testing.T) {
	tests := []struct {
		name        string
		validator   ScrapeRequestValidator
		wantErr     bool
		errContains string
	}{
		{
			name: "valid request with all fields",
			validator: ScrapeRequestValidator{
				URL:         "https://example.com",
				Timeout:     30,
				AuthProfile: "test-profile",
			},
			wantErr: false,
		},
		{
			name: "valid request with zero timeout",
			validator: ScrapeRequestValidator{
				URL:         "https://example.com",
				Timeout:     0,
				AuthProfile: "test-profile",
			},
			wantErr: false,
		},
		{
			name: "invalid URL - wrong scheme",
			validator: ScrapeRequestValidator{
				URL:         "ftp://example.com",
				Timeout:     30,
				AuthProfile: "test-profile",
			},
			wantErr:     true,
			errContains: "invalid url: must be http or https and have a host",
		},
		{
			name: "invalid URL - missing host",
			validator: ScrapeRequestValidator{
				URL:         "https://",
				Timeout:     30,
				AuthProfile: "test-profile",
			},
			wantErr:     true,
			errContains: "invalid url: must have a host",
		},
		{
			name: "empty URL",
			validator: ScrapeRequestValidator{
				URL:         "",
				Timeout:     30,
				AuthProfile: "test-profile",
			},
			wantErr:     true,
			errContains: "url is required",
		},
		{
			name: "invalid timeout - too low",
			validator: ScrapeRequestValidator{
				URL:         "https://example.com",
				Timeout:     4,
				AuthProfile: "test-profile",
			},
			wantErr:     true,
			errContains: "timeoutSeconds must be between 5 and 300",
		},
		{
			name: "invalid timeout - too high",
			validator: ScrapeRequestValidator{
				URL:         "https://example.com",
				Timeout:     301,
				AuthProfile: "test-profile",
			},
			wantErr:     true,
			errContains: "timeoutSeconds must be between 5 and 300",
		},
		{
			name: "invalid authProfile - invalid chars",
			validator: ScrapeRequestValidator{
				URL:         "https://example.com",
				Timeout:     30,
				AuthProfile: "test profile",
			},
			wantErr:     true,
			errContains: "invalid authProfile: only alphanumeric, hyphens, and underscores allowed",
		},
		{
			name: "empty authProfile is valid",
			validator: ScrapeRequestValidator{
				URL:         "https://example.com",
				Timeout:     30,
				AuthProfile: "",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.validator.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errContains != "" {
				if !containsString(err.Error(), tt.errContains) {
					t.Errorf("Error = %v, want containing %v", err.Error(), tt.errContains)
				}
			}
		})
	}
}

func TestCrawlRequestValidator(t *testing.T) {
	tests := []struct {
		name        string
		validator   CrawlRequestValidator
		wantErr     bool
		errContains string
	}{
		{
			name: "valid request with all fields",
			validator: CrawlRequestValidator{
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
			validator: CrawlRequestValidator{
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
			validator: CrawlRequestValidator{
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
			validator: CrawlRequestValidator{
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
			validator: CrawlRequestValidator{
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
			validator: CrawlRequestValidator{
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
			validator: CrawlRequestValidator{
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
			validator: CrawlRequestValidator{
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
			validator: CrawlRequestValidator{
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
			validator: CrawlRequestValidator{
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
			err := tt.validator.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errContains != "" {
				if !containsString(err.Error(), tt.errContains) {
					t.Errorf("Error = %v, want containing %v", err.Error(), tt.errContains)
				}
			}
		})
	}
}

func TestResearchRequestValidator(t *testing.T) {
	tests := []struct {
		name        string
		validator   ResearchRequestValidator
		wantErr     bool
		errContains string
	}{
		{
			name: "valid request with multiple URLs",
			validator: ResearchRequestValidator{
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
			validator: ResearchRequestValidator{
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
			validator: ResearchRequestValidator{
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
			validator: ResearchRequestValidator{
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
			validator: ResearchRequestValidator{
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
			validator: ResearchRequestValidator{
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
			validator: ResearchRequestValidator{
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
			validator: ResearchRequestValidator{
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
			err := tt.validator.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
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
