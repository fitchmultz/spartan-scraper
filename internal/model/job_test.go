package model

import "testing"

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
