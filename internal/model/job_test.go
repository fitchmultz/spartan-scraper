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

func TestWebhookConfig_ExtractFromParams(t *testing.T) {
	tests := []struct {
		name     string
		params   map[string]interface{}
		expected *WebhookConfig
	}{
		{
			name:     "no webhook configured",
			params:   map[string]interface{}{"url": "http://example.com"},
			expected: nil,
		},
		{
			name: "webhook with defaults",
			params: map[string]interface{}{
				"webhookURL": "https://example.com/webhook",
			},
			expected: &WebhookConfig{
				URL:    "https://example.com/webhook",
				Events: []string{"completed"},
			},
		},
		{
			name: "webhook with custom events",
			params: map[string]interface{}{
				"webhookURL":    "https://example.com/webhook",
				"webhookEvents": []string{"started", "completed", "failed"},
			},
			expected: &WebhookConfig{
				URL:    "https://example.com/webhook",
				Events: []string{"started", "completed", "failed"},
			},
		},
		{
			name: "webhook with secret",
			params: map[string]interface{}{
				"webhookURL":    "https://example.com/webhook",
				"webhookSecret": "my-secret",
			},
			expected: &WebhookConfig{
				URL:    "https://example.com/webhook",
				Events: []string{"completed"},
				Secret: "my-secret",
			},
		},
		{
			name: "webhook with all options",
			params: map[string]interface{}{
				"webhookURL":    "https://example.com/webhook",
				"webhookEvents": []string{"all"},
				"webhookSecret": "super-secret",
			},
			expected: &WebhookConfig{
				URL:    "https://example.com/webhook",
				Events: []string{"all"},
				Secret: "super-secret",
			},
		},
		{
			name: "webhook events as []interface{}",
			params: map[string]interface{}{
				"webhookURL":    "https://example.com/webhook",
				"webhookEvents": []interface{}{"started", "completed"},
			},
			expected: &WebhookConfig{
				URL:    "https://example.com/webhook",
				Events: []string{"started", "completed"},
			},
		},
		{
			name: "empty webhook URL",
			params: map[string]interface{}{
				"webhookURL": "",
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := Job{Params: tt.params}
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
