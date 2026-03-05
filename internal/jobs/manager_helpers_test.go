// Package jobs provides tests for manager helper functions.
// Tests cover toInt, toBool, and decodeScreenshot type conversion utilities.
// Does NOT test manager lifecycle or job operations.
package jobs

import (
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/fetch"
)

func TestToInt(t *testing.T) {
	tests := []struct {
		input    interface{}
		fallback int
		expected int
	}{
		{10, 5, 10},
		{0, 5, 5},
		{-1, 5, 5},
		{10.0, 5, 10},
		{"10", 5, 5},
	}

	for _, tt := range tests {
		got := toInt(tt.input, tt.fallback)
		if got != tt.expected {
			t.Errorf("toInt(%v, %d) = %d; want %d", tt.input, tt.fallback, got, tt.expected)
		}
	}
}

func TestToBool(t *testing.T) {
	tests := []struct {
		input    interface{}
		fallback bool
		expected bool
	}{
		{true, false, true},
		{false, true, false},
		{"true", true, true},
		{1, false, false},
	}

	for _, tt := range tests {
		got := toBool(tt.input, tt.fallback)
		if got != tt.expected {
			t.Errorf("toBool(%v, %v) = %v; want %v", tt.input, tt.fallback, got, tt.expected)
		}
	}
}

func TestDecodeScreenshot(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected *fetch.ScreenshotConfig
	}{
		{
			name:     "nil input returns nil",
			input:    nil,
			expected: nil,
		},
		{
			name: "pointer input returns same pointer",
			input: &fetch.ScreenshotConfig{
				Enabled:  true,
				FullPage: true,
				Format:   fetch.ScreenshotFormatPNG,
			},
			expected: &fetch.ScreenshotConfig{
				Enabled:  true,
				FullPage: true,
				Format:   fetch.ScreenshotFormatPNG,
			},
		},
		{
			name: "value input returns pointer to copy",
			input: fetch.ScreenshotConfig{
				Enabled: true,
				Format:  fetch.ScreenshotFormatJPEG,
				Quality: 85,
			},
			expected: &fetch.ScreenshotConfig{
				Enabled: true,
				Format:  fetch.ScreenshotFormatJPEG,
				Quality: 85,
			},
		},
		{
			name: "map[string]interface{} decodes correctly",
			input: map[string]interface{}{
				"enabled":  true,
				"fullPage": false,
				"format":   "png",
				"quality":  90,
				"width":    1920,
				"height":   1080,
			},
			expected: &fetch.ScreenshotConfig{
				Enabled:  true,
				FullPage: false,
				Format:   fetch.ScreenshotFormatPNG,
				Quality:  90,
				Width:    1920,
				Height:   1080,
			},
		},
		{
			name:     "invalid type returns nil",
			input:    "invalid string",
			expected: nil,
		},
		{
			name:     "empty map returns empty config",
			input:    map[string]string{},
			expected: &fetch.ScreenshotConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeScreenshot(tt.input)
			if tt.expected == nil {
				if got != nil {
					t.Errorf("decodeScreenshot(%v) = %+v; want nil", tt.input, got)
				}
				return
			}
			if got == nil {
				t.Errorf("decodeScreenshot(%v) = nil; want %+v", tt.input, tt.expected)
				return
			}
			if got.Enabled != tt.expected.Enabled ||
				got.FullPage != tt.expected.FullPage ||
				got.Format != tt.expected.Format ||
				got.Quality != tt.expected.Quality ||
				got.Width != tt.expected.Width ||
				got.Height != tt.expected.Height {
				t.Errorf("decodeScreenshot(%v) = %+v; want %+v", tt.input, got, tt.expected)
			}
		})
	}
}
