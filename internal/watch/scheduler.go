// Package watch provides content change monitoring functionality.
//
// This file is responsible for:
// - Scheduling periodic watch checks
// - Managing the watch check loop
// - Coordinating concurrent watch execution
// - Graceful shutdown handling
//
// This file does NOT handle:
// - Individual watch execution (watch.go handles this)
// - Watch storage (storage.go handles this)
// - Diff generation (diff package handles this)
//
// Invariants:
// - Checks are executed concurrently with a limit
// - Scheduler can be stopped gracefully
// - Failed checks are logged but don't stop the scheduler
package watch

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Scheduler manages the periodic execution of watches.
type Scheduler struct {
	watcher       *Watcher
	storage       Storage
	interval      time.Duration
	maxConcurrent int
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

// SchedulerConfig holds configuration for the scheduler.
type SchedulerConfig struct {
	// Interval is how often to check for due watches (default: 10s)
	Interval time.Duration
	// MaxConcurrent is the maximum number of concurrent watch checks (default: 5)
	MaxConcurrent int
}

// DefaultSchedulerConfig returns default scheduler configuration.
func DefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		Interval:      10 * time.Second,
		MaxConcurrent: 5,
	}
}

// NewScheduler creates a new watch scheduler.
func NewScheduler(watcher *Watcher, storage Storage, cfg SchedulerConfig) *Scheduler {
	if cfg.Interval <= 0 {
		cfg.Interval = 10 * time.Second
	}
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = 5
	}

	return &Scheduler{
		watcher:       watcher,
		storage:       storage,
		interval:      cfg.Interval,
		maxConcurrent: cfg.MaxConcurrent,
		stopCh:        make(chan struct{}),
	}
}

// Run starts the scheduler and blocks until stopped.
func (s *Scheduler) Run(ctx context.Context) error {
	slog.Info("watch scheduler started", "interval", s.interval, "maxConcurrent", s.maxConcurrent)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Run immediately on start
	s.runChecks(ctx)

	for {
		select {
		case <-ctx.Done():
			slog.Info("watch scheduler stopping due to context cancellation")
			s.wg.Wait()
			return ctx.Err()
		case <-s.stopCh:
			slog.Info("watch scheduler stopping")
			s.wg.Wait()
			return nil
		case <-ticker.C:
			s.runChecks(ctx)
		}
	}
}

// Stop gracefully stops the scheduler.
// Safe to call multiple times; only the first call has effect.
func (s *Scheduler) Stop() {
	select {
	case <-s.stopCh:
		// Already closed
	default:
		close(s.stopCh)
	}
}

// runChecks executes all due watches concurrently.
func (s *Scheduler) runChecks(ctx context.Context) {
	watches, err := s.storage.ListEnabled()
	if err != nil {
		slog.Error("failed to list enabled watches", "error", err)
		return
	}

	// Filter to only due watches
	var dueWatches []Watch
	for _, watch := range watches {
		if watch.IsDue() {
			dueWatches = append(dueWatches, watch)
		}
	}

	if len(dueWatches) == 0 {
		return
	}

	slog.Debug("running watch checks", "count", len(dueWatches))

	// Create semaphore for limiting concurrency
	sem := make(chan struct{}, s.maxConcurrent)

	for _, watch := range dueWatches {
		s.wg.Add(1)
		go func(w Watch) {
			defer s.wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			result, err := s.watcher.Check(ctx, &w)
			if err != nil {
				slog.Error("watch check failed",
					"watchID", w.ID,
					"url", w.URL,
					"error", err,
				)
				return
			}

			if result.Changed {
				prevHash := result.PreviousHash
				if len(prevHash) > 8 {
					prevHash = prevHash[:8]
				}
				currHash := result.CurrentHash
				if len(currHash) > 8 {
					currHash = currHash[:8]
				}
				slog.Info("content changed detected",
					"watchID", w.ID,
					"url", w.URL,
					"previousHash", prevHash,
					"currentHash", currHash,
				)
			} else {
				slog.Debug("no change detected",
					"watchID", w.ID,
					"url", w.URL,
				)
			}
		}(watch)
	}
}

// RunOnce runs all due watches once and returns the results.
func (s *Scheduler) RunOnce(ctx context.Context) ([]*WatchCheckResult, error) {
	return s.watcher.CheckAll(ctx)
}
