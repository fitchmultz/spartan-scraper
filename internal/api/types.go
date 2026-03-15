// Package api provides HTTP request and response types for the Spartan Scraper API.
// These types are used for JSON encoding/decoding of API requests and responses.
//
// Purpose:
// - Define stable request and response contracts shared across REST handlers and MCP adapters.
//
// Responsibilities:
// - Hold operator-facing request payload types for scrape, crawl, research, schedules, and batches.
// - Hold stable job and batch response envelope types used across transports.
//
// Scope:
// - JSON contracts only; handler logic and response construction live elsewhere in this package.
//
// Usage:
// - Imported by REST handlers, MCP handlers, tests, and generated OpenAPI maintenance work.
//
// Invariants/Assumptions:
// - Job and batch automation surfaces should reuse the same response envelope shapes.
// - Response envelopes expose sanitized jobs rather than persisted raw records.
package api

import (
	"encoding/json"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
	"github.com/fitchmultz/spartan-scraper/internal/submission"
)

// ComponentStatus represents the health of a single system component.
type ComponentStatus struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Details interface{} `json:"details,omitempty"`
}

// HealthResponse represents the overall health of the system.
type HealthResponse struct {
	Status     string                     `json:"status"`
	Version    string                     `json:"version"`
	Components map[string]ComponentStatus `json:"components"`
}

// ErrorResponse represents a standard error response.
type ErrorResponse struct {
	Error     string `json:"error"`
	RequestID string `json:"requestId,omitempty"`
}

// StatusResponse represents a generic success response.
type StatusResponse struct {
	Status    string `json:"status"`
	RequestID string `json:"requestId,omitempty"`
}

// WebhookConfig represents webhook configuration for job notifications.
type WebhookConfig = submission.WebhookConfig

// ScrapeRequest represents a request to scrape a single page.
type ScrapeRequest = submission.ScrapeRequest

// CrawlRequest represents a request to crawl a website.
type CrawlRequest = submission.CrawlRequest

// ResearchRequest represents a request to perform deep research across multiple URLs.
type ResearchRequest = submission.ResearchRequest

// ScheduleRequest represents a request to add a scheduled job.
type ScheduleRequest struct {
	Kind            string          `json:"kind"`
	IntervalSeconds int             `json:"intervalSeconds"`
	Request         json.RawMessage `json:"request"`
}

// ScheduleResponse represents a schedule in the response.
type ScheduleResponse struct {
	ID              string `json:"id"`
	Kind            string `json:"kind"`
	IntervalSeconds int    `json:"intervalSeconds"`
	NextRun         string `json:"nextRun"`
	Request         any    `json:"request"`
}

// BatchJobRequest represents a single job within a batch.
type BatchJobRequest struct {
	URL         string            `json:"url"`
	Method      string            `json:"method,omitempty"`
	Body        string            `json:"body,omitempty"`
	ContentType string            `json:"contentType,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
}

// BatchScrapeRequest creates multiple scrape jobs.
type BatchScrapeRequest struct {
	Jobs             []BatchJobRequest             `json:"jobs"`
	OutputFormat     string                        `json:"outputFormat,omitempty"`
	ExtractionName   string                        `json:"extractionName,omitempty"`
	ExtractionMode   string                        `json:"extractionMode,omitempty"`
	Headless         bool                          `json:"headless"`
	Playwright       *bool                         `json:"playwright,omitempty"`
	TimeoutSeconds   int                           `json:"timeoutSeconds"`
	AuthProfile      string                        `json:"authProfile,omitempty"`
	Auth             *fetch.AuthOptions            `json:"auth,omitempty"`
	Extract          *extract.ExtractOptions       `json:"extract,omitempty"`
	Pipeline         *pipeline.Options             `json:"pipeline,omitempty"`
	Incremental      *bool                         `json:"incremental,omitempty"`
	Webhook          *WebhookConfig                `json:"webhook,omitempty"`
	Screenshot       *fetch.ScreenshotConfig       `json:"screenshot,omitempty"`
	Device           *fetch.DeviceEmulation        `json:"device,omitempty"`
	NetworkIntercept *fetch.NetworkInterceptConfig `json:"networkIntercept,omitempty"`
}

// BatchCrawlRequest creates multiple crawl jobs.
type BatchCrawlRequest struct {
	Jobs             []BatchJobRequest             `json:"jobs"`
	MaxDepth         int                           `json:"maxDepth"`
	MaxPages         int                           `json:"maxPages"`
	Headless         bool                          `json:"headless"`
	Playwright       *bool                         `json:"playwright,omitempty"`
	TimeoutSeconds   int                           `json:"timeoutSeconds"`
	AuthProfile      string                        `json:"authProfile,omitempty"`
	Auth             *fetch.AuthOptions            `json:"auth,omitempty"`
	Extract          *extract.ExtractOptions       `json:"extract,omitempty"`
	Pipeline         *pipeline.Options             `json:"pipeline,omitempty"`
	Incremental      *bool                         `json:"incremental,omitempty"`
	SitemapURL       string                        `json:"sitemapURL,omitempty"`
	SitemapOnly      *bool                         `json:"sitemapOnly,omitempty"`
	IncludePatterns  []string                      `json:"includePatterns,omitempty"`
	ExcludePatterns  []string                      `json:"excludePatterns,omitempty"`
	RespectRobotsTxt *bool                         `json:"respectRobotsTxt,omitempty"`
	SkipDuplicates   *bool                         `json:"skipDuplicates,omitempty"`
	SimHashThreshold *int                          `json:"simHashThreshold,omitempty"`
	Webhook          *WebhookConfig                `json:"webhook,omitempty"`
	Screenshot       *fetch.ScreenshotConfig       `json:"screenshot,omitempty"`
	Device           *fetch.DeviceEmulation        `json:"device,omitempty"`
	NetworkIntercept *fetch.NetworkInterceptConfig `json:"networkIntercept,omitempty"`
}

// BatchResearchRequest creates multiple research jobs.
type BatchResearchRequest struct {
	Jobs             []BatchJobRequest             `json:"jobs"`
	Query            string                        `json:"query"`
	MaxDepth         int                           `json:"maxDepth"`
	MaxPages         int                           `json:"maxPages"`
	Headless         bool                          `json:"headless"`
	Playwright       *bool                         `json:"playwright,omitempty"`
	TimeoutSeconds   int                           `json:"timeoutSeconds"`
	AuthProfile      string                        `json:"authProfile,omitempty"`
	Auth             *fetch.AuthOptions            `json:"auth,omitempty"`
	Extract          *extract.ExtractOptions       `json:"extract,omitempty"`
	Pipeline         *pipeline.Options             `json:"pipeline,omitempty"`
	Webhook          *WebhookConfig                `json:"webhook,omitempty"`
	Screenshot       *fetch.ScreenshotConfig       `json:"screenshot,omitempty"`
	Device           *fetch.DeviceEmulation        `json:"device,omitempty"`
	NetworkIntercept *fetch.NetworkInterceptConfig `json:"networkIntercept,omitempty"`
	Agentic          *model.ResearchAgenticConfig  `json:"agentic,omitempty"`
}

// JobResponse represents a single sanitized job envelope.
type JobResponse struct {
	Job model.Job `json:"job"`
}

// JobListResponse represents a paginated collection of sanitized jobs.
type JobListResponse struct {
	Jobs   []model.Job `json:"jobs"`
	Total  int         `json:"total"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
}

// BatchSummary represents aggregate batch metadata and status.
type BatchSummary struct {
	ID        string              `json:"id"`
	Kind      string              `json:"kind"`
	Status    string              `json:"status"`
	JobCount  int                 `json:"jobCount"`
	Stats     model.BatchJobStats `json:"stats"`
	CreatedAt time.Time           `json:"createdAt"`
	UpdatedAt time.Time           `json:"updatedAt"`
}

// BatchResponse represents a stable batch envelope shared by create/get/cancel flows.
type BatchResponse struct {
	Batch  BatchSummary `json:"batch"`
	Jobs   []model.Job  `json:"jobs"`
	Total  int          `json:"total"`
	Limit  int          `json:"limit"`
	Offset int          `json:"offset"`
}

// RetentionStatusResponse represents the retention system status.
type RetentionStatusResponse struct {
	Enabled          bool  `json:"enabled"`
	JobRetentionDays int   `json:"jobRetentionDays"`
	CrawlStateDays   int   `json:"crawlStateDays"`
	MaxJobs          int   `json:"maxJobs"`
	MaxStorageGB     int   `json:"maxStorageGB"`
	TotalJobs        int64 `json:"totalJobs"`
	JobsEligible     int64 `json:"jobsEligible"`
	StorageUsedMB    int64 `json:"storageUsedMB"`
}

// RetentionCleanupRequest represents a request to run retention cleanup.
type RetentionCleanupRequest struct {
	DryRun    bool   `json:"dryRun"`
	Force     bool   `json:"force,omitempty"`
	OlderThan *int   `json:"olderThan,omitempty"` // days
	Kind      string `json:"kind,omitempty"`      // scrape|crawl|research
}

// RetentionCleanupResponse represents the result of a retention cleanup operation.
type RetentionCleanupResponse struct {
	JobsDeleted        int      `json:"jobsDeleted"`
	JobsAttempted      int      `json:"jobsAttempted"`
	CrawlStatesDeleted int64    `json:"crawlStatesDeleted"`
	SpaceReclaimedMB   int64    `json:"spaceReclaimedMB"`
	DurationMs         int64    `json:"durationMs"`
	FailedJobIDs       []string `json:"failedJobIDs,omitempty"`
	Errors             []string `json:"errors,omitempty"`
	DryRun             bool     `json:"dryRun"`
}
