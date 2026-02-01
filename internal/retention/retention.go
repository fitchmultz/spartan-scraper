// Package retention provides data retention policy enforcement.
//
// This package is responsible for:
// - Evaluating retention policies against current data
// - Determining which jobs/crawl states should be cleaned up
// - Executing cleanup operations (with dry-run support)
// - Reporting cleanup results and statistics
//
// This package does NOT handle:
// - Scheduled execution (scheduler handles this)
// - Database operations directly (store layer handles this)
//
// Invariants:
// - Cleanup respects dry-run mode - no deletions when enabled
// - Failed jobs are prioritized for cleanup over succeeded jobs
// - Oldest data is cleaned first when count/size limits are exceeded
// - All operations are context-aware and can be cancelled
package retention

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

// Engine orchestrates retention policy enforcement.
type Engine struct {
	store  *store.Store
	cfg    config.Config
	logger *slog.Logger
}

// NewEngine creates a retention engine.
func NewEngine(st *store.Store, cfg config.Config) *Engine {
	return &Engine{
		store:  st,
		cfg:    cfg,
		logger: slog.Default().With("component", "retention"),
	}
}

// CleanupOptions controls cleanup behavior.
type CleanupOptions struct {
	DryRun    bool        // Preview only, don't delete
	Force     bool        // Ignore retention enabled setting
	OlderThan *time.Time  // Override age threshold
	Kind      *model.Kind // Only cleanup specific kind
}

// CleanupResult reports what was cleaned up.
type CleanupResult struct {
	JobsDeleted        int
	CrawlStatesDeleted int64
	SpaceReclaimedMB   int64
	Errors             []error
	Duration           time.Duration
	FailedJobIDs       []string // Jobs whose artifacts failed to delete
}

// RunCleanup executes retention policies.
func (e *Engine) RunCleanup(ctx context.Context, opts CleanupOptions) (CleanupResult, error) {
	start := time.Now()
	result := CleanupResult{}

	// Check if retention is enabled (unless forced)
	if !e.cfg.RetentionEnabled && !opts.Force {
		e.logger.Info("retention cleanup skipped: not enabled (use --force to override)")
		return result, nil
	}

	// Phase 1: Age-based cleanup for jobs
	if e.cfg.RetentionJobDays > 0 || opts.OlderThan != nil {
		ageResult, err := e.cleanupByAge(ctx, opts)
		result.JobsDeleted += ageResult.JobsDeleted
		result.SpaceReclaimedMB += ageResult.SpaceReclaimedMB
		result.FailedJobIDs = append(result.FailedJobIDs, ageResult.FailedJobIDs...)
		if err != nil {
			result.Errors = append(result.Errors, err)
		}
	}

	// Phase 2: Count-based cleanup
	if e.cfg.RetentionMaxJobs > 0 {
		countResult, err := e.cleanupByCount(ctx, opts)
		result.JobsDeleted += countResult.JobsDeleted
		result.SpaceReclaimedMB += countResult.SpaceReclaimedMB
		result.FailedJobIDs = append(result.FailedJobIDs, countResult.FailedJobIDs...)
		if err != nil {
			result.Errors = append(result.Errors, err)
		}
	}

	// Phase 3: Storage-based cleanup
	if e.cfg.RetentionMaxStorageGB > 0 {
		storageResult, err := e.cleanupByStorage(ctx, opts)
		result.JobsDeleted += storageResult.JobsDeleted
		result.SpaceReclaimedMB += storageResult.SpaceReclaimedMB
		result.FailedJobIDs = append(result.FailedJobIDs, storageResult.FailedJobIDs...)
		if err != nil {
			result.Errors = append(result.Errors, err)
		}
	}

	// Phase 4: Crawl state cleanup
	if e.cfg.RetentionCrawlStateDays > 0 {
		crawlResult, err := e.cleanupCrawlStates(ctx, opts)
		result.CrawlStatesDeleted += crawlResult.CrawlStatesDeleted
		if err != nil {
			result.Errors = append(result.Errors, err)
		}
	}

	result.Duration = time.Since(start)

	// Log summary
	if opts.DryRun {
		e.logger.Info("retention dry-run completed",
			"jobsWouldDelete", result.JobsDeleted,
			"crawlStatesWouldDelete", result.CrawlStatesDeleted,
			"spaceWouldReclaimMB", result.SpaceReclaimedMB,
			"duration", result.Duration,
		)
	} else {
		logArgs := []any{
			"jobsDeleted", result.JobsDeleted,
			"crawlStatesDeleted", result.CrawlStatesDeleted,
			"spaceReclaimedMB", result.SpaceReclaimedMB,
			"duration", result.Duration,
		}
		if len(result.FailedJobIDs) > 0 {
			logArgs = append(logArgs, "failedArtifactDeletions", len(result.FailedJobIDs))
		}
		e.logger.Info("retention cleanup completed", logArgs...)
	}

	return result, nil
}

// cleanupByAge deletes jobs older than the configured threshold.
func (e *Engine) cleanupByAge(ctx context.Context, opts CleanupOptions) (CleanupResult, error) {
	result := CleanupResult{}

	var cutoff time.Time
	if opts.OlderThan != nil {
		cutoff = *opts.OlderThan
	} else {
		cutoff = time.Now().AddDate(0, 0, -e.cfg.RetentionJobDays)
	}

	// Prioritize: failed jobs first, then succeeded, then others
	statuses := []model.Status{model.StatusFailed, model.StatusSucceeded, model.StatusCanceled}

	for _, status := range statuses {
		offset := 0
		for {
			// Get batch of old jobs with this status
			jobs, err := e.store.ListJobsByStatusAndAge(ctx, status, cutoff, store.ListOptions{Limit: 100, Offset: offset})
			if err != nil {
				return result, err
			}
			if len(jobs) == 0 {
				break
			}

			// Filter by kind if specified
			var toDelete []string
			for _, job := range jobs {
				if opts.Kind != nil && job.Kind != *opts.Kind {
					continue
				}
				toDelete = append(toDelete, job.ID)
			}

			if len(toDelete) == 0 {
				offset += len(jobs)
				if len(jobs) < 100 {
					break
				}
				continue
			}

			if opts.DryRun {
				result.JobsDeleted += len(toDelete)
				for _, id := range toDelete {
					size, _ := e.store.GetJobStorageSize(ctx, id)
					result.SpaceReclaimedMB += (size + 1024*1024 - 1) / (1024 * 1024)
				}
			} else {
				deleted, spaceMB, failedIDs, err := e.store.DeleteJobsWithArtifactsBatch(ctx, toDelete)
				if err != nil {
					result.Errors = append(result.Errors, err)
				}
				result.JobsDeleted += deleted
				result.SpaceReclaimedMB += spaceMB
				if len(failedIDs) > 0 {
					result.FailedJobIDs = append(result.FailedJobIDs, failedIDs...)
					e.logger.Warn("artifact deletions failed",
						"count", len(failedIDs),
						"jobIDs", failedIDs,
					)
				}
			}

			offset += len(jobs)
			if len(jobs) < 100 {
				break
			}
		}
	}

	return result, nil
}

// cleanupByCount deletes oldest jobs when total count exceeds limit.
func (e *Engine) cleanupByCount(ctx context.Context, opts CleanupOptions) (CleanupResult, error) {
	result := CleanupResult{}

	// Get current count
	stats, err := e.store.GetStorageStats(ctx)
	if err != nil {
		return result, err
	}

	if stats.TotalJobs <= int64(e.cfg.RetentionMaxJobs) {
		return result, nil
	}

	toDeleteCount := int(stats.TotalJobs) - e.cfg.RetentionMaxJobs
	e.logger.Info("cleanup by count triggered", "current", stats.TotalJobs, "limit", e.cfg.RetentionMaxJobs, "toDelete", toDeleteCount)

	// Get oldest jobs (prioritize terminal statuses)
	deleted := 0
	offset := 0
	for deleted < toDeleteCount {
		batchSize := 100
		if toDeleteCount-deleted < batchSize {
			batchSize = toDeleteCount - deleted
		}

		// Get oldest jobs using offset
		jobs, err := e.store.ListJobsOlderThan(ctx, time.Now(), store.ListOptions{Limit: batchSize, Offset: offset})
		if err != nil {
			return result, err
		}

		if len(jobs) == 0 {
			break
		}

		// Filter by kind if specified
		var toDelete []string
		for _, job := range jobs {
			if opts.Kind != nil && job.Kind != *opts.Kind {
				continue
			}
			toDelete = append(toDelete, job.ID)
		}

		if len(toDelete) == 0 {
			offset += len(jobs)
			continue
		}

		if opts.DryRun {
			result.JobsDeleted += len(toDelete)
			for _, id := range toDelete {
				size, _ := e.store.GetJobStorageSize(ctx, id)
				result.SpaceReclaimedMB += (size + 1024*1024 - 1) / (1024 * 1024)
			}
		} else {
			d, spaceMB, failedIDs, err := e.store.DeleteJobsWithArtifactsBatch(ctx, toDelete)
			if err != nil {
				result.Errors = append(result.Errors, err)
			}
			result.JobsDeleted += d
			result.SpaceReclaimedMB += spaceMB
			if len(failedIDs) > 0 {
				result.FailedJobIDs = append(result.FailedJobIDs, failedIDs...)
				e.logger.Warn("artifact deletions failed",
					"count", len(failedIDs),
					"jobIDs", failedIDs,
				)
			}
		}

		deleted += len(toDelete)
		offset += len(jobs)
	}

	return result, nil
}

// cleanupByStorage deletes oldest jobs when storage exceeds limit.
func (e *Engine) cleanupByStorage(ctx context.Context, opts CleanupOptions) (CleanupResult, error) {
	result := CleanupResult{}

	// Get current storage
	stats, err := e.store.GetStorageStats(ctx)
	if err != nil {
		return result, err
	}

	maxStorageMB := int64(e.cfg.RetentionMaxStorageGB) * 1024
	if stats.TotalStorageMB <= maxStorageMB {
		return result, nil
	}

	targetReclaim := stats.TotalStorageMB - maxStorageMB
	e.logger.Info("cleanup by storage triggered", "currentMB", stats.TotalStorageMB, "limitGB", e.cfg.RetentionMaxStorageGB, "targetReclaimMB", targetReclaim)

	reclaimed := int64(0)
	offset := 0
	for reclaimed < targetReclaim {
		// Get oldest jobs
		jobs, err := e.store.ListJobsOlderThan(ctx, time.Now(), store.ListOptions{Limit: 100, Offset: offset})
		if err != nil {
			return result, err
		}
		if len(jobs) == 0 {
			break
		}

		// Filter by kind if specified
		var toDelete []string
		for _, job := range jobs {
			if opts.Kind != nil && job.Kind != *opts.Kind {
				continue
			}
			toDelete = append(toDelete, job.ID)
		}

		if len(toDelete) == 0 {
			offset += len(jobs)
			continue
		}

		if opts.DryRun {
			for _, id := range toDelete {
				size, _ := e.store.GetJobStorageSize(ctx, id)
				reclaimed += (size + 1024*1024 - 1) / (1024 * 1024)
				result.JobsDeleted++
				result.SpaceReclaimedMB += (size + 1024*1024 - 1) / (1024 * 1024)
			}
		} else {
			d, spaceMB, failedIDs, err := e.store.DeleteJobsWithArtifactsBatch(ctx, toDelete)
			if err != nil {
				result.Errors = append(result.Errors, err)
			}
			result.JobsDeleted += d
			result.SpaceReclaimedMB += spaceMB
			reclaimed += spaceMB
			if len(failedIDs) > 0 {
				result.FailedJobIDs = append(result.FailedJobIDs, failedIDs...)
				e.logger.Warn("artifact deletions failed",
					"count", len(failedIDs),
					"jobIDs", failedIDs,
				)
			}
		}
		offset += len(jobs)
	}

	return result, nil
}

// cleanupCrawlStates deletes old crawl states.
func (e *Engine) cleanupCrawlStates(ctx context.Context, opts CleanupOptions) (CleanupResult, error) {
	result := CleanupResult{}

	cutoff := time.Now().AddDate(0, 0, -e.cfg.RetentionCrawlStateDays)

	if opts.DryRun {
		// Count would-be deleted crawl states
		// Note: This is approximate since we don't have a count method
		// In dry-run mode we just report that crawl states would be cleaned
		e.logger.Info("crawl state cleanup would run", "olderThan", cutoff)
		return result, nil
	}

	deleted, err := e.store.DeleteCrawlStatesOlderThan(ctx, cutoff)
	if err != nil {
		return result, err
	}

	result.CrawlStatesDeleted = deleted
	e.logger.Info("crawl states cleaned", "deleted", deleted)

	return result, nil
}

// GetStatus returns current retention status.
func (e *Engine) GetStatus(ctx context.Context) (config.RetentionStatus, error) {
	stats, err := e.store.GetStorageStats(ctx)
	if err != nil {
		return config.RetentionStatus{}, err
	}

	// Calculate eligible jobs
	var eligible int64
	if e.cfg.RetentionJobDays > 0 {
		cutoff := time.Now().AddDate(0, 0, -e.cfg.RetentionJobDays)
		count, err := e.store.CountJobsOlderThan(ctx, cutoff)
		if err != nil {
			return config.RetentionStatus{}, err
		}
		eligible = count
	}

	status := config.RetentionStatus{
		Enabled:          e.cfg.RetentionEnabled,
		JobRetentionDays: e.cfg.RetentionJobDays,
		CrawlStateDays:   e.cfg.RetentionCrawlStateDays,
		MaxJobs:          e.cfg.RetentionMaxJobs,
		MaxStorageGB:     e.cfg.RetentionMaxStorageGB,
		TotalJobs:        stats.TotalJobs,
		JobsEligible:     eligible,
		StorageUsedMB:    stats.TotalStorageMB,
	}

	return status, nil
}

// EvaluatePolicies returns jobs that should be deleted based on policies.
// This is used for preview/dry-run purposes.
func (e *Engine) EvaluatePolicies(ctx context.Context, kind *model.Kind) ([]string, error) {
	if !e.cfg.RetentionEnabled {
		return nil, apperrors.Validation("retention is not enabled")
	}

	var toDelete []string
	seen := make(map[string]bool)

	// Age-based evaluation
	if e.cfg.RetentionJobDays > 0 {
		cutoff := time.Now().AddDate(0, 0, -e.cfg.RetentionJobDays)
		for {
			jobs, err := e.store.ListJobsOlderThan(ctx, cutoff, store.ListOptions{Limit: 100})
			if err != nil {
				return nil, err
			}
			if len(jobs) == 0 {
				break
			}

			for _, job := range jobs {
				if kind != nil && job.Kind != *kind {
					continue
				}
				if !seen[job.ID] {
					toDelete = append(toDelete, job.ID)
					seen[job.ID] = true
				}
			}

			if len(jobs) < 100 {
				break
			}
		}
	}

	return toDelete, nil
}

// FormatResult returns a human-readable summary of cleanup results.
func FormatResult(result CleanupResult, dryRun bool) string {
	mode := "Deleted"
	if dryRun {
		mode = "Would delete"
	}

	msg := fmt.Sprintf("%s %d jobs, %d crawl states, reclaimed %d MB in %v",
		mode, result.JobsDeleted, result.CrawlStatesDeleted, result.SpaceReclaimedMB, result.Duration)

	if len(result.FailedJobIDs) > 0 {
		msg += fmt.Sprintf(" (%d artifact deletions failed)", len(result.FailedJobIDs))
	}

	return msg
}
