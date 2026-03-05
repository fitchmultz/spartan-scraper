// Package hostmatch provides tests for host extraction and pattern matching.
//
// Tests cover:
// - Host extraction from various URL formats
// - Pattern matching with exact, wildcard subdomain, and wildcard prefix patterns
// - Edge cases (empty inputs, case insensitivity, whitespace handling)
// - Pattern validation
//
// Does NOT test:
// - DNS resolution
// - Internationalized domain names (IDN)
// - IPv6 addresses
package hostmatch

import (
	"testing"
)

func TestHostFromURL(t *testing.T) {
	tests := []struct {
		name   string
		rawURL string
		want   string
	}{
		{
			name:   "https URL",
			rawURL: "https://example.com/path",
			want:   "example.com",
		},
		{
			name:   "http URL",
			rawURL: "http://example.com/path",
			want:   "example.com",
		},
		{
			name:   "URL without scheme",
			rawURL: "example.com/path",
			want:   "example.com",
		},
		{
			name:   "URL with port",
			rawURL: "https://example.com:8080/path",
			want:   "example.com",
		},
		{
			name:   "subdomain",
			rawURL: "https://sub.example.com",
			want:   "sub.example.com",
		},
		{
			name:   "with query params",
			rawURL: "https://example.com?foo=bar",
			want:   "example.com",
		},
		{
			name:   "empty string",
			rawURL: "",
			want:   "",
		},
		{
			name:   "whitespace only",
			rawURL: "   ",
			want:   "",
		},
		{
			name:   "with whitespace",
			rawURL: "  https://example.com  ",
			want:   "example.com",
		},
		{
			name:   "uppercase",
			rawURL: "https://EXAMPLE.COM",
			want:   "example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HostFromURL(tt.rawURL)
			if got != tt.want {
				t.Errorf("HostFromURL(%q) = %q, want %q", tt.rawURL, got, tt.want)
			}
		})
	}
}

func TestHostMatchesAnyPattern(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		patterns []string
		want     bool
	}{
		{
			name:     "exact match",
			host:     "example.com",
			patterns: []string{"example.com"},
			want:     true,
		},
		{
			name:     "exact match - case insensitive",
			host:     "EXAMPLE.COM",
			patterns: []string{"example.com"},
			want:     true,
		},
		{
			name:     "exact match - pattern uppercase",
			host:     "example.com",
			patterns: []string{"EXAMPLE.COM"},
			want:     true,
		},
		{
			name:     "no match",
			host:     "example.com",
			patterns: []string{"other.com"},
			want:     false,
		},
		{
			name:     "wildcard subdomain match",
			host:     "sub.example.com",
			patterns: []string{"*.example.com"},
			want:     true,
		},
		{
			name:     "wildcard subdomain - root domain should NOT match",
			host:     "example.com",
			patterns: []string{"*.example.com"},
			want:     false,
		},
		{
			name:     "wildcard subdomain - deep subdomain",
			host:     "deep.sub.example.com",
			patterns: []string{"*.example.com"},
			want:     true,
		},
		{
			name:     "wildcard prefix match",
			host:     "example.org",
			patterns: []string{"example.*"},
			want:     true,
		},
		{
			name:     "wildcard prefix - multiple TLDs",
			host:     "example.co.uk",
			patterns: []string{"example.*"},
			want:     true,
		},
		{
			name:     "multiple patterns - first matches",
			host:     "example.com",
			patterns: []string{"example.com", "other.com"},
			want:     true,
		},
		{
			name:     "multiple patterns - second matches",
			host:     "other.com",
			patterns: []string{"example.com", "other.com"},
			want:     true,
		},
		{
			name:     "multiple patterns - none match",
			host:     "third.com",
			patterns: []string{"example.com", "other.com"},
			want:     false,
		},
		{
			name:     "empty host",
			host:     "",
			patterns: []string{"example.com"},
			want:     false,
		},
		{
			name:     "empty patterns",
			host:     "example.com",
			patterns: []string{},
			want:     false,
		},
		{
			name:     "nil patterns",
			host:     "example.com",
			patterns: nil,
			want:     false,
		},
		{
			name:     "pattern with whitespace - trimmed",
			host:     "example.com",
			patterns: []string{"  example.com  "},
			want:     true,
		},
		{
			name:     "host with whitespace - trimmed",
			host:     "  example.com  ",
			patterns: []string{"example.com"},
			want:     true,
		},
		{
			name:     "empty pattern in list - skipped",
			host:     "example.com",
			patterns: []string{"", "example.com"},
			want:     true,
		},
		{
			name:     "whitespace-only pattern - skipped",
			host:     "example.com",
			patterns: []string{"   ", "example.com"},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HostMatchesAnyPattern(tt.host, tt.patterns)
			if got != tt.want {
				t.Errorf("HostMatchesAnyPattern(%q, %v) = %v, want %v", tt.host, tt.patterns, got, tt.want)
			}
		})
	}
}

func TestValidateHostPatterns(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		wantErr  bool
	}{
		{
			name:     "valid single pattern",
			patterns: []string{"example.com"},
			wantErr:  false,
		},
		{
			name:     "valid multiple patterns",
			patterns: []string{"example.com", "*.example.com", "other.*"},
			wantErr:  false,
		},
		{
			name:     "empty list is valid",
			patterns: []string{},
			wantErr:  false,
		},
		{
			name:     "nil list is valid",
			patterns: nil,
			wantErr:  false,
		},
		{
			name:     "empty pattern - error",
			patterns: []string{"example.com", ""},
			wantErr:  true,
		},
		{
			name:     "whitespace-only pattern - error",
			patterns: []string{"example.com", "   "},
			wantErr:  true,
		},
		{
			name:     "pattern with internal space - error",
			patterns: []string{"example .com"},
			wantErr:  true,
		},
		{
			name:     "pattern with tab - error",
			patterns: []string{"example\t.com"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHostPatterns(tt.patterns)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateHostPatterns(%v) error = %v, wantErr %v", tt.patterns, err, tt.wantErr)
			}
		})
	}
}

func TestValidationError_Error(t *testing.T) {
	err := &ValidationError{
		Field:   "hostPatterns",
		Index:   2,
		Message: "pattern cannot be empty",
	}
	want := "hostPatterns[2]: pattern cannot be empty"
	if got := err.Error(); got != want {
		t.Errorf("ValidationError.Error() = %q, want %q", got, want)
	}

	// Test without index (using -1 sentinel)
	err2 := &ValidationError{
		Field:   "name",
		Index:   -1,
		Message: "name is required",
	}
	want2 := "name: name is required"
	if got := err2.Error(); got != want2 {
		t.Errorf("ValidationError.Error() = %q, want %q", got, want2)
	}
}
