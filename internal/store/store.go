// Package store provides SQLite-backed persistent storage for jobs and crawl states.
// It handles job CRUD operations, status tracking, and incremental crawling state.
// It does NOT handle job execution or scheduling.
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/fsutil"
	"github.com/fitchmultz/spartan-scraper/internal/model"

	_ "modernc.org/sqlite"
)

// ListOptions specifies pagination parameters for Store.ListOpts.
type ListOptions struct {
	Limit  int
	Offset int
}

// Defaults returns options with safe defaults applied.
// Limit defaults to 100, max is 1000. Offset defaults to 0.
func (o ListOptions) Defaults() ListOptions {
	if o.Limit <= 0 {
		o.Limit = 100
	}
	if o.Limit > 1000 {
		o.Limit = 1000
	}
	if o.Offset < 0 {
		o.Offset = 0
	}
	return o
}

// ListByStatusOptions specifies pagination parameters for Store.ListByStatus.
type ListByStatusOptions struct {
	Limit  int
	Offset int
}

// Defaults returns options with safe defaults applied.
// Limit defaults to 100, max is 1000. Offset defaults to 0.
func (o ListByStatusOptions) Defaults() ListByStatusOptions {
	if o.Limit <= 0 {
		o.Limit = 100
	}
	if o.Limit > 1000 {
		o.Limit = 1000
	}
	if o.Offset < 0 {
		o.Offset = 0
	}
	return o
}

// ListCrawlStatesOptions specifies pagination parameters for Store.ListCrawlStates.
type ListCrawlStatesOptions struct {
	Limit  int
	Offset int
}

// Defaults returns options with safe defaults applied.
// Limit defaults to 100, max is 1000. Offset defaults to 0.
func (o ListCrawlStatesOptions) Defaults() ListCrawlStatesOptions {
	if o.Limit <= 0 {
		o.Limit = 100
	}
	if o.Limit > 1000 {
		o.Limit = 1000
	}
	if o.Offset < 0 {
		o.Offset = 0
	}
	return o
}

// ListByStatus returns all jobs with the given status, ordered by created_at.
// If no options are provided, it uses safe defaults (limit 100, offset 0).
func (s *Store) ListByStatus(ctx context.Context, status model.Status, opts ListByStatusOptions) ([]model.Job, error) {
	opts = opts.Defaults()
	rows, err := s.db.QueryContext(ctx,
		`select id, kind, status, created_at, updated_at, params, result_path, error
		 from jobs where status = ? order by created_at desc limit ? offset ?`, status, opts.Limit, opts.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []model.Job{}
	for rows.Next() {
		var job model.Job
		var createdAt, updatedAt string
		var params string
		if err := rows.Scan(&job.ID, &job.Kind, &job.Status, &createdAt, &updatedAt, &params, &job.ResultPath, &job.Error); err != nil {
			return nil, err
		}
		var parseErr error
		job.CreatedAt, parseErr = time.Parse(time.RFC3339Nano, createdAt)
		if parseErr != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to parse job created_at", parseErr)
		}
		job.UpdatedAt, parseErr = time.Parse(time.RFC3339Nano, updatedAt)
		if parseErr != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to parse job updated_at", parseErr)
		}
		if params != "" {
			if err := json.Unmarshal([]byte(params), &job.Params); err != nil {
				return nil, apperrors.Wrap(apperrors.KindInternal, "failed to unmarshal job params", err)
			}
		}
		results = append(results, job)
	}
	return results, rows.Err()
}

// CountJobs returns the total number of jobs, optionally filtered by status.
func (s *Store) CountJobs(ctx context.Context, status model.Status) (int, error) {
	var query string
	var args []interface{}
	if status != "" {
		query = "select count(*) from jobs where status = ?"
		args = append(args, status)
	} else {
		query = "select count(*) from jobs"
	}

	var count int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

type Store struct {
	db      *sql.DB
	dataDir string

	// Prepared statements
	insertJobStmt            *sql.Stmt
	updateJobStatusStmt      *sql.Stmt
	getJobStmt               *sql.Stmt
	getCrawlStateStmt        *sql.Stmt
	upsertCrawlStateStmt     *sql.Stmt
	deleteCrawlStateStmt     *sql.Stmt
	deleteAllCrawlStatesStmt *sql.Stmt
}

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

	// Connection pooling configuration
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

func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// columnExists returns true if the specified column exists in the given table.
// Uses PRAGMA table_info to query schema information.
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

	// Migration: add depth and job_id columns to crawl_states if they don't exist
	// These columns were added in a later version; existing databases need migration.
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

func (s *Store) Create(ctx context.Context, job model.Job) error {
	params, err := json.Marshal(job.Params)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to marshal job params", err)
	}
	_, err = s.insertJobStmt.ExecContext(
		ctx,
		job.ID,
		job.Kind,
		job.Status,
		job.CreatedAt.Format(time.RFC3339Nano),
		job.UpdatedAt.Format(time.RFC3339Nano),
		string(params),
		job.ResultPath,
		job.Error,
	)
	return err
}

func (s *Store) UpdateStatus(ctx context.Context, id string, status model.Status, errMsg string) error {
	_, err := s.updateJobStatusStmt.ExecContext(
		ctx,
		status,
		time.Now().Format(time.RFC3339Nano),
		errMsg,
		id,
	)
	return err
}

func (s *Store) Get(ctx context.Context, id string) (model.Job, error) {
	row := s.getJobStmt.QueryRowContext(ctx, id)
	var job model.Job
	var createdAt, updatedAt string
	var params string
	if err := row.Scan(&job.ID, &job.Kind, &job.Status, &createdAt, &updatedAt, &params, &job.ResultPath, &job.Error); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Job{}, apperrors.NotFound("job not found")
		}
		return model.Job{}, err
	}
	var err error
	job.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return model.Job{}, apperrors.Wrap(apperrors.KindInternal, "failed to parse job created_at", err)
	}
	job.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return model.Job{}, apperrors.Wrap(apperrors.KindInternal, "failed to parse job updated_at", err)
	}
	if params != "" {
		if err := json.Unmarshal([]byte(params), &job.Params); err != nil {
			return model.Job{}, apperrors.Wrap(apperrors.KindInternal, "failed to unmarshal job params", err)
		}
	}
	return job, nil
}

func (s *Store) List(ctx context.Context) ([]model.Job, error) {
	return s.ListOpts(ctx, ListOptions{})
}

func (s *Store) ListOpts(ctx context.Context, opts ListOptions) ([]model.Job, error) {
	opts = opts.Defaults()
	rows, err := s.db.QueryContext(ctx, `select id, kind, status, created_at, updated_at, params, result_path, error from jobs order by created_at desc limit ? offset ?`, opts.Limit, opts.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []model.Job{}
	for rows.Next() {
		var job model.Job
		var createdAt, updatedAt string
		var params string
		if err := rows.Scan(&job.ID, &job.Kind, &job.Status, &createdAt, &updatedAt, &params, &job.ResultPath, &job.Error); err != nil {
			return nil, err
		}
		var parseErr error
		job.CreatedAt, parseErr = time.Parse(time.RFC3339Nano, createdAt)
		if parseErr != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to parse job created_at", parseErr)
		}
		job.UpdatedAt, parseErr = time.Parse(time.RFC3339Nano, updatedAt)
		if parseErr != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to parse job updated_at", parseErr)
		}
		if params != "" {
			if err := json.Unmarshal([]byte(params), &job.Params); err != nil {
				return nil, apperrors.Wrap(apperrors.KindInternal, "failed to unmarshal job params", err)
			}
		}
		results = append(results, job)
	}
	return results, rows.Err()
}

func (s *Store) GetCrawlState(ctx context.Context, url string) (model.CrawlState, error) {
	row := s.getCrawlStateStmt.QueryRowContext(ctx, url)
	var state model.CrawlState
	var lastScraped string
	if err := row.Scan(&state.URL, &state.ETag, &state.LastModified, &state.ContentHash, &lastScraped, &state.Depth, &state.JobID); err != nil {
		if err == sql.ErrNoRows {
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
		`select url, etag, last_modified, content_hash, last_scraped, depth, job_id
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
			&state.ContentHash, &lastScraped, &state.Depth, &state.JobID); err != nil {
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

// Delete permanently removes a job from the store.
func (s *Store) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM jobs WHERE id = ?", id)
	return err
}

// DeleteWithArtifacts permanently removes a job from store and deletes its result file and directory.
// This is used for force delete operations.
func (s *Store) DeleteWithArtifacts(ctx context.Context, id string) error {
	if err := s.Delete(ctx, id); err != nil {
		return err
	}

	// Delete the job directory (includes result file)
	// Directory path: {dataDir}/jobs/{id}
	jobDir := filepath.Join(s.dataDir, "jobs", id)
	if err := os.RemoveAll(jobDir); err != nil {
		// Log warning but don't fail if directory removal fails
		// The DB record is gone, which is the critical part
		slog.Warn("failed to delete job directory", "id", id, "path", jobDir, "error", err)
	}

	return nil
}

func (s *Store) Close() error {
	// Try to checkpoint WAL before closing
	_, _ = s.db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")

	// Close prepared statements
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

func (s *Store) Checkpoint(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "PRAGMA wal_checkpoint(PASSIVE)")
	return err
}

// UpdateResultPath updates the result_path field for a job.
func (s *Store) UpdateResultPath(ctx context.Context, id string, resultPath string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE jobs SET result_path = ? WHERE id = ?", resultPath, id)
	return err
}

// DataDir returns the data directory path.
func (s *Store) DataDir() string {
	return s.dataDir
}
