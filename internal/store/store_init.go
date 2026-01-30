// Package store provides SQLite database initialization and connection management.
//
// This file is responsible for:
// - Opening and initializing the SQLite database
// - Running schema migrations (table creation, column additions)
// - Preparing SQL statements for reuse
// - Connection pool configuration (max conns, timeouts)
// - WAL mode and checkpointing for durability
// - Store lifecycle (Open, Close, Checkpoint)
//
// This file does NOT handle:
// - Job CRUD operations (store_jobs.go handles this)
// - Crawl state operations (store_crawl_states.go handles this)
// - Business logic
//
// Invariants:
// - Uses SQLite with WAL mode (journal_mode=WAL)
// - Database file is jobs.db in the data directory
// - All prepared statements are closed on Close()
// - columnExists helper used for safe migrations
package store

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/fsutil"

	_ "modernc.org/sqlite"
)

// Open creates and initializes a new Store at the given data directory.
// Creates the database file if it doesn't exist, runs migrations, and prepares statements.
func Open(dataDir string) (*Store, error) {
	if err := fsutil.EnsureDataDir(dataDir); err != nil {
		return nil, err
	}
	path := filepath.Join(dataDir, "jobs.db")
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)", url.PathEscape(path))
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(1 * time.Hour)
	db.SetConnMaxIdleTime(30 * time.Minute)

	store := &Store{db: db, dataDir: dataDir}
	if err := store.init(); err != nil {
		return nil, err
	}

	if err := store.prepareStatements(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *Store) prepareStatements() error {
	var err error
	s.insertJobStmt, err = s.db.Prepare(`insert into jobs (id, kind, status, created_at, updated_at, params, result_path, error)
			values (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare insert job statement", err)
	}

	s.updateJobStatusStmt, err = s.db.Prepare(`update jobs set status = ?, updated_at = ?, error = ? where id = ?`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare update job status statement", err)
	}

	s.getJobStmt, err = s.db.Prepare(`select id, kind, status, created_at, updated_at, params, result_path, error from jobs where id = ?`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare get job statement", err)
	}

	s.getCrawlStateStmt, err = s.db.Prepare(`select url, etag, last_modified, content_hash, last_scraped, depth, job_id from crawl_states where url = ?`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare get crawl state statement", err)
	}

	s.upsertCrawlStateStmt, err = s.db.Prepare(`insert into crawl_states (url, etag, last_modified, content_hash, last_scraped, depth, job_id)
		values (?, ?, ?, ?, ?, ?, ?)
		on conflict(url) do update set
			etag = excluded.etag,
			last_modified = excluded.last_modified,
			content_hash = excluded.content_hash,
			last_scraped = excluded.last_scraped,
			depth = excluded.depth,
			job_id = excluded.job_id`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare upsert crawl state statement", err)
	}

	s.deleteCrawlStateStmt, err = s.db.Prepare(`delete from crawl_states where url = ?`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare delete crawl state statement", err)
	}

	s.deleteAllCrawlStatesStmt, err = s.db.Prepare(`delete from crawl_states`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare delete all crawl states statement", err)
	}

	return nil
}

// Ping checks if the database connection is alive.
func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func columnExists(db *sql.DB, tableName, columnName string) (bool, error) {
	query := fmt.Sprintf("PRAGMA table_info(%s)", tableName)
	rows, err := db.Query(query)
	if err != nil {
		return false, apperrors.Wrap(apperrors.KindInternal,
			fmt.Sprintf("failed to query table schema for %s", tableName), err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, datatype string
		var notnull, pk int
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &datatype, &notnull, &dfltValue, &pk); err != nil {
			return false, apperrors.Wrap(apperrors.KindInternal,
				"failed to scan table_info row", err)
		}
		if name == columnName {
			return true, nil
		}
	}

	if err := rows.Err(); err != nil {
		return false, apperrors.Wrap(apperrors.KindInternal,
			"error iterating table_info results", err)
	}

	return false, nil
}

func (s *Store) init() error {
	_, err := s.db.Exec(`
		create table if not exists jobs (
			id text primary key,
			kind text not null,
			status text not null,
			created_at text not null,
			updated_at text not null,
			params text,
			result_path text,
			error text
		);
		create index if not exists idx_jobs_status_created on jobs(status, created_at desc);
		create index if not exists idx_jobs_created on jobs(created_at desc);

		create table if not exists crawl_states (
			url text primary key,
			etag text,
			last_modified text,
			content_hash text,
			last_scraped text,
			depth integer,
			job_id text
		);
		create index if not exists idx_crawl_states_last_scraped on crawl_states(last_scraped desc);
	`)
	if err != nil {
		return err
	}

	depthExists, err := columnExists(s.db, "crawl_states", "depth")
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal,
			"failed to check for depth column migration", err)
	}
	if !depthExists {
		_, err = s.db.Exec("alter table crawl_states add column depth integer")
		if err != nil {
			return apperrors.Wrap(apperrors.KindInternal,
				"failed to add depth column to crawl_states", err)
		}
	}

	jobIDExists, err := columnExists(s.db, "crawl_states", "job_id")
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal,
			"failed to check for job_id column migration", err)
	}
	if !jobIDExists {
		_, err = s.db.Exec("alter table crawl_states add column job_id text")
		if err != nil {
			return apperrors.Wrap(apperrors.KindInternal,
				"failed to add job_id column to crawl_states", err)
		}
	}

	return nil
}

// Close closes the database connection and all prepared statements.
// Attempts to checkpoint the WAL before closing.
func (s *Store) Close() error {
	_, _ = s.db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")

	if s.insertJobStmt != nil {
		s.insertJobStmt.Close()
	}
	if s.updateJobStatusStmt != nil {
		s.updateJobStatusStmt.Close()
	}
	if s.getJobStmt != nil {
		s.getJobStmt.Close()
	}
	if s.getCrawlStateStmt != nil {
		s.getCrawlStateStmt.Close()
	}
	if s.upsertCrawlStateStmt != nil {
		s.upsertCrawlStateStmt.Close()
	}
	if s.deleteCrawlStateStmt != nil {
		s.deleteCrawlStateStmt.Close()
	}
	if s.deleteAllCrawlStatesStmt != nil {
		s.deleteAllCrawlStatesStmt.Close()
	}

	return s.db.Close()
}

// Checkpoint checkpoints the WAL file to the main database.
func (s *Store) Checkpoint(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "PRAGMA wal_checkpoint(PASSIVE)")
	return err
}

// DataDir returns the data directory path.
func (s *Store) DataDir() string {
	return s.dataDir
}
