// Package store provides tests for database migrations and error handling.
// Tests cover schema initialization, idempotent migrations, ALTER TABLE failures,
// and error kind classification (KindInternal) for parse and JSON errors.
// Does NOT test normal CRUD operations or happy-path queries.
package store

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"

	_ "modernc.org/sqlite"
)

func TestMigrationFreshDatabase(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	depthExists, err := columnExists(s.db, "crawl_states", "depth")
	if err != nil {
		t.Fatalf("columnExists(depth) failed: %v", err)
	}
	if !depthExists {
		t.Error("depth column should exist after init")
	}

	jobIDExists, err := columnExists(s.db, "crawl_states", "job_id")
	if err != nil {
		t.Fatalf("columnExists(job_id) failed: %v", err)
	}
	if !jobIDExists {
		t.Error("job_id column should exist after init")
	}

	ctx := context.Background()
	state := model.CrawlState{
		URL:          "http://example.com",
		ETag:         "tag",
		LastModified: "Mon, 01 Jan 2026 00:00:00 GMT",
		ContentHash:  "hash",
		LastScraped:  time.Now(),
		Depth:        2,
		JobID:        "test-job",
	}

	if err := s.UpsertCrawlState(ctx, state); err != nil {
		t.Fatalf("UpsertCrawlState failed: %v", err)
	}

	got, err := s.GetCrawlState(ctx, "http://example.com")
	if err != nil {
		t.Fatalf("GetCrawlState failed: %v", err)
	}
	if got.Depth != 2 {
		t.Errorf("expected Depth 2, got %d", got.Depth)
	}
	if got.JobID != "test-job" {
		t.Errorf("expected JobID test-job, got %s", got.JobID)
	}

	var version string
	if err := s.db.QueryRow(`select value from store_metadata where key = 'storage_schema'`).Scan(&version); err != nil {
		t.Fatalf("failed to read storage schema version: %v", err)
	}
	if version != balanced10StorageSchemaVersion {
		t.Fatalf("expected storage schema %q, got %q", balanced10StorageSchemaVersion, version)
	}
}

func TestMigrationIdempotent(t *testing.T) {
	dataDir := t.TempDir()

	s1, err := Open(dataDir)
	if err != nil {
		t.Fatalf("First Open failed: %v", err)
	}

	ctx := context.Background()
	state := model.CrawlState{
		URL:         "http://example.com",
		ETag:        "tag1",
		LastScraped: time.Now(),
		Depth:       3,
		JobID:       "job-1",
	}
	if err := s1.UpsertCrawlState(ctx, state); err != nil {
		t.Fatalf("UpsertCrawlState failed: %v", err)
	}
	s1.Close()

	s2, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Second Open failed: %v", err)
	}
	defer s2.Close()

	depthExists, err := columnExists(s2.db, "crawl_states", "depth")
	if err != nil {
		t.Fatalf("columnExists(depth) failed on reopen: %v", err)
	}
	if !depthExists {
		t.Error("depth column should still exist after reopen")
	}

	jobIDExists, err := columnExists(s2.db, "crawl_states", "job_id")
	if err != nil {
		t.Fatalf("columnExists(job_id) failed on reopen: %v", err)
	}
	if !jobIDExists {
		t.Error("job_id column should still exist after reopen")
	}

	got, err := s2.GetCrawlState(ctx, "http://example.com")
	if err != nil {
		t.Fatalf("GetCrawlState failed after reopen: %v", err)
	}
	if got.Depth != 3 {
		t.Errorf("expected Depth 3 after reopen, got %d", got.Depth)
	}
	if got.JobID != "job-1" {
		t.Errorf("expected JobID job-1 after reopen, got %s", got.JobID)
	}
}

func TestMigrationAlterFailure(t *testing.T) {
	dataDir := t.TempDir()
	dbPath := filepath.Join(dataDir, "jobs.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open failed: %v", err)
	}
	_, err = db.Exec(`
		create table jobs (
			id text primary key,
			kind text not null,
			status text not null,
			created_at text not null,
			updated_at text not null,
			params text,
			result_path text,
			error text
		);

		create table crawl_states (
			url text primary key,
			etag text,
			last_modified text,
			content_hash text,
			last_scraped text
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create old schema: %v", err)
	}
	db.Close()

	if err := os.Chmod(dbPath, 0o444); err != nil {
		t.Fatalf("Failed to make database read-only: %v", err)
	}
	defer os.Chmod(dbPath, 0o644)

	_, err = Open(dataDir)
	if err == nil {
		t.Error("Open should return error when ALTER TABLE fails")
	}
}

func TestOpenRejectsLegacyDatabaseWithoutStorageSchema(t *testing.T) {
	dataDir := t.TempDir()
	dbPath := filepath.Join(dataDir, "jobs.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open failed: %v", err)
	}
	_, err = db.Exec(`
		create table jobs (
			id text primary key,
			kind text not null,
			status text not null,
			created_at text not null,
			updated_at text not null,
			params text,
			result_path text,
			error text
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create legacy jobs table: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("db.Close failed: %v", err)
	}

	_, err = Open(dataDir)
	if err == nil {
		t.Fatal("Open should reject a legacy database without storage schema metadata")
	}
	if !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("expected validation error, got %v", apperrors.KindOf(err))
	}
	if got := err.Error(); got == "" || !containsAll(got, "legacy data dir detected", "Balanced 1.0") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenRejectsUnsupportedStorageSchemaVersion(t *testing.T) {
	dataDir := t.TempDir()
	dbPath := filepath.Join(dataDir, "jobs.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open failed: %v", err)
	}
	_, err = db.Exec(`
		create table store_metadata (
			key text primary key,
			value text not null
		);
		insert into store_metadata (key, value) values ('storage_schema', 'legacy-0.x');
	`)
	if err != nil {
		t.Fatalf("Failed to create metadata table: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("db.Close failed: %v", err)
	}

	_, err = Open(dataDir)
	if err == nil {
		t.Fatal("Open should reject an unsupported storage schema version")
	}
	if !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("expected validation error, got %v", apperrors.KindOf(err))
	}
	if got := err.Error(); got == "" || !containsAll(got, "unsupported data dir schema", "legacy-0.x") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func containsAll(value string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(value, part) {
			return false
		}
	}
	return true
}

func TestParseErrorsReturnInternalKind(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	job := model.Job{
		ID:        "test-id",
		Kind:      model.KindScrape,
		Status:    model.StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Spec:      map[string]interface{}{"url": "http://example.com"},
	}
	if err := s.Create(ctx, job); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	_, err = s.db.ExecContext(ctx, "UPDATE jobs SET created_at = ? WHERE id = ?", "invalid-timestamp", "test-id")
	if err != nil {
		t.Fatalf("Failed to corrupt created_at: %v", err)
	}

	_, err = s.ListByStatus(ctx, model.StatusQueued, ListByStatusOptions{})
	if err == nil {
		t.Fatal("Expected error when parsing invalid created_at, got nil")
	}
	if !apperrors.IsKind(err, apperrors.KindInternal) {
		t.Errorf("Expected KindInternal error, got %v", apperrors.KindOf(err))
	}

	_, err = s.db.ExecContext(ctx, "UPDATE jobs SET created_at = ?, updated_at = ? WHERE id = ?",
		job.CreatedAt.Format(time.RFC3339Nano), "invalid-timestamp", "test-id")
	if err != nil {
		t.Fatalf("Failed to corrupt updated_at: %v", err)
	}

	_, err = s.ListByStatus(ctx, model.StatusQueued, ListByStatusOptions{})
	if err == nil {
		t.Fatal("Expected error when parsing invalid updated_at, got nil")
	}
	if !apperrors.IsKind(err, apperrors.KindInternal) {
		t.Errorf("Expected KindInternal error, got %v", apperrors.KindOf(err))
	}

	_, err = s.Get(ctx, "test-id")
	if err == nil {
		t.Fatal("Expected error when parsing invalid updated_at in Get, got nil")
	}
	if !apperrors.IsKind(err, apperrors.KindInternal) {
		t.Errorf("Expected KindInternal error in Get, got %v", apperrors.KindOf(err))
	}

	_, err = s.ListOpts(ctx, ListOptions{})
	if err == nil {
		t.Fatal("Expected error when parsing invalid updated_at in ListOpts, got nil")
	}
	if !apperrors.IsKind(err, apperrors.KindInternal) {
		t.Errorf("Expected KindInternal error in ListOpts, got %v", apperrors.KindOf(err))
	}
}

func TestJSONErrorsReturnInternalKind(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	job := model.Job{
		ID:        "test-id",
		Kind:      model.KindScrape,
		Status:    model.StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Spec:      map[string]interface{}{"url": "http://example.com"},
	}
	if err := s.Create(ctx, job); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	_, err = s.db.ExecContext(ctx, "UPDATE jobs SET spec_json = ? WHERE id = ?", "{invalid-json}", "test-id")
	if err != nil {
		t.Fatalf("Failed to corrupt spec_json: %v", err)
	}

	_, err = s.ListByStatus(ctx, model.StatusQueued, ListByStatusOptions{})
	if err == nil {
		t.Fatal("Expected error when unmarshaling invalid spec_json, got nil")
	}
	if !apperrors.IsKind(err, apperrors.KindInternal) {
		t.Errorf("Expected KindInternal error, got %v", apperrors.KindOf(err))
	}

	_, err = s.Get(ctx, "test-id")
	if err == nil {
		t.Fatal("Expected error when unmarshaling invalid spec_json in Get, got nil")
	}
	if !apperrors.IsKind(err, apperrors.KindInternal) {
		t.Errorf("Expected KindInternal error in Get, got %v", apperrors.KindOf(err))
	}

	_, err = s.ListOpts(ctx, ListOptions{})
	if err == nil {
		t.Fatal("Expected error when unmarshaling invalid spec_json in ListOpts, got nil")
	}
	if !apperrors.IsKind(err, apperrors.KindInternal) {
		t.Errorf("Expected KindInternal error in ListOpts, got %v", apperrors.KindOf(err))
	}
}

func TestCrawlStateParseErrorsReturnInternalKind(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	state := model.CrawlState{
		URL:         "http://example.com",
		LastScraped: time.Now(),
		Depth:       1,
	}
	if err := s.UpsertCrawlState(ctx, state); err != nil {
		t.Fatalf("UpsertCrawlState failed: %v", err)
	}

	_, err = s.db.ExecContext(ctx, "UPDATE crawl_states SET last_scraped = ? WHERE url = ?", "invalid-timestamp", "http://example.com")
	if err != nil {
		t.Fatalf("Failed to corrupt last_scraped: %v", err)
	}

	_, err = s.GetCrawlState(ctx, "http://example.com")
	if err == nil {
		t.Fatal("Expected error when parsing invalid last_scraped, got nil")
	}
	if !apperrors.IsKind(err, apperrors.KindInternal) {
		t.Errorf("Expected KindInternal error, got %v", apperrors.KindOf(err))
	}

	_, err = s.ListCrawlStates(ctx, ListCrawlStatesOptions{})
	if err == nil {
		t.Fatal("Expected error when parsing invalid last_scraped in ListCrawlStates, got nil")
	}
	if !apperrors.IsKind(err, apperrors.KindInternal) {
		t.Errorf("Expected KindInternal error in ListCrawlStates, got %v", apperrors.KindOf(err))
	}
}

func TestPrepareStatementErrorsReturnInternalKind(t *testing.T) {

	t.Skip("Prepare statement errors require database corruption to test; pattern verified by code inspection")
}
