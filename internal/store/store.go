package store

import (
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

type Store struct {
	db *sql.DB
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
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	store := &Store{db: db}
	if err := store.init(); err != nil {
		return nil, err
	}
	return store, nil
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

func (s *Store) Create(job model.Job) error {
	params, _ := json.Marshal(job.Params)
	_, err := s.db.Exec(
		`insert into jobs (id, kind, status, created_at, updated_at, params, result_path, error)
			values (?, ?, ?, ?, ?, ?, ?, ?)`,
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

func (s *Store) UpdateStatus(id string, status model.Status, errMsg string) error {
	_, err := s.db.Exec(
		`update jobs set status = ?, updated_at = ?, error = ? where id = ?`,
		status,
		time.Now().Format(time.RFC3339Nano),
		errMsg,
		id,
	)
	return err
}

func (s *Store) Get(id string) (model.Job, error) {
	row := s.db.QueryRow(`select id, kind, status, created_at, updated_at, params, result_path, error from jobs where id = ?`, id)
	var job model.Job
	var createdAt, updatedAt string
	var params string
	if err := row.Scan(&job.ID, &job.Kind, &job.Status, &createdAt, &updatedAt, &params, &job.ResultPath, &job.Error); err != nil {
		return model.Job{}, err
	}
	job.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	job.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
	if params != "" {
		_ = json.Unmarshal([]byte(params), &job.Params)
	}
	return job, nil
}

func (s *Store) List() ([]model.Job, error) {
	rows, err := s.db.Query(`select id, kind, status, created_at, updated_at, params, result_path, error from jobs order by created_at desc`)
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
		job.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		job.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
		if params != "" {
			_ = json.Unmarshal([]byte(params), &job.Params)
		}
		results = append(results, job)
	}
	return results, nil
}

func (s *Store) GetCrawlState(url string) (model.CrawlState, error) {
	row := s.db.QueryRow(`select url, etag, last_modified, content_hash, last_scraped from crawl_states where url = ?`, url)
	var state model.CrawlState
	var lastScraped string
	if err := row.Scan(&state.URL, &state.ETag, &state.LastModified, &state.ContentHash, &lastScraped); err != nil {
		if err == sql.ErrNoRows {
			return model.CrawlState{}, nil
		}
		return model.CrawlState{}, err
	}
	state.LastScraped, _ = time.Parse(time.RFC3339Nano, lastScraped)
	return state, nil
}

func (s *Store) UpsertCrawlState(state model.CrawlState) error {
	_, err := s.db.Exec(
		`insert into crawl_states (url, etag, last_modified, content_hash, last_scraped)
		values (?, ?, ?, ?, ?)
		on conflict(url) do update set
			etag = excluded.etag,
			last_modified = excluded.last_modified,
			content_hash = excluded.content_hash,
			last_scraped = excluded.last_scraped`,
		state.URL,
		state.ETag,
		state.LastModified,
		state.ContentHash,
		state.LastScraped.Format(time.RFC3339Nano),
	)
	return err
}

func (s *Store) Close() error {
	return s.db.Close()
}
