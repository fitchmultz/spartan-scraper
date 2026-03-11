// Package scheduler provides the main scheduling loop for executing scheduled jobs.
//
// This file is responsible for:
// - Running the main scheduler loop with 1-second ticker polling
// - Checking schedules and enqueuing jobs when NextRun is due
// - Updating NextRun after successful job enqueue
// - Building JobSpec from schedule parameters for job creation
// - Running periodic retention cleanup based on configuration
//
// This file does NOT handle:
// - Schedule persistence (storage.go does this)
// - File watching for hot reloads (watcher.go does this)
// - In-memory schedule caching (cached_scheduler.go does this)
// - Retention policy evaluation (retention package does this)
//
// Invariants:
// - Ticker polls every 1 second for schedule evaluation
// - Schedules are copied under RLock before processing
// - NextRun is updated only after successful job enqueue
// - Uses jobs.Manager for job creation and enqueueing
// - Retention cleanup runs on configured interval (default 24h)
package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/auth"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/retention"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

func Run(ctx context.Context, dataDir string, manager *jobs.Manager, cfg config.Config) error {
	cs, err := NewCachedScheduler(dataDir, manager)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to create cached scheduler", err)
	}
	defer cs.watcher.Close()

	cs.startWatcher(ctx)

	go cs.reloadLoop(ctx)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Setup retention cleanup ticker
	var cleanupTicker *time.Ticker
	var cleanupChan <-chan time.Time
	if cfg.RetentionEnabled && cfg.RetentionCleanupIntervalHours > 0 {
		cleanupTicker = time.NewTicker(time.Duration(cfg.RetentionCleanupIntervalHours) * time.Hour)
		cleanupChan = cleanupTicker.C
		defer cleanupTicker.Stop()
	}

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-ticker.C:
			now := time.Now()

			cs.mu.RLock()
			schedules := make([]Schedule, len(cs.schedules))
			copy(schedules, cs.schedules)
			cs.mu.RUnlock()

			changed := false
			for i := range schedules {
				if schedules[i].NextRun.After(now) {
					continue
				}
				err := enqueue(ctx, manager, dataDir, schedules[i])
				if err == nil {
					schedules[i].NextRun = now.Add(time.Duration(schedules[i].IntervalSeconds) * time.Second)
					changed = true
				} else {
					slog.Error("failed to enqueue scheduled job",
						"scheduleID", schedules[i].ID,
						"scheduleKind", schedules[i].Kind,
						"error", err,
					)
				}
			}

			if changed {
				if err := SaveAll(dataDir, schedules); err != nil {
					slog.Error("failed to save schedules", "error", err)
				} else {
					cs.mu.Lock()
					cs.schedules = schedules
					cs.mu.Unlock()
				}
			}

		case <-cleanupChan:
			if cfg.RetentionEnabled {
				runRetentionCleanup(ctx, dataDir, cfg)
			}
		}
	}
}

// runRetentionCleanup executes the retention cleanup process.
func runRetentionCleanup(ctx context.Context, dataDir string, cfg config.Config) {
	slog.Info("running scheduled retention cleanup")

	st, err := store.Open(dataDir)
	if err != nil {
		slog.Error("failed to open store for retention cleanup", "error", err)
		return
	}
	defer st.Close()

	engine := retention.NewEngine(st, cfg)

	result, err := engine.RunCleanup(ctx, retention.CleanupOptions{DryRun: false})
	if err != nil {
		slog.Error("retention cleanup failed", "error", err)
		return
	}

	logArgs := []any{
		"jobsDeleted", result.JobsDeleted,
		"jobsAttempted", result.JobsAttempted,
		"crawlStatesDeleted", result.CrawlStatesDeleted,
		"spaceReclaimedMB", result.SpaceReclaimedMB,
		"duration", result.Duration,
	}
	if len(result.FailedJobIDs) > 0 {
		logArgs = append(logArgs, "failedArtifactDeletions", len(result.FailedJobIDs))
	}
	slog.Info("retention cleanup completed", logArgs...)

	if len(result.Errors) > 0 {
		slog.Warn("retention cleanup completed with errors", "errorCount", len(result.Errors))
	}
}

func enqueue(ctx context.Context, manager *jobs.Manager, dataDir string, schedule Schedule) error {
	exec, err := executionSpecForSchedule(schedule)
	if err != nil {
		return err
	}

	authOptions, err := resolveScheduleAuth(schedule, dataDir, auth.EnvOverrides{})
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to resolve auth for schedule", err)
	}

	spec := jobs.JobSpec{
		Kind:             schedule.Kind,
		Headless:         exec.Headless,
		UsePlaywright:    exec.UsePlaywright,
		AuthProfile:      exec.AuthProfile,
		Auth:             authOptions,
		TimeoutSeconds:   exec.TimeoutSeconds,
		Extract:          exec.Extract,
		Pipeline:         exec.Pipeline,
		Screenshot:       exec.Screenshot,
		Device:           exec.Device,
		NetworkIntercept: exec.NetworkIntercept,
	}
	if spec.TimeoutSeconds <= 0 {
		spec.TimeoutSeconds = manager.DefaultTimeoutSeconds()
	}

	if webhook := model.ExtractWebhookSpec(schedule.Spec); webhook != nil {
		spec.WebhookURL = webhook.URL
		spec.WebhookEvents = webhook.Events
		spec.WebhookSecret = webhook.Secret
	}

	switch typed := schedule.Spec.(type) {
	case model.ScrapeSpecV1:
		spec.URL = typed.URL
		spec.Method = typed.Method
		spec.Body = typed.Body
		spec.ContentType = typed.ContentType
		spec.Incremental = typed.Incremental
	case *model.ScrapeSpecV1:
		spec.URL = typed.URL
		spec.Method = typed.Method
		spec.Body = typed.Body
		spec.ContentType = typed.ContentType
		spec.Incremental = typed.Incremental
	case model.CrawlSpecV1:
		spec.URL = typed.URL
		spec.MaxDepth = typed.MaxDepth
		spec.MaxPages = typed.MaxPages
		spec.Incremental = typed.Incremental
		spec.SitemapURL = typed.SitemapURL
		spec.SitemapOnly = typed.SitemapOnly
		spec.IncludePatterns = typed.IncludePatterns
		spec.ExcludePatterns = typed.ExcludePatterns
		spec.RespectRobotsTxt = typed.RespectRobotsTxt
		spec.SkipDuplicates = typed.SkipDuplicates
		spec.SimHashThreshold = typed.SimHashThreshold
	case *model.CrawlSpecV1:
		spec.URL = typed.URL
		spec.MaxDepth = typed.MaxDepth
		spec.MaxPages = typed.MaxPages
		spec.Incremental = typed.Incremental
		spec.SitemapURL = typed.SitemapURL
		spec.SitemapOnly = typed.SitemapOnly
		spec.IncludePatterns = typed.IncludePatterns
		spec.ExcludePatterns = typed.ExcludePatterns
		spec.RespectRobotsTxt = typed.RespectRobotsTxt
		spec.SkipDuplicates = typed.SkipDuplicates
		spec.SimHashThreshold = typed.SimHashThreshold
	case model.ResearchSpecV1:
		spec.Query = typed.Query
		spec.URLs = typed.URLs
		spec.MaxDepth = typed.MaxDepth
		spec.MaxPages = typed.MaxPages
	case *model.ResearchSpecV1:
		spec.Query = typed.Query
		spec.URLs = typed.URLs
		spec.MaxDepth = typed.MaxDepth
		spec.MaxPages = typed.MaxPages
	default:
		return apperrors.Validation("schedule spec is invalid")
	}

	job, err := manager.CreateJob(ctx, spec)
	if err != nil {
		return err
	}
	return manager.Enqueue(job)
}
