// Package watch provides content change monitoring functionality.
//
// This file contains tests for watch types.
package watch

import (
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestWatchIsDue(t *testing.T) {
	tests := []struct {
		name     string
		watch    Watch
		expected bool
	}{
		{
			name: "disabled watch",
			watch: Watch{
				Enabled:         false,
				IntervalSeconds: 60,
				LastCheckedAt:   time.Now().Add(-2 * time.Minute),
			},
			expected: false,
		},
		{
			name: "never checked",
			watch: Watch{
				Enabled:         true,
				IntervalSeconds: 60,
				LastCheckedAt:   time.Time{},
			},
			expected: true,
		},
		{
			name: "checked recently",
			watch: Watch{
				Enabled:         true,
				IntervalSeconds: 3600,
				LastCheckedAt:   time.Now().Add(-5 * time.Minute),
			},
			expected: false,
		},
		{
			name: "due for check",
			watch: Watch{
				Enabled:         true,
				IntervalSeconds: 60,
				LastCheckedAt:   time.Now().Add(-2 * time.Minute),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.watch.IsDue()
			if result != tt.expected {
				t.Errorf("IsDue() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestWatchNextRun(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		watch    Watch
		expected time.Time
	}{
		{
			name: "never checked",
			watch: Watch{
				IntervalSeconds: 60,
				LastCheckedAt:   time.Time{},
			},
			expected: now,
		},
		{
			name: "checked before",
			watch: Watch{
				IntervalSeconds: 3600,
				LastCheckedAt:   now,
			},
			expected: now.Add(time.Hour),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.watch.NextRun()
			// Allow some tolerance for "now" comparison
			diff := result.Sub(tt.expected)
			if diff < -time.Second || diff > time.Second {
				t.Errorf("NextRun() = %v, want %v (diff: %v)", result, tt.expected, diff)
			}
		})
	}
}

func TestWatchValidate(t *testing.T) {
	tests := []struct {
		name    string
		watch   Watch
		wantErr bool
	}{
		{
			name: "valid watch",
			watch: Watch{
				URL:             "https://example.com",
				IntervalSeconds: 3600,
			},
			wantErr: false,
		},
		{
			name: "missing URL",
			watch: Watch{
				URL:             "",
				IntervalSeconds: 3600,
			},
			wantErr: true,
		},
		{
			name: "zero interval",
			watch: Watch{
				URL:             "https://example.com",
				IntervalSeconds: 0,
			},
			wantErr: true,
		},
		{
			name: "negative interval",
			watch: Watch{
				URL:             "https://example.com",
				IntervalSeconds: -1,
			},
			wantErr: true,
		},
		{
			name: "interval too small",
			watch: Watch{
				URL:             "https://example.com",
				IntervalSeconds: 30,
			},
			wantErr: true,
		},
		{
			name: "minimum valid interval",
			watch: Watch{
				URL:             "https://example.com",
				IntervalSeconds: 60,
			},
			wantErr: false,
		},
		{
			name: "valid webhook url",
			watch: Watch{
				URL:             "https://example.com",
				IntervalSeconds: 3600,
				WebhookConfig:   &model.WebhookSpec{URL: "https://hooks.example.com/watch"},
			},
			wantErr: false,
		},
		{
			name: "invalid webhook url scheme",
			watch: Watch{
				URL:             "https://example.com",
				IntervalSeconds: 3600,
				WebhookConfig:   &model.WebhookSpec{URL: "ftp://hooks.example.com/watch"},
			},
			wantErr: true,
		},
		{
			name: "present webhook config requires url",
			watch: Watch{
				URL:             "https://example.com",
				IntervalSeconds: 3600,
				WebhookConfig:   &model.WebhookSpec{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.watch.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{
		Field:   "url",
		Message: "URL is required",
	}

	expected := "url: URL is required"
	if err.Error() != expected {
		t.Errorf("Error() = %v, want %v", err.Error(), expected)
	}
}

func TestWatchGetStatus(t *testing.T) {
	tests := []struct {
		name     string
		watch    Watch
		expected Status
	}{
		{
			name:     "enabled watch",
			watch:    Watch{Enabled: true},
			expected: StatusActive,
		},
		{
			name:     "disabled watch",
			watch:    Watch{Enabled: false},
			expected: StatusDisabled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.watch.GetStatus()
			if result != tt.expected {
				t.Errorf("GetStatus() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestWatchCheckResult(t *testing.T) {
	now := time.Now()
	result := WatchCheckResult{
		WatchID:      "watch-123",
		URL:          "https://example.com",
		CheckedAt:    now,
		Changed:      true,
		PreviousHash: "abc123",
		CurrentHash:  "def456",
		DiffText:     "-old\n+new",
		DiffHTML:     "<div>...</div>",
		Selector:     "#price",
	}

	if result.WatchID != "watch-123" {
		t.Errorf("WatchID = %v, want watch-123", result.WatchID)
	}

	if result.URL != "https://example.com" {
		t.Errorf("URL = %v, want https://example.com", result.URL)
	}

	if result.CheckedAt != now {
		t.Error("CheckedAt mismatch")
	}

	if !result.Changed {
		t.Error("Expected Changed to be true")
	}

	if result.PreviousHash != "abc123" {
		t.Errorf("PreviousHash = %v, want abc123", result.PreviousHash)
	}

	if result.CurrentHash != "def456" {
		t.Errorf("CurrentHash = %v, want def456", result.CurrentHash)
	}

	if result.DiffText != "-old\n+new" {
		t.Errorf("DiffText = %v, want -old\\n+new", result.DiffText)
	}

	if result.DiffHTML != "<div>...</div>" {
		t.Errorf("DiffHTML = %v, want <div>...</div>", result.DiffHTML)
	}

	if result.Selector != "#price" {
		t.Errorf("Selector = %v, want #price", result.Selector)
	}
}

func TestWatchWithSelector(t *testing.T) {
	watch := Watch{
		ID:              "watch-456",
		URL:             "https://example.com/products",
		Selector:        "#price",
		IntervalSeconds: 300,
		Enabled:         true,
	}

	if err := watch.Validate(); err != nil {
		t.Errorf("Expected valid watch, got error: %v", err)
	}

	if watch.ID != "watch-456" {
		t.Errorf("ID = %v, want watch-456", watch.ID)
	}

	if watch.URL != "https://example.com/products" {
		t.Errorf("URL = %v, want https://example.com/products", watch.URL)
	}

	if watch.IntervalSeconds != 300 {
		t.Errorf("IntervalSeconds = %v, want 300", watch.IntervalSeconds)
	}

	if !watch.Enabled {
		t.Error("Expected Enabled to be true")
	}

	if watch.Selector != "#price" {
		t.Errorf("Selector = %v, want #price", watch.Selector)
	}
}

func TestWatchWithWebhook(t *testing.T) {
	watch := Watch{
		ID:              "watch-789",
		URL:             "https://example.com",
		IntervalSeconds: 3600,
		Enabled:         true,
		NotifyOnChange:  true,
	}

	if watch.ID != "watch-789" {
		t.Errorf("ID = %v, want watch-789", watch.ID)
	}

	if watch.URL != "https://example.com" {
		t.Errorf("URL = %v, want https://example.com", watch.URL)
	}

	if watch.IntervalSeconds != 3600 {
		t.Errorf("IntervalSeconds = %v, want 3600", watch.IntervalSeconds)
	}

	if !watch.Enabled {
		t.Error("Expected Enabled to be true")
	}

	if !watch.NotifyOnChange {
		t.Error("Expected NotifyOnChange to be true")
	}
}
