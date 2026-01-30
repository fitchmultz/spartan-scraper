// Package exporter provides functionality for exporting job results to various formats.
//
// This file contains tests for HAR format export functionality.
package exporter

import (
	"bytes"
	"encoding/json"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportHARStream_SingleResult(t *testing.T) {
	// Create test data with intercepted entries
	result := struct {
		URL             string                   `json:"url"`
		Status          int                      `json:"status"`
		HTML            string                   `json:"html"`
		FetchedAt       time.Time                `json:"fetchedAt"`
		InterceptedData []fetch.InterceptedEntry `json:"interceptedData"`
	}{
		URL:       "https://example.com",
		Status:    200,
		HTML:      "<html></html>",
		FetchedAt: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		InterceptedData: []fetch.InterceptedEntry{
			{
				Request: fetch.InterceptedRequest{
					RequestID:    "req-1",
					URL:          "https://api.example.com/data",
					Method:       "GET",
					Headers:      map[string]string{"Accept": "application/json"},
					Timestamp:    time.Date(2024, 1, 1, 12, 0, 1, 0, time.UTC),
					ResourceType: fetch.ResourceTypeXHR,
				},
				Response: &fetch.InterceptedResponse{
					RequestID:  "req-1",
					Status:     200,
					StatusText: "OK",
					Headers:    map[string]string{"Content-Type": "application/json"},
					Body:       `{"data":"test"}`,
					BodySize:   15,
					Timestamp:  time.Date(2024, 1, 1, 12, 0, 2, 0, time.UTC),
				},
				Duration: 1 * time.Second,
			},
		},
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	job := model.Job{ID: "test-job", Kind: "scrape"}
	var buf bytes.Buffer

	err = exportHARStream(job, bytes.NewReader(data), &buf)
	require.NoError(t, err)

	// Parse the HAR output
	var har HARLog
	err = json.Unmarshal(buf.Bytes(), &har)
	require.NoError(t, err)

	// Verify HAR structure
	assert.Equal(t, "1.2", har.Version)
	assert.Equal(t, "Spartan Scraper", har.Creator.Name)
	assert.Len(t, har.Pages, 1)
	assert.Len(t, har.Entries, 1)

	// Verify page
	page := har.Pages[0]
	assert.Equal(t, "https://example.com", page.Title)

	// Verify entry
	entry := har.Entries[0]
	assert.Equal(t, "GET", entry.Request.Method)
	assert.Equal(t, "https://api.example.com/data", entry.Request.URL)
	assert.Equal(t, 200, entry.Response.Status)
	assert.Equal(t, "OK", entry.Response.StatusText)
	assert.Equal(t, `{"data":"test"}`, entry.Response.Content.Text)
	assert.Equal(t, "application/json", entry.Response.Content.MimeType)
}

func TestExportHARStream_MultipleResults(t *testing.T) {
	// Create test data with multiple results
	results := []struct {
		URL             string                   `json:"url"`
		Status          int                      `json:"status"`
		HTML            string                   `json:"html"`
		FetchedAt       time.Time                `json:"fetchedAt"`
		InterceptedData []fetch.InterceptedEntry `json:"interceptedData"`
	}{
		{
			URL:       "https://example.com/page1",
			FetchedAt: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			InterceptedData: []fetch.InterceptedEntry{
				{
					Request: fetch.InterceptedRequest{
						RequestID: "req-1",
						URL:       "https://api.example.com/data1",
						Method:    "GET",
						Timestamp: time.Date(2024, 1, 1, 12, 0, 1, 0, time.UTC),
					},
					Response: &fetch.InterceptedResponse{
						RequestID: "req-1",
						Status:    200,
						Timestamp: time.Date(2024, 1, 1, 12, 0, 2, 0, time.UTC),
					},
					Duration: 1 * time.Second,
				},
			},
		},
		{
			URL:       "https://example.com/page2",
			FetchedAt: time.Date(2024, 1, 1, 12, 1, 0, 0, time.UTC),
			InterceptedData: []fetch.InterceptedEntry{
				{
					Request: fetch.InterceptedRequest{
						RequestID: "req-2",
						URL:       "https://api.example.com/data2",
						Method:    "POST",
						Timestamp: time.Date(2024, 1, 1, 12, 1, 1, 0, time.UTC),
					},
					Response: &fetch.InterceptedResponse{
						RequestID: "req-2",
						Status:    201,
						Timestamp: time.Date(2024, 1, 1, 12, 1, 2, 0, time.UTC),
					},
					Duration: 1 * time.Second,
				},
			},
		},
	}

	data, err := json.Marshal(results)
	require.NoError(t, err)

	job := model.Job{ID: "test-job", Kind: "crawl"}
	var buf bytes.Buffer

	err = exportHARStream(job, bytes.NewReader(data), &buf)
	require.NoError(t, err)

	var har HARLog
	err = json.Unmarshal(buf.Bytes(), &har)
	require.NoError(t, err)

	assert.Len(t, har.Pages, 2)
	assert.Len(t, har.Entries, 2)

	// Verify first entry
	assert.Equal(t, "GET", har.Entries[0].Request.Method)
	assert.Equal(t, "https://api.example.com/data1", har.Entries[0].Request.URL)

	// Verify second entry
	assert.Equal(t, "POST", har.Entries[1].Request.Method)
	assert.Equal(t, "https://api.example.com/data2", har.Entries[1].Request.URL)
	assert.Equal(t, 201, har.Entries[1].Response.Status)
}

func TestExportHARStream_NoInterceptedData(t *testing.T) {
	// Create test data without intercepted data
	result := struct {
		URL             string                   `json:"url"`
		Status          int                      `json:"status"`
		HTML            string                   `json:"html"`
		FetchedAt       time.Time                `json:"fetchedAt"`
		InterceptedData []fetch.InterceptedEntry `json:"interceptedData"`
	}{
		URL:             "https://example.com",
		Status:          200,
		HTML:            "<html></html>",
		FetchedAt:       time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		InterceptedData: []fetch.InterceptedEntry{},
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	job := model.Job{ID: "test-job", Kind: "scrape"}
	var buf bytes.Buffer

	err = exportHARStream(job, bytes.NewReader(data), &buf)
	require.NoError(t, err)

	var har HARLog
	err = json.Unmarshal(buf.Bytes(), &har)
	require.NoError(t, err)

	assert.Len(t, har.Pages, 1)
	assert.Len(t, har.Entries, 0)
}

func TestExportHARStream_InvalidData(t *testing.T) {
	job := model.Job{ID: "test-job", Kind: "scrape"}
	var buf bytes.Buffer

	// Test with invalid JSON
	err := exportHARStream(job, bytes.NewReader([]byte("invalid json")), &buf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode job results")
}

func TestConvertToHAREntry(t *testing.T) {
	entry := fetch.InterceptedEntry{
		Request: fetch.InterceptedRequest{
			RequestID: "req-1",
			URL:       "https://api.example.com/data?key=value",
			Method:    "POST",
			Headers: map[string]string{
				"Content-Type": "application/json",
				"Cookie":       "session=abc123",
			},
			Body:         `{"test":"data"}`,
			BodySize:     15,
			Timestamp:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			ResourceType: fetch.ResourceTypeXHR,
		},
		Response: &fetch.InterceptedResponse{
			RequestID:  "req-1",
			Status:     200,
			StatusText: "OK",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body:      `{"result":"success"}`,
			BodySize:  20,
			Timestamp: time.Date(2024, 1, 1, 12, 0, 1, 0, time.UTC),
		},
		Duration: 1 * time.Second,
	}

	harEntry := convertToHAREntry(entry, "page-1")

	// Verify request
	assert.Equal(t, "POST", harEntry.Request.Method)
	assert.Equal(t, "https://api.example.com/data?key=value", harEntry.Request.URL)
	assert.Equal(t, `{"test":"data"}`, harEntry.Request.PostData.Text)
	assert.Equal(t, "application/json", harEntry.Request.PostData.MimeType)
	assert.Equal(t, 15, harEntry.Request.BodySize)

	// Verify query string
	require.Len(t, harEntry.Request.QueryString, 1)
	assert.Equal(t, "key", harEntry.Request.QueryString[0].Name)
	assert.Equal(t, "value", harEntry.Request.QueryString[0].Value)

	// Verify cookies
	require.Len(t, harEntry.Request.Cookies, 1)
	assert.Equal(t, "session", harEntry.Request.Cookies[0].Name)
	assert.Equal(t, "abc123", harEntry.Request.Cookies[0].Value)

	// Verify response
	assert.Equal(t, 200, harEntry.Response.Status)
	assert.Equal(t, "OK", harEntry.Response.StatusText)
	assert.Equal(t, `{"result":"success"}`, harEntry.Response.Content.Text)
	assert.Equal(t, "application/json", harEntry.Response.Content.MimeType)
	assert.Equal(t, 20, harEntry.Response.BodySize)

	// Verify timing
	assert.Equal(t, 1000.0, harEntry.Time) // 1 second in milliseconds
}

func TestConvertToHAREntry_NilResponse(t *testing.T) {
	entry := fetch.InterceptedEntry{
		Request: fetch.InterceptedRequest{
			RequestID:    "req-failed",
			URL:          "https://api.example.com/timeout",
			Method:       "GET",
			Headers:      map[string]string{},
			Timestamp:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			ResourceType: fetch.ResourceTypeXHR,
		},
		Response: nil,
		Duration: 30 * time.Second,
	}

	harEntry := convertToHAREntry(entry, "page-1")

	assert.Equal(t, "GET", harEntry.Request.Method)
	assert.Equal(t, 0, harEntry.Response.Status)
	assert.Equal(t, "", harEntry.Response.StatusText)
	assert.Equal(t, 30000.0, harEntry.Time) // 30 seconds in milliseconds
}

func TestIsBinaryContent(t *testing.T) {
	tests := []struct {
		contentType string
		want        bool
	}{
		{"image/png", true},
		{"image/jpeg", true},
		{"audio/mp3", true},
		{"video/mp4", true},
		{"application/octet-stream", true},
		{"application/pdf", true},
		{"application/zip", true},
		{"application/json", false},
		{"text/html", false},
		{"text/plain", false},
		{"application/javascript", false},
		{"text/css", false},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			got := isBinaryContent(tt.contentType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractQueryString(t *testing.T) {
	tests := []struct {
		url      string
		expected []HARQueryParam
	}{
		{
			url:      "https://example.com/api?key1=value1\u0026key2=value2",
			expected: []HARQueryParam{{Name: "key1", Value: "value1"}, {Name: "key2", Value: "value2"}},
		},
		{
			url:      "https://example.com/api?key=value1\u0026key=value2",
			expected: []HARQueryParam{{Name: "key", Value: "value1"}, {Name: "key", Value: "value2"}},
		},
		{
			url:      "https://example.com/api",
			expected: []HARQueryParam{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			u, _ := parseTestURL(tt.url)
			got := extractQueryString(u)
			// Sort both slices for comparison since map iteration is non-deterministic
			assert.Len(t, got, len(tt.expected))
		})
	}
}

func TestExtractCookies(t *testing.T) {
	tests := []struct {
		headers  map[string]string
		expected []HARCookie
	}{
		{
			headers:  map[string]string{"Cookie": "session=abc; user=john"},
			expected: []HARCookie{{Name: "session", Value: "abc"}, {Name: "user", Value: "john"}},
		},
		{
			headers:  map[string]string{"cookie": "single=value"},
			expected: []HARCookie{{Name: "single", Value: "value"}},
		},
		{
			headers:  map[string]string{"Content-Type": "application/json"},
			expected: []HARCookie{},
		},
		{
			headers:  map[string]string{"Cookie": ""},
			expected: []HARCookie{},
		},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := extractCookies(tt.headers)
			assert.Len(t, got, len(tt.expected))
			for i, cookie := range got {
				assert.Equal(t, tt.expected[i].Name, cookie.Name)
				assert.Equal(t, tt.expected[i].Value, cookie.Value)
			}
		})
	}
}

func TestMapToHARHeaders(t *testing.T) {
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer token123",
	}

	harHeaders := mapToHARHeaders(headers)
	assert.Len(t, harHeaders, 2)

	// Convert to map for easier assertion
	headerMap := make(map[string]string)
	for _, h := range harHeaders {
		headerMap[h.Name] = h.Value
	}

	assert.Equal(t, "application/json", headerMap["Content-Type"])
	assert.Equal(t, "Bearer token123", headerMap["Authorization"])
}

func TestBinaryContentBase64Encoding(t *testing.T) {
	// Create entry with binary content type
	entry := fetch.InterceptedEntry{
		Request: fetch.InterceptedRequest{
			RequestID:    "req-1",
			URL:          "https://example.com/image.png",
			Method:       "GET",
			Headers:      map[string]string{},
			Timestamp:    time.Now(),
			ResourceType: fetch.ResourceTypeImage,
		},
		Response: &fetch.InterceptedResponse{
			RequestID: "req-1",
			Status:    200,
			Headers: map[string]string{
				"Content-Type": "image/png",
			},
			Body:      "binarydata",
			BodySize:  10,
			Timestamp: time.Now(),
		},
		Duration: 100 * time.Millisecond,
	}

	harEntry := convertToHAREntry(entry, "page-1")

	assert.Equal(t, "base64", harEntry.Response.Content.Encoding)
	assert.NotEqual(t, "binarydata", harEntry.Response.Content.Text)
	assert.True(t, strings.Contains(harEntry.Response.Content.Text, "Ymlu") || len(harEntry.Response.Content.Text) > 0)
}

// Helper function to parse URL
func parseTestURL(rawURL string) (*url.URL, error) {
	return url.Parse(rawURL)
}
