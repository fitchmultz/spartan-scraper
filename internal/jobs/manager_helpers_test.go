package jobs

import (
	"testing"
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
