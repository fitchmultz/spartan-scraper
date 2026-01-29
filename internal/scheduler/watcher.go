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
