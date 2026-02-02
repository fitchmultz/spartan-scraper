// Package feed provides RSS/Atom feed monitoring and parsing functionality.
//
// This file contains tests for the feed scheduler.
package feed

import (
	"context"
	"testing"
	"time"
)

func TestDefaultSchedulerConfig(t *testing.T) {
	cfg := DefaultSchedulerConfig()

	if cfg.Interval != 10*time.Second {
		t.Errorf("Interval = %v, want %v", cfg.Interval, 10*time.Second)
	}

	if cfg.MaxConcurrent != 5 {
		t.Errorf("MaxConcurrent = %d, want 5", cfg.MaxConcurrent)
	}
}

func TestNewScheduler(t *testing.T) {
	storage := &mockStorage{}
	checker := NewChecker(storage, nil, nil)

	cfg := SchedulerConfig{
		Interval:      30 * time.Second,
		MaxConcurrent: 10,
	}

	scheduler := NewScheduler(checker, storage, cfg)

	if scheduler.interval != 30*time.Second {
		t.Errorf("interval = %v, want %v", scheduler.interval, 30*time.Second)
	}

	if scheduler.maxConcurrent != 10 {
		t.Errorf("maxConcurrent = %d, want 10", scheduler.maxConcurrent)
	}
}

func TestNewScheduler_Defaults(t *testing.T) {
	storage := &mockStorage{}
	checker := NewChecker(storage, nil, nil)

	cfg := SchedulerConfig{
		Interval:      0, // Should default to 10s
		MaxConcurrent: 0, // Should default to 5
	}

	scheduler := NewScheduler(checker, storage, cfg)

	if scheduler.interval != 10*time.Second {
		t.Errorf("interval = %v, want %v", scheduler.interval, 10*time.Second)
	}

	if scheduler.maxConcurrent != 5 {
		t.Errorf("maxConcurrent = %d, want 5", scheduler.maxConcurrent)
	}
}

func TestScheduler_Stop(t *testing.T) {
	storage := &mockStorage{}
	checker := NewChecker(storage, nil, nil)
	scheduler := NewScheduler(checker, storage, DefaultSchedulerConfig())

	// Stop should not panic even if scheduler hasn't started
	scheduler.Stop()

	// Stop should be safe to call multiple times
	scheduler.Stop()
	scheduler.Stop()
}

func TestScheduler_RunOnce(t *testing.T) {
	storage := &mockStorage{}
	seenStorage := newMockSeenStorage()
	checker := NewChecker(storage, seenStorage, nil)
	scheduler := NewScheduler(checker, storage, DefaultSchedulerConfig())

	ctx := context.Background()

	// Add some enabled feeds that are due
	now := time.Now()
	storage.feeds = []Feed{
		{
			ID:              "feed-1",
			URL:             "https://example.com/feed1.xml",
			Enabled:         true,
			IntervalSeconds: 3600,
			LastCheckedAt:   now.Add(-2 * time.Hour),
		},
		{
			ID:              "feed-2",
			URL:             "https://example.com/feed2.xml",
			Enabled:         false, // Disabled
			IntervalSeconds: 3600,
			LastCheckedAt:   now.Add(-2 * time.Hour),
		},
	}

	results, err := scheduler.RunOnce(ctx)
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}

	// Should have results for enabled feeds only
	// Note: The actual check will fail because there's no HTTP server
	// but we should still get a result entry
	if len(results) == 0 {
		t.Error("RunOnce() returned no results, expected at least one")
	}
}

func TestScheduler_Run_ContextCancellation(t *testing.T) {
	storage := &mockStorage{}
	checker := NewChecker(storage, nil, nil)
	scheduler := NewScheduler(checker, storage, SchedulerConfig{
		Interval:      100 * time.Millisecond,
		MaxConcurrent: 1,
	})

	ctx, cancel := context.WithCancel(context.Background())

	// Start scheduler in background
	done := make(chan error, 1)
	go func() {
		done <- scheduler.Run(ctx)
	}()

	// Cancel context after a short delay
	time.Sleep(200 * time.Millisecond)
	cancel()

	// Wait for scheduler to stop
	select {
	case err := <-done:
		if err != context.Canceled {
			t.Errorf("Run() error = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Run() did not stop after context cancellation")
	}
}

func TestScheduler_StopWhileRunning(t *testing.T) {
	storage := &mockStorage{}
	checker := NewChecker(storage, nil, nil)
	scheduler := NewScheduler(checker, storage, SchedulerConfig{
		Interval:      100 * time.Millisecond,
		MaxConcurrent: 1,
	})

	ctx := context.Background()

	// Start scheduler in background
	done := make(chan error, 1)
	go func() {
		done <- scheduler.Run(ctx)
	}()

	// Let it run briefly
	time.Sleep(150 * time.Millisecond)

	// Stop the scheduler
	scheduler.Stop()

	// Wait for scheduler to stop
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run() error = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Run() did not stop after Stop() called")
	}
}
