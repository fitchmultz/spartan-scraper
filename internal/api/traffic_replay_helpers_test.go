// Package api provides tests for traffic replay helper functions.
//
// This file contains tests for helper functions including filtering,
// URL pattern matching, URL transformation, header conversion, and response comparison.
package api

import (
	"net/http"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/fetch"
)

func TestFilterEntries(t *testing.T) {
	entries := []fetch.InterceptedEntry{
		{
			Request: fetch.InterceptedRequest{
				URL:          "https://example.com/api/users",
				Method:       "GET",
				ResourceType: fetch.ResourceTypeXHR,
			},
			Response: &fetch.InterceptedResponse{Status: 200},
		},
		{
			Request: fetch.InterceptedRequest{
				URL:          "https://example.com/api/posts",
				Method:       "POST",
				ResourceType: fetch.ResourceTypeFetch,
			},
			Response: &fetch.InterceptedResponse{Status: 201},
		},
		{
			Request: fetch.InterceptedRequest{
				URL:          "https://example.com/script.js",
				Method:       "GET",
				ResourceType: fetch.ResourceTypeScript,
			},
			Response: &fetch.InterceptedResponse{Status: 200},
		},
	}

	tests := []struct {
		name     string
		filter   *TrafficReplayFilter
		expected int
	}{
		{
			name:     "no filter",
			filter:   nil,
			expected: 3,
		},
		{
			name: "filter by method GET",
			filter: &TrafficReplayFilter{
				Methods: []string{"GET"},
			},
			expected: 2,
		},
		{
			name: "filter by method POST",
			filter: &TrafficReplayFilter{
				Methods: []string{"POST"},
			},
			expected: 1,
		},
		{
			name: "filter by resource type XHR",
			filter: &TrafficReplayFilter{
				ResourceTypes: []string{"xhr"},
			},
			expected: 1,
		},
		{
			name: "filter by status code 200",
			filter: &TrafficReplayFilter{
				StatusCodes: []int{200},
			},
			expected: 2,
		},
		{
			name: "filter by URL pattern",
			filter: &TrafficReplayFilter{
				URLPatterns: []string{"*api*"},
			},
			expected: 2,
		},
		{
			name: "filter with no matches",
			filter: &TrafficReplayFilter{
				Methods: []string{"DELETE"},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterEntries(entries, tt.filter)
			if len(result) != tt.expected {
				t.Errorf("expected %d entries, got %d", tt.expected, len(result))
			}
		})
	}
}

func TestMatchURLPattern(t *testing.T) {
	tests := []struct {
		url     string
		pattern string
		want    bool
	}{
		{"https://example.com/api/users", "*api*", true},
		{"https://example.com/api/users", "*users", true},
		{"https://example.com/api/users", "*posts", false},
		{"https://example.com/api/users/123", "*api/**", true},
		{"https://example.com/api/users", "https://example.com/api/*", true},
		{"https://example.com/api/users", "https://*.com/api/*", true},
		{"https://example.com/api/users", "*", true},
		{"https://example.com/api/users", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			got := matchURLPattern(tt.url, tt.pattern)
			if got != tt.want {
				t.Errorf("matchURLPattern(%q, %q) = %v, want %v", tt.url, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestTransformURL(t *testing.T) {
	tests := []struct {
		original string
		target   string
		want     string
		wantErr  bool
	}{
		{
			original: "https://prod.example.com/api/users",
			target:   "https://staging.example.com",
			want:     "https://staging.example.com/api/users",
			wantErr:  false,
		},
		{
			original: "https://prod.example.com/api/users?id=123",
			target:   "https://staging.example.com",
			want:     "https://staging.example.com/api/users?id=123",
			wantErr:  false,
		},
		{
			original: "https://prod.example.com/api/users#section",
			target:   "https://staging.example.com",
			want:     "https://staging.example.com/api/users#section",
			wantErr:  false,
		},
		{
			original: "://invalid-url",
			target:   "https://staging.example.com",
			want:     "",
			wantErr:  true,
		},
		{
			original: "https://prod.example.com/api/users",
			target:   "://invalid-target",
			want:     "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.original, func(t *testing.T) {
			got, err := transformURL(tt.original, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("transformURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("transformURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHeadersToMap(t *testing.T) {
	headers := http.Header{
		"Content-Type": []string{"application/json"},
		"X-Custom":     []string{"value1", "value2"}, // Multiple values, should take first
	}

	result := headersToMap(headers)

	if result["Content-Type"] != "application/json" {
		t.Errorf("expected Content-Type to be 'application/json', got %q", result["Content-Type"])
	}

	if result["X-Custom"] != "value1" {
		t.Errorf("expected X-Custom to be 'value1', got %q", result["X-Custom"])
	}
}

func TestCompareResponse(t *testing.T) {
	tests := []struct {
		name     string
		original *fetch.InterceptedResponse
		replayed ReplayResponseInfo
		wantDiff bool
	}{
		{
			name: "identical responses",
			original: &fetch.InterceptedResponse{
				Status:   200,
				Headers:  map[string]string{"Content-Type": "application/json"},
				BodySize: 100,
			},
			replayed: ReplayResponseInfo{
				Status:   200,
				Headers:  map[string]string{"Content-Type": "application/json"},
				BodySize: 100,
			},
			wantDiff: false,
		},
		{
			name: "different status",
			original: &fetch.InterceptedResponse{
				Status:   200,
				Headers:  map[string]string{"Content-Type": "application/json"},
				BodySize: 100,
			},
			replayed: ReplayResponseInfo{
				Status:   404,
				Headers:  map[string]string{"Content-Type": "application/json"},
				BodySize: 100,
			},
			wantDiff: true,
		},
		{
			name: "different body size",
			original: &fetch.InterceptedResponse{
				Status:   200,
				Headers:  map[string]string{"Content-Type": "application/json"},
				BodySize: 100,
				Body:     "original body",
			},
			replayed: ReplayResponseInfo{
				Status:   200,
				Headers:  map[string]string{"Content-Type": "application/json"},
				BodySize: 150,
				Body:     "different body content here",
			},
			wantDiff: true,
		},
		{
			name: "different headers",
			original: &fetch.InterceptedResponse{
				Status:   200,
				Headers:  map[string]string{"Content-Type": "application/json"},
				BodySize: 100,
			},
			replayed: ReplayResponseInfo{
				Status:   200,
				Headers:  map[string]string{"Content-Type": "text/html"},
				BodySize: 100,
			},
			wantDiff: true,
		},
		{
			name: "new header in replay",
			original: &fetch.InterceptedResponse{
				Status:   200,
				Headers:  map[string]string{"Content-Type": "application/json"},
				BodySize: 100,
			},
			replayed: ReplayResponseInfo{
				Status:   200,
				Headers:  map[string]string{"Content-Type": "application/json", "X-New": "value"},
				BodySize: 100,
			},
			wantDiff: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := compareResponse(tt.original, tt.replayed)
			if tt.wantDiff && diff == nil {
				t.Error("expected diff, got nil")
			}
			if !tt.wantDiff && diff != nil {
				t.Errorf("expected no diff, got %+v", diff)
			}
		})
	}
}
