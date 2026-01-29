package scheduler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/auth"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/store"
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

	if _, err := Add(dataDir, s1); err != nil {
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
			_, err := Add(testDataDir, tt.schedule)
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

func TestExtractConfigPersistence(t *testing.T) {
	dataDir := t.TempDir()

	// Create temporary extract config file
	extractConfigPath := filepath.Join(dataDir, "extract.json")
	extractConfig := `{
		"name": "test-template",
		"selectors": [
			{
				"name": "title",
				"selector": "h1",
				"attr": "text"
			}
		]
	}`
	if err := os.WriteFile(extractConfigPath, []byte(extractConfig), 0o644); err != nil {
		t.Fatalf("failed to write extract config: %v", err)
	}

	schedule := Schedule{
		Kind:            model.KindScrape,
		IntervalSeconds: 60,
		Params: map[string]interface{}{
			"url":             "https://example.com",
			"extractTemplate": "product",
			"extractConfig":   extractConfigPath,
			"extractValidate": true,
		},
	}

	if _, err := Add(dataDir, schedule); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Load and verify
	loaded, err := LoadAll(dataDir)
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 schedule, got %d", len(loaded))
	}

	extractOpts := loadExtract(loaded[0].Params)
	if extractOpts.Template != "product" {
		t.Errorf("expected extractTemplate 'product', got %s", extractOpts.Template)
	}
	if !extractOpts.Validate {
		t.Error("expected extractValidate true")
	}
	if extractOpts.Inline == nil {
		t.Error("expected inline template to be loaded from extractConfig")
	}
	if extractOpts.Inline.Name != "test-template" {
		t.Errorf("expected inline template name 'test-template', got %s", extractOpts.Inline.Name)
	}
	if len(extractOpts.Inline.Selectors) != 1 {
		t.Errorf("expected 1 selector, got %d", len(extractOpts.Inline.Selectors))
	}
}

func TestIncrementalModePersistence(t *testing.T) {
	testCases := []struct {
		name       string
		kind       model.Kind
		params     map[string]interface{}
		wantResult bool
	}{
		{
			name: "scrape with incremental enabled",
			kind: model.KindScrape,
			params: map[string]interface{}{
				"url":         "https://example.com",
				"incremental": true,
			},
			wantResult: true,
		},
		{
			name: "crawl with incremental enabled",
			kind: model.KindCrawl,
			params: map[string]interface{}{
				"url":         "https://example.com",
				"incremental": true,
			},
			wantResult: true,
		},
		{
			name: "scrape with incremental disabled",
			kind: model.KindScrape,
			params: map[string]interface{}{
				"url":         "https://example.com",
				"incremental": false,
			},
			wantResult: false,
		},
		{
			name: "scrape without incremental flag",
			kind: model.KindScrape,
			params: map[string]interface{}{
				"url": "https://example.com",
			},
			wantResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testDataDir := t.TempDir()

			schedule := Schedule{
				Kind:            tc.kind,
				IntervalSeconds: 60,
				Params:          tc.params,
			}

			if _, err := Add(testDataDir, schedule); err != nil {
				t.Fatalf("Add failed: %v", err)
			}

			loaded, err := LoadAll(testDataDir)
			if err != nil {
				t.Fatalf("LoadAll failed: %v", err)
			}
			if len(loaded) != 1 {
				t.Fatalf("expected 1 schedule, got %d", len(loaded))
			}

			got := boolParam(loaded[0].Params, "incremental")
			if got != tc.wantResult {
				t.Errorf("incremental = %v, want %v", got, tc.wantResult)
			}
		})
	}
}

func TestBoolParamDefault(t *testing.T) {
	tests := []struct {
		name     string
		params   map[string]interface{}
		key      string
		fallback bool
		want     bool
	}{
		{
			name:     "nil params - returns fallback",
			params:   nil,
			key:      "playwright",
			fallback: true,
			want:     true,
		},
		{
			name:     "key absent - returns fallback",
			params:   map[string]interface{}{},
			key:      "playwright",
			fallback: true,
			want:     true,
		},
		{
			name:     "key present with true - returns true",
			params:   map[string]interface{}{"playwright": true},
			key:      "playwright",
			fallback: false,
			want:     true,
		},
		{
			name:     "key present with false - returns false (not fallback)",
			params:   map[string]interface{}{"playwright": false},
			key:      "playwright",
			fallback: true,
			want:     false,
		},
		{
			name:     "key present with non-bool - returns fallback",
			params:   map[string]interface{}{"playwright": "invalid"},
			key:      "playwright",
			fallback: true,
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := boolParamDefault(tt.params, tt.key, tt.fallback)
			if got != tt.want {
				t.Errorf("boolParamDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthOverridePersistence(t *testing.T) {
	testCases := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name: "auth with headers",
			params: map[string]interface{}{
				"url": "https://example.com",
				"headers": []auth.HeaderKV{
					{Key: "X-API-Key", Value: "secret"},
					{Key: "Authorization", Value: "Bearer token"},
				},
			},
			wantErr: false,
		},
		{
			name: "auth with cookies",
			params: map[string]interface{}{
				"url": "https://example.com",
				"cookies": []auth.Cookie{
					{Name: "session", Value: "abc123"},
				},
			},
			wantErr: false,
		},
		{
			name: "auth with basic token",
			params: map[string]interface{}{
				"url":       "https://example.com",
				"authBasic": "user:pass",
			},
			wantErr: false,
		},
		{
			name: "auth with bearer tokens",
			params: map[string]interface{}{
				"url":       "https://example.com",
				"tokenKind": "bearer",
				"tokens":    []string{"token1", "token2"},
			},
			wantErr: false,
		},
		{
			name: "auth with login flow",
			params: map[string]interface{}{
				"url":                 "https://example.com",
				"loginURL":            "https://example.com/login",
				"loginUserSelector":   "#username",
				"loginPassSelector":   "#password",
				"loginSubmitSelector": "button[type=submit]",
				"loginUser":           "test@example.com",
				"loginPass":           "password",
			},
			wantErr: false,
		},
		{
			name: "auth with profile and overrides",
			params: map[string]interface{}{
				"url":         "https://example.com",
				"authProfile": "base",
				"headers": []auth.HeaderKV{
					{Key: "X-Custom", Value: "value"},
				},
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testDataDir := t.TempDir()

			// Create a base auth profile for tests that need it
			if _, hasProfile := tc.params["authProfile"]; hasProfile {
				profile := auth.Profile{
					Name: "base",
				}
				if err := auth.UpsertProfile(testDataDir, profile); err != nil {
					t.Fatalf("failed to create profile: %v", err)
				}
			}

			schedule := Schedule{
				Kind:            model.KindScrape,
				IntervalSeconds: 60,
				Params:          tc.params,
			}

			_, err := Add(testDataDir, schedule)
			if (err != nil) != tc.wantErr {
				t.Errorf("Add() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if err != nil {
				return
			}

			loaded, err := LoadAll(testDataDir)
			if err != nil {
				t.Fatalf("LoadAll failed: %v", err)
			}
			if len(loaded) != 1 {
				t.Fatalf("expected 1 schedule, got %d", len(loaded))
			}

			// Verify auth overrides are loaded correctly
			input := loadAuthOverrides(loaded[0].Params)

			if headers, hasHeaders := tc.params["headers"]; hasHeaders {
				if len(input.Headers) != len(headers.([]auth.HeaderKV)) {
					t.Errorf("expected %d headers, got %d", len(headers.([]auth.HeaderKV)), len(input.Headers))
				}
			}

			if cookies, hasCookies := tc.params["cookies"]; hasCookies {
				if len(input.Cookies) != len(cookies.([]auth.Cookie)) {
					t.Errorf("expected %d cookies, got %d", len(cookies.([]auth.Cookie)), len(input.Cookies))
				}
			}

			if tokens, hasTokens := tc.params["tokens"]; hasTokens {
				if len(input.Tokens) != len(tokens.([]string)) {
					t.Errorf("expected %d tokens, got %d", len(tokens.([]string)), len(input.Tokens))
				}
			}

			if _, hasBasic := tc.params["authBasic"]; hasBasic {
				if len(input.Tokens) == 0 {
					t.Error("expected basic auth token")
				}
			}

			if loginURL, hasLogin := tc.params["loginURL"]; hasLogin {
				if input.Login == nil {
					t.Error("expected login flow")
				}
				if input.Login.URL != loginURL {
					t.Errorf("expected login URL %s, got %s", loginURL, input.Login.URL)
				}
			}
		})
	}
}

func TestCachedSchedulerInit(t *testing.T) {
	dataDir := t.TempDir()
	manager, _, cleanup := setupTestManager(t)
	defer cleanup()

	cs, err := NewCachedScheduler(dataDir, manager)
	if err != nil {
		t.Fatalf("NewCachedScheduler failed: %v", err)
	}
	defer cs.watcher.Close()

	cs.mu.RLock()
	if len(cs.schedules) != 0 {
		cs.mu.RUnlock()
		t.Errorf("expected 0 schedules, got %d", len(cs.schedules))
	}
	cs.mu.RUnlock()
}

// TestCachedSchedulerManualReload tests that loadSchedules updates the cache correctly.
func TestCachedSchedulerManualReload(t *testing.T) {
	dataDir := t.TempDir()
	manager, _, cleanup := setupTestManager(t)
	defer cleanup()

	cs, err := NewCachedScheduler(dataDir, manager)
	if err != nil {
		t.Fatalf("NewCachedScheduler failed: %v", err)
	}
	defer cs.watcher.Close()

	schedule := Schedule{
		Kind:            model.KindScrape,
		IntervalSeconds: 60,
		Params:          map[string]interface{}{"url": "http://example.com"},
	}
	if _, err := Add(dataDir, schedule); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if err := cs.loadSchedules(); err != nil {
		t.Fatalf("loadSchedules failed: %v", err)
	}

	cs.mu.RLock()
	scheduleCount := len(cs.schedules)
	cs.mu.RUnlock()
	if scheduleCount != 1 {
		t.Errorf("expected 1 schedule in cache, got %d", scheduleCount)
	}
}

func TestCachedSchedulerRun(t *testing.T) {
	dataDir := t.TempDir()
	manager, st, cleanup := setupTestManager(t)
	defer cleanup()

	schedule := Schedule{
		Kind:            model.KindScrape,
		IntervalSeconds: 60,
		NextRun:         time.Now().Add(-1 * time.Second),
		Params:          map[string]interface{}{"url": "http://example.com"},
	}
	if _, err := Add(dataDir, schedule); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	cs, err := NewCachedScheduler(dataDir, manager)
	if err != nil {
		t.Fatalf("NewCachedScheduler failed: %v", err)
	}
	defer cs.watcher.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- Run(ctx, dataDir, manager)
	}()

	time.Sleep(1500 * time.Millisecond)

	jobs, err := st.List(context.Background())
	if err != nil {
		t.Fatalf("failed to list jobs: %v", err)
	}
	if len(jobs) == 0 {
		t.Error("expected at least one job to be enqueued")
	}

	cancel()
	if err := <-errCh; err != nil && err != context.Canceled {
		t.Errorf("Run failed: %v", err)
	}
}

func TestCachedSchedulerConcurrentAccess(t *testing.T) {
	dataDir := t.TempDir()
	manager, _, cleanup := setupTestManager(t)
	defer cleanup()

	cs, err := NewCachedScheduler(dataDir, manager)
	if err != nil {
		t.Fatalf("NewCachedScheduler failed: %v", err)
	}
	defer cs.watcher.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go cs.reloadLoop(ctx)

	for i := 0; i < 5; i++ {
		schedule := Schedule{
			Kind:            model.KindScrape,
			IntervalSeconds: 60,
			Params:          map[string]interface{}{"url": fmt.Sprintf("http://example%d.com", i)},
		}
		if _, err := Add(dataDir, schedule); err != nil {
			t.Fatalf("Add failed: %v", err)
		}
	}

	if err := cs.loadSchedules(); err != nil {
		t.Fatalf("loadSchedules failed: %v", err)
	}

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				cs.mu.RLock()
				_ = len(cs.schedules)
				cs.mu.RUnlock()
			}
			done <- struct{}{}
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	cs.mu.RLock()
	scheduleCount := len(cs.schedules)
	cs.mu.RUnlock()

	if scheduleCount < 5 {
		t.Errorf("expected at least 5 schedules, got %d", scheduleCount)
	}
}

func TestCachedSchedulerFileWatcher(t *testing.T) {
	dataDir := t.TempDir()
	manager, _, cleanup := setupTestManager(t)
	defer cleanup()

	cs, err := NewCachedScheduler(dataDir, manager)
	if err != nil {
		t.Fatalf("NewCachedScheduler failed: %v", err)
	}
	defer cs.watcher.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cs.startWatcher(ctx)
	go cs.reloadLoop(ctx)

	schedule := Schedule{
		Kind:            model.KindScrape,
		IntervalSeconds: 60,
		Params:          map[string]interface{}{"url": "http://example.com"},
	}
	if _, err := Add(dataDir, schedule); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	cs.mu.RLock()
	scheduleCount := len(cs.schedules)
	cs.mu.RUnlock()

	if scheduleCount != 1 {
		t.Errorf("expected 1 schedule after file change, got %d", scheduleCount)
	}
}
