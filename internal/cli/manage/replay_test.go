// Package manage provides tests for traffic replay CLI functionality.
// Tests cover argument parsing, request building, and output formatting.
package manage

import (
	"testing"
)

func TestSplitCommaSeparated(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{
			input:    "a,b,c",
			expected: []string{"a", "b", "c"},
		},
		{
			input:    "a, b, c",
			expected: []string{"a", "b", "c"},
		},
		{
			input:    "  a  ,  b  ,  c  ",
			expected: []string{"a", "b", "c"},
		},
		{
			input:    "single",
			expected: []string{"single"},
		},
		{
			input:    "",
			expected: []string{},
		},
		{
			input:    "a,,c",
			expected: []string{"a", "c"},
		},
		{
			input:    "GET, POST, PUT",
			expected: []string{"GET", "POST", "PUT"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := splitCommaSeparated(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("splitCommaSeparated(%q) = %v, want %v", tt.input, result, tt.expected)
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("splitCommaSeparated(%q)[%d] = %q, want %q", tt.input, i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		s      string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello world", 8, "hello..."},
		{"", 5, ""},
		{"test", 4, "test"},
		{"exactly", 7, "exactly"},
		{"longer string here", 10, "longer ..."},
		{"hello world", 5, "he..."},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			got := truncateString(tt.s, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestTruncateStringEdgeCases(t *testing.T) {
	// Test with maxLen less than 3 (ellipsis length)
	result := truncateString("hello", 2)
	if result != "he" {
		t.Errorf("expected 'he' for maxLen=2, got %q", result)
	}

	// Test with exact length
	result = truncateString("abc", 3)
	if result != "abc" {
		t.Errorf("expected 'abc' for exact length, got %q", result)
	}

	// Test with single character
	result = truncateString("x", 1)
	if result != "x" {
		t.Errorf("expected 'x' for single char, got %q", result)
	}
}
