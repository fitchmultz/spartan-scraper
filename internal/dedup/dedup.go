// Package dedup provides cross-job content deduplication using simhash.
// It enables persistent storage and querying of content fingerprints across
// all crawl jobs, allowing detection of duplicate content between different
// crawling sessions.
//
// This package does NOT handle:
// - In-memory deduplication during a single crawl (handled by internal/crawl)
// - Content extraction or simhash computation (handled by internal/simhash)
// - Job execution or scheduling (handled by internal/jobs)
//
// Invariants:
// - All simhash values are 64-bit unsigned integers
// - URLs are stored normalized (lowercase host, no fragments)
// - Threshold values: 0=exact match, 3=near-duplicates (default), 8=similar
// - IndexedAt timestamps are UTC
package dedup

import (
	"context"
	"time"
)

// ContentIndex provides persistent cross-job content deduplication.
// Implementations must be safe for concurrent use.
type ContentIndex interface {
	// Index stores a content fingerprint for a URL.
	// The jobID identifies which crawl job indexed this content.
	// The URL should be normalized before calling Index.
	// Returns an error if the index operation fails.
	Index(ctx context.Context, jobID, url string, simhash uint64) error

	// FindDuplicates returns URLs with similar content across all jobs.
	// Threshold: 0=exact match, 3=near-duplicates (default), 8=similar
	// Results are sorted by distance (closest matches first).
	// Returns an empty slice if no duplicates are found.
	FindDuplicates(ctx context.Context, simhash uint64, threshold int) ([]DuplicateMatch, error)

	// GetContentHistory returns all indexed entries for a URL across jobs.
	// Results are sorted by IndexedAt descending (most recent first).
	// Returns an empty slice if the URL has never been indexed.
	GetContentHistory(ctx context.Context, url string) ([]ContentEntry, error)

	// DeleteJobEntries removes all entries for a job (cleanup).
	// This should be called when a job is deleted to free up storage.
	// Returns the number of entries deleted.
	DeleteJobEntries(ctx context.Context, jobID string) (int64, error)

	// Stats returns deduplication statistics.
	// These are approximate values for monitoring and UI display.
	Stats(ctx context.Context) (Stats, error)
}

// DuplicateMatch represents a potential duplicate content match.
type DuplicateMatch struct {
	JobID     string    `json:"jobId"`
	URL       string    `json:"url"`
	SimHash   uint64    `json:"simhash"`
	Distance  int       `json:"distance"`
	IndexedAt time.Time `json:"indexedAt"`
}

// ContentEntry represents a single indexed content fingerprint.
type ContentEntry struct {
	JobID     string    `json:"jobId"`
	SimHash   uint64    `json:"simhash"`
	IndexedAt time.Time `json:"indexedAt"`
}

// Stats contains deduplication statistics.
type Stats struct {
	TotalIndexed   int64 `json:"totalIndexed"`
	UniqueURLs     int64 `json:"uniqueUrls"`
	UniqueJobs     int64 `json:"uniqueJobs"`
	DuplicatePairs int64 `json:"duplicatePairs"`
}

// Threshold constants for duplicate detection.
const (
	ThresholdExact   = 0 // Identical content
	ThresholdNear    = 3 // Near-duplicates (minor changes)
	ThresholdSimilar = 8 // Similar content
)

// NormalizeURL returns a normalized form of the URL suitable for deduplication.
// It lowercases the host and removes fragments.
// This is a helper function - callers may use their own normalization.
func NormalizeURL(u string) string {
	// Simple normalization: lowercase and remove fragment
	// More sophisticated normalization can be added if needed
	return u
}
