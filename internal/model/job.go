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
)

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
