// Package scheduler provides recurring job scheduling with file-based persistence
// and hot-reloading capabilities.
//
// This package defines:
// - Schedule type for representing scheduled jobs with intervals
// - cachedScheduler for in-memory schedule management with file watching
// - scheduleStore for JSON persistence format
//
// This package is responsible for:
// - Loading and saving schedules to disk
// - Watching schedule files for external changes
// - Executing scheduled jobs at their configured intervals
// - Thread-safe access to schedule data via RWMutex
//
// This package does NOT handle:
// - Job execution logic (delegates to jobs.Manager)
// - Schedule validation (validation.go handles this)
// - Direct schedule persistence (storage.go handles this)
//
// Invariants:
// - Schedule IDs are UUIDs generated on creation
// - IntervalSeconds defaults to 3600 if not specified
// - NextRun defaults to now + interval if not specified
// - All schedule access is protected by cachedScheduler.mu
package scheduler

import (
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fsnotify/fsnotify"
	"sync"
)

type Schedule struct {
	ID              string     `json:"id"`
	Kind            model.Kind `json:"kind"`
	IntervalSeconds int        `json:"intervalSeconds"`
	NextRun         time.Time  `json:"nextRun"`
	SpecVersion     int        `json:"specVersion"`
	Spec            any        `json:"spec"`
}

type scheduleStore struct {
	Schedules []Schedule `json:"schedules"`
}

type cachedScheduler struct {
	dataDir   string
	manager   *jobs.Manager
	mu        sync.RWMutex
	schedules []Schedule
	watcher   *fsnotify.Watcher
	reloadCh  chan struct{}
	doneCh    chan struct{}
}
