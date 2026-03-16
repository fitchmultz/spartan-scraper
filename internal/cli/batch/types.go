// Package batch provides CLI commands for batch job operations.
//
// Purpose:
// - Reuse the canonical operator-facing batch request and response contracts in the CLI.
//
// Responsibilities:
// - Alias the shared batch request payloads used by API and direct CLI execution.
// - Alias the stable batch response envelopes returned across transports.
// - Keep CLI batch request shapes aligned with the shared submission contract.
//
// Scope:
// - Batch CLI request and response contract aliases only.
//
// Usage:
// - Imported by CLI batch parsing, submission, direct execution, and tests.
//
// Invariants/Assumptions:
// - CLI batch requests should stay wire-compatible with the REST batch API.
// - Batch request-to-spec conversion lives outside this file.
package batch

import (
	spartanapi "github.com/fitchmultz/spartan-scraper/internal/api"
	"github.com/fitchmultz/spartan-scraper/internal/submission"
)

// BatchJobRequest represents a single job in a batch.
type BatchJobRequest = submission.BatchJobRequest

// BatchScrapeRequest represents a batch scrape request.
type BatchScrapeRequest = submission.BatchScrapeRequest

// BatchCrawlRequest represents a batch crawl request.
type BatchCrawlRequest = submission.BatchCrawlRequest

// BatchResearchRequest represents a batch research request.
type BatchResearchRequest = submission.BatchResearchRequest

// BatchSummary aliases the stable aggregate batch metadata used across surfaces.
type BatchSummary = spartanapi.BatchSummary

// BatchResponse represents the stable batch envelope used by API and direct CLI flows.
type BatchResponse = spartanapi.BatchResponse

// BatchStatusResponse aliases the same stable batch envelope for legacy CLI call sites.
type BatchStatusResponse = spartanapi.BatchResponse

// BatchListResponse aliases the stable paginated batch-summary collection envelope.
type BatchListResponse = spartanapi.BatchListResponse
