// Package api provides HTTP request and response types for the Spartan Scraper API.
// These types are used for JSON encoding/decoding of API requests and responses.
package api

import (
	"encoding/json"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
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
type WebhookConfig struct {
	URL    string   `json:"url,omitempty"`
	Events []string `json:"events,omitempty"`
	Secret string   `json:"secret,omitempty"`
}

// ScrapeRequest represents a request to scrape a single page.
type ScrapeRequest struct {
	URL              string                        `json:"url"`
	Method           string                        `json:"method,omitempty"`      // HTTP method, default GET
	Body             string                        `json:"body,omitempty"`        // Request body (base64 for binary data)
	ContentType      string                        `json:"contentType,omitempty"` // Content-Type header for request body
	Headless         bool                          `json:"headless"`
	Playwright       *bool                         `json:"playwright"`
	TimeoutSeconds   int                           `json:"timeoutSeconds"`
	AuthProfile      string                        `json:"authProfile,omitempty"`
	Auth             *fetch.AuthOptions            `json:"auth"`
	Extract          *extract.ExtractOptions       `json:"extract"`
	Pipeline         *pipeline.Options             `json:"pipeline"`
	Incremental      *bool                         `json:"incremental"`
	Webhook          *WebhookConfig                `json:"webhook,omitempty"`
	Screenshot       *fetch.ScreenshotConfig       `json:"screenshot,omitempty"`
	Device           *fetch.DeviceEmulation        `json:"device,omitempty"`
	NetworkIntercept *fetch.NetworkInterceptConfig `json:"networkIntercept,omitempty"`
}

// CrawlRequest represents a request to crawl a website.
type CrawlRequest struct {
	URL              string                        `json:"url"`
	MaxDepth         int                           `json:"maxDepth"`
	MaxPages         int                           `json:"maxPages"`
	Headless         bool                          `json:"headless"`
	Playwright       *bool                         `json:"playwright"`
	TimeoutSeconds   int                           `json:"timeoutSeconds"`
	AuthProfile      string                        `json:"authProfile,omitempty"`
	Auth             *fetch.AuthOptions            `json:"auth"`
	Extract          *extract.ExtractOptions       `json:"extract"`
	Pipeline         *pipeline.Options             `json:"pipeline"`
	Incremental      *bool                         `json:"incremental"`
	SitemapURL       string                        `json:"sitemapURL,omitempty"`
	SitemapOnly      *bool                         `json:"sitemapOnly,omitempty"`
	Webhook          *WebhookConfig                `json:"webhook,omitempty"`
	Screenshot       *fetch.ScreenshotConfig       `json:"screenshot,omitempty"`
	Device           *fetch.DeviceEmulation        `json:"device,omitempty"`
	NetworkIntercept *fetch.NetworkInterceptConfig `json:"networkIntercept,omitempty"`
}

// ResearchRequest represents a request to perform deep research across multiple URLs.
type ResearchRequest struct {
	Query            string                        `json:"query"`
	URLs             []string                      `json:"urls"`
	MaxDepth         int                           `json:"maxDepth"`
	MaxPages         int                           `json:"maxPages"`
	Headless         bool                          `json:"headless"`
	Playwright       *bool                         `json:"playwright"`
	TimeoutSeconds   int                           `json:"timeoutSeconds"`
	AuthProfile      string                        `json:"authProfile,omitempty"`
	Auth             *fetch.AuthOptions            `json:"auth"`
	Extract          *extract.ExtractOptions       `json:"extract"`
	Pipeline         *pipeline.Options             `json:"pipeline"`
	Webhook          *WebhookConfig                `json:"webhook,omitempty"`
	Screenshot       *fetch.ScreenshotConfig       `json:"screenshot,omitempty"`
	Device           *fetch.DeviceEmulation        `json:"device,omitempty"`
	NetworkIntercept *fetch.NetworkInterceptConfig `json:"networkIntercept,omitempty"`
	Agentic          *model.ResearchAgenticConfig  `json:"agentic,omitempty"`
}

// ScheduleRequest represents a request to add a scheduled job.
type ScheduleRequest struct {
	Kind            string          `json:"kind"`
	IntervalSeconds int             `json:"intervalSeconds"`
	SpecVersion     int             `json:"specVersion"`
	Spec            json.RawMessage `json:"spec"`
}

// ScheduleResponse represents a schedule in the response.
type ScheduleResponse struct {
	ID              string `json:"id"`
	Kind            string `json:"kind"`
	IntervalSeconds int    `json:"intervalSeconds"`
	NextRun         string `json:"nextRun"`
	SpecVersion     int    `json:"specVersion"`
	Spec            any    `json:"spec"`
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
	Jobs           []BatchJobRequest       `json:"jobs"`
	OutputFormat   string                  `json:"outputFormat,omitempty"`
	ExtractionName string                  `json:"extractionName,omitempty"`
	ExtractionMode string                  `json:"extractionMode,omitempty"`
	Headless       bool                    `json:"headless"`
	Playwright     *bool                   `json:"playwright,omitempty"`
	TimeoutSeconds int                     `json:"timeoutSeconds"`
	AuthProfile    string                  `json:"authProfile,omitempty"`
	Auth           *fetch.AuthOptions      `json:"auth,omitempty"`
	Extract        *extract.ExtractOptions `json:"extract,omitempty"`
	Pipeline       *pipeline.Options       `json:"pipeline,omitempty"`
	Incremental    *bool                   `json:"incremental,omitempty"`
	Webhook        *WebhookConfig          `json:"webhook,omitempty"`
	Screenshot     *fetch.ScreenshotConfig `json:"screenshot,omitempty"`
	Device         *fetch.DeviceEmulation  `json:"device,omitempty"`
}

// BatchCrawlRequest creates multiple crawl jobs.
type BatchCrawlRequest struct {
	Jobs           []BatchJobRequest       `json:"jobs"`
	MaxDepth       int                     `json:"maxDepth"`
	MaxPages       int                     `json:"maxPages"`
	Headless       bool                    `json:"headless"`
	Playwright     *bool                   `json:"playwright,omitempty"`
	TimeoutSeconds int                     `json:"timeoutSeconds"`
	AuthProfile    string                  `json:"authProfile,omitempty"`
	Auth           *fetch.AuthOptions      `json:"auth,omitempty"`
	Extract        *extract.ExtractOptions `json:"extract,omitempty"`
	Pipeline       *pipeline.Options       `json:"pipeline,omitempty"`
	Incremental    *bool                   `json:"incremental,omitempty"`
	SitemapURL     string                  `json:"sitemapURL,omitempty"`
	SitemapOnly    *bool                   `json:"sitemapOnly,omitempty"`
	Webhook        *WebhookConfig          `json:"webhook,omitempty"`
	Screenshot     *fetch.ScreenshotConfig `json:"screenshot,omitempty"`
	Device         *fetch.DeviceEmulation  `json:"device,omitempty"`
}

// BatchResearchRequest creates multiple research jobs.
type BatchResearchRequest struct {
	Jobs           []BatchJobRequest            `json:"jobs"`
	Query          string                       `json:"query"`
	MaxDepth       int                          `json:"maxDepth"`
	MaxPages       int                          `json:"maxPages"`
	Headless       bool                         `json:"headless"`
	Playwright     *bool                        `json:"playwright,omitempty"`
	TimeoutSeconds int                          `json:"timeoutSeconds"`
	AuthProfile    string                       `json:"authProfile,omitempty"`
	Auth           *fetch.AuthOptions           `json:"auth,omitempty"`
	Extract        *extract.ExtractOptions      `json:"extract,omitempty"`
	Pipeline       *pipeline.Options            `json:"pipeline,omitempty"`
	Webhook        *WebhookConfig               `json:"webhook,omitempty"`
	Screenshot     *fetch.ScreenshotConfig      `json:"screenshot,omitempty"`
	Device         *fetch.DeviceEmulation       `json:"device,omitempty"`
	Agentic        *model.ResearchAgenticConfig `json:"agentic,omitempty"`
}

// BatchResponse represents a created batch.
type BatchResponse struct {
	ID        string      `json:"id"`
	Kind      string      `json:"kind"`
	Status    string      `json:"status"`
	JobCount  int         `json:"jobCount"`
	Jobs      []model.Job `json:"jobs"`
	CreatedAt time.Time   `json:"createdAt"`
}

// BatchStatusResponse represents batch status with aggregated stats.
type BatchStatusResponse struct {
	ID        string              `json:"id"`
	Kind      string              `json:"kind"`
	Status    string              `json:"status"`
	JobCount  int                 `json:"jobCount"`
	Stats     model.BatchJobStats `json:"stats"`
	Jobs      []model.Job         `json:"jobs,omitempty"`
	CreatedAt time.Time           `json:"createdAt"`
	UpdatedAt time.Time           `json:"updatedAt"`
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
