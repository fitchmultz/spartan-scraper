// Package manage provides tests for backup functionality.
package manage

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/fsutil"
)

func TestCreateBackupArchive(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	outputDir := filepath.Join(tempDir, "output")

	if err := fsutil.EnsureDataDir(dataDir); err != nil {
		t.Fatalf("failed to create data dir: %v", err)
	}
	if err := fsutil.EnsureDataDir(outputDir); err != nil {
		t.Fatalf("failed to create output dir: %v", err)
	}

	// Create test files
	testFiles := map[string]string{
		"jobs.db":                "test database content",
		"auth_vault.json":        `{"profiles": {}}`,
		"render_profiles.json":   `[]`,
		"extract_templates.json": `[]`,
		"pipeline_js.json":       `{}`,
	}

	for name, content := range testFiles {
		path := filepath.Join(dataDir, name)
		if err := fsutil.WriteFileSecure(path, []byte(content)); err != nil {
			t.Fatalf("failed to create test file %s: %v", name, err)
		}
	}

	// Create jobs directory with test content
	jobsDir := filepath.Join(dataDir, "jobs")
	if err := fsutil.MkdirAllSecure(jobsDir); err != nil {
		t.Fatalf("failed to create jobs dir: %v", err)
	}
	jobFile := filepath.Join(jobsDir, "test-job.json")
	if err := fsutil.WriteFileSecure(jobFile, []byte(`{"id": "test"}`)); err != nil {
		t.Fatalf("failed to create job file: %v", err)
	}

	// Test creating backup
	outputPath := filepath.Join(outputDir, "test-backup.tar.gz")
	if err := createBackupArchive(dataDir, outputPath, false); err != nil {
		t.Fatalf("createBackupArchive failed: %v", err)
	}

	// Verify backup file exists
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("backup file not created: %v", err)
	}

	// Verify backup file has secure permissions
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("failed to stat backup file: %v", err)
	}
	mode := info.Mode().Perm()
	if mode != fsutil.FileMode {
		t.Errorf("backup file permissions = %o, want %o", mode, fsutil.FileMode)
	}

	// Verify archive contents
	files, err := listArchiveContents(outputPath)
	if err != nil {
		t.Fatalf("failed to list archive contents: %v", err)
	}

	// Check expected files are present
	expectedFiles := []string{"jobs.db", "auth_vault.json", "render_profiles.json"}
	for _, expected := range expectedFiles {
		found := false
		for _, f := range files {
			if f == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected file %s not found in archive", expected)
		}
	}

	// Check jobs directory is included
	jobsFound := false
	for _, f := range files {
		if strings.HasPrefix(f, "jobs/") {
			jobsFound = true
			break
		}
	}
	if !jobsFound {
		t.Error("jobs directory not found in archive")
	}
}

func TestCreateBackupArchiveExcludeJobs(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	outputDir := filepath.Join(tempDir, "output")

	if err := fsutil.EnsureDataDir(dataDir); err != nil {
		t.Fatalf("failed to create data dir: %v", err)
	}
	if err := fsutil.EnsureDataDir(outputDir); err != nil {
		t.Fatalf("failed to create output dir: %v", err)
	}

	// Create test database file
	dbPath := filepath.Join(dataDir, "jobs.db")
	if err := fsutil.WriteFileSecure(dbPath, []byte("test database")); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create jobs directory with test content
	jobsDir := filepath.Join(dataDir, "jobs")
	if err := fsutil.MkdirAllSecure(jobsDir); err != nil {
		t.Fatalf("failed to create jobs dir: %v", err)
	}
	jobFile := filepath.Join(jobsDir, "test-job.json")
	if err := fsutil.WriteFileSecure(jobFile, []byte(`{"id": "test"}`)); err != nil {
		t.Fatalf("failed to create job file: %v", err)
	}

	// Test creating backup with --exclude-jobs
	outputPath := filepath.Join(outputDir, "test-backup.tar.gz")
	if err := createBackupArchive(dataDir, outputPath, true); err != nil {
		t.Fatalf("createBackupArchive failed: %v", err)
	}

	// Verify archive contents
	files, err := listArchiveContents(outputPath)
	if err != nil {
		t.Fatalf("failed to list archive contents: %v", err)
	}

	// Check jobs directory is NOT included
	for _, f := range files {
		if strings.HasPrefix(f, "jobs/") {
			t.Errorf("jobs directory should not be in archive, found: %s", f)
		}
	}

	// But database should be there
	dbFound := false
	for _, f := range files {
		if f == "jobs.db" {
			dbFound = true
			break
		}
	}
	if !dbFound {
		t.Error("jobs.db should be in archive")
	}
}

func TestCreateBackupArchiveMissingFiles(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	outputDir := filepath.Join(tempDir, "output")

	if err := fsutil.EnsureDataDir(dataDir); err != nil {
		t.Fatalf("failed to create data dir: %v", err)
	}
	if err := fsutil.EnsureDataDir(outputDir); err != nil {
		t.Fatalf("failed to create output dir: %v", err)
	}

	// Create only the essential database file
	dbPath := filepath.Join(dataDir, "jobs.db")
	if err := fsutil.WriteFileSecure(dbPath, []byte("test database")); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Test creating backup (should succeed even with missing optional files)
	outputPath := filepath.Join(outputDir, "test-backup.tar.gz")
	if err := createBackupArchive(dataDir, outputPath, false); err != nil {
		t.Fatalf("createBackupArchive failed: %v", err)
	}

	// Verify archive contents
	files, err := listArchiveContents(outputPath)
	if err != nil {
		t.Fatalf("failed to list archive contents: %v", err)
	}

	// Check database is present
	dbFound := false
	for _, f := range files {
		if f == "jobs.db" {
			dbFound = true
			break
		}
	}
	if !dbFound {
		t.Error("jobs.db should be in archive")
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1024 * 1024, "1.00 MB"},
		{1024 * 1024 * 1024, "1.00 GB"},
		{2 * 1024 * 1024 * 1024, "2.00 GB"},
	}

	for _, tt := range tests {
		result := formatBytes(tt.bytes)
		if result != tt.expected {
			t.Errorf("formatBytes(%d) = %s, want %s", tt.bytes, result, tt.expected)
		}
	}
}

func TestFindBackups(t *testing.T) {
	tempDir := t.TempDir()

	// Create test backup files
	backups := []string{
		"spartan-backup-20240115-120000.tar.gz",
		"spartan-backup-20240116-130000.tar.gz",
		"not-a-backup.txt",
		"other-file.tar.gz",
	}

	for _, name := range backups {
		path := filepath.Join(tempDir, name)
		content := []byte("test backup content")
		if err := os.WriteFile(path, content, 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	result, err := findBackups(tempDir)
	if err != nil {
		t.Fatalf("findBackups failed: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 backups, got %d", len(result))
	}

	// Verify backup names
	for _, b := range result {
		if !strings.HasPrefix(b.Name, "spartan-backup-") {
			t.Errorf("unexpected backup name: %s", b.Name)
		}
	}
}

func TestFindBackupsEmptyDir(t *testing.T) {
	tempDir := t.TempDir()

	result, err := findBackups(tempDir)
	if err != nil {
		t.Fatalf("findBackups failed: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected 0 backups, got %d", len(result))
	}
}

// Helper function to list archive contents
func listArchiveContents(archivePath string) ([]string, error) {
	file, err := os.Open(archivePath)
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

	var files []string
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		files = append(files, header.Name)
	}

	return files, nil
}
