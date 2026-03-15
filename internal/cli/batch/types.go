// Package batch provides CLI commands for batch job operations.
//
// This file contains type definitions for batch requests and responses.
//
// Responsibilities:
// - Define request/response structs for batch operations
// - Define shared constants for batch operations
//
// Does NOT handle:
// - Business logic or parsing
// - API client implementations
// - Direct execution logic
package batch

import (
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

const maxBatchSize = 100

// BatchJobRequest represents a single job in a batch.
type BatchJobRequest struct {
	URL         string            `json:"url"`
	Method      string            `json:"method,omitempty"`
	Body        string            `json:"body,omitempty"`
	ContentType string            `json:"contentType,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
}

// BatchScrapeRequest represents a batch scrape request.
type BatchScrapeRequest struct {
	Jobs             []BatchJobRequest             `json:"jobs"`
	Headless         bool                          `json:"headless,omitempty"`
	Playwright       *bool                         `json:"playwright,omitempty"`
	TimeoutSeconds   int                           `json:"timeoutSeconds,omitempty"`
	AuthProfile      string                        `json:"authProfile,omitempty"`
	Auth             *fetch.AuthOptions            `json:"auth,omitempty"`
	Extract          *extract.ExtractOptions       `json:"extract,omitempty"`
	Pipeline         *pipeline.Options             `json:"pipeline,omitempty"`
	Incremental      *bool                         `json:"incremental,omitempty"`
	Webhook          *model.WebhookSpec            `json:"webhook,omitempty"`
	Screenshot       *fetch.ScreenshotConfig       `json:"screenshot,omitempty"`
	Device           *fetch.DeviceEmulation        `json:"device,omitempty"`
	NetworkIntercept *fetch.NetworkInterceptConfig `json:"networkIntercept,omitempty"`
}

// BatchCrawlRequest represents a batch crawl request.
type BatchCrawlRequest struct {
	Jobs             []BatchJobRequest             `json:"jobs"`
	MaxDepth         int                           `json:"maxDepth,omitempty"`
	MaxPages         int                           `json:"maxPages,omitempty"`
	Headless         bool                          `json:"headless,omitempty"`
	Playwright       *bool                         `json:"playwright,omitempty"`
	TimeoutSeconds   int                           `json:"timeoutSeconds,omitempty"`
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
	Webhook          *model.WebhookSpec            `json:"webhook,omitempty"`
	Screenshot       *fetch.ScreenshotConfig       `json:"screenshot,omitempty"`
	Device           *fetch.DeviceEmulation        `json:"device,omitempty"`
	NetworkIntercept *fetch.NetworkInterceptConfig `json:"networkIntercept,omitempty"`
}

// BatchResearchRequest represents a batch research request.
type BatchResearchRequest struct {
	Jobs             []BatchJobRequest             `json:"jobs"`
	Query            string                        `json:"query"`
	MaxDepth         int                           `json:"maxDepth,omitempty"`
	MaxPages         int                           `json:"maxPages,omitempty"`
	Headless         bool                          `json:"headless,omitempty"`
	Playwright       *bool                         `json:"playwright,omitempty"`
	TimeoutSeconds   int                           `json:"timeoutSeconds,omitempty"`
	AuthProfile      string                        `json:"authProfile,omitempty"`
	Auth             *fetch.AuthOptions            `json:"auth,omitempty"`
	Extract          *extract.ExtractOptions       `json:"extract,omitempty"`
	Pipeline         *pipeline.Options             `json:"pipeline,omitempty"`
	Webhook          *model.WebhookSpec            `json:"webhook,omitempty"`
	Screenshot       *fetch.ScreenshotConfig       `json:"screenshot,omitempty"`
	Device           *fetch.DeviceEmulation        `json:"device,omitempty"`
	NetworkIntercept *fetch.NetworkInterceptConfig `json:"networkIntercept,omitempty"`
	Agentic          *model.ResearchAgenticConfig  `json:"agentic,omitempty"`
}

// BatchResponse represents a batch creation response.
type BatchResponse struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Status    string    `json:"status"`
	JobCount  int       `json:"jobCount"`
	Jobs      []JobInfo `json:"jobs,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// BatchStatusResponse represents batch status with statistics.
type BatchStatusResponse struct {
	ID        string              `json:"id"`
	Kind      string              `json:"kind"`
	Status    string              `json:"status"`
	JobCount  int                 `json:"jobCount"`
	Stats     model.BatchJobStats `json:"stats"`
	Jobs      []JobInfo           `json:"jobs,omitempty"`
	CreatedAt time.Time           `json:"createdAt"`
	UpdatedAt time.Time           `json:"updatedAt"`
}

// JobInfo represents a job in the batch response.
type JobInfo struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
