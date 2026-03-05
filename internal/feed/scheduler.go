// Package feed provides RSS/Atom feed monitoring and parsing functionality.
//
// This file is responsible for:
// - Scheduling periodic feed checks
// - Managing the feed check loop
// - Coordinating concurrent feed execution
// - Graceful shutdown handling
//
// This file does NOT handle:
// - Individual feed execution (feed.go handles this)
// - Feed storage (storage.go handles this)
// - Seen item tracking (seen_storage.go handles this)
//
// Invariants:
// - Checks are executed concurrently with a limit
// - Scheduler can be stopped gracefully
// - Failed checks are logged but don't stop the scheduler
package feed

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Scheduler manages the periodic execution of feed checks.
type Scheduler struct {
	checker       *Checker
	storage       Storage
	interval      time.Duration
	maxConcurrent int
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

// SchedulerConfig holds configuration for the scheduler.
type SchedulerConfig struct {
	// Interval is how often to check for due feeds (default: 10s)
	Interval time.Duration
	// MaxConcurrent is the maximum number of concurrent feed checks (default: 5)
	MaxConcurrent int
}

// DefaultSchedulerConfig returns default scheduler configuration.
func DefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		Interval:      10 * time.Second,
		MaxConcurrent: 5,
	}
}

// NewScheduler creates a new feed scheduler.
func NewScheduler(checker *Checker, storage Storage, cfg SchedulerConfig) *Scheduler {
	if cfg.Interval <= 0 {
		cfg.Interval = 10 * time.Second
	}
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = 5
	}

	return &Scheduler{
		checker:       checker,
		storage:       storage,
		interval:      cfg.Interval,
		maxConcurrent: cfg.MaxConcurrent,
		stopCh:        make(chan struct{}),
	}
}

// Run starts the scheduler and blocks until stopped.
func (s *Scheduler) Run(ctx context.Context) error {
	slog.Info("feed scheduler started", "interval", s.interval, "maxConcurrent", s.maxConcurrent)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Run immediately on start
	s.runChecks(ctx)

	for {
		select {
		case <-ctx.Done():
			slog.Info("feed scheduler stopping due to context cancellation")
			s.wg.Wait()
			return ctx.Err()
		case <-s.stopCh:
			slog.Info("feed scheduler stopping")
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

// runChecks executes all due feeds concurrently.
func (s *Scheduler) runChecks(ctx context.Context) {
	feeds, err := s.storage.ListEnabled()
	if err != nil {
		slog.Error("failed to list enabled feeds", "error", err)
		return
	}

	// Filter to only due feeds
	var dueFeeds []Feed
	for _, feed := range feeds {
		if feed.IsDue() {
			dueFeeds = append(dueFeeds, feed)
		}
	}

	if len(dueFeeds) == 0 {
		return
	}

	slog.Debug("running feed checks", "count", len(dueFeeds))

	// Create semaphore for limiting concurrency
	sem := make(chan struct{}, s.maxConcurrent)

	for _, feed := range dueFeeds {
		s.wg.Add(1)
		go func(f Feed) {
			defer s.wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			result, err := s.checker.Check(ctx, &f)
			if err != nil {
				slog.Error("feed check failed",
					"feedID", f.ID,
					"url", f.URL,
					"error", err,
				)
				return
			}

			if len(result.NewItems) > 0 {
				slog.Info("feed has new items",
					"feedID", f.ID,
					"url", f.URL,
					"newItems", len(result.NewItems),
					"totalItems", result.TotalItems,
				)
			} else {
				slog.Debug("no new items in feed",
					"feedID", f.ID,
					"url", f.URL,
					"totalItems", result.TotalItems,
				)
			}
		}(feed)
	}
}

// RunOnce runs all due feeds once and returns the results.
func (s *Scheduler) RunOnce(ctx context.Context) ([]*FeedCheckResult, error) {
	return s.checker.CheckAll(ctx)
}
