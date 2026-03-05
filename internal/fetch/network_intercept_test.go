// Package fetch provides HTTP and headless browser content fetching capabilities.
//
// This file contains tests for network interception functionality.
package fetch

import (
	"testing"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/stretchr/testify/assert"
)

func TestNetworkInterceptConfigDefaults(t *testing.T) {
	config := DefaultNetworkInterceptConfig()

	assert.False(t, config.Enabled, "default should be disabled")
	assert.Empty(t, config.URLPatterns, "default should have no URL patterns")
	assert.Equal(t, []InterceptedResourceType{ResourceTypeXHR, ResourceTypeFetch}, config.ResourceTypes)
	assert.True(t, config.CaptureRequestBody, "default should capture request body")
	assert.True(t, config.CaptureResponseBody, "default should capture response body")
	assert.Equal(t, int64(1024*1024), config.MaxBodySize, "default max body size should be 1MB")
	assert.Equal(t, 1000, config.MaxEntries, "default max entries should be 1000")
}

func TestNetworkInterceptorShouldIntercept(t *testing.T) {
	tests := []struct {
		name         string
		config       NetworkInterceptConfig
		url          string
		resourceType network.ResourceType
		want         bool
	}{
		{
			name: "intercept all when no patterns or types",
			config: NetworkInterceptConfig{
				Enabled:       true,
				URLPatterns:   []string{},
				ResourceTypes: []InterceptedResourceType{},
			},
			url:          "https://api.example.com/data",
			resourceType: network.ResourceTypeXHR,
			want:         true,
		},
		{
			name: "match URL pattern",
			config: NetworkInterceptConfig{
				Enabled:       true,
				URLPatterns:   []string{"**/api/**"},
				ResourceTypes: []InterceptedResourceType{},
			},
			url:          "https://example.com/api/users",
			resourceType: network.ResourceTypeXHR,
			want:         true,
		},
		{
			name: "no match URL pattern",
			config: NetworkInterceptConfig{
				Enabled:       true,
				URLPatterns:   []string{"**/api/**"},
				ResourceTypes: []InterceptedResourceType{},
			},
			url:          "https://example.com/static/image.png",
			resourceType: network.ResourceTypeXHR,
			want:         false,
		},
		{
			name: "match resource type",
			config: NetworkInterceptConfig{
				Enabled:       true,
				URLPatterns:   []string{},
				ResourceTypes: []InterceptedResourceType{ResourceTypeXHR, ResourceTypeFetch},
			},
			url:          "https://example.com/data",
			resourceType: "XHR", // network.ResourceTypeXHR = "XHR"
			want:         true,
		},
		{
			name: "no match resource type",
			config: NetworkInterceptConfig{
				Enabled:       true,
				URLPatterns:   []string{},
				ResourceTypes: []InterceptedResourceType{ResourceTypeXHR},
			},
			url:          "https://example.com/script.js",
			resourceType: "Script", // network.ResourceTypeScript = "Script"
			want:         false,
		},
		{
			name: "match both URL and resource type",
			config: NetworkInterceptConfig{
				Enabled:       true,
				URLPatterns:   []string{"**/api/**"},
				ResourceTypes: []InterceptedResourceType{ResourceTypeXHR},
			},
			url:          "https://example.com/api/users",
			resourceType: "XHR",
			want:         true,
		},
		{
			name: "match URL but not resource type",
			config: NetworkInterceptConfig{
				Enabled:       true,
				URLPatterns:   []string{"**/api/**"},
				ResourceTypes: []InterceptedResourceType{ResourceTypeXHR},
			},
			url:          "https://example.com/api/image.png",
			resourceType: "Image", // network.ResourceTypeImage = "Image"
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := newNetworkInterceptor(tt.config)
			got := interceptor.shouldIntercept(tt.url, tt.resourceType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		pattern string
		s       string
		want    bool
	}{
		{"**/*", "anything", true},
		{"*", "anything", true},
		{"**/api/**", "https://example.com/api/users", true},
		{"**/api/**", "https://example.com/api/v1/data", true},
		{"**/api/**", "https://example.com/static/api.js", false}, // api.js is not /api/ path
		{"*.json", "https://example.com/data.json", true},
		{"*.json", "https://example.com/api/users", false},
		{"https://*.example.com/*", "https://api.example.com/data", true},
		{"https://*.example.com/*", "https://other.test.com/data", false},
		{"/api/", "https://example.com/api/users", true},
		{"/api/", "https://example.com/static/data", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.s, func(t *testing.T) {
			got := matchGlob(tt.pattern, tt.s)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNetworkInterceptorGetEntries(t *testing.T) {
	config := NetworkInterceptConfig{
		Enabled:             true,
		MaxEntries:          10,
		CaptureRequestBody:  true,
		CaptureResponseBody: true,
	}

	interceptor := newNetworkInterceptor(config)

	// Add some test entries
	interceptor.mu.Lock()
	interceptor.entries = []InterceptedEntry{
		{
			Request: InterceptedRequest{
				RequestID: "1",
				URL:       "https://example.com/api/1",
				Method:    "GET",
				Timestamp: time.Now(),
			},
			Response: &InterceptedResponse{
				RequestID: "1",
				Status:    200,
				Timestamp: time.Now(),
			},
			Duration: 100 * time.Millisecond,
		},
	}
	interceptor.mu.Unlock()

	entries := interceptor.getEntries()
	assert.Len(t, entries, 1)
	assert.Equal(t, "1", entries[0].Request.RequestID)
	assert.Equal(t, "https://example.com/api/1", entries[0].Request.URL)
	assert.Equal(t, 200, entries[0].Response.Status)
}

func TestInterceptedEntryTypes(t *testing.T) {
	// Test that all resource types are defined
	types := []InterceptedResourceType{
		ResourceTypeXHR,
		ResourceTypeFetch,
		ResourceTypeDocument,
		ResourceTypeScript,
		ResourceTypeStylesheet,
		ResourceTypeImage,
		ResourceTypeMedia,
		ResourceTypeFont,
		ResourceTypeWebSocket,
		ResourceTypeOther,
	}

	for _, rt := range types {
		assert.NotEmpty(t, string(rt), "resource type should not be empty")
	}
}

func TestInterceptedRequestResponse(t *testing.T) {
	// Test request struct
	req := InterceptedRequest{
		RequestID:    "req-123",
		URL:          "https://api.example.com/data",
		Method:       "POST",
		Headers:      map[string]string{"Content-Type": "application/json"},
		Body:         `{"key":"value"}`,
		BodySize:     15,
		Timestamp:    time.Now(),
		ResourceType: ResourceTypeXHR,
	}

	assert.Equal(t, "req-123", req.RequestID)
	assert.Equal(t, "https://api.example.com/data", req.URL)
	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, "application/json", req.Headers["Content-Type"])
	assert.Equal(t, `{"key":"value"}`, req.Body)
	assert.Equal(t, int64(15), req.BodySize)
	assert.Equal(t, ResourceTypeXHR, req.ResourceType)

	// Test response struct
	resp := InterceptedResponse{
		RequestID:  "req-123",
		Status:     200,
		StatusText: "OK",
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       `{"result":"success"}`,
		BodySize:   20,
		Timestamp:  time.Now(),
	}

	assert.Equal(t, "req-123", resp.RequestID)
	assert.Equal(t, 200, resp.Status)
	assert.Equal(t, "OK", resp.StatusText)
	assert.Equal(t, "application/json", resp.Headers["Content-Type"])
	assert.Equal(t, `{"result":"success"}`, resp.Body)
	assert.Equal(t, int64(20), resp.BodySize)

	// Test entry
	entry := InterceptedEntry{
		Request:  req,
		Response: &resp,
		Duration: 150 * time.Millisecond,
	}

	assert.Equal(t, req.RequestID, entry.Request.RequestID)
	assert.Equal(t, resp.Status, entry.Response.Status)
	assert.Equal(t, 150*time.Millisecond, entry.Duration)
}

func TestInterceptedEntryWithNilResponse(t *testing.T) {
	// Test entry with failed request (nil response)
	entry := InterceptedEntry{
		Request: InterceptedRequest{
			RequestID: "req-failed",
			URL:       "https://api.example.com/timeout",
			Method:    "GET",
			Timestamp: time.Now(),
		},
		Response: nil,
		Duration: 30 * time.Second,
	}

	assert.Equal(t, "req-failed", entry.Request.RequestID)
	assert.Nil(t, entry.Response)
	assert.Equal(t, 30*time.Second, entry.Duration)
}
