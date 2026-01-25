package scheduler

import (
	"context"
	"strings"
	"testing"
	"time"

	"spartan-scraper/internal/jobs"
	"spartan-scraper/internal/model"
	"spartan-scraper/internal/store"
)

func setupTestManager(t *testing.T) (*jobs.Manager, *store.Store, func()) {
	t.Helper()
	dataDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}

	m := jobs.NewManager(
		st,
		dataDir,
		"TestAgent/1.0",
		30*time.Second,
		2,
		10,
		20,
		3,
		100*time.Millisecond,
		10*1024*1024,
		false,
	)

	cleanup := func() {
		st.Close()
	}

	return m, st, cleanup
}

func TestEnqueueAuthResolutionFailure(t *testing.T) {
	tests := []struct {
		name     string
		schedule Schedule
	}{
		{
			name: "scrape with invalid auth profile",
			schedule: Schedule{
				ID:              "scrape-test-id",
				Kind:            model.KindScrape,
				IntervalSeconds: 60,
				Params: map[string]interface{}{
					"url":         "https://example.com",
					"authProfile": "non-existent-profile",
				},
			},
		},
		{
			name: "crawl with invalid auth profile",
			schedule: Schedule{
				ID:              "crawl-test-id",
				Kind:            model.KindCrawl,
				IntervalSeconds: 60,
				Params: map[string]interface{}{
					"url":         "https://example.com",
					"authProfile": "missing-profile",
				},
			},
		},
		{
			name: "research with invalid auth profile",
			schedule: Schedule{
				ID:              "research-test-id",
				Kind:            model.KindResearch,
				IntervalSeconds: 60,
				Params: map[string]interface{}{
					"query":       "test query",
					"urls":        []string{"https://example.com"},
					"authProfile": "bad-profile",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dataDir := t.TempDir()
			manager, _, cleanup := setupTestManager(t)
			defer cleanup()

			ctx := context.Background()
			err := enqueue(ctx, manager, dataDir, tt.schedule)

			if err == nil {
				t.Errorf("expected error for invalid auth profile, got nil")
			}
			if !strings.Contains(err.Error(), "failed to resolve auth") {
				t.Errorf("error message should mention auth resolution failure: %v", err)
			}
			if !strings.Contains(err.Error(), tt.schedule.ID) {
				t.Errorf("error message should include schedule ID %s: %v", tt.schedule.ID, err)
			}
		})
	}
}

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
				if !strings.Contains(err.Error(), tt.errContains) {
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
