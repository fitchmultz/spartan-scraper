// Package fetch provides tests for render profile utilities.
// Tests cover profile path generation and extended host pattern matching scenarios.
// Does NOT test profile storage or retrieval.
package fetch

import (
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/hostmatch"
)

func TestRenderProfilesPath(t *testing.T) {
	tests := []struct {
		name    string
		dataDir string
		want    string
	}{
		{
			name:    "non-empty dataDir",
			dataDir: "/custom/data",
			want:    "/custom/data/render_profiles.json",
		},
		{
			name:    "empty dataDir uses default",
			dataDir: "",
			want:    ".data/render_profiles.json",
		},
		{
			name:    "whitespace-only dataDir uses default",
			dataDir: "   \t\n",
			want:    ".data/render_profiles.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RenderProfilesPath(tt.dataDir); got != tt.want {
				t.Errorf("RenderProfilesPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHostMatchesAnyPattern_Extended(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		patterns []string
		match    bool
	}{
		{
			name:     "empty patterns",
			host:     "example.com",
			patterns: []string{},
			match:    false,
		},
		{
			name:     "empty patterns with whitespace skipped",
			host:     "example.com",
			patterns: []string{"  ", "\t"},
			match:    false,
		},
		{
			name:     "empty host",
			host:     "",
			patterns: []string{"example.com"},
			match:    false,
		},
		{
			name:     "case insensitive host",
			host:     "EXAMPLE.COM",
			patterns: []string{"example.com"},
			match:    true,
		},
		{
			name:     "case insensitive patterns",
			host:     "example.com",
			patterns: []string{"EXAMPLE.COM"},
			match:    true,
		},
		{
			name:     "both case insensitive",
			host:     "SUB.EXAMPLE.COM",
			patterns: []string{"*.EXAMPLE.COM"},
			match:    true,
		},
		{
			name:     "mixed case wildcard",
			host:     "API.TEST.CO.UK",
			patterns: []string{"*.test.co.uk"},
			match:    true,
		},
		{
			name:     "wildcard prefix matches",
			host:     "example.com",
			patterns: []string{"example.*"},
			match:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hostmatch.HostMatchesAnyPattern(tt.host, tt.patterns); got != tt.match {
				t.Errorf("HostMatchesAnyPattern(%q, %v) = %v, want %v", tt.host, tt.patterns, got, tt.match)
			}
		})
	}
}
