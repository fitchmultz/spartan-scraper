// Package api provides HTTP request and response types for the Spartan Scraper API.
// These types are used for JSON encoding/decoding of API requests and responses.
package api

import (
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
	URL            string                  `json:"url"`
	Method         string                  `json:"method,omitempty"`      // HTTP method, default GET
	Body           string                  `json:"body,omitempty"`        // Request body (base64 for binary data)
	ContentType    string                  `json:"contentType,omitempty"` // Content-Type header for request body
	Headless       bool                    `json:"headless"`
	Playwright     *bool                   `json:"playwright"`
	TimeoutSeconds int                     `json:"timeoutSeconds"`
	AuthProfile    string                  `json:"authProfile,omitempty"`
	Auth           *fetch.AuthOptions      `json:"auth"`
	Extract        *extract.ExtractOptions `json:"extract"`
	Pipeline       *pipeline.Options       `json:"pipeline"`
	Incremental    *bool                   `json:"incremental"`
	Webhook        *WebhookConfig          `json:"webhook,omitempty"`
	Screenshot     *fetch.ScreenshotConfig `json:"screenshot,omitempty"`
	Device         *fetch.DeviceEmulation  `json:"device,omitempty"`
}

// CrawlRequest represents a request to crawl a website.
type CrawlRequest struct {
	URL            string                  `json:"url"`
	MaxDepth       int                     `json:"maxDepth"`
	MaxPages       int                     `json:"maxPages"`
	Headless       bool                    `json:"headless"`
	Playwright     *bool                   `json:"playwright"`
	TimeoutSeconds int                     `json:"timeoutSeconds"`
	AuthProfile    string                  `json:"authProfile,omitempty"`
	Auth           *fetch.AuthOptions      `json:"auth"`
	Extract        *extract.ExtractOptions `json:"extract"`
	Pipeline       *pipeline.Options       `json:"pipeline"`
	Incremental    *bool                   `json:"incremental"`
	SitemapURL     string                  `json:"sitemapURL,omitempty"`
	SitemapOnly    *bool                   `json:"sitemapOnly,omitempty"`
	Webhook        *WebhookConfig          `json:"webhook,omitempty"`
	Screenshot     *fetch.ScreenshotConfig `json:"screenshot,omitempty"`
	Device         *fetch.DeviceEmulation  `json:"device,omitempty"`
}

// ResearchRequest represents a request to perform deep research across multiple URLs.
type ResearchRequest struct {
	Query          string                  `json:"query"`
	URLs           []string                `json:"urls"`
	MaxDepth       int                     `json:"maxDepth"`
	MaxPages       int                     `json:"maxPages"`
	Headless       bool                    `json:"headless"`
	Playwright     *bool                   `json:"playwright"`
	TimeoutSeconds int                     `json:"timeoutSeconds"`
	AuthProfile    string                  `json:"authProfile,omitempty"`
	Auth           *fetch.AuthOptions      `json:"auth"`
	Extract        *extract.ExtractOptions `json:"extract"`
	Pipeline       *pipeline.Options       `json:"pipeline"`
	Webhook        *WebhookConfig          `json:"webhook,omitempty"`
	Screenshot     *fetch.ScreenshotConfig `json:"screenshot,omitempty"`
	Device         *fetch.DeviceEmulation  `json:"device,omitempty"`
}

// ScheduleRequest represents a request to add a scheduled job.
type ScheduleRequest struct {
	Kind            string                  `json:"kind"`
	IntervalSeconds int                     `json:"intervalSeconds"`
	URL             *string                 `json:"url,omitempty"`
	Query           *string                 `json:"query,omitempty"`
	URLs            []string                `json:"urls,omitempty"`
	MaxDepth        *int                    `json:"maxDepth,omitempty"`
	MaxPages        *int                    `json:"maxPages,omitempty"`
	Headless        bool                    `json:"headless"`
	Playwright      *bool                   `json:"playwright"`
	TimeoutSeconds  int                     `json:"timeoutSeconds"`
	AuthProfile     *string                 `json:"authProfile,omitempty"`
	Auth            *fetch.AuthOptions      `json:"auth"`
	Extract         *extract.ExtractOptions `json:"extract"`
	Pipeline        *pipeline.Options       `json:"pipeline"`
	Incremental     *bool                   `json:"incremental"`
	SitemapURL      *string                 `json:"sitemapURL,omitempty"`
	SitemapOnly     *bool                   `json:"sitemapOnly,omitempty"`
	Screenshot      *fetch.ScreenshotConfig `json:"screenshot,omitempty"`
	Device          *fetch.DeviceEmulation  `json:"device,omitempty"`
}

// ScheduleResponse represents a schedule in the response.
type ScheduleResponse struct {
	ID              string                 `json:"id"`
	Kind            string                 `json:"kind"`
	IntervalSeconds int                    `json:"intervalSeconds"`
	NextRun         string                 `json:"nextRun"`
	Params          map[string]interface{} `json:"params"`
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
	Jobs           []BatchJobRequest       `json:"jobs"`
	Query          string                  `json:"query"`
	MaxDepth       int                     `json:"maxDepth"`
	MaxPages       int                     `json:"maxPages"`
	Headless       bool                    `json:"headless"`
	Playwright     *bool                   `json:"playwright,omitempty"`
	TimeoutSeconds int                     `json:"timeoutSeconds"`
	AuthProfile    string                  `json:"authProfile,omitempty"`
	Auth           *fetch.AuthOptions      `json:"auth,omitempty"`
	Extract        *extract.ExtractOptions `json:"extract,omitempty"`
	Pipeline       *pipeline.Options       `json:"pipeline,omitempty"`
	Webhook        *WebhookConfig          `json:"webhook,omitempty"`
	Screenshot     *fetch.ScreenshotConfig `json:"screenshot,omitempty"`
	Device         *fetch.DeviceEmulation  `json:"device,omitempty"`
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

// TrafficReplayRequest configures replay of captured traffic.
type TrafficReplayRequest struct {
	JobID            string                `json:"jobId"`
	TargetBaseURL    string                `json:"targetBaseUrl"`
	Filter           *TrafficReplayFilter  `json:"filter,omitempty"`
	Modifications    *TrafficModifications `json:"modifications,omitempty"`
	CompareResponses bool                  `json:"compareResponses"`
	Timeout          int                   `json:"timeout,omitempty"`
}

// TrafficReplayFilter defines which requests to replay.
type TrafficReplayFilter struct {
	URLPatterns   []string `json:"urlPatterns,omitempty"`
	Methods       []string `json:"methods,omitempty"`
	ResourceTypes []string `json:"resourceTypes,omitempty"`
	StatusCodes   []int    `json:"statusCodes,omitempty"`
}

// TrafficModifications defines request modifications for replay.
type TrafficModifications struct {
	Headers       map[string]string `json:"headers,omitempty"`
	RemoveHeaders []string          `json:"removeHeaders,omitempty"`
	BodyTransform string            `json:"bodyTransform,omitempty"`
}

// TrafficReplayResponse contains replay results.
type TrafficReplayResponse struct {
	JobID         string            `json:"jobId"`
	TotalRequests int               `json:"totalRequests"`
	Successful    int               `json:"successful"`
	Failed        int               `json:"failed"`
	Results       []ReplayResult    `json:"results"`
	Comparison    *ReplayComparison `json:"comparison,omitempty"`
	Duration      int               `json:"durationMs"`
}

// ReplayResult represents a single replayed request result.
type ReplayResult struct {
	OriginalRequest  ReplayRequestInfo  `json:"originalRequest"`
	ReplayedRequest  ReplayRequestInfo  `json:"replayedRequest"`
	ReplayedResponse ReplayResponseInfo `json:"replayedResponse"`
	Error            string             `json:"error,omitempty"`
	Duration         int                `json:"durationMs"`
}

// ReplayRequestInfo represents request details in replay.
type ReplayRequestInfo struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body,omitempty"`
}

// ReplayResponseInfo represents response details in replay.
type ReplayResponseInfo struct {
	Status     int               `json:"status"`
	StatusText string            `json:"statusText"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body,omitempty"`
	BodySize   int               `json:"bodySize"`
}

// ReplayComparison contains response comparison results.
type ReplayComparison struct {
	TotalCompared int            `json:"totalCompared"`
	Matches       int            `json:"matches"`
	Mismatches    int            `json:"mismatches"`
	Differences   []ResponseDiff `json:"differences"`
}

// ResponseDiff represents a single response difference.
type ResponseDiff struct {
	RequestID   string       `json:"requestId"`
	URL         string       `json:"url"`
	StatusDiff  *StatusDiff  `json:"statusDiff,omitempty"`
	HeaderDiffs []HeaderDiff `json:"headerDiffs,omitempty"`
	BodyDiff    *BodyDiff    `json:"bodyDiff,omitempty"`
}

// StatusDiff represents status code difference.
type StatusDiff struct {
	Original int `json:"original"`
	Replayed int `json:"replayed"`
}

// HeaderDiff represents header difference.
type HeaderDiff struct {
	Name     string `json:"name"`
	Original string `json:"original,omitempty"`
	Replayed string `json:"replayed,omitempty"`
}

// BodyDiff represents body difference.
type BodyDiff struct {
	OriginalSize int    `json:"originalSize"`
	ReplayedSize int    `json:"replayedSize"`
	Preview      string `json:"preview,omitempty"`
}
