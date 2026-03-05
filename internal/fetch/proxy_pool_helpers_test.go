// Package fetch provides HTTP and headless browser content fetching capabilities.
package fetch

import (
	"testing"
)

func TestContainsString(t *testing.T) {
	tests := []struct {
		slice []string
		s     string
		want  bool
	}{
		{[]string{"a", "b", "c"}, "b", true},
		{[]string{"a", "b", "c"}, "d", false},
		{[]string{}, "a", false},
		{[]string{"a"}, "a", true},
	}

	for _, tt := range tests {
		got := containsString(tt.slice, tt.s)
		if got != tt.want {
			t.Errorf("containsString(%v, %q) = %v, want %v", tt.slice, tt.s, got, tt.want)
		}
	}
}

func TestHasAllTags(t *testing.T) {
	tests := []struct {
		proxyTags    []string
		requiredTags []string
		want         bool
	}{
		{[]string{"a", "b", "c"}, []string{"b"}, true},
		{[]string{"a", "b", "c"}, []string{"b", "c"}, true},
		{[]string{"a", "b", "c"}, []string{"d"}, false},
		{[]string{"a", "b", "c"}, []string{"a", "d"}, false},
		{[]string{}, []string{"a"}, false},
		{[]string{"a"}, []string{}, true},
	}

	for _, tt := range tests {
		got := hasAllTags(tt.proxyTags, tt.requiredTags)
		if got != tt.want {
			t.Errorf("hasAllTags(%v, %v) = %v, want %v", tt.proxyTags, tt.requiredTags, got, tt.want)
		}
	}
}
