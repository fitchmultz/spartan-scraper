// Package webhook provides shared test fixtures and helpers for dispatcher tests.
//
// This file contains common utilities used across multiple dispatcher test files.
// Import this file's package to access shared test helpers.
//
// Does NOT contain actual test cases - only shared infrastructure.
package webhook

import (
	"time"
)

// testPayload returns a standard test payload for webhook dispatch tests.
// Callers can override specific fields as needed.
func testPayload() Payload {
	return Payload{
		EventID:   "evt-123",
		EventType: EventJobCompleted,
		Timestamp: time.Now(),
		JobID:     "job-456",
		JobKind:   "scrape",
		Status:    "succeeded",
	}
}

// testPayloadWithEvent returns a test payload with a specific event type and status.
func testPayloadWithEvent(eventType EventType, status string) Payload {
	return Payload{
		EventID:   "evt-123",
		EventType: eventType,
		Timestamp: time.Now(),
		JobID:     "job-456",
		JobKind:   "scrape",
		Status:    status,
	}
}
