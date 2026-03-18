// Package server tests startup preflight inspection and guided-recovery messaging.
//
// Purpose:
// - Verify setup-mode detection for legacy or unsupported data directories.
//
// Responsibilities:
// - Assert structured setup status is returned when required.
// - Assert current data directories pass preflight.
// - Assert rendered recovery guidance includes the reset command.
//
// Scope:
// - Preflight inspection only.
//
// Usage:
// - Run with `go test ./internal/cli/server`.
//
// Invariants/Assumptions:
// - Preflight should never silently ignore legacy persisted state.
package server

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/config"

	_ "modernc.org/sqlite"
)

func TestCurrentCommandName(t *testing.T) {
	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})

	t.Run("normalizes go run temp binary path", func(t *testing.T) {
		os.Args = []string{"/var/folders/test/go-build1234/b001/exe/spartan"}
		if got := currentCommandName(); got != "go run ./cmd/spartan" {
			t.Fatalf("currentCommandName() = %q, want %q", got, "go run ./cmd/spartan")
		}
	})

	t.Run("preserves relative project binary path", func(t *testing.T) {
		os.Args = []string{"./bin/spartan"}
		if got := currentCommandName(); got != "./bin/spartan" {
			t.Fatalf("currentCommandName() = %q, want %q", got, "./bin/spartan")
		}
	})

	t.Run("falls back to executable basename", func(t *testing.T) {
		os.Args = []string{"/usr/local/bin/spartan"}
		if got := currentCommandName(); got != "spartan" {
			t.Fatalf("currentCommandName() = %q, want %q", got, "spartan")
		}
	})
}

func TestInspectStartupPreflight(t *testing.T) {
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

		status, err := inspectStartupPreflight(config.Config{DataDir: dataDir}, "./bin/spartan")
		if err != nil {
			t.Fatalf("inspectStartupPreflight failed: %v", err)
		}
		if status == nil || !status.Required {
			t.Fatalf("expected setup status, got %#v", status)
		}
		if status.Code != "legacy_data_dir" {
			t.Fatalf("expected legacy_data_dir code, got %#v", status)
		}
		if rendered := renderSetupStatus(status, "./bin/spartan"); !strings.Contains(rendered, "./bin/spartan reset-data") {
			t.Fatalf("rendered setup guidance %q does not include reset-data guidance", rendered)
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

		status, err := inspectStartupPreflight(config.Config{DataDir: dataDir}, "./bin/spartan")
		if err != nil {
			t.Fatalf("inspectStartupPreflight failed: %v", err)
		}
		if status != nil {
			t.Fatalf("status = %#v, want nil", status)
		}
	})
}
