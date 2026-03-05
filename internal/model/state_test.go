// Package model provides tests for CrawlState serialization and field handling.
//
// Tests cover:
// - JSON marshaling/unmarshaling of CrawlState
// - Field type verification (URL, ETag, LastModified, ContentHash, LastScraped)
// - Handling of fully populated, zero value, and partial state objects
// - Time serialization in RFC3339 format
//
// Does NOT test:
// - Job model behaviors (see job_test.go)
// - Job sanitization (see job_sanitize_test.go)
// - Database persistence of crawl states
//
// Assumes:
// - CrawlState fields use json tags for proper serialization
// - LastScraped is stored as RFC3339 formatted string in JSON
package model

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCrawlState_JSONSerialization(t *testing.T) {
	tests := []struct {
		name  string
		state CrawlState
	}{
		{
			name: "fully populated",
			state: CrawlState{
				URL:          "https://example.com",
				ETag:         "\"abc123\"",
				LastModified: "Wed, 21 Oct 2015 07:28:00 GMT",
				ContentHash:  "sha256:1234",
				LastScraped:  time.Date(2025, 1, 26, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "zero values",
			state: CrawlState{
				URL:          "",
				ETag:         "",
				LastModified: "",
				ContentHash:  "",
				LastScraped:  time.Time{},
			},
		},
		{
			name: "partial fields",
			state: CrawlState{
				URL:         "https://example.com",
				LastScraped: time.Date(2025, 1, 26, 12, 30, 45, 0, time.UTC),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.state)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded CrawlState
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.URL != tt.state.URL {
				t.Errorf("URL = %s; want %s", decoded.URL, tt.state.URL)
			}
			if decoded.ETag != tt.state.ETag {
				t.Errorf("ETag = %s; want %s", decoded.ETag, tt.state.ETag)
			}
			if decoded.LastModified != tt.state.LastModified {
				t.Errorf("LastModified = %s; want %s", decoded.LastModified, tt.state.LastModified)
			}
			if decoded.ContentHash != tt.state.ContentHash {
				t.Errorf("ContentHash = %s; want %s", decoded.ContentHash, tt.state.ContentHash)
			}
			if !decoded.LastScraped.Equal(tt.state.LastScraped) {
				t.Errorf("LastScraped = %v; want %v", decoded.LastScraped, tt.state.LastScraped)
			}
		})
	}
}

func TestCrawlState_FieldTypes(t *testing.T) {
	state := CrawlState{
		URL:          "https://example.com",
		ETag:         "test-etag",
		LastModified: "test-date",
		ContentHash:  "test-hash",
		LastScraped:  time.Date(2025, 1, 26, 12, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var jsonMap map[string]interface{}
	if err := json.Unmarshal(data, &jsonMap); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	tests := []struct {
		field   string
		wantURL string
	}{
		{"url", "https://example.com"},
		{"etag", "test-etag"},
		{"lastModified", "test-date"},
		{"contentHash", "test-hash"},
	}

	for _, tt := range tests {
		if val, ok := jsonMap[tt.field]; !ok {
			t.Errorf("field %s not found in JSON", tt.field)
		} else if str, ok := val.(string); !ok {
			t.Errorf("field %s is not a string", tt.field)
		} else if str != tt.wantURL {
			t.Errorf("field %s = %s; want %s", tt.field, str, tt.wantURL)
		}
	}

	if lastScrapedStr, ok := jsonMap["lastScraped"].(string); !ok {
		t.Errorf("lastScraped is not a string in JSON")
	} else if lastScrapedStr != state.LastScraped.Format(time.RFC3339) {
		t.Errorf("lastScraped = %s; want %s", lastScrapedStr, state.LastScraped.Format(time.RFC3339))
	}
}
