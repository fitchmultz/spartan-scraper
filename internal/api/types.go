// Package api provides HTTP request and response types for the Spartan Scraper API.
// These types are used for JSON encoding/decoding of API requests and responses.
package api

import (
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
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
	Components map[string]ComponentStatus `json:"components"`
}

// ErrorResponse represents a standard error response.
type ErrorResponse struct {
	Error string `json:"error"`
}

// StatusResponse represents a generic success response.
type StatusResponse struct {
	Status string `json:"status"`
}

// ScrapeRequest represents a request to scrape a single page.
type ScrapeRequest struct {
	URL            string                  `json:"url"`
	Headless       bool                    `json:"headless"`
	Playwright     *bool                   `json:"playwright"`
	TimeoutSeconds int                     `json:"timeoutSeconds"`
	AuthProfile    string                  `json:"authProfile,omitempty"`
	Auth           *fetch.AuthOptions      `json:"auth"`
	Extract        *extract.ExtractOptions `json:"extract"`
	Pipeline       *pipeline.Options       `json:"pipeline"`
	Incremental    *bool                   `json:"incremental"`
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
	Incremental    *bool                   `json:"incremental"`
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
}

// ScheduleResponse represents a schedule in the response.
type ScheduleResponse struct {
	ID              string                 `json:"id"`
	Kind            string                 `json:"kind"`
	IntervalSeconds int                    `json:"intervalSeconds"`
	NextRun         string                 `json:"nextRun"`
	Params          map[string]interface{} `json:"params"`
}
