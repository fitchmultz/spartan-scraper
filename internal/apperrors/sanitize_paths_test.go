package apperrors

import (
	"strings"
	"testing"
)

func TestRedactString_FilesystemPaths(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantPath string
	}{
		{
			name:     "macOS user path",
			input:    "Error writing to /Users/john/Documents/data.json",
			wantPath: "/Users/john/Documents/data.json",
		},
		{
			name:     "Linux home path",
			input:    "Failed to read /home/admin/secrets.txt",
			wantPath: "/home/admin/secrets.txt",
		},
		{
			name:     "Linux var path",
			input:    "Log at /var/log/app/error.log",
			wantPath: "/var/log/app/error.log",
		},
		{
			name:     "tmp path",
			input:    "Temp file /tmp/scratch/data.tmp",
			wantPath: "/tmp/scratch/data.tmp",
		},
		{
			name:     "opt path",
			input:    "Binary at /opt/app/bin/server",
			wantPath: "/opt/app/bin/server",
		},
		{
			name:     "usr path",
			input:    "Config at /usr/local/etc/config.yaml",
			wantPath: "/usr/local/etc/config.yaml",
		},
		{
			name:     "etc path",
			input:    "Reading /etc/passwd",
			wantPath: "/etc/passwd",
		},
		{
			name:     "file URL",
			input:    "Loading file:///Users/admin/secret.txt",
			wantPath: "file:///Users/admin/secret.txt",
		},
		{
			name:     "Windows backslash path",
			input:    "File at C:\\Users\\Admin\\Documents\\file.txt",
			wantPath: "C:\\Users\\Admin\\Documents\\file.txt",
		},
		{
			name:     "Windows forward slash path",
			input:    "File at D:/Data/Results/output.json",
			wantPath: "D:/Data/Results/output.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RedactString(tt.input)
			if strings.Contains(got, tt.wantPath) {
				t.Errorf("RedactString() should redact path %q, got: %s", tt.wantPath, got)
			}
			if !strings.Contains(got, "[REDACTED]") {
				t.Errorf("RedactString() should contain [REDACTED], got: %s", got)
			}
		})
	}
}

func TestRedactString_APIPathsNotRedacted(t *testing.T) {
	// API paths should NOT be redacted
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "API v1 jobs path",
			input: "GET /v1/jobs returned 200",
		},
		{
			name:  "API v1 scrape path",
			input: "POST /v1/scrape created job",
		},
		{
			name:  "URL path component",
			input: "Fetching https://api.example.com/v1/users",
		},
		{
			name:  "relative path in URL",
			input: "Resource at /assets/images/logo.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RedactString(tt.input)
			if got != tt.input {
				t.Errorf("RedactString() should not modify API paths, got: %s, want: %s", got, tt.input)
			}
		})
	}
}

func TestRedactString_CombinedSecretsAndPaths(t *testing.T) {
	input := "Bearer token=abc123, error at /Users/admin/.data/secret.txt"
	got := RedactString(input)

	// Should redact the token
	if strings.Contains(got, "abc123") {
		t.Errorf("Should redact token, got: %s", got)
	}

	// Should contain Bearer (not redacted)
	if !strings.Contains(got, "Bearer") {
		t.Errorf("Should preserve Bearer, got: %s", got)
	}
}

func TestRedactString_EmptyString(t *testing.T) {
	got := RedactString("")
	if got != "" {
		t.Errorf("RedactString(\"\") should return empty string, got: %s", got)
	}
}

func TestRedactString_NoSensitiveContent(t *testing.T) {
	input := "This is a normal error message without secrets or paths"
	got := RedactString(input)
	if got != input {
		t.Errorf("RedactString() should not modify clean strings, got: %s, want: %s", got, input)
	}
}
