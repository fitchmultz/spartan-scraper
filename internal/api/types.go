// Package api provides HTTP request and response types for the Spartan Scraper API.
//
// Purpose:
// - Define stable request and response contracts shared across REST handlers, MCP adapters, and CLI direct-mode helpers.
//
// Responsibilities:
// - Hold operator-facing request payload types for scrape, crawl, research, schedules, and batches.
// - Hold stable job and batch response envelope types used across transports.
// - Define derived observability payloads for recent runs, queue progression, and failure context.
//
// Scope:
// - JSON contracts only; handler logic and response construction live elsewhere in this package.
//
// Usage:
// - Imported by REST handlers, MCP handlers, CLI direct-mode helpers, tests, and OpenAPI maintenance work.
//
// Invariants/Assumptions:
// - Job and batch automation surfaces should reuse the same response envelope shapes.
// - Response envelopes expose sanitized jobs rather than persisted raw records.
// - Derived run-history fields are transport-safe and never reveal host-local artifact paths.
package api

import (
	"encoding/json"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/submission"
)

const (
	ActionKindRoute        = "route"
	ActionKindCommand      = "command"
	ActionKindEnv          = "env"
	ActionKindCopy         = "copy"
	ActionKindDoc          = "doc"
	ActionKindExternalLink = "external-link"
	ActionKindOneClick     = "one-click"
)

// RecommendedAction describes an operator-facing next step for setup or recovery.
type RecommendedAction struct {
	Label string `json:"label"`
	Kind  string `json:"kind"`
	Value string `json:"value,omitempty"`
}

// RuntimeNotice summarizes a non-fatal setup, config, or runtime issue.
type RuntimeNotice struct {
	ID       string              `json:"id"`
	Scope    string              `json:"scope"`
	Severity string              `json:"severity"`
	Title    string              `json:"title"`
	Message  string              `json:"message"`
	Actions  []RecommendedAction `json:"actions,omitempty"`
}

// SetupStatus describes guided recovery information when the server starts in setup mode.
type SetupStatus struct {
	Required      bool                `json:"required"`
	Code          string              `json:"code,omitempty"`
	Title         string              `json:"title,omitempty"`
	Message       string              `json:"message,omitempty"`
	DataDir       string              `json:"dataDir,omitempty"`
	SchemaVersion string              `json:"schemaVersion,omitempty"`
	Actions       []RecommendedAction `json:"actions,omitempty"`
}

// ComponentStatus represents the health of a single system component.
type ComponentStatus struct {
	Status  string              `json:"status"`
	Message string              `json:"message,omitempty"`
	Details interface{}         `json:"details,omitempty"`
	Actions []RecommendedAction `json:"actions,omitempty"`
}

// DiagnosticActionResponse represents the result of a safe read-only diagnostic action.
type DiagnosticActionResponse struct {
	Status  string              `json:"status"`
	Title   string              `json:"title,omitempty"`
	Message string              `json:"message"`
	Details interface{}         `json:"details,omitempty"`
	Actions []RecommendedAction `json:"actions,omitempty"`
}

// HealthResponse represents the overall health of the system.
type HealthResponse struct {
	Status     string                     `json:"status"`
	Version    string                     `json:"version"`
	Components map[string]ComponentStatus `json:"components"`
	Notices    []RuntimeNotice            `json:"notices,omitempty"`
	Setup      *SetupStatus               `json:"setup,omitempty"`
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
type BatchJobRequest = submission.BatchJobRequest

// BatchScrapeRequest creates multiple scrape jobs.
type BatchScrapeRequest = submission.BatchScrapeRequest

// BatchCrawlRequest creates multiple crawl jobs.
type BatchCrawlRequest = submission.BatchCrawlRequest

// BatchResearchRequest creates multiple research jobs.
type BatchResearchRequest = submission.BatchResearchRequest

// JobFailureContext summarizes operator-meaningful terminal failure details.
type JobFailureContext struct {
	Category  string `json:"category"`
	Summary   string `json:"summary"`
	Retryable bool   `json:"retryable"`
	Terminal  bool   `json:"terminal"`
}

// JobQueueProgress summarizes a job's position within a persisted batch queue.
type JobQueueProgress struct {
	BatchID   string `json:"batchId,omitempty"`
	Index     int    `json:"index,omitempty"`
	Total     int    `json:"total,omitempty"`
	Completed int    `json:"completed,omitempty"`
	Remaining int    `json:"remaining,omitempty"`
	Queued    int    `json:"queued,omitempty"`
	Running   int    `json:"running,omitempty"`
	Percent   int    `json:"percent,omitempty"`
}

// JobRunSummary exposes derived lifecycle timing, queue, and failure details for a job.
type JobRunSummary struct {
	WaitMs  int64              `json:"waitMs"`
	RunMs   int64              `json:"runMs"`
	TotalMs int64              `json:"totalMs"`
	Queue   *JobQueueProgress  `json:"queue,omitempty"`
	Failure *JobFailureContext `json:"failure,omitempty"`
}

// InspectableJob represents a sanitized job with derived observability fields.
type InspectableJob struct {
	model.Job
	Run JobRunSummary `json:"run"`
}

// JobResponse represents a single sanitized inspectable job envelope.
type JobResponse struct {
	Job InspectableJob `json:"job"`
}

// JobListResponse represents a paginated collection of sanitized inspectable jobs.
type JobListResponse struct {
	Jobs   []InspectableJob `json:"jobs"`
	Total  int              `json:"total"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
}

// BatchProgress exposes explicit queue progress for a batch.
type BatchProgress struct {
	Completed int `json:"completed"`
	Remaining int `json:"remaining"`
	Percent   int `json:"percent"`
}

// BatchSummary represents aggregate batch metadata and status.
type BatchSummary struct {
	ID        string              `json:"id"`
	Kind      string              `json:"kind"`
	Status    string              `json:"status"`
	JobCount  int                 `json:"jobCount"`
	Stats     model.BatchJobStats `json:"stats"`
	Progress  BatchProgress       `json:"progress"`
	CreatedAt time.Time           `json:"createdAt"`
	UpdatedAt time.Time           `json:"updatedAt"`
}

// BatchResponse represents a stable batch envelope shared by create/get/cancel flows.
type BatchResponse struct {
	Batch  BatchSummary     `json:"batch"`
	Jobs   []InspectableJob `json:"jobs"`
	Total  int              `json:"total"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
}

// BatchListResponse represents a paginated collection of aggregate batch summaries.
type BatchListResponse struct {
	Batches []BatchSummary `json:"batches"`
	Total   int            `json:"total"`
	Limit   int            `json:"limit"`
	Offset  int            `json:"offset"`
}

// CapabilityGuidance describes a capability-aware explanation and follow-up actions.
type CapabilityGuidance struct {
	Status  string              `json:"status"`
	Title   string              `json:"title,omitempty"`
	Message string              `json:"message,omitempty"`
	Actions []RecommendedAction `json:"actions,omitempty"`
}

// RetentionStatusResponse represents the retention system status.
type RetentionStatusResponse struct {
	Enabled          bool                `json:"enabled"`
	JobRetentionDays int                 `json:"jobRetentionDays"`
	CrawlStateDays   int                 `json:"crawlStateDays"`
	MaxJobs          int                 `json:"maxJobs"`
	MaxStorageGB     int                 `json:"maxStorageGB"`
	TotalJobs        int64               `json:"totalJobs"`
	JobsEligible     int64               `json:"jobsEligible"`
	StorageUsedMB    int64               `json:"storageUsedMB"`
	Guidance         *CapabilityGuidance `json:"guidance,omitempty"`
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
