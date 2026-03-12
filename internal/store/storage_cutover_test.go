package store

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestInspectDataDir(t *testing.T) {
	t.Run("missing database", func(t *testing.T) {
		report, err := InspectDataDir(t.TempDir())
		if err != nil {
			t.Fatalf("InspectDataDir failed: %v", err)
		}
		if report.Status != DataDirStatusMissing {
			t.Fatalf("status = %q, want %q", report.Status, DataDirStatusMissing)
		}
	})

	t.Run("current schema", func(t *testing.T) {
		dataDir := t.TempDir()
		s, err := Open(dataDir)
		if err != nil {
			t.Fatalf("Open failed: %v", err)
		}
		s.Close()

		report, err := InspectDataDir(dataDir)
		if err != nil {
			t.Fatalf("InspectDataDir failed: %v", err)
		}
		if report.Status != DataDirStatusCurrent {
			t.Fatalf("status = %q, want %q", report.Status, DataDirStatusCurrent)
		}
		if report.SchemaVersion != balanced10StorageSchemaVersion {
			t.Fatalf("schema = %q, want %q", report.SchemaVersion, balanced10StorageSchemaVersion)
		}
	})

	t.Run("legacy without metadata", func(t *testing.T) {
		dataDir := t.TempDir()
		db := openSQLiteForTest(t, filepath.Join(dataDir, "jobs.db"))
		if _, err := db.Exec(`create table jobs (id text primary key)`); err != nil {
			t.Fatalf("failed to create legacy jobs table: %v", err)
		}
		if err := db.Close(); err != nil {
			t.Fatalf("db.Close failed: %v", err)
		}

		report, err := InspectDataDir(dataDir)
		if err != nil {
			t.Fatalf("InspectDataDir failed: %v", err)
		}
		if report.Status != DataDirStatusLegacy {
			t.Fatalf("status = %q, want %q", report.Status, DataDirStatusLegacy)
		}
	})

	t.Run("unsupported schema", func(t *testing.T) {
		dataDir := t.TempDir()
		db := openSQLiteForTest(t, filepath.Join(dataDir, "jobs.db"))
		if _, err := db.Exec(`
			create table store_metadata (
				key text primary key,
				value text not null
			);
			insert into store_metadata (key, value) values ('storage_schema', 'legacy-0.x');
		`); err != nil {
			t.Fatalf("failed to create metadata table: %v", err)
		}
		if err := db.Close(); err != nil {
			t.Fatalf("db.Close failed: %v", err)
		}

		report, err := InspectDataDir(dataDir)
		if err != nil {
			t.Fatalf("InspectDataDir failed: %v", err)
		}
		if report.Status != DataDirStatusUnsupported {
			t.Fatalf("status = %q, want %q", report.Status, DataDirStatusUnsupported)
		}
		if report.SchemaVersion != "legacy-0.x" {
			t.Fatalf("schema = %q, want legacy-0.x", report.SchemaVersion)
		}
	})
}

func openSQLiteForTest(t *testing.T, dbPath string) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open failed: %v", err)
	}
	return db
}
