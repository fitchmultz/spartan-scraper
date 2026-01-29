package scheduler

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

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
			if !apperrors.IsKind(err, apperrors.KindInternal) {
				t.Errorf("error kind = %v, want %v", apperrors.KindOf(err), apperrors.KindInternal)
			}
			if !strings.Contains(err.Error(), "failed to resolve auth") {
				t.Errorf("error message should mention auth resolution failure: %v", err)
			}
			if strings.Contains(apperrors.SafeMessage(err), tt.schedule.ID) {
				t.Errorf("safe message should not include schedule ID %s", tt.schedule.ID)
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

func TestSchedulerWatcherErrors(t *testing.T) {
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
