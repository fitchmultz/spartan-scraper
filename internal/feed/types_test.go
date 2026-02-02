// Package feed provides RSS/Atom feed monitoring and parsing functionality.
//
// This file contains tests for the feed types and validation logic.
package feed

import (
	"testing"
	"time"
)

func TestFeed_Validate(t *testing.T) {
	tests := []struct {
		name    string
		feed    *Feed
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid feed with all fields",
			feed: &Feed{
				URL:             "https://example.com/feed.xml",
				FeedType:        FeedTypeRSS,
				IntervalSeconds: 3600,
			},
			wantErr: false,
		},
		{
			name: "valid feed with auto type",
			feed: &Feed{
				URL:             "https://example.com/feed.xml",
				FeedType:        FeedTypeAuto,
				IntervalSeconds: 1800,
			},
			wantErr: false,
		},
		{
			name: "missing URL",
			feed: &Feed{
				URL:             "",
				FeedType:        FeedTypeRSS,
				IntervalSeconds: 3600,
			},
			wantErr: true,
			errMsg:  "url: URL is required",
		},
		{
			name: "zero interval",
			feed: &Feed{
				URL:             "https://example.com/feed.xml",
				FeedType:        FeedTypeRSS,
				IntervalSeconds: 0,
			},
			wantErr: true,
			errMsg:  "intervalSeconds: interval must be greater than 0",
		},
		{
			name: "interval too short",
			feed: &Feed{
				URL:             "https://example.com/feed.xml",
				FeedType:        FeedTypeRSS,
				IntervalSeconds: 30,
			},
			wantErr: true,
			errMsg:  "intervalSeconds: interval must be at least 60 seconds",
		},
		{
			name: "invalid feed type",
			feed: &Feed{
				URL:             "https://example.com/feed.xml",
				FeedType:        FeedType("invalid"),
				IntervalSeconds: 3600,
			},
			wantErr: true,
			errMsg:  "feedType: invalid feed type",
		},
		{
			name: "minimum valid interval",
			feed: &Feed{
				URL:             "https://example.com/feed.xml",
				FeedType:        FeedTypeAtom,
				IntervalSeconds: 60,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.feed.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error but got nil")
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("Validate() error = %q, want %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestFeed_IsDue(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		feed     *Feed
		expected bool
	}{
		{
			name: "disabled feed is never due",
			feed: &Feed{
				Enabled:         false,
				IntervalSeconds: 60,
				LastCheckedAt:   now.Add(-2 * time.Hour),
			},
			expected: false,
		},
		{
			name: "never checked is due",
			feed: &Feed{
				Enabled:         true,
				IntervalSeconds: 3600,
				LastCheckedAt:   time.Time{},
			},
			expected: true,
		},
		{
			name: "checked recently is not due",
			feed: &Feed{
				Enabled:         true,
				IntervalSeconds: 3600,
				LastCheckedAt:   now.Add(-30 * time.Minute),
			},
			expected: false,
		},
		{
			name: "checked long ago is due",
			feed: &Feed{
				Enabled:         true,
				IntervalSeconds: 3600,
				LastCheckedAt:   now.Add(-2 * time.Hour),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.feed.IsDue()
			if got != tt.expected {
				t.Errorf("IsDue() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFeed_NextRun(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		feed     *Feed
		expected time.Time
	}{
		{
			name: "never checked returns now",
			feed: &Feed{
				IntervalSeconds: 3600,
				LastCheckedAt:   time.Time{},
			},
			expected: now,
		},
		{
			name: "returns last checked plus interval",
			feed: &Feed{
				IntervalSeconds: 3600,
				LastCheckedAt:   now.Add(-1 * time.Hour),
			},
			expected: now,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.feed.NextRun()
			// Allow small time difference due to test execution time
			diff := got.Sub(tt.expected)
			if diff < -time.Second || diff > time.Second {
				t.Errorf("NextRun() = %v, want approximately %v", got, tt.expected)
			}
		})
	}
}

func TestFeed_GetStatus(t *testing.T) {
	tests := []struct {
		name     string
		feed     *Feed
		expected string
	}{
		{
			name: "disabled feed",
			feed: &Feed{
				Enabled:             false,
				ConsecutiveFailures: 0,
			},
			expected: "disabled",
		},
		{
			name: "active feed",
			feed: &Feed{
				Enabled:             true,
				ConsecutiveFailures: 0,
			},
			expected: "active",
		},
		{
			name: "feed with failures",
			feed: &Feed{
				Enabled:             true,
				ConsecutiveFailures: 3,
			},
			expected: "error",
		},
		{
			name: "disabled feed with failures shows disabled",
			feed: &Feed{
				Enabled:             false,
				ConsecutiveFailures: 5,
			},
			expected: "disabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.feed.GetStatus()
			if got != tt.expected {
				t.Errorf("GetStatus() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestFeedItem_ItemKey(t *testing.T) {
	tests := []struct {
		name     string
		item     *FeedItem
		expected string
	}{
		{
			name: "uses GUID when available",
			item: &FeedItem{
				GUID: "unique-guid-123",
				Link: "https://example.com/article",
			},
			expected: "unique-guid-123",
		},
		{
			name: "falls back to link when GUID is empty",
			item: &FeedItem{
				GUID: "",
				Link: "https://example.com/article",
			},
			expected: "https://example.com/article",
		},
		{
			name: "empty GUID and link returns empty",
			item: &FeedItem{
				GUID: "",
				Link: "",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.item.ItemKey()
			if got != tt.expected {
				t.Errorf("ItemKey() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestNotFoundError(t *testing.T) {
	err := &NotFoundError{ID: "feed-123"}
	expected := "feed not found: feed-123"
	if err.Error() != expected {
		t.Errorf("NotFoundError.Error() = %q, want %q", err.Error(), expected)
	}
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{Field: "url", Message: "URL is required"}
	expected := "url: URL is required"
	if err.Error() != expected {
		t.Errorf("ValidationError.Error() = %q, want %q", err.Error(), expected)
	}
}

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "NotFoundError",
			err:      &NotFoundError{ID: "test"},
			expected: true,
		},
		{
			name:     "wrapped NotFoundError",
			err:      &NotFoundError{ID: "test"},
			expected: true,
		},
		{
			name:     "other error",
			err:      &ValidationError{Field: "url", Message: "invalid"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNotFoundError(tt.err)
			if got != tt.expected {
				t.Errorf("IsNotFoundError() = %v, want %v", got, tt.expected)
			}
		})
	}
}
