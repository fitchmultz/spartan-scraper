// Package submission defines the canonical operator-facing job request contract.
//
// Purpose:
//   - Hold the shared scrape, crawl, and research request shapes used across REST,
//     schedules, chains, watches, CLI, MCP, and other operator-facing submission flows.
//
// Responsibilities:
// - Define stable request payloads for single-job submissions.
// - Provide the webhook request shape reused by those payloads.
// - Keep automation surfaces on the same contract as live job submissions.
//
// Scope:
// - Request payload types only. Validation and conversion live in requests.go.
//
// Usage:
// - Imported by API, MCP, scheduler, watch automation, and chain automation code.
//
// Invariants/Assumptions:
// - These request types describe operator input, not persisted typed specs.
// - The payloads intentionally mirror the public REST contract.
package submission

import (
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

// WebhookConfig represents webhook configuration for job notifications.
type WebhookConfig struct {
	URL    string   `json:"url,omitempty"`
	Events []string `json:"events,omitempty"`
	Secret string   `json:"secret,omitempty"`
}

// ScrapeRequest represents an operator-facing scrape job request.
type ScrapeRequest struct {
	URL              string                        `json:"url"`
	Method           string                        `json:"method,omitempty"`
	Body             string                        `json:"body,omitempty"`
	ContentType      string                        `json:"contentType,omitempty"`
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

// CrawlRequest represents an operator-facing crawl job request.
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

// ResearchRequest represents an operator-facing research job request.
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
