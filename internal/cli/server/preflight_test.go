package server

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/config"

	_ "modernc.org/sqlite"
)

func TestStartupPreflightMessage(t *testing.T) {
	t.Run("legacy directory guidance", func(t *testing.T) {
		dataDir := t.TempDir()
		db, err := sql.Open("sqlite", filepath.Join(dataDir, "jobs.db"))
		if err != nil {
			t.Fatalf("sql.Open failed: %v", err)
		}
		if _, err := db.Exec(`create table jobs (id text primary key)`); err != nil {
			t.Fatalf("failed to create legacy jobs table: %v", err)
		}
		if err := db.Close(); err != nil {
			t.Fatalf("db.Close failed: %v", err)
		}

		msg, err := startupPreflightMessage(config.Config{DataDir: dataDir}, "./bin/spartan")
		if err != nil {
			t.Fatalf("startupPreflightMessage failed: %v", err)
		}
		if !strings.Contains(msg, "./bin/spartan reset-data") {
			t.Fatalf("message %q does not include reset-data guidance", msg)
		}
		if !strings.Contains(msg, "fresh data directory") {
			t.Fatalf("message %q does not explain the cutover", msg)
		}
	})

	t.Run("current directory passes", func(t *testing.T) {
		dataDir := t.TempDir()
		db, err := sql.Open("sqlite", filepath.Join(dataDir, "jobs.db"))
		if err != nil {
			t.Fatalf("sql.Open failed: %v", err)
		}
		if _, err := db.Exec(`
			create table store_metadata (
				key text primary key,
				value text not null
			);
			insert into store_metadata (key, value) values ('storage_schema', 'balanced-1.0-2026-03-11');
		`); err != nil {
			t.Fatalf("failed to create metadata table: %v", err)
		}
		if err := db.Close(); err != nil {
			t.Fatalf("db.Close failed: %v", err)
		}

		msg, err := startupPreflightMessage(config.Config{DataDir: dataDir}, "./bin/spartan")
		if err != nil {
			t.Fatalf("startupPreflightMessage failed: %v", err)
		}
		if msg != "" {
			t.Fatalf("message = %q, want empty", msg)
		}
	})
}
