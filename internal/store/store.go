package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"spartan-scraper/internal/model"

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

type Store struct {
	db *sql.DB

	// Prepared statements
	insertJobStmt        *sql.Stmt
	updateJobStatusStmt  *sql.Stmt
	getJobStmt           *sql.Stmt
	getCrawlStateStmt    *sql.Stmt
	upsertCrawlStateStmt *sql.Stmt
}

func Open(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
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

	store := &Store{db: db}
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
		return fmt.Errorf("failed to prepare insertJobStmt: %w", err)
	}

	s.updateJobStatusStmt, err = s.db.Prepare(`update jobs set status = ?, updated_at = ?, error = ? where id = ?`)
	if err != nil {
		return fmt.Errorf("failed to prepare updateJobStatusStmt: %w", err)
	}

	s.getJobStmt, err = s.db.Prepare(`select id, kind, status, created_at, updated_at, params, result_path, error from jobs where id = ?`)
	if err != nil {
		return fmt.Errorf("failed to prepare getJobStmt: %w", err)
	}

	s.getCrawlStateStmt, err = s.db.Prepare(`select url, etag, last_modified, content_hash, last_scraped from crawl_states where url = ?`)
	if err != nil {
		return fmt.Errorf("failed to prepare getCrawlStateStmt: %w", err)
	}

	s.upsertCrawlStateStmt, err = s.db.Prepare(`insert into crawl_states (url, etag, last_modified, content_hash, last_scraped)
		values (?, ?, ?, ?, ?)
		on conflict(url) do update set
			etag = excluded.etag,
			last_modified = excluded.last_modified,
			content_hash = excluded.content_hash,
			last_scraped = excluded.last_scraped`)
	if err != nil {
		return fmt.Errorf("failed to prepare upsertCrawlStateStmt: %w", err)
	}

	return nil
}

func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
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
		create table if not exists crawl_states (
			url text primary key,
			etag text,
			last_modified text,
			content_hash text,
			last_scraped text
		);
	`)
	return err
}

func (s *Store) Create(ctx context.Context, job model.Job) error {
	params, err := json.Marshal(job.Params)
	if err != nil {
		return fmt.Errorf("failed to marshal job params: %w", err)
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
		return model.Job{}, err
	}
	var err error
	job.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return model.Job{}, fmt.Errorf("failed to parse created_at: %w", err)
	}
	job.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return model.Job{}, fmt.Errorf("failed to parse updated_at: %w", err)
	}
	if params != "" {
		if err := json.Unmarshal([]byte(params), &job.Params); err != nil {
			return model.Job{}, fmt.Errorf("failed to unmarshal params: %w", err)
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
			return nil, fmt.Errorf("failed to parse created_at for job %s: %w", job.ID, parseErr)
		}
		job.UpdatedAt, parseErr = time.Parse(time.RFC3339Nano, updatedAt)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse updated_at for job %s: %w", job.ID, parseErr)
		}
		if params != "" {
			if err := json.Unmarshal([]byte(params), &job.Params); err != nil {
				return nil, fmt.Errorf("failed to unmarshal params for job %s: %w", job.ID, err)
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
	if err := row.Scan(&state.URL, &state.ETag, &state.LastModified, &state.ContentHash, &lastScraped); err != nil {
		if err == sql.ErrNoRows {
			return model.CrawlState{}, nil
		}
		return model.CrawlState{}, err
	}
	if lastScraped != "" {
		var err error
		state.LastScraped, err = time.Parse(time.RFC3339Nano, lastScraped)
		if err != nil {
			return model.CrawlState{}, fmt.Errorf("failed to parse last_scraped: %w", err)
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
	)
	return err
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

	return s.db.Close()
}

func (s *Store) Checkpoint(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "PRAGMA wal_checkpoint(PASSIVE)")
	return err
}
