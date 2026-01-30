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
	s.insertJobStmt, err = s.db.Prepare(`insert into jobs (id, kind, status, created_at, updated_at, params, result_path, error, depends_on, dependency_status, chain_id)
			values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare insert job statement", err)
	}

	s.updateJobStatusStmt, err = s.db.Prepare(`update jobs set status = ?, updated_at = ?, error = ? where id = ?`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare update job status statement", err)
	}

	s.getJobStmt, err = s.db.Prepare(`select id, kind, status, created_at, updated_at, params, result_path, error, depends_on, dependency_status, chain_id from jobs where id = ?`)
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

	// Chain statements
	s.stmtCreateChain, err = s.db.Prepare(`insert into job_chains (id, name, description, definition, created_at, updated_at) values (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare create chain statement", err)
	}

	s.stmtGetChain, err = s.db.Prepare(`select id, name, description, definition, created_at, updated_at from job_chains where id = ?`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare get chain statement", err)
	}

	s.stmtGetChainByName, err = s.db.Prepare(`select id, name, description, definition, created_at, updated_at from job_chains where name = ?`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare get chain by name statement", err)
	}

	s.stmtUpdateChain, err = s.db.Prepare(`update job_chains set name = ?, description = ?, definition = ?, updated_at = ? where id = ?`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare update chain statement", err)
	}

	s.stmtDeleteChain, err = s.db.Prepare(`delete from job_chains where id = ?`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare delete chain statement", err)
	}

	s.stmtListChains, err = s.db.Prepare(`select id, name, description, definition, created_at, updated_at from job_chains order by created_at desc`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare list chains statement", err)
	}

	// Dependency statements
	s.stmtGetJobsByDependencyStatus, err = s.db.Prepare(`select id, kind, status, created_at, updated_at, params, result_path, error, depends_on, dependency_status, chain_id from jobs where dependency_status = ? order by created_at desc`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare get jobs by dependency status statement", err)
	}

	s.stmtUpdateDependencyStatus, err = s.db.Prepare(`update jobs set dependency_status = ? where id = ?`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare update dependency status statement", err)
	}

	s.stmtGetDependentJobs, err = s.db.Prepare(`select id, kind, status, created_at, updated_at, params, result_path, error, depends_on, dependency_status, chain_id from jobs where depends_on like ?`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare get dependent jobs statement", err)
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

	// Initialize batch tables
	if err := s.initBatchTables(); err != nil {
		return err
	}

	// Initialize dependency and chain tables
	if err := s.initDependencyTables(); err != nil {
		return err
	}

	return nil
}

// initBatchTables creates the batch-related tables if they don't exist.
// This is called during store initialization.
func (s *Store) initBatchTables() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS batches (
			id TEXT PRIMARY KEY,
			kind TEXT NOT NULL,
			status TEXT NOT NULL,
			job_count INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_batches_status ON batches(status);
		CREATE INDEX IF NOT EXISTS idx_batches_created ON batches(created_at DESC);

		CREATE TABLE IF NOT EXISTS batch_jobs (
			batch_id TEXT NOT NULL,
			job_id TEXT NOT NULL PRIMARY KEY,
			idx INTEGER NOT NULL,
			FOREIGN KEY (batch_id) REFERENCES batches(id) ON DELETE CASCADE,
			FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE
		);
		CREATE INDEX IF NOT EXISTS idx_batch_jobs_batch_id ON batch_jobs(batch_id);
	`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to create batch tables", err)
	}
	return nil
}

// initDependencyTables creates the dependency and chain-related tables.
// This is called during store initialization.
func (s *Store) initDependencyTables() error {
	// Add dependency columns to jobs table
	dependsOnExists, err := columnExists(s.db, "jobs", "depends_on")
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to check for depends_on column", err)
	}
	if !dependsOnExists {
		_, err = s.db.Exec("ALTER TABLE jobs ADD COLUMN depends_on TEXT")
		if err != nil {
			return apperrors.Wrap(apperrors.KindInternal, "failed to add depends_on column", err)
		}
	}

	depStatusExists, err := columnExists(s.db, "jobs", "dependency_status")
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to check for dependency_status column", err)
	}
	if !depStatusExists {
		_, err = s.db.Exec("ALTER TABLE jobs ADD COLUMN dependency_status TEXT DEFAULT 'ready'")
		if err != nil {
			return apperrors.Wrap(apperrors.KindInternal, "failed to add dependency_status column", err)
		}
	}

	chainIDExists, err := columnExists(s.db, "jobs", "chain_id")
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to check for chain_id column", err)
	}
	if !chainIDExists {
		_, err = s.db.Exec("ALTER TABLE jobs ADD COLUMN chain_id TEXT")
		if err != nil {
			return apperrors.Wrap(apperrors.KindInternal, "failed to add chain_id column", err)
		}
	}

	// Create job_chains table
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS job_chains (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			description TEXT,
			definition TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_job_chains_name ON job_chains(name);
	`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to create job_chains table", err)
	}

	// Create indexes for dependency lookups
	_, err = s.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_jobs_chain_id ON jobs(chain_id);
		CREATE INDEX IF NOT EXISTS idx_jobs_dependency_status ON jobs(dependency_status);
	`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to create dependency indexes", err)
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

	// Close chain statements
	if s.stmtCreateChain != nil {
		s.stmtCreateChain.Close()
	}
	if s.stmtGetChain != nil {
		s.stmtGetChain.Close()
	}
	if s.stmtGetChainByName != nil {
		s.stmtGetChainByName.Close()
	}
	if s.stmtUpdateChain != nil {
		s.stmtUpdateChain.Close()
	}
	if s.stmtDeleteChain != nil {
		s.stmtDeleteChain.Close()
	}
	if s.stmtListChains != nil {
		s.stmtListChains.Close()
	}

	// Close dependency statements
	if s.stmtGetJobsByDependencyStatus != nil {
		s.stmtGetJobsByDependencyStatus.Close()
	}
	if s.stmtUpdateDependencyStatus != nil {
		s.stmtUpdateDependencyStatus.Close()
	}
	if s.stmtGetDependentJobs != nil {
		s.stmtGetDependentJobs.Close()
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
