package fetch

import (
	"testing"
)

// TestCSSEscape tests the CSS escaping function.
func TestCSSEscape(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with-dash", "with-dash"},
		{"with_underscore", "with_underscore"},
		{"with'quote", "with\\'quote"},
		{"with\"double", "with\\\"double"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := CSSEscape(tt.input)
			if result != tt.expected {
				t.Errorf("CSSEscape(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}
