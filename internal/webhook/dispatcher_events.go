// Package webhook defines event types and event-matching rules for outbound webhook delivery.
//
// Purpose:
// - Centralize webhook event type constants and event-matching logic shared across dispatch, API, CLI, and MCP surfaces.
//
// Responsibilities:
// - Define all EventType constants used in webhook payloads.
// - Determine whether a given event type and job status should trigger a webhook delivery.
//
// Scope:
// - Event classification and filtering only; delivery mechanics live in dispatcher.go.
//
// Usage:
// - Call ShouldSendEvent(eventType, status, configuredEvents) before dispatching.
// - Reference EventType constants when building payloads or configuring webhook event filters.
//
// Invariants/Assumptions:
// - An empty configuredEvents slice defaults to ["completed"].
// - The "all" catch-all overrides all other event filters.
package webhook

// EventType represents the type of webhook event.
type EventType string

const (
	EventJobCreated      EventType = "job.created"
	EventJobStarted      EventType = "job.started"
	EventJobCompleted    EventType = "job.completed"
	EventContentChanged  EventType = "content.changed"
	EventPageCrawled     EventType = "page.crawled"
	EventRetryAttempted  EventType = "retry.attempted"
	EventExportCompleted EventType = "export.completed"
	EventVisualChanged   EventType = "visual.changed"
)

// ShouldSendEvent checks if the given event type matches the configured events.
// Supported events: "completed", "failed", "canceled", "started", "created", "succeeded",
// "content_changed", "page_crawled", "retry_attempted", "export_completed", "all".
// An empty configuredEvents slice defaults to ["completed"].
func ShouldSendEvent(eventType EventType, status string, configuredEvents []string) bool {
	if len(configuredEvents) == 0 {
		// Default: only send on terminal states (completed)
		return eventType == EventJobCompleted
	}

	for _, e := range configuredEvents {
		switch e {
		case "all":
			return true
		case "started":
			if eventType == EventJobStarted {
				return true
			}
		case "created":
			if eventType == EventJobCreated {
				return true
			}
		case "completed":
			if eventType == EventJobCompleted {
				return true
			}
		case "failed":
			if eventType == EventJobCompleted && status == "failed" {
				return true
			}
		case "canceled":
			if eventType == EventJobCompleted && status == "canceled" {
				return true
			}
		case "succeeded":
			if eventType == EventJobCompleted && status == "succeeded" {
				return true
			}
		case "content_changed":
			if eventType == EventContentChanged {
				return true
			}
		case "page_crawled":
			if eventType == EventPageCrawled {
				return true
			}
		case "retry_attempted":
			if eventType == EventRetryAttempted {
				return true
			}
		case "export_completed":
			if eventType == EventExportCompleted {
				return true
			}
		case "visual_changed":
			if eventType == EventVisualChanged {
				return true
			}
		}
	}

	return false
}
