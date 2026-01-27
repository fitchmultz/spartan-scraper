// Package model defines shared domain types for jobs, crawling, and state tracking.
// It handles type definitions for Job, Kind, Status, and CrawlState.
// It does NOT handle job persistence, execution, or state transitions.
package model

import "time"

type Kind string

type Status string

const (
	KindScrape   Kind = "scrape"
	KindCrawl    Kind = "crawl"
	KindResearch Kind = "research"

	StatusQueued    Status = "queued"
	StatusRunning   Status = "running"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusCanceled  Status = "canceled"
)

var validStatuses = map[Status]bool{
	StatusQueued:    true,
	StatusRunning:   true,
	StatusSucceeded: true,
	StatusFailed:    true,
	StatusCanceled:  true,
}

func (s Status) IsTerminal() bool {
	return s == StatusSucceeded || s == StatusFailed || s == StatusCanceled
}

func (s Status) IsValid() bool {
	return validStatuses[s]
}

func ValidStatuses() []Status {
	return []Status{StatusQueued, StatusRunning, StatusSucceeded, StatusFailed, StatusCanceled}
}

type Job struct {
	ID         string                 `json:"id"`
	Kind       Kind                   `json:"kind"`
	Status     Status                 `json:"status"`
	CreatedAt  time.Time              `json:"createdAt"`
	UpdatedAt  time.Time              `json:"updatedAt"`
	Params     map[string]interface{} `json:"params"`
	ResultPath string                 `json:"resultPath"`
	Error      string                 `json:"error"`
}
