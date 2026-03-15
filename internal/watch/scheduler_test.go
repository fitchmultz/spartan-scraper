// Package watch provides content change monitoring functionality.
//
// This file contains tests for the watch scheduler.
package watch

import (
	"context"
	"testing"
	"time"
)

func TestSchedulerStop(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFileStorage(tmpDir)

	watcher := NewWatcher(storage, nil, tmpDir, nil, nil)
	scheduler := NewScheduler(watcher, storage, DefaultSchedulerConfig())

	// Test that Stop can be called multiple times without panic
	done := make(chan struct{})
	go func() {
		defer close(done)
		// First stop
		scheduler.Stop()
		// Second stop should not panic
		scheduler.Stop()
	}()

	select {
	case <-done:
		// Success - no panic
	case <-time.After(2 * time.Second):
		t.Error("Stop() calls timed out or panicked")
	}
}

func TestSchedulerRunAndStop(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFileStorage(tmpDir)

	// Add a watch
	watch := &Watch{
		URL:             "https://example.com",
		IntervalSeconds: 3600,
		Enabled:         true,
	}
	_, err := storage.Add(watch)
	if err != nil {
		t.Fatalf("Failed to add watch: %v", err)
	}

	watcher := NewWatcher(storage, nil, tmpDir, nil, nil)
	scheduler := NewScheduler(watcher, storage, SchedulerConfig{
		Interval:      100 * time.Millisecond,
		MaxConcurrent: 1,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Run scheduler in background
	done := make(chan error)
	go func() {
		done <- scheduler.Run(ctx)
	}()

	// Let it run briefly
	time.Sleep(200 * time.Millisecond)

	// Stop the scheduler
	scheduler.Stop()

	// Wait for completion
	select {
	case err := <-done:
		if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
			t.Errorf("Unexpected error from Run(): %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Scheduler did not stop in time")
	}
}

func TestDefaultSchedulerConfig(t *testing.T) {
	cfg := DefaultSchedulerConfig()

	if cfg.Interval != 10*time.Second {
		t.Errorf("Default interval = %v, want %v", cfg.Interval, 10*time.Second)
	}

	if cfg.MaxConcurrent != 5 {
		t.Errorf("Default maxConcurrent = %d, want 5", cfg.MaxConcurrent)
	}
}

func TestNewSchedulerDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFileStorage(tmpDir)
	watcher := NewWatcher(storage, nil, tmpDir, nil, nil)

	// Test with zero values - should use defaults
	scheduler := NewScheduler(watcher, storage, SchedulerConfig{
		Interval:      0,
		MaxConcurrent: 0,
	})

	if scheduler.interval != 10*time.Second {
		t.Errorf("interval = %v, want %v", scheduler.interval, 10*time.Second)
	}

	if scheduler.maxConcurrent != 5 {
		t.Errorf("maxConcurrent = %d, want 5", scheduler.maxConcurrent)
	}
}

func TestSchedulerRunOnce(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewFileStorage(tmpDir)

	watcher := NewWatcher(storage, nil, tmpDir, nil, nil)
	scheduler := NewScheduler(watcher, storage, DefaultSchedulerConfig())

	ctx := context.Background()

	// RunOnce should not fail even with no watches
	results, err := scheduler.RunOnce(ctx)
	if err != nil {
		t.Errorf("RunOnce() error = %v", err)
	}

	if len(results) != 0 {
		t.Errorf("RunOnce() returned %d results, want 0", len(results))
	}
}
