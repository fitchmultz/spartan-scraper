package validate

import (
	"testing"
)

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid http", "http://example.com", false},
		{"valid https", "https://example.com", false},
		{"valid https with path", "https://example.com/path", false},
		{"valid https with query", "https://example.com?q=test", false},
		{"empty url", "", true},
		{"missing scheme", "example.com", true},
		{"invalid scheme", "ftp://example.com", true},
		{"missing host", "https://", true},
		{"malformed url", "://example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateURLs(t *testing.T) {
	tests := []struct {
		name    string
		urls    []string
		wantErr bool
	}{
		{"valid urls", []string{"http://example.com", "https://example.org"}, false},
		{"empty list", []string{}, true},
		{"nil list", nil, true},
		{"invalid url in list", []string{"http://example.com", "ftp://example.org"}, true},
		{"empty url in list", []string{"http://example.com", ""}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURLs(tt.urls)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURLs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout int
		wantErr bool
	}{
		{"zero value", 0, false},
		{"valid min", 5, false},
		{"valid mid", 30, false},
		{"valid max", 300, false},
		{"below min", 4, true},
		{"above max", 301, true},
		{"negative", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTimeout(tt.timeout)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTimeout() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateMaxDepth(t *testing.T) {
	tests := []struct {
		name    string
		depth   int
		wantErr bool
	}{
		{"zero value", 0, false},
		{"valid min", 1, false},
		{"valid mid", 5, false},
		{"valid max", 10, false},
		{"above max", 11, true},
		{"negative", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMaxDepth(tt.depth)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMaxDepth() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateMaxPages(t *testing.T) {
	tests := []struct {
		name    string
		pages   int
		wantErr bool
	}{
		{"zero value", 0, false},
		{"valid min", 1, false},
		{"valid mid", 100, false},
		{"valid max", 10000, false},
		{"above max", 10001, true},
		{"negative", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMaxPages(tt.pages)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMaxPages() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateAuthProfileName(t *testing.T) {
	tests := []struct {
		name    string
		profile string
		wantErr bool
	}{
		{"empty string", "", false},
		{"valid alphanumeric", "profile123", false},
		{"valid with hyphens", "my-profile", false},
		{"valid with underscores", "my_profile", false},
		{"valid mixed", "my_profile-123", false},
		{"invalid spaces", "my profile", true},
		{"invalid special chars", "my.profile", true},
		{"invalid at sign", "my@profile", true},
		{"invalid slash", "my/profile", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAuthProfileName(tt.profile)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAuthProfileName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
