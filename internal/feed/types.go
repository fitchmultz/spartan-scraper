// Package feed provides RSS/Atom feed monitoring and parsing functionality.
//
// This package is responsible for:
// - Defining feed configuration types
// - Storing and retrieving feed configurations
// - Fetching and parsing RSS/Atom feeds
// - Tracking seen feed items for deduplication
// - Scheduling periodic feed checks
// - Creating scrape jobs for new feed items
//
// This file does NOT handle:
// - Content fetching (uses fetch package)
// - Job execution (jobs package handles this)
// - Webhook delivery (webhook package handles this)
//
// Invariants:
// - Feed IDs are UUIDs
// - Feed URLs are normalized before storage
// - Intervals are validated to be reasonable (>= 60 seconds)
// - Item deduplication uses GUID as primary key, falls back to link URL
package feed

import (
	"errors"
	"time"
)

// FeedType represents the type of feed.
type FeedType string

const (
	FeedTypeRSS  FeedType = "rss"
	FeedTypeAtom FeedType = "atom"
	FeedTypeAuto FeedType = "auto"
)

// Feed represents an RSS/Atom feed configuration.
type Feed struct {
	ID                  string            `json:"id"`
	URL                 string            `json:"url"`
	FeedType            FeedType          `json:"feedType"`
	IntervalSeconds     int               `json:"intervalSeconds"`
	Enabled             bool              `json:"enabled"`
	AutoScrape          bool              `json:"autoScrape"`               // Create scrape jobs for new items
	ExtractOptions      map[string]string `json:"extractOptions,omitempty"` // Options for auto-scrape jobs
	CreatedAt           time.Time         `json:"createdAt"`
	LastCheckedAt       time.Time         `json:"lastCheckedAt,omitempty"`
	LastError           string            `json:"lastError,omitempty"`
	ConsecutiveFailures int               `json:"consecutiveFailures"`
}

// FeedItem represents a parsed feed item/entry.
type FeedItem struct {
	GUID        string    `json:"guid"`
	Title       string    `json:"title"`
	Link        string    `json:"link"`
	Description string    `json:"description,omitempty"`
	Content     string    `json:"content,omitempty"`
	PubDate     time.Time `json:"pubDate,omitempty"`
	Author      string    `json:"author,omitempty"`
	Categories  []string  `json:"categories,omitempty"`
}

// FeedCheckResult contains the outcome of a feed check.
type FeedCheckResult struct {
	FeedID     string     `json:"feedId"`
	CheckedAt  time.Time  `json:"checkedAt"`
	NewItems   []FeedItem `json:"newItems"`
	TotalItems int        `json:"totalItems"`
	FeedTitle  string     `json:"feedTitle,omitempty"`
	FeedDesc   string     `json:"feedDesc,omitempty"`
	Error      string     `json:"error,omitempty"`
}

// SeenItem tracks when a feed item was first seen for deduplication.
type SeenItem struct {
	GUID   string    `json:"guid"`
	Link   string    `json:"link"`
	Title  string    `json:"title,omitempty"`
	SeenAt time.Time `json:"seenAt"`
}

// ValidationError represents a validation error for a feed field.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// IsNotFoundError checks if an error is a NotFoundError.
func IsNotFoundError(err error) bool {
	var notFoundErr *NotFoundError
	return errors.As(err, &notFoundErr)
}

// NotFoundError is returned when a feed is not found.
type NotFoundError struct {
	ID string
}

func (e *NotFoundError) Error() string {
	return "feed not found: " + e.ID
}

// Validate checks if the feed configuration is valid.
func (f *Feed) Validate() error {
	if f.URL == "" {
		return &ValidationError{Field: "url", Message: "URL is required"}
	}
	if f.IntervalSeconds <= 0 {
		return &ValidationError{Field: "intervalSeconds", Message: "interval must be greater than 0"}
	}
	if f.IntervalSeconds < 60 {
		return &ValidationError{Field: "intervalSeconds", Message: "interval must be at least 60 seconds"}
	}
	if f.FeedType != "" && f.FeedType != FeedTypeRSS && f.FeedType != FeedTypeAtom && f.FeedType != FeedTypeAuto {
		return &ValidationError{Field: "feedType", Message: "invalid feed type"}
	}
	return nil
}

// IsDue returns true if the feed is due for a check based on its interval.
func (f *Feed) IsDue() bool {
	if !f.Enabled {
		return false
	}
	if f.LastCheckedAt.IsZero() {
		return true
	}
	return time.Since(f.LastCheckedAt) >= time.Duration(f.IntervalSeconds)*time.Second
}

// NextRun returns the time when the feed should next be checked.
func (f *Feed) NextRun() time.Time {
	if f.LastCheckedAt.IsZero() {
		return time.Now()
	}
	return f.LastCheckedAt.Add(time.Duration(f.IntervalSeconds) * time.Second)
}

// GetStatus returns the current status of the feed.
func (f *Feed) GetStatus() string {
	if !f.Enabled {
		return "disabled"
	}
	if f.ConsecutiveFailures > 0 {
		return "error"
	}
	return "active"
}

// ItemKey returns the deduplication key for a feed item.
// Uses GUID if available, otherwise falls back to link URL.
func (fi *FeedItem) ItemKey() string {
	if fi.GUID != "" {
		return fi.GUID
	}
	return fi.Link
}
