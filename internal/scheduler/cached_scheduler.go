package scheduler

import (
	"path/filepath"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fsnotify/fsnotify"
)

func NewCachedScheduler(dataDir string, manager *jobs.Manager) (*cachedScheduler, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to create file watcher", err)
	}

	cs := &cachedScheduler{
		dataDir:  dataDir,
		manager:  manager,
		watcher:  watcher,
		reloadCh: make(chan struct{}, 1),
		doneCh:   make(chan struct{}),
	}

	if err := cs.loadSchedules(); err != nil {
		watcher.Close()
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to load initial schedules", err)
	}

	schedulesDir := filepath.Dir(schedulesPath(dataDir))
	if err := watcher.Add(schedulesDir); err != nil {
		watcher.Close()
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to watch schedules directory", err)
	}

	return cs, nil
}

func (cs *cachedScheduler) loadSchedules() error {
	schedules, err := LoadAll(cs.dataDir)
	if err != nil {
		return err
	}

	cs.mu.Lock()
	cs.schedules = schedules
	cs.mu.Unlock()
	return nil
}
