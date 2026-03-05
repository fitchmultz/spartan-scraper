// Package dedup provides SQLite-backed implementation of the ContentIndex interface.
//
// This file implements persistent storage for content fingerprints using SQLite.
// It uses the existing store.DB connection and follows the same patterns as
// other store implementations in the codebase.
//
// Performance notes:
//   - Index operations are O(1) with prepared statements
//   - FindDuplicates is O(N) where N = total fingerprints (brute force)
//     Future optimization: simhash bucketing or LSH for sub-linear lookup
//   - Storage: ~50 bytes per fingerprint
package dedup

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/simhash"
)

// SQLiteIndex implements ContentIndex using SQLite.
type SQLiteIndex struct {
	db *sql.DB

	// Prepared statements
	stmtIndex       *sql.Stmt
	stmtFindAll     *sql.Stmt
	stmtGetHistory  *sql.Stmt
	stmtDeleteByJob *sql.Stmt
	stmtStats       *sql.Stmt
}

// NewSQLiteIndex creates a new SQLite-backed content index.
// The db parameter should be the same database connection used by the store package.
func NewSQLiteIndex(db *sql.DB) (*SQLiteIndex, error) {
	idx := &SQLiteIndex{db: db}
	if err := idx.init(); err != nil {
		return nil, err
	}
	if err := idx.prepareStatements(); err != nil {
		return nil, err
	}
	return idx, nil
}

// init creates the content_fingerprints table if it doesn't exist.
func (idx *SQLiteIndex) init() error {
	_, err := idx.db.Exec(`
		CREATE TABLE IF NOT EXISTS content_fingerprints (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			job_id TEXT NOT NULL,
			url TEXT NOT NULL,
			simhash INTEGER NOT NULL,
			indexed_at TEXT NOT NULL,
			UNIQUE(job_id, url)
		);
		CREATE INDEX IF NOT EXISTS idx_content_fingerprints_simhash ON content_fingerprints(simhash);
		CREATE INDEX IF NOT EXISTS idx_content_fingerprints_job_id ON content_fingerprints(job_id);
		CREATE INDEX IF NOT EXISTS idx_content_fingerprints_url ON content_fingerprints(url);
	`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to create content_fingerprints table", err)
	}
	return nil
}

// prepareStatements prepares SQL statements for reuse.
func (idx *SQLiteIndex) prepareStatements() error {
	var err error

	idx.stmtIndex, err = idx.db.Prepare(`
		INSERT INTO content_fingerprints (job_id, url, simhash, indexed_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(job_id, url) DO UPDATE SET
			simhash = excluded.simhash,
			indexed_at = excluded.indexed_at
	`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare index statement", err)
	}

	idx.stmtFindAll, err = idx.db.Prepare(`
		SELECT job_id, url, simhash, indexed_at
		FROM content_fingerprints
		ORDER BY indexed_at DESC
	`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare find all statement", err)
	}

	idx.stmtGetHistory, err = idx.db.Prepare(`
		SELECT job_id, simhash, indexed_at
		FROM content_fingerprints
		WHERE url = ?
		ORDER BY indexed_at DESC
	`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare get history statement", err)
	}

	idx.stmtDeleteByJob, err = idx.db.Prepare(`
		DELETE FROM content_fingerprints WHERE job_id = ?
	`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare delete by job statement", err)
	}

	idx.stmtStats, err = idx.db.Prepare(`
		SELECT
			(SELECT COUNT(*) FROM content_fingerprints) as total_indexed,
			(SELECT COUNT(DISTINCT url) FROM content_fingerprints) as unique_urls,
			(SELECT COUNT(DISTINCT job_id) FROM content_fingerprints) as unique_jobs
	`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare stats statement", err)
	}

	return nil
}

// Close closes all prepared statements.
func (idx *SQLiteIndex) Close() error {
	if idx.stmtIndex != nil {
		idx.stmtIndex.Close()
	}
	if idx.stmtFindAll != nil {
		idx.stmtFindAll.Close()
	}
	if idx.stmtGetHistory != nil {
		idx.stmtGetHistory.Close()
	}
	if idx.stmtDeleteByJob != nil {
		idx.stmtDeleteByJob.Close()
	}
	if idx.stmtStats != nil {
		idx.stmtStats.Close()
	}
	return nil
}

// Index stores a content fingerprint for a URL.
func (idx *SQLiteIndex) Index(ctx context.Context, jobID, url string, simhash uint64) error {
	if jobID == "" {
		return apperrors.Validation("job_id is required")
	}
	if url == "" {
		return apperrors.Validation("url is required")
	}

	indexedAt := time.Now().UTC().Format(time.RFC3339)
	_, err := idx.stmtIndex.ExecContext(ctx, jobID, url, int64(simhash), indexedAt)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to index content fingerprint", err)
	}
	return nil
}

// FindDuplicates returns URLs with similar content across all jobs.
func (idx *SQLiteIndex) FindDuplicates(ctx context.Context, targetHash uint64, threshold int) ([]DuplicateMatch, error) {
	if threshold < 0 {
		threshold = ThresholdNear
	}

	// Fetch all entries - brute force approach
	// Future optimization: use simhash bucketing or LSH
	rows, err := idx.stmtFindAll.QueryContext(ctx)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to query content fingerprints", err)
	}
	defer rows.Close()

	var matches []DuplicateMatch
	for rows.Next() {
		var jobID, url string
		var simhashVal int64
		var indexedAtStr string

		if err := rows.Scan(&jobID, &url, &simhashVal, &indexedAtStr); err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to scan content fingerprint row", err)
		}

		distance := simhash.HammingDistance(targetHash, uint64(simhashVal))
		if distance <= threshold {
			indexedAt, _ := time.Parse(time.RFC3339, indexedAtStr)
			matches = append(matches, DuplicateMatch{
				JobID:     jobID,
				URL:       url,
				SimHash:   uint64(simhashVal),
				Distance:  distance,
				IndexedAt: indexedAt,
			})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "error iterating content fingerprint rows", err)
	}

	return matches, nil
}

// GetContentHistory returns all indexed entries for a URL across jobs.
func (idx *SQLiteIndex) GetContentHistory(ctx context.Context, url string) ([]ContentEntry, error) {
	if url == "" {
		return nil, apperrors.Validation("url is required")
	}

	rows, err := idx.stmtGetHistory.QueryContext(ctx, url)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to query content history", err)
	}
	defer rows.Close()

	var entries []ContentEntry
	for rows.Next() {
		var jobID string
		var simhashVal int64
		var indexedAtStr string

		if err := rows.Scan(&jobID, &simhashVal, &indexedAtStr); err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to scan content history row", err)
		}

		indexedAt, _ := time.Parse(time.RFC3339, indexedAtStr)
		entries = append(entries, ContentEntry{
			JobID:     jobID,
			SimHash:   uint64(simhashVal),
			IndexedAt: indexedAt,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "error iterating content history rows", err)
	}

	return entries, nil
}

// DeleteJobEntries removes all entries for a job.
func (idx *SQLiteIndex) DeleteJobEntries(ctx context.Context, jobID string) (int64, error) {
	if jobID == "" {
		return 0, apperrors.Validation("job_id is required")
	}

	result, err := idx.stmtDeleteByJob.ExecContext(ctx, jobID)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.KindInternal, "failed to delete job entries", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, apperrors.Wrap(apperrors.KindInternal, "failed to get rows affected", err)
	}

	return rowsAffected, nil
}

// Stats returns deduplication statistics.
func (idx *SQLiteIndex) Stats(ctx context.Context) (Stats, error) {
	var stats Stats
	var err error

	err = idx.stmtStats.QueryRowContext(ctx).Scan(&stats.TotalIndexed, &stats.UniqueURLs, &stats.UniqueJobs)
	if err != nil {
		return stats, apperrors.Wrap(apperrors.KindInternal, "failed to query dedup stats", err)
	}

	// Count duplicate pairs by finding entries with the same URL but different job IDs
	// This is an approximation - true duplicate pairs would require comparing all simhashes
	stats.DuplicatePairs, err = idx.countDuplicatePairs(ctx)
	if err != nil {
		// Non-fatal - just log and continue with 0
		stats.DuplicatePairs = 0
	}

	return stats, nil
}

// countDuplicatePairs counts URLs that appear in multiple jobs.
func (idx *SQLiteIndex) countDuplicatePairs(ctx context.Context) (int64, error) {
	query := `
		SELECT COUNT(*) FROM (
			SELECT url, COUNT(DISTINCT job_id) as job_count
			FROM content_fingerprints
			GROUP BY url
			HAVING job_count > 1
		)
	`
	var count int64
	err := idx.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// FindDuplicatesByURL finds duplicates for a given URL by looking up its simhash.
// Returns matches that have the same/similar content.
// Note: This returns ALL matches including the queried URL itself.
// Callers should filter by job ID if they want to exclude specific jobs.
func (idx *SQLiteIndex) FindDuplicatesByURL(ctx context.Context, url string, threshold int) ([]DuplicateMatch, error) {
	if url == "" {
		return nil, apperrors.Validation("url is required")
	}

	// Get the most recent simhash for this URL
	history, err := idx.GetContentHistory(ctx, url)
	if err != nil {
		return nil, err
	}
	if len(history) == 0 {
		return nil, nil
	}

	// Use the most recent simhash
	latestHash := history[0].SimHash
	matches, err := idx.FindDuplicates(ctx, latestHash, threshold)
	if err != nil {
		return nil, err
	}

	return matches, nil
}

// String returns a string representation of a DuplicateMatch for debugging.
func (m DuplicateMatch) String() string {
	return fmt.Sprintf("DuplicateMatch{JobID: %s, URL: %s, Distance: %d}", m.JobID, m.URL, m.Distance)
}

// String returns a string representation of a ContentEntry for debugging.
func (e ContentEntry) String() string {
	return fmt.Sprintf("ContentEntry{JobID: %s, SimHash: %d}", e.JobID, e.SimHash)
}
