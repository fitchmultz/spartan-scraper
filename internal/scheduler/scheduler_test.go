package scheduler

import (
	"testing"

	"spartan-scraper/internal/model"
)

func TestSchedulerStorage(t *testing.T) {
	dataDir := t.TempDir()

	schedules, err := LoadAll(dataDir)
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	if len(schedules) != 0 {
		t.Errorf("expected 0 schedules, got %d", len(schedules))
	}

	s1 := Schedule{
		Kind:            model.KindScrape,
		IntervalSeconds: 60,
		Params:          map[string]interface{}{"url": "http://example.com"},
	}

	if err := Add(dataDir, s1); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	list, _ := List(dataDir)
	if len(list) != 1 {
		t.Errorf("expected 1 schedule, got %d", len(list))
	}
	if list[0].Kind != model.KindScrape {
		t.Errorf("expected kind scrape, got %v", list[0].Kind)
	}

	id := list[0].ID
	if err := Delete(dataDir, id); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	list, _ = List(dataDir)
	if len(list) != 0 {
		t.Errorf("expected 0 schedules after delete, got %d", len(list))
	}
}

func TestScheduleValidation(t *testing.T) {
	tests := []struct {
		name        string
		schedule    Schedule
		wantErr     bool
		errContains string
	}{
		{
			name: "valid scrape schedule",
			schedule: Schedule{
				Kind:            model.KindScrape,
				IntervalSeconds: 60,
				Params: map[string]interface{}{
					"url":     "https://example.com",
					"timeout": 30,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid scrape schedule - invalid URL",
			schedule: Schedule{
				Kind:            model.KindScrape,
				IntervalSeconds: 60,
				Params: map[string]interface{}{
					"url": "ftp://example.com",
				},
			},
			wantErr:     true,
			errContains: "invalid scrape schedule",
		},
		{
			name: "invalid scrape schedule - timeout too low",
			schedule: Schedule{
				Kind:            model.KindScrape,
				IntervalSeconds: 60,
				Params: map[string]interface{}{
					"url":     "https://example.com",
					"timeout": 4,
				},
			},
			wantErr:     true,
			errContains: "invalid scrape schedule",
		},
		{
			name: "valid crawl schedule",
			schedule: Schedule{
				Kind:            model.KindCrawl,
				IntervalSeconds: 60,
				Params: map[string]interface{}{
					"url":      "https://example.com",
					"maxDepth": 3,
					"maxPages": 100,
					"timeout":  30,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid crawl schedule - maxDepth too high",
			schedule: Schedule{
				Kind:            model.KindCrawl,
				IntervalSeconds: 60,
				Params: map[string]interface{}{
					"url":      "https://example.com",
					"maxDepth": 11,
					"maxPages": 100,
				},
			},
			wantErr:     true,
			errContains: "invalid crawl schedule",
		},
		{
			name: "invalid crawl schedule - maxPages too high",
			schedule: Schedule{
				Kind:            model.KindCrawl,
				IntervalSeconds: 60,
				Params: map[string]interface{}{
					"url":      "https://example.com",
					"maxDepth": 3,
					"maxPages": 10001,
				},
			},
			wantErr:     true,
			errContains: "invalid crawl schedule",
		},
		{
			name: "valid research schedule",
			schedule: Schedule{
				Kind:            model.KindResearch,
				IntervalSeconds: 60,
				Params: map[string]interface{}{
					"query": "test query",
					"urls":  []string{"https://example.com", "https://example.org"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid research schedule - empty query",
			schedule: Schedule{
				Kind:            model.KindResearch,
				IntervalSeconds: 60,
				Params: map[string]interface{}{
					"query": "",
					"urls":  []string{"https://example.com"},
				},
			},
			wantErr:     true,
			errContains: "invalid research schedule",
		},
		{
			name: "invalid research schedule - invalid URL in list",
			schedule: Schedule{
				Kind:            model.KindResearch,
				IntervalSeconds: 60,
				Params: map[string]interface{}{
					"query": "test query",
					"urls":  []string{"https://example.com", "ftp://example.org"},
				},
			},
			wantErr:     true,
			errContains: "invalid research schedule",
		},
		{
			name: "unknown schedule kind",
			schedule: Schedule{
				Kind:            "unknown",
				IntervalSeconds: 60,
				Params:          map[string]interface{}{},
			},
			wantErr:     true,
			errContains: "unknown schedule kind",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDataDir := t.TempDir()
			err := Add(testDataDir, tt.schedule)
			if (err != nil) != tt.wantErr {
				t.Errorf("Add() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errContains != "" {
				if !containsTestString(err.Error(), tt.errContains) {
					t.Errorf("Error = %v, want containing %v", err.Error(), tt.errContains)
				}
			}

			if tt.wantErr {
				list, _ := List(testDataDir)
				for _, s := range list {
					if s.Kind == tt.schedule.Kind {
						t.Errorf("Invalid schedule should not have been added")
					}
				}
			}
		})
	}
}

func containsTestString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findTestSubstring(s, substr))
}

func findTestSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
