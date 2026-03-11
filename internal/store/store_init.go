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
	"os"
	"path/filepath"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/fsutil"

	_ "modernc.org/sqlite"
)

const balanced10StorageSchemaVersion = "balanced-1.0-2026-03-11"

// Open creates and initializes a new Store at the given data directory.
// Creates the database file if it doesn't exist, runs migrations, and prepares statements.
func Open(dataDir string) (*Store, error) {
	if err := fsutil.EnsureDataDir(dataDir); err != nil {
		return nil, err
	}
	path := filepath.Join(dataDir, "jobs.db")
	_, statErr := os.Stat(path)
	dbExisted := statErr == nil
	if statErr != nil && !os.IsNotExist(statErr) {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to inspect jobs database", statErr)
	}
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
	if err := store.init(dbExisted); err != nil {
		return nil, err
	}

	if err := store.prepareStatements(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *Store) prepareStatements() error {
	prepare := func(dest **sql.Stmt, query, message string) error {
		stmt, err := s.db.Prepare(query)
		if err != nil {
			return apperrors.Wrap(apperrors.KindInternal, message, err)
		}
		*dest = stmt
		return nil
	}

	statements := []struct {
		dest    **sql.Stmt
		query   string
		message string
	}{
		{&s.insertJobStmt, `insert into jobs (id, kind, status, created_at, updated_at, params, result_path, error, depends_on, dependency_status, chain_id)
			values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, "failed to prepare insert job statement"},
		{&s.updateJobStatusStmt, `update jobs set status = ?, updated_at = ?, error = ? where id = ?`, "failed to prepare update job status statement"},
		{&s.getJobStmt, `select id, kind, status, created_at, updated_at, params, result_path, error, depends_on, dependency_status, chain_id from jobs where id = ?`, "failed to prepare get job statement"},
		{&s.getCrawlStateStmt, `select url, etag, last_modified, content_hash, last_scraped, depth, job_id, previous_content, content_snapshot from crawl_states where url = ?`, "failed to prepare get crawl state statement"},
		{&s.upsertCrawlStateStmt, `insert into crawl_states (url, etag, last_modified, content_hash, last_scraped, depth, job_id, previous_content, content_snapshot)
		values (?, ?, ?, ?, ?, ?, ?, ?, ?)
		on conflict(url) do update set
			etag = excluded.etag,
			last_modified = excluded.last_modified,
			content_hash = excluded.content_hash,
			last_scraped = excluded.last_scraped,
			depth = excluded.depth,
			job_id = excluded.job_id,
			previous_content = excluded.previous_content,
			content_snapshot = excluded.content_snapshot`, "failed to prepare upsert crawl state statement"},
		{&s.deleteCrawlStateStmt, `delete from crawl_states where url = ?`, "failed to prepare delete crawl state statement"},
		{&s.deleteAllCrawlStatesStmt, `delete from crawl_states`, "failed to prepare delete all crawl states statement"},
		{&s.stmtCreateChain, `insert into job_chains (id, name, description, definition, created_at, updated_at) values (?, ?, ?, ?, ?, ?)`, "failed to prepare create chain statement"},
		{&s.stmtGetChain, `select id, name, description, definition, created_at, updated_at from job_chains where id = ?`, "failed to prepare get chain statement"},
		{&s.stmtGetChainByName, `select id, name, description, definition, created_at, updated_at from job_chains where name = ?`, "failed to prepare get chain by name statement"},
		{&s.stmtUpdateChain, `update job_chains set name = ?, description = ?, definition = ?, updated_at = ? where id = ?`, "failed to prepare update chain statement"},
		{&s.stmtDeleteChain, `delete from job_chains where id = ?`, "failed to prepare delete chain statement"},
		{&s.stmtListChains, `select id, name, description, definition, created_at, updated_at from job_chains order by created_at desc`, "failed to prepare list chains statement"},
		{&s.stmtGetJobsByDependencyStatus, `select id, kind, status, created_at, updated_at, params, result_path, error, depends_on, dependency_status, chain_id from jobs where dependency_status = ? order by created_at desc`, "failed to prepare get jobs by dependency status statement"},
		{&s.stmtUpdateDependencyStatus, `update jobs set dependency_status = ? where id = ?`, "failed to prepare update dependency status statement"},
		{&s.stmtGetDependentJobs, `select id, kind, status, created_at, updated_at, params, result_path, error, depends_on, dependency_status, chain_id from jobs where depends_on is not null and depends_on != '' and exists (select 1 from json_each(depends_on) where value = ?)`, "failed to prepare get dependent jobs statement"},
	}

	for _, statement := range statements {
		if err := prepare(statement.dest, statement.query, statement.message); err != nil {
			return err
		}
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

func (s *Store) init(dbExisted bool) error {
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
		create table if not exists store_metadata (
			key text primary key,
			value text not null
		);
	`)
	if err != nil {
		return err
	}

	if err := s.ensureStorageSchemaVersion(dbExisted); err != nil {
		return err
	}

	migrations := []struct {
		table      string
		column     string
		query      string
		checkError string
		applyError string
	}{
		{"crawl_states", "depth", "alter table crawl_states add column depth integer", "failed to check for depth column migration", "failed to add depth column to crawl_states"},
		{"crawl_states", "job_id", "alter table crawl_states add column job_id text", "failed to check for job_id column migration", "failed to add job_id column to crawl_states"},
		{"crawl_states", "previous_content", "alter table crawl_states add column previous_content text", "failed to check for previous_content column migration", "failed to add previous_content column to crawl_states"},
		{"crawl_states", "content_snapshot", "alter table crawl_states add column content_snapshot text", "failed to check for content_snapshot column migration", "failed to add content_snapshot column to crawl_states"},
		{"crawl_states", "screenshot_path", "alter table crawl_states add column screenshot_path text", "failed to check for screenshot_path column migration", "failed to add screenshot_path column to crawl_states"},
		{"crawl_states", "visual_hash", "alter table crawl_states add column visual_hash text", "failed to check for visual_hash column migration", "failed to add visual_hash column to crawl_states"},
	}
	for _, migration := range migrations {
		exists, err := columnExists(s.db, migration.table, migration.column)
		if err != nil {
			return apperrors.Wrap(apperrors.KindInternal, migration.checkError, err)
		}
		if exists {
			continue
		}
		if _, err := s.db.Exec(migration.query); err != nil {
			return apperrors.Wrap(apperrors.KindInternal, migration.applyError, err)
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

	// Initialize analytics tables
	if err := s.initAnalyticsTables(); err != nil {
		return err
	}

	return nil
}

func (s *Store) ensureStorageSchemaVersion(dbExisted bool) error {
	var version string
	err := s.db.QueryRow(`select value from store_metadata where key = 'storage_schema'`).Scan(&version)
	switch {
	case err == nil:
		if version != balanced10StorageSchemaVersion {
			return apperrors.Validation(
				fmt.Sprintf("unsupported data dir schema %q; back up %s and reset the data directory for Balanced 1.0", version, s.dataDir),
			)
		}
		return nil
	case err != sql.ErrNoRows:
		return apperrors.Wrap(apperrors.KindInternal, "failed to read store schema version", err)
	}

	if dbExisted {
		return apperrors.Validation(
			fmt.Sprintf("legacy data dir detected at %s; back it up and reset the data directory before starting Balanced 1.0", s.dataDir),
		)
	}

	_, err = s.db.Exec(
		`insert into store_metadata (key, value) values ('storage_schema', ?)`,
		balanced10StorageSchemaVersion,
	)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to initialize store schema version", err)
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

	for _, stmt := range []*sql.Stmt{
		s.insertJobStmt,
		s.updateJobStatusStmt,
		s.getJobStmt,
		s.getCrawlStateStmt,
		s.upsertCrawlStateStmt,
		s.deleteCrawlStateStmt,
		s.deleteAllCrawlStatesStmt,
		s.stmtCreateChain,
		s.stmtGetChain,
		s.stmtGetChainByName,
		s.stmtUpdateChain,
		s.stmtDeleteChain,
		s.stmtListChains,
		s.stmtGetJobsByDependencyStatus,
		s.stmtUpdateDependencyStatus,
		s.stmtGetDependentJobs,
	} {
		if stmt != nil {
			stmt.Close()
		}
	}

	// Close analytics statements
	if err := s.closeAnalyticsStatements(); err != nil {
		return err
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
