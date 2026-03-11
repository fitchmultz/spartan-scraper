// Package model provides tests for domain model behaviors and constants.
//
// Tests cover:
// - Status state machine methods (IsTerminal, IsValid)
// - Valid status enumeration
// - Job kind constants
// - Webhook configuration extraction from typed job specs
//
// Does NOT test:
// - Job sanitization (see job_sanitize_test.go)
// - Crawl state serialization (see state_test.go)
// - Database persistence
//
// Assumes:
// - Status constants are defined in the model package
// - Webhook config extraction follows documented parameter naming conventions
package model

import (
	"reflect"
	"testing"
)

func TestStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		name     string
		status   Status
		expected bool
	}{
		{"StatusSucceeded", StatusSucceeded, true},
		{"StatusFailed", StatusFailed, true},
		{"StatusCanceled", StatusCanceled, true},
		{"StatusQueued", StatusQueued, false},
		{"StatusRunning", StatusRunning, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.status.IsTerminal()
			if got != tt.expected {
				t.Errorf("IsTerminal() = %v; want %v", got, tt.expected)
			}
		})
	}
}

func TestStatus_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		status   Status
		expected bool
	}{
		{"StatusQueued", StatusQueued, true},
		{"StatusRunning", StatusRunning, true},
		{"StatusSucceeded", StatusSucceeded, true},
		{"StatusFailed", StatusFailed, true},
		{"StatusCanceled", StatusCanceled, true},
		{"Invalid status", Status(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.status.IsValid()
			if got != tt.expected {
				t.Errorf("IsValid() = %v; want %v", got, tt.expected)
			}
		})
	}
}

func TestValidStatuses(t *testing.T) {
	statuses := ValidStatuses()

	expectedLength := 5
	if len(statuses) != expectedLength {
		t.Fatalf("ValidStatuses() returned %d statuses; want %d", len(statuses), expectedLength)
	}

	expected := []Status{StatusQueued, StatusRunning, StatusSucceeded, StatusFailed, StatusCanceled}
	for i, want := range expected {
		if statuses[i] != want {
			t.Errorf("ValidStatuses()[%d] = %s; want %s", i, statuses[i], want)
		}
	}
}

func TestKind_Constants(t *testing.T) {
	tests := []struct {
		name     string
		kind     Kind
		expected string
	}{
		{"KindScrape", KindScrape, "scrape"},
		{"KindCrawl", KindCrawl, "crawl"},
		{"KindResearch", KindResearch, "research"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(tt.kind)
			if got != tt.expected {
				t.Errorf("Kind value = %s; want %s", got, tt.expected)
			}
		})
	}
}

func TestWebhookConfig_ExtractFromSpec(t *testing.T) {
	tests := []struct {
		name     string
		spec     any
		expected *WebhookSpec
	}{
		{
			name:     "no webhook configured",
			spec:     ScrapeSpecV1{Version: JobSpecVersion1, URL: "http://example.com"},
			expected: nil,
		},
		{
			name: "webhook with defaults",
			spec: ScrapeSpecV1{
				Version: JobSpecVersion1,
				URL:     "http://example.com",
				Execution: ExecutionSpec{
					Webhook: &WebhookSpec{URL: "https://example.com/webhook", Events: []string{"completed"}},
				},
			},
			expected: &WebhookSpec{
				URL:    "https://example.com/webhook",
				Events: []string{"completed"},
			},
		},
		{
			name: "webhook with custom events",
			spec: ScrapeSpecV1{
				Version: JobSpecVersion1,
				URL:     "http://example.com",
				Execution: ExecutionSpec{
					Webhook: &WebhookSpec{URL: "https://example.com/webhook", Events: []string{"started", "completed", "failed"}},
				},
			},
			expected: &WebhookSpec{
				URL:    "https://example.com/webhook",
				Events: []string{"started", "completed", "failed"},
			},
		},
		{
			name: "webhook with secret",
			spec: ScrapeSpecV1{
				Version: JobSpecVersion1,
				URL:     "http://example.com",
				Execution: ExecutionSpec{
					Webhook: &WebhookSpec{URL: "https://example.com/webhook", Events: []string{"completed"}, Secret: "my-secret"},
				},
			},
			expected: &WebhookSpec{
				URL:    "https://example.com/webhook",
				Events: []string{"completed"},
				Secret: "my-secret",
			},
		},
		{
			name: "webhook with all options",
			spec: ScrapeSpecV1{
				Version: JobSpecVersion1,
				URL:     "http://example.com",
				Execution: ExecutionSpec{
					Webhook: &WebhookSpec{URL: "https://example.com/webhook", Events: []string{"all"}, Secret: "super-secret"},
				},
			},
			expected: &WebhookSpec{
				URL:    "https://example.com/webhook",
				Events: []string{"all"},
				Secret: "super-secret",
			},
		},
		{
			name:     "empty webhook URL",
			spec:     ScrapeSpecV1{Version: JobSpecVersion1, URL: "http://example.com", Execution: ExecutionSpec{Webhook: &WebhookSpec{URL: ""}}},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := Job{SpecVersion: JobSpecVersion1, Spec: tt.spec}
			got := job.ExtractWebhookConfig()

			if tt.expected == nil {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
				return
			}

			if got == nil {
				t.Fatalf("expected %+v, got nil", tt.expected)
			}

			if got.URL != tt.expected.URL {
				t.Errorf("URL: expected %q, got %q", tt.expected.URL, got.URL)
			}
			if got.Secret != tt.expected.Secret {
				t.Errorf("Secret: expected %q, got %q", tt.expected.Secret, got.Secret)
			}
			if !reflect.DeepEqual(got.Events, tt.expected.Events) {
				t.Errorf("Events: expected %v, got %v", tt.expected.Events, got.Events)
			}
		})
	}
}
