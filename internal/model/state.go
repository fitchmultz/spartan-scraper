// Package model defines shared domain types for jobs, crawling, and state tracking.
// It handles type definitions for Job, Kind, Status, and CrawlState.
// It does NOT handle job persistence, execution, or state transitions.
package model

import "time"

type CrawlState struct {
	URL             string    `json:"url"`
	ETag            string    `json:"etag"`
	LastModified    string    `json:"lastModified"`
	ContentHash     string    `json:"contentHash"`
	LastScraped     time.Time `json:"lastScraped"`
	Depth           int       `json:"depth"`
	JobID           string    `json:"jobId"`
	PreviousContent string    `json:"previousContent,omitempty"` // Previous content snapshot for diff generation
	ContentSnapshot string    `json:"contentSnapshot,omitempty"` // Current full content snapshot
}
