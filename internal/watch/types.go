// Package watch provides content change monitoring functionality.
//
// This package is responsible for:
// - Defining watch configuration types
// - Storing and retrieving watch configurations
// - Executing watch checks and detecting changes
// - Scheduling periodic watch checks
//
// This file does NOT handle:
// - Content fetching (fetch package handles this)
// - Diff generation (diff package handles this)
// - Webhook delivery (webhook package handles this)
//
// Invariants:
// - Watch IDs are UUIDs
// - URLs are normalized before storage
// - Intervals are validated to be reasonable (> 0)
package watch

import (
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

// Watch represents a content monitoring configuration.
type Watch struct {
	ID              string    `json:"id"`
	URL             string    `json:"url"`
	Selector        string    `json:"selector,omitempty"` // CSS selector for targeted monitoring
	IntervalSeconds int       `json:"intervalSeconds"`    // Check interval
	Enabled         bool      `json:"enabled"`
	CreatedAt       time.Time `json:"createdAt"`
	LastCheckedAt   time.Time `json:"lastCheckedAt,omitempty"`
	LastChangedAt   time.Time `json:"lastChangedAt,omitempty"`
	ChangeCount     int       `json:"changeCount"` // Total number of changes detected

	// Configuration
	DiffFormat     string               `json:"diffFormat"` // unified, html-side-by-side, html-inline
	WebhookConfig  *model.WebhookConfig `json:"webhookConfig"`
	NotifyOnChange bool                 `json:"notifyOnChange"`

	// Filters
	MinChangeSize  int      `json:"minChangeSize,omitempty"`  // Ignore changes smaller than N bytes
	IgnorePatterns []string `json:"ignorePatterns,omitempty"` // Regex patterns to ignore

	// Content extraction options
	Headless      bool   `json:"headless"`              // Use headless browser
	UsePlaywright bool   `json:"usePlaywright"`         // Use Playwright instead of chromedp
	ExtractMode   string `json:"extractMode,omitempty"` // Extraction mode (text, html, markdown)

	// Screenshot configuration for visual change detection
	ScreenshotConfig    *fetch.ScreenshotConfig `json:"screenshotConfig,omitempty"`    // Screenshot capture options
	ScreenshotEnabled   bool                    `json:"screenshotEnabled"`             // Enable visual change detection
	VisualDiffThreshold float64                 `json:"visualDiffThreshold,omitempty"` // Pixel difference threshold (0-1, default 0.1 = 10%)
}

// WatchCheckResult contains the outcome of a watch check.
type WatchCheckResult struct {
	WatchID      string    `json:"watchId"`
	URL          string    `json:"url"`
	CheckedAt    time.Time `json:"checkedAt"`
	Changed      bool      `json:"changed"`
	PreviousHash string    `json:"previousHash,omitempty"`
	CurrentHash  string    `json:"currentHash,omitempty"`
	DiffText     string    `json:"diffText,omitempty"`
	DiffHTML     string    `json:"diffHtml,omitempty"`
	Error        string    `json:"error,omitempty"`
	Selector     string    `json:"selector,omitempty"`

	// Visual change detection fields
	ScreenshotPath         string  `json:"screenshotPath,omitempty"`         // Path to current screenshot
	PreviousScreenshotPath string  `json:"previousScreenshotPath,omitempty"` // Path to previous screenshot
	VisualDiffPath         string  `json:"visualDiffPath,omitempty"`         // Path to generated visual diff image
	VisualHash             string  `json:"visualHash,omitempty"`             // Perceptual hash of current screenshot
	PreviousVisualHash     string  `json:"previousVisualHash,omitempty"`     // Perceptual hash of previous screenshot
	VisualChanged          bool    `json:"visualChanged"`                    // True if visual change detected
	VisualSimilarity       float64 `json:"visualSimilarity,omitempty"`       // Similarity score (0-1)
}

// IsDue returns true if the watch is due for a check based on its interval.
func (w *Watch) IsDue() bool {
	if !w.Enabled {
		return false
	}
	if w.LastCheckedAt.IsZero() {
		return true
	}
	return time.Since(w.LastCheckedAt) >= time.Duration(w.IntervalSeconds)*time.Second
}

// NextRun returns the time when the watch should next be checked.
func (w *Watch) NextRun() time.Time {
	if w.LastCheckedAt.IsZero() {
		return time.Now()
	}
	return w.LastCheckedAt.Add(time.Duration(w.IntervalSeconds) * time.Second)
}

// Validate checks if the watch configuration is valid.
func (w *Watch) Validate() error {
	if w.URL == "" {
		return &ValidationError{Field: "url", Message: "URL is required"}
	}
	if w.IntervalSeconds <= 0 {
		return &ValidationError{Field: "intervalSeconds", Message: "interval must be greater than 0"}
	}
	if w.IntervalSeconds < 60 {
		return &ValidationError{Field: "intervalSeconds", Message: "interval must be at least 60 seconds"}
	}
	if w.VisualDiffThreshold < 0 || w.VisualDiffThreshold > 1 {
		return &ValidationError{Field: "visualDiffThreshold", Message: "visual diff threshold must be between 0 and 1"}
	}
	return nil
}

// ValidationError represents a validation error for a watch field.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// Status represents the current status of a watch.
type Status string

const (
	StatusActive   Status = "active"
	StatusPaused   Status = "paused"
	StatusError    Status = "error"
	StatusDisabled Status = "disabled"
)

// GetStatus returns the current status of the watch.
func (w *Watch) GetStatus() Status {
	if !w.Enabled {
		return StatusDisabled
	}
	return StatusActive
}
