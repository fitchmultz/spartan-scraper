// Package scheduler provides tests for schedule parameter helpers.
//
// Purpose:
// - Verify scheduler parameter helpers mirror the shared decoder semantics.
//
// Responsibilities:
// - Cover bool fallback behavior.
// - Cover positive-int and string-slice extraction from persisted params.
//
// Scope:
// - Helper-level tests for schedule parameter decoding only.
//
// Usage:
// - Run with `go test ./internal/scheduler`.
//
// Invariants/Assumptions:
// - Missing or invalid values use explicit defaults.
// - JSON-style arrays decode into Go string slices.
package scheduler

import (
	"testing"
)

func TestBoolParamDefault(t *testing.T) {
	tests := []struct {
		name     string
		params   map[string]interface{}
		key      string
		fallback bool
		want     bool
	}{
		{
			name:     "nil params - returns fallback",
			params:   nil,
			key:      "playwright",
			fallback: true,
			want:     true,
		},
		{
			name:     "key absent - returns fallback",
			params:   map[string]interface{}{},
			key:      "playwright",
			fallback: true,
			want:     true,
		},
		{
			name:     "key present with true - returns true",
			params:   map[string]interface{}{"playwright": true},
			key:      "playwright",
			fallback: false,
			want:     true,
		},
		{
			name:     "key present with false - returns false (not fallback)",
			params:   map[string]interface{}{"playwright": false},
			key:      "playwright",
			fallback: true,
			want:     false,
		},
		{
			name:     "key present with non-bool - returns fallback",
			params:   map[string]interface{}{"playwright": "invalid"},
			key:      "playwright",
			fallback: true,
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := boolParamDefault(tt.params, tt.key, tt.fallback)
			if got != tt.want {
				t.Errorf("boolParamDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIntParam(t *testing.T) {
	tests := []struct {
		name     string
		params   map[string]interface{}
		fallback int
		want     int
	}{
		{name: "typed int", params: map[string]interface{}{"timeout": 30}, fallback: 5, want: 30},
		{name: "json float", params: map[string]interface{}{"timeout": 45.0}, fallback: 5, want: 45},
		{name: "zero falls back", params: map[string]interface{}{"timeout": 0}, fallback: 5, want: 5},
		{name: "invalid falls back", params: map[string]interface{}{"timeout": "bad"}, fallback: 5, want: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := intParam(tt.params, "timeout", tt.fallback); got != tt.want {
				t.Fatalf("intParam() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestStringSliceParam(t *testing.T) {
	params := map[string]interface{}{
		"urls": []interface{}{"https://example.com", "https://example.com/docs", 3},
	}

	got := stringSliceParam(params, "urls")
	want := []string{"https://example.com", "https://example.com/docs"}
	if len(got) != len(want) {
		t.Fatalf("stringSliceParam() len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("stringSliceParam()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
