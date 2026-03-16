// Package submission defines the canonical operator-facing batch request contract.
//
// Purpose:
//   - Hold the shared batch scrape, crawl, and research request shapes used across REST,
//     direct CLI batch flows, and any other operator-facing batch submission surface.
//
// Responsibilities:
// - Define stable batch payloads and the per-item request shape they reuse.
// - Reuse the same shared webhook and execution option shapes as single-job requests.
// - Keep batch transports aligned on one request contract.
//
// Scope:
// - Batch request payload types only. Validation and conversion live in batch_requests.go.
//
// Usage:
// - Imported by API handlers, CLI batch commands, tests, and future batch-capable transports.
//
// Invariants/Assumptions:
// - Batch item URLs are validated before jobs are created.
// - Research batches collapse submitted URLs into a single research job.
package submission

import (
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

// BatchJobRequest represents a single operator-supplied job within a batch.
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
