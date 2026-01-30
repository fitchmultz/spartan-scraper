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

// WebhookConfig holds webhook notification settings for a job.
// These values are stored in Job.Params to avoid database schema changes.
type WebhookConfig struct {
	URL    string   `json:"url,omitempty"`
	Events []string `json:"events,omitempty"`
	Secret string   `json:"secret,omitempty"`
}

// ExtractWebhookConfig extracts webhook configuration from job params.
// Returns nil if no webhook is configured.
func (j Job) ExtractWebhookConfig() *WebhookConfig {
	url, _ := j.Params["webhookURL"].(string)
	if url == "" {
		return nil
	}

	cfg := &WebhookConfig{
		URL:    url,
		Events: []string{"completed"}, // default
	}

	if events, ok := j.Params["webhookEvents"].([]string); ok && len(events) > 0 {
		cfg.Events = events
	}
	if events, ok := j.Params["webhookEvents"].([]interface{}); ok && len(events) > 0 {
		cfg.Events = make([]string, 0, len(events))
		for _, e := range events {
			if s, ok := e.(string); ok {
				cfg.Events = append(cfg.Events, s)
			}
		}
	}
	if secret, ok := j.Params["webhookSecret"].(string); ok && secret != "" {
		cfg.Secret = secret
	}

	return cfg
}

type Job struct {
	ID         string                 `json:"id"`
	Kind       Kind                   `json:"kind"`
	Status     Status                 `json:"status"`
	CreatedAt  time.Time              `json:"createdAt"`
	UpdatedAt  time.Time              `json:"updatedAt"`
	Params     map[string]interface{} `json:"params"`
	ResultPath string                 `json:"resultPath,omitempty"`
	Error      string                 `json:"error"`
}
