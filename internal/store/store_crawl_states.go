// Package store provides persistent storage for crawl states used in deduplication.
//
// This file is responsible for:
// - Crawl state CRUD operations (Get, Upsert, Delete)
// - Crawl state listing with pagination
// - Counting crawl states for monitoring
// - SQLite UPSERT for efficient insert-or-update
//
// This file does NOT handle:
// - Crawl logic or URL discovery (crawl package handles this)
// - Job operations (store_jobs.go handles this)
// - Store initialization (store_init.go handles this)
//
// Invariants:
// - Returns empty state (not error) when crawl state not found
// - Timestamps are stored as RFC3339Nano strings
// - Uses prepared statements for all operations
// - URL is the primary key for crawl states
package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

// GetCrawlState retrieves the crawl state for a given URL.
// Returns nil state if not found (no error).
func (s *Store) GetCrawlState(ctx context.Context, url string) (model.CrawlState, error) {
	row := s.getCrawlStateStmt.QueryRowContext(ctx, url)
	var state model.CrawlState
	var lastScraped string
	if err := row.Scan(&state.URL, &state.ETag, &state.LastModified, &state.ContentHash, &lastScraped, &state.Depth, &state.JobID, &state.PreviousContent, &state.ContentSnapshot); err != nil {
		if err == sql.ErrNoRows || err.Error() == "sql: no rows in result set" {
			return model.CrawlState{}, nil
		}
		return model.CrawlState{}, err
	}
	if lastScraped != "" {
		var err error
		state.LastScraped, err = time.Parse(time.RFC3339Nano, lastScraped)
		if err != nil {
			return model.CrawlState{}, apperrors.Wrap(apperrors.KindInternal, "failed to parse crawl state last_scraped", err)
		}
	}
	return state, nil
}

// UpsertCrawlState inserts or updates the crawl state for a URL.
// Uses SQLite UPSERT (insert on conflict do update).
func (s *Store) UpsertCrawlState(ctx context.Context, state model.CrawlState) error {
	_, err := s.upsertCrawlStateStmt.ExecContext(
		ctx,
		state.URL,
		state.ETag,
		state.LastModified,
		state.ContentHash,
		state.LastScraped.Format(time.RFC3339Nano),
		state.Depth,
		state.JobID,
		state.PreviousContent,
		state.ContentSnapshot,
	)
	return err
}

// DeleteCrawlState removes a specific crawl state by URL.
func (s *Store) DeleteCrawlState(ctx context.Context, url string) error {
	_, err := s.deleteCrawlStateStmt.ExecContext(ctx, url)
	return err
}

// DeleteAllCrawlStates removes all crawl states from the store.
func (s *Store) DeleteAllCrawlStates(ctx context.Context) error {
	_, err := s.deleteAllCrawlStatesStmt.ExecContext(ctx)
	return err
}

// ListCrawlStates returns all crawl states, ordered by last_scraped DESC.
// If no options are provided, it uses safe defaults (limit 100, offset 0).
func (s *Store) ListCrawlStates(ctx context.Context, opts ListCrawlStatesOptions) ([]model.CrawlState, error) {
	opts = opts.Defaults()
	rows, err := s.db.QueryContext(ctx,
		`select url, etag, last_modified, content_hash, last_scraped, depth, job_id, previous_content, content_snapshot
		 from crawl_states order by last_scraped desc limit ? offset ?`,
		opts.Limit, opts.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []model.CrawlState{}
	for rows.Next() {
		var state model.CrawlState
		var lastScraped string
		if err := rows.Scan(&state.URL, &state.ETag, &state.LastModified,
			&state.ContentHash, &lastScraped, &state.Depth, &state.JobID, &state.PreviousContent, &state.ContentSnapshot); err != nil {
			return nil, err
		}
		if lastScraped != "" {
			var parseErr error
			state.LastScraped, parseErr = time.Parse(time.RFC3339Nano, lastScraped)
			if parseErr != nil {
				return nil, apperrors.Wrap(apperrors.KindInternal, "failed to parse crawl state last_scraped", parseErr)
			}
		}
		results = append(results, state)
	}
	return results, rows.Err()
}

// CountCrawlStates returns the total number of crawl states.
func (s *Store) CountCrawlStates(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, "select count(*) from crawl_states").Scan(&count)
	return count, err
}
