package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Clear env vars that might interfere
	os.Unsetenv("PORT")
	os.Unsetenv("DATA_DIR")

	cfg := Load()
	if cfg.Port != "8741" {
		t.Errorf("expected default port 8741, got %s", cfg.Port)
	}

	os.Setenv("PORT", "9999")
	cfg = Load()
	if cfg.Port != "9999" {
		t.Errorf("expected port 9999, got %s", cfg.Port)
	}
}

func TestGetenvBool(t *testing.T) {
	tests := []struct {
		val      string
		expected bool
	}{
		{"1", true},
		{"true", true},
		{"yes", true},
		{"y", true},
		{"0", false},
		{"false", false},
		{"", false}, // fallback
	}

	for _, tt := range tests {
		os.Setenv("TEST_BOOL", tt.val)
		got := getenvBool("TEST_BOOL", false)
		if got != tt.expected {
			t.Errorf("getenvBool(%q) = %v; want %v", tt.val, got, tt.expected)
		}
	}
}

func TestNormalizeAuthKeySuffix(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"USER_AGENT", "USER-AGENT"},
		{"X__API__KEY", "X-API-KEY"},
		{"", ""},
	}

	for _, tt := range tests {
		got := normalizeAuthKeySuffix(tt.input)
		if got != tt.expected {
			t.Errorf("normalizeAuthKeySuffix(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}
