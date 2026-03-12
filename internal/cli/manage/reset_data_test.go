package manage

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/config"

	_ "modernc.org/sqlite"
)

func TestRunResetDataArchivesLegacyDirAndRecreatesFreshDirectory(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "data")
	if err := os.MkdirAll(filepath.Join(dataDir, "jobs"), 0o700); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	db, err := sql.Open("sqlite", filepath.Join(dataDir, "jobs.db"))
	if err != nil {
		t.Fatalf("sql.Open failed: %v", err)
	}
	if _, err := db.Exec(`create table jobs (id text primary key, params text)`); err != nil {
		t.Fatalf("failed to create legacy jobs table: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("db.Close failed: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dataDir, "schedules.json"), []byte(`{"legacy":true}`), 0o600); err != nil {
		t.Fatalf("WriteFile schedules.json failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "jobs", "artifact.txt"), []byte("artifact"), 0o600); err != nil {
		t.Fatalf("WriteFile artifact failed: %v", err)
	}

	backupDir := filepath.Join(t.TempDir(), "backups")
	cfg := config.Config{DataDir: dataDir}

	exitCode := RunResetData(context.Background(), cfg, []string{"--backup-dir", backupDir})
	if exitCode != 0 {
		t.Fatalf("RunResetData exitCode = %d, want 0", exitCode)
	}

	isEmpty, err := isDirEmptyForTest(dataDir)
	if err != nil {
		t.Fatalf("failed to inspect reset data dir: %v", err)
	}
	if !isEmpty {
		t.Fatal("expected recreated data directory to be empty")
	}

	matches, err := filepath.Glob(filepath.Join(backupDir, "spartan-cutover-*.tar.gz"))
	if err != nil {
		t.Fatalf("Glob failed: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 cutover archive, got %d", len(matches))
	}

	entries, err := tarEntries(matches[0])
	if err != nil {
		t.Fatalf("tarEntries failed: %v", err)
	}
	for _, want := range []string{"jobs.db", "jobs/artifact.txt", "schedules.json"} {
		if !entries[want] {
			t.Fatalf("archive missing %s", want)
		}
	}
}

func tarEntries(path string) (map[string]bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	entries := make(map[string]bool)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return entries, nil
		}
		if err != nil {
			return nil, err
		}
		entries[header.Name] = true
	}
}

func isDirEmptyForTest(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}
