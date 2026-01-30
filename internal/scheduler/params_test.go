// Package scheduler provides tests for schedule parameter helpers.
// Tests cover bool parameter extraction with fallback defaults.
// Does NOT test parameter validation or other parameter types.
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
