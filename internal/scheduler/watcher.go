// Package scheduler provides file system watching and hot-reloading for schedules.
//
// This file is responsible for:
// - Watching the schedules file for external changes (write, create, remove)
// - Triggering reloads via channel when file changes are detected
// - Fallback polling via ticker every 5 seconds
// - Handling fsnotify errors gracefully
//
// This file does NOT handle:
// - Schedule persistence (storage.go does this)
// - Schedule loading logic (cached_scheduler.go does this)
// - Schedule execution (scheduler.go does this)
//
// Invariants:
// - Uses fsnotify.Watcher for file system events
// - Non-blocking send on reloadCh with default case
// - Context cancellation stops both watcher and reload loop
package scheduler

import (
	"context"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

func (cs *cachedScheduler) startWatcher(ctx context.Context) {
	schedulesFilePath := schedulesPath(cs.dataDir)

	go func() {
		defer close(cs.doneCh)

		for {
			select {
			case <-ctx.Done():
				return

			case event, ok := <-cs.watcher.Events:
				if !ok {
					return
				}

				if filepath.Clean(event.Name) == schedulesFilePath {
					if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Remove) {
						select {
						case cs.reloadCh <- struct{}{}:
						default:
						}
					}
				}

			case err, ok := <-cs.watcher.Errors:
				if !ok {
					return
				}
				slog.Error("scheduler file watcher error", "error", err)
			}
		}
	}()
}

func (cs *cachedScheduler) reloadLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-cs.reloadCh:
			if err := cs.loadSchedules(); err != nil {
				slog.Error("failed to reload schedules from disk", "error", err)
			}

		case <-ticker.C:
			if err := cs.loadSchedules(); err != nil {
				slog.Error("failed to reload schedules (fallback)", "error", err)
			}
		}
	}
}
