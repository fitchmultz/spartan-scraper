// Package manage provides tests for restore functionality.
package manage

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/fsutil"
)

func TestValidateBackup(t *testing.T) {
	tempDir := t.TempDir()

	// Test valid backup
	validBackup := filepath.Join(tempDir, "valid-backup.tar.gz")
	if err := createTestBackup(validBackup, []string{"jobs.db", "auth_vault.json"}); err != nil {
		t.Fatalf("failed to create test backup: %v", err)
	}

	if err := validateBackup(validBackup); err != nil {
		t.Errorf("validateBackup failed for valid backup: %v", err)
	}

	// Test backup missing required file
	invalidBackup := filepath.Join(tempDir, "invalid-backup.tar.gz")
	if err := createTestBackup(invalidBackup, []string{"auth_vault.json"}); err != nil {
		t.Fatalf("failed to create test backup: %v", err)
	}

	if err := validateBackup(invalidBackup); err == nil {
		t.Error("validateBackup should fail for backup missing jobs.db")
	}

	// Test non-existent backup
	if err := validateBackup("/nonexistent/backup.tar.gz"); err == nil {
		t.Error("validateBackup should fail for non-existent file")
	}

	// Test invalid gzip file
	invalidGzip := filepath.Join(tempDir, "invalid.gz")
	if err := os.WriteFile(invalidGzip, []byte("not a gzip file"), 0644); err != nil {
		t.Fatalf("failed to create invalid gzip file: %v", err)
	}

	if err := validateBackup(invalidGzip); err == nil {
		t.Error("validateBackup should fail for invalid gzip file")
	}
}

func TestValidateBackupPathTraversal(t *testing.T) {
	tempDir := t.TempDir()

	// Create backup with path traversal attempt
	backupPath := filepath.Join(tempDir, "traversal-backup.tar.gz")
	if err := createTestBackupWithTraversal(backupPath); err != nil {
		t.Fatalf("failed to create test backup: %v", err)
	}

	if err := validateBackup(backupPath); err == nil {
		t.Error("validateBackup should fail for archive with path traversal")
	}
}

func TestRestoreFromArchive(t *testing.T) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")

	// Create test backup
	backupPath := filepath.Join(tempDir, "test-backup.tar.gz")
	testFiles := map[string]string{
		"jobs.db":              "test database content",
		"auth_vault.json":      `{"profiles": {}}`,
		"render_profiles.json": `[]`,
	}

	if err := createTestBackupWithContent(backupPath, testFiles); err != nil {
		t.Fatalf("failed to create test backup: %v", err)
	}

	// Test restore
	if err := restoreFromArchive(backupPath, dataDir, false); err != nil {
		t.Fatalf("restoreFromArchive failed: %v", err)
	}

	// Verify files were restored
	for name, expectedContent := range testFiles {
		path := filepath.Join(dataDir, name)
		content, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("restored file %s not found: %v", name, err)
			continue
		}
		if string(content) != expectedContent {
			t.Errorf("restored file %s content mismatch: got %q, want %q", name, string(content), expectedContent)
		}
	}
}

func TestRestoreFromArchiveDryRun(t *testing.T) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")

	// Create test backup
	backupPath := filepath.Join(tempDir, "test-backup.tar.gz")
	if err := createTestBackup(backupPath, []string{"jobs.db"}); err != nil {
		t.Fatalf("failed to create test backup: %v", err)
	}

	// Test dry-run restore
	if err := restoreFromArchive(backupPath, dataDir, true); err != nil {
		t.Fatalf("restoreFromArchive (dry-run) failed: %v", err)
	}

	// Verify no files were restored
	if _, err := os.Stat(dataDir); !os.IsNotExist(err) {
		t.Error("data directory should not exist after dry-run")
	}
}

func TestIsDataDirInUse(t *testing.T) {
	tempDir := t.TempDir()

	// Test with empty directory (should not be in use)
	if isDataDirInUse(tempDir) {
		t.Error("isDataDirInUse should return false for empty directory")
	}

	// Test with lock file present
	lockFile := filepath.Join(tempDir, ".server-lock")
	if err := os.WriteFile(lockFile, []byte("locked"), 0644); err != nil {
		t.Fatalf("failed to create lock file: %v", err)
	}

	if !isDataDirInUse(tempDir) {
		t.Error("isDataDirInUse should return true when lock file exists")
	}

	// Clean up lock file
	os.Remove(lockFile)

	// Test with existing test file (simulating another process)
	testFile := filepath.Join(tempDir, ".write-test-restore")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	if !isDataDirInUse(tempDir) {
		t.Error("isDataDirInUse should return true when test file exists")
	}
}

func TestContainsPathTraversal(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"jobs.db", false},
		{"auth_vault.json", false},
		{"jobs/test.json", false},
		{"../etc/passwd", true},
		{"foo/../bar", true},
		{"/absolute/path", true},
		{"\\windows\\path", true},
		{"normal-file.txt", false},
	}

	for _, tt := range tests {
		result := containsPathTraversal(tt.path)
		if result != tt.expected {
			t.Errorf("containsPathTraversal(%q) = %v, want %v", tt.path, result, tt.expected)
		}
	}
}

// Helper function to create a test backup with specified files
func createTestBackup(archivePath string, files []string) error {
	file, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	for _, name := range files {
		header := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len("test content")),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if _, err := tarWriter.Write([]byte("test content")); err != nil {
			return err
		}
	}

	return nil
}

// Helper function to create a test backup with specific content
func createTestBackupWithContent(archivePath string, files map[string]string) error {
	file, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	for name, content := range files {
		header := &tar.Header{
			Name: name,
			Mode: int64(fsutil.FileMode),
			Size: int64(len(content)),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if _, err := tarWriter.Write([]byte(content)); err != nil {
			return err
		}
	}

	return nil
}

// Helper function to create a test backup with path traversal attempt
func createTestBackupWithTraversal(archivePath string) error {
	file, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Add a file with path traversal
	header := &tar.Header{
		Name: "../etc/passwd",
		Mode: 0644,
		Size: int64(len("test content")),
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}
	if _, err := tarWriter.Write([]byte("test content")); err != nil {
		return err
	}

	return nil
}
