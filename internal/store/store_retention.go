// Package store provides retention-related database operations.
//
// This file is responsible for:
// - Querying jobs by age, status, and count for retention decisions
// - Batch deletion of jobs and their artifacts
// - Storage size calculations
// - Crawl state cleanup by age
//
// This file does NOT handle:
// - Policy evaluation (retention package handles this)
// - Scheduled cleanup execution (scheduler handles this)
//
// Invariants:
// - Uses transactions for batch operations
// - Reuses path traversal protection from DeleteWithArtifacts
// - Returns accurate counts of deleted items and reclaimed space
package store

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

// RetentionStats holds statistics for retention decisions.
type RetentionStats struct {
	TotalJobs      int64
	JobsByStatus   map[model.Status]int64
	JobsOlderThan  map[int]int64 // days -> count
	TotalStorageMB int64
	OldestJobAge   time.Duration
}

// ListJobsOlderThan returns jobs created before the given time, ordered by created_at ASC.
// This returns oldest jobs first, useful for cleanup prioritization.
func (s *Store) ListJobsOlderThan(ctx context.Context, before time.Time, opts ListOptions) ([]model.Job, error) {
	opts = opts.Defaults()
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, kind, status, created_at, updated_at, params, result_path, error, depends_on, dependency_status, chain_id
		 FROM jobs WHERE created_at < ? ORDER BY created_at ASC LIMIT ? OFFSET ?`,
		before.Format(time.RFC3339Nano), opts.Limit, opts.Offset)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to query jobs older than", err)
	}
	defer rows.Close()

	return s.scanJobsWithDependencies(rows)
}

// ListJobsByStatusAndAge returns jobs with given status created before the given time.
func (s *Store) ListJobsByStatusAndAge(ctx context.Context, status model.Status, before time.Time, opts ListOptions) ([]model.Job, error) {
	opts = opts.Defaults()
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, kind, status, created_at, updated_at, params, result_path, error, depends_on, dependency_status, chain_id
		 FROM jobs WHERE status = ? AND created_at < ? ORDER BY created_at ASC LIMIT ? OFFSET ?`,
		status, before.Format(time.RFC3339Nano), opts.Limit, opts.Offset)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to query jobs by status and age", err)
	}
	defer rows.Close()

	return s.scanJobsWithDependencies(rows)
}

// CountJobsOlderThan returns count of jobs older than given time.
func (s *Store) CountJobsOlderThan(ctx context.Context, before time.Time) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM jobs WHERE created_at < ?`,
		before.Format(time.RFC3339Nano)).Scan(&count)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.KindInternal, "failed to count jobs older than", err)
	}
	return count, nil
}

// CountJobsByStatus returns count of jobs with the given status.
func (s *Store) CountJobsByStatus(ctx context.Context, status model.Status) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM jobs WHERE status = ?`, status).Scan(&count)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.KindInternal, "failed to count jobs by status", err)
	}
	return count, nil
}

// GetStorageStats returns storage usage statistics.
func (s *Store) GetStorageStats(ctx context.Context) (RetentionStats, error) {
	stats := RetentionStats{
		JobsByStatus:  make(map[model.Status]int64),
		JobsOlderThan: make(map[int]int64),
	}

	// Total jobs
	var total int64
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM jobs`).Scan(&total)
	if err != nil {
		return stats, apperrors.Wrap(apperrors.KindInternal, "failed to count total jobs", err)
	}
	stats.TotalJobs = total

	// Jobs by status
	for _, status := range model.ValidStatuses() {
		count, err := s.CountJobsByStatus(ctx, status)
		if err != nil {
			return stats, err
		}
		stats.JobsByStatus[status] = count
	}

	// Jobs older than thresholds
	for _, days := range []int{7, 30, 90, 180, 365} {
		before := time.Now().AddDate(0, 0, -days)
		count, err := s.CountJobsOlderThan(ctx, before)
		if err != nil {
			return stats, err
		}
		stats.JobsOlderThan[days] = count
	}

	// Total storage
	storageMB, err := s.getTotalStorageSizeMB()
	if err != nil {
		return stats, err
	}
	stats.TotalStorageMB = storageMB

	// Oldest job age
	oldest, err := s.getOldestJobTime(ctx)
	if err != nil {
		return stats, err
	}
	if !oldest.IsZero() {
		stats.OldestJobAge = time.Since(oldest)
	}

	return stats, nil
}

// getTotalStorageSizeMB calculates total storage size of all job artifacts in MB.
func (s *Store) getTotalStorageSizeMB() (int64, error) {
	jobsDir := filepath.Join(s.dataDir, "jobs")
	var totalBytes int64

	err := filepath.WalkDir(jobsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Directory might not exist, which is fine
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			totalBytes += info.Size()
		}
		return nil
	})

	if err != nil {
		return 0, apperrors.Wrap(apperrors.KindInternal, "failed to calculate storage size", err)
	}

	// Convert to MB (round up)
	return (totalBytes + 1024*1024 - 1) / (1024 * 1024), nil
}

// getOldestJobTime returns the creation time of the oldest job.
func (s *Store) getOldestJobTime(ctx context.Context) (time.Time, error) {
	var createdAt string
	err := s.db.QueryRowContext(ctx,
		`SELECT created_at FROM jobs ORDER BY created_at ASC LIMIT 1`).Scan(&createdAt)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return time.Time{}, nil
		}
		return time.Time{}, apperrors.Wrap(apperrors.KindInternal, "failed to get oldest job time", err)
	}
	return time.Parse(time.RFC3339Nano, createdAt)
}

// DeleteJobsBatch deletes multiple jobs by ID in a transaction.
// Returns the number of jobs actually deleted.
func (s *Store) DeleteJobsBatch(ctx context.Context, ids []string) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.KindInternal, "failed to begin transaction", err)
	}
	defer tx.Rollback()

	// Build placeholders for IN clause
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	result, err := tx.ExecContext(ctx,
		fmt.Sprintf(`DELETE FROM jobs WHERE id IN (%s)`, strings.Join(placeholders, ",")),
		args...)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.KindInternal, "failed to delete jobs batch", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, apperrors.Wrap(apperrors.KindInternal, "failed to commit transaction", err)
	}

	deleted, _ := result.RowsAffected()
	return int(deleted), nil
}

// DeleteJobsWithArtifactsBatch deletes jobs and their artifacts in batch.
// Returns the number of jobs deleted, actual MB of space reclaimed, and any job IDs whose artifacts failed to delete.
// Note: Jobs are deleted from the database even if their artifact deletion fails.
func (s *Store) DeleteJobsWithArtifactsBatch(ctx context.Context, ids []string) (deletedCount int, spaceReclaimedMB int64, failedIDs []string, err error) {
	if len(ids) == 0 {
		return 0, 0, nil, nil
	}

	// Delete jobs from database first
	deleted, err := s.DeleteJobsBatch(ctx, ids)
	if err != nil {
		return 0, 0, nil, err
	}

	// Delete artifacts and track actual space reclaimed and failures
	var totalBytes int64
	for _, id := range ids {
		// Get size before attempting deletion for accurate tracking
		size, err := s.GetJobStorageSize(ctx, id)
		if err != nil {
			// Job might not have artifacts - this is not a failure
			size = 0
		}

		if err := s.deleteJobArtifacts(id); err != nil {
			// Artifact deletion failed - track the ID but don't fail the whole operation
			failedIDs = append(failedIDs, id)
		} else {
			// Artifact deletion succeeded - add to reclaimed space
			totalBytes += size
		}
	}

	spaceReclaimedMB = (totalBytes + 1024*1024 - 1) / (1024 * 1024)
	return deleted, spaceReclaimedMB, failedIDs, nil
}

// deleteJobArtifacts removes the artifact directory for a job.
// Uses path traversal protection.
func (s *Store) deleteJobArtifacts(jobID string) error {
	jobDir := filepath.Join(s.dataDir, "jobs", jobID)
	cleanPath := filepath.Clean(jobDir)
	baseDir := filepath.Clean(filepath.Join(s.dataDir, "jobs"))

	// Ensure the cleaned path is within the jobs directory
	if !strings.HasPrefix(cleanPath, baseDir+string(filepath.Separator)) && cleanPath != baseDir {
		return apperrors.Permission("invalid job id: path traversal detected")
	}

	if err := os.RemoveAll(cleanPath); err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to remove job artifacts", err)
	}

	return nil
}

// GetJobStorageSize returns the size of a job's artifacts in bytes.
func (s *Store) GetJobStorageSize(ctx context.Context, jobID string) (int64, error) {
	jobDir := filepath.Join(s.dataDir, "jobs", jobID)
	cleanPath := filepath.Clean(jobDir)
	baseDir := filepath.Clean(filepath.Join(s.dataDir, "jobs"))

	// Ensure the cleaned path is within the jobs directory
	if !strings.HasPrefix(cleanPath, baseDir+string(filepath.Separator)) && cleanPath != baseDir {
		return 0, apperrors.Permission("invalid job id: path traversal detected")
	}

	var totalBytes int64
	err := filepath.WalkDir(cleanPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			totalBytes += info.Size()
		}
		return nil
	})

	if err != nil {
		return 0, apperrors.Wrap(apperrors.KindInternal, "failed to calculate job storage size", err)
	}

	return totalBytes, nil
}

// DeleteCrawlStatesOlderThan deletes crawl states not updated since given time.
// Returns the number of crawl states deleted.
func (s *Store) DeleteCrawlStatesOlderThan(ctx context.Context, before time.Time) (int64, error) {
	result, err := s.db.ExecContext(ctx,
		`DELETE FROM crawl_states WHERE last_scraped < ? OR (last_scraped IS NULL AND rowid IN (
			SELECT rowid FROM crawl_states WHERE last_scraped IS NULL LIMIT 1000
		))`,
		before.Format(time.RFC3339Nano))
	if err != nil {
		return 0, apperrors.Wrap(apperrors.KindInternal, "failed to delete old crawl states", err)
	}

	deleted, _ := result.RowsAffected()
	return deleted, nil
}

// ListJobsByKind returns jobs filtered by kind, ordered by created_at ASC.
func (s *Store) ListJobsByKind(ctx context.Context, kind model.Kind, opts ListOptions) ([]model.Job, error) {
	opts = opts.Defaults()
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, kind, status, created_at, updated_at, params, result_path, error, depends_on, dependency_status, chain_id
		 FROM jobs WHERE kind = ? ORDER BY created_at ASC LIMIT ? OFFSET ?`,
		kind, opts.Limit, opts.Offset)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to query jobs by kind", err)
	}
	defer rows.Close()

	return s.scanJobsWithDependencies(rows)
}

// CountJobsByKind returns the count of jobs with the given kind.
func (s *Store) CountJobsByKind(ctx context.Context, kind model.Kind) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM jobs WHERE kind = ?`, kind).Scan(&count)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.KindInternal, "failed to count jobs by kind", err)
	}
	return count, nil
}
