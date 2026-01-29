package scheduler

import (
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fsnotify/fsnotify"
	"sync"
)

type Schedule struct {
	ID              string                 `json:"id"`
	Kind            model.Kind             `json:"kind"`
	IntervalSeconds int                    `json:"intervalSeconds"`
	NextRun         time.Time              `json:"nextRun"`
	Params          map[string]interface{} `json:"params"`
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
