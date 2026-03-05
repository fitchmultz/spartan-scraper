// Package manage provides tests for restore functionality.
package manage

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/config"
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
		// Normal files (should be allowed)
		{"jobs.db", false},
		{"auth_vault.json", false},
		{"jobs/test.json", false},
		{"subdir/nested/file.txt", false},
		{"normal-file.txt", false},
		{"./jobs.db", false},       // Explicit current dir (harmless)
		{"foo/./bar", false},       // Current dir in middle
		{".hiddenfile", false},     // Hidden file
		{"dir/.hiddenfile", false}, // Hidden file in directory

		// Path traversal attacks (should be blocked)
		{"../etc/passwd", true},               // Unix traversal
		{"..\\Windows\\System32", true},       // Windows traversal (TAR uses /, but check both)
		{"foo/../bar", true},                  // Mid-path traversal
		{"foo/bar/../../../etc/passwd", true}, // Nested traversal
		{"../jobs.db", true},                  // Simple parent reference
		{"a/b/c/../../../../d", true},         // Deep traversal

		// Absolute paths (should be blocked)
		{"/absolute/path", true},        // Unix absolute
		{"/etc/passwd", true},           // Unix absolute system file
		{"/", true},                     // Root directory
		{"\\windows\\path", true},       // Windows backslash absolute
		{"C:\\Windows\\System32", true}, // Windows drive letter
		{"\\\\server\\share", true},     // UNC path (Windows network)

		// Edge cases with traversal
		{"foo//../bar", true},    // Double slashes with traversal
		{"/../etc/passwd", true}, // Absolute with traversal
		{"../", true},            // Trailing slash after traversal
		{"..", true},             // Just parent reference
		{"foo/..", true},         // Parent at end
		{"foo/bar/..", true},     // Parent at end of nested path
		{".../file", false},      // Triple dot (not traversal)
		{"file...txt", false},    // Triple dot in filename
		{"..file", false},        // Double dot prefix
		{"file..", false},        // Double dot suffix
		{"file..txt", false},     // Double dot in middle
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

// TestRunRestore_NonEmptyDir tests restore behavior with non-empty data directories.
func TestRunRestore_NonEmptyDir(t *testing.T) {
	tempDir := t.TempDir()

	// Create test backup
	backupPath := filepath.Join(tempDir, "test-backup.tar.gz")
	if err := createTestBackup(backupPath, []string{"jobs.db"}); err != nil {
		t.Fatalf("failed to create test backup: %v", err)
	}

	ctx := context.Background()

	// Test 1: Restore into empty directory succeeds
	t.Run("EmptyDirSucceeds", func(t *testing.T) {
		dataDir := filepath.Join(tempDir, "empty-data")
		// Create the empty data directory
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			t.Fatalf("failed to create empty data dir: %v", err)
		}
		cfg := config.Config{DataDir: dataDir}

		exitCode := RunRestore(ctx, cfg, []string{"--from", backupPath})
		if exitCode != 0 {
			t.Errorf("expected exit code 0 for empty dir, got %d", exitCode)
		}
	})

	// Test 2: Restore into non-empty directory without --force fails
	t.Run("NonEmptyDirWithoutForceFails", func(t *testing.T) {
		dataDir := filepath.Join(tempDir, "nonempty-data")
		cfg := config.Config{DataDir: dataDir}

		// Create directory with a file
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			t.Fatalf("failed to create data dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dataDir, "existing.txt"), []byte("data"), 0644); err != nil {
			t.Fatalf("failed to create existing file: %v", err)
		}

		exitCode := RunRestore(ctx, cfg, []string{"--from", backupPath})
		if exitCode != 1 {
			t.Errorf("expected exit code 1 for non-empty dir without force, got %d", exitCode)
		}
	})

	// Test 3: Restore into non-empty directory with --force succeeds
	t.Run("NonEmptyDirWithForceSucceeds", func(t *testing.T) {
		dataDir := filepath.Join(tempDir, "force-data")
		cfg := config.Config{DataDir: dataDir}

		// Create directory with a file
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			t.Fatalf("failed to create data dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dataDir, "existing.txt"), []byte("data"), 0644); err != nil {
			t.Fatalf("failed to create existing file: %v", err)
		}

		exitCode := RunRestore(ctx, cfg, []string{"--from", backupPath, "--force"})
		if exitCode != 0 {
			t.Errorf("expected exit code 0 for non-empty dir with force, got %d", exitCode)
		}
	})

	// Test 4: Dry-run should succeed even with non-empty directory (no --force needed)
	t.Run("DryRunNonEmptyDirSucceeds", func(t *testing.T) {
		dataDir := filepath.Join(tempDir, "dryrun-data")
		cfg := config.Config{DataDir: dataDir}

		// Create directory with a file
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			t.Fatalf("failed to create data dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dataDir, "existing.txt"), []byte("data"), 0644); err != nil {
			t.Fatalf("failed to create existing file: %v", err)
		}

		exitCode := RunRestore(ctx, cfg, []string{"--from", backupPath, "--dry-run"})
		if exitCode != 0 {
			t.Errorf("expected exit code 0 for dry-run with non-empty dir, got %d", exitCode)
		}
	})
}

// TestRestoreFromArchivePathTraversal tests that secondary validation in
// restoreFromArchive blocks path traversal attempts even if validateBackup
// were bypassed or the archive is modified between validation and extraction.
func TestRestoreFromArchivePathTraversal(t *testing.T) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")

	// Test case 1: Direct path traversal attempt
	t.Run("DirectTraversalBlocked", func(t *testing.T) {
		backupPath := filepath.Join(tempDir, "traversal-backup.tar.gz")
		if err := createTestBackupWithContent(backupPath, map[string]string{
			"jobs.db":       "valid content",
			"../etc/passwd": "malicious content",
		}); err != nil {
			t.Fatalf("failed to create test backup: %v", err)
		}

		err := restoreFromArchive(backupPath, dataDir, false)
		if err == nil {
			t.Error("restoreFromArchive should fail for archive with path traversal")
		}
	})

	// Test case 2: Mid-path traversal attempt
	t.Run("MidPathTraversalBlocked", func(t *testing.T) {
		backupPath := filepath.Join(tempDir, "mid-traversal-backup.tar.gz")
		if err := createTestBackupWithContent(backupPath, map[string]string{
			"jobs.db":              "valid content",
			"foo/../../etc/shadow": "malicious content",
		}); err != nil {
			t.Fatalf("failed to create test backup: %v", err)
		}

		err := restoreFromArchive(backupPath, dataDir, false)
		if err == nil {
			t.Error("restoreFromArchive should fail for archive with mid-path traversal")
		}
	})

	// Test case 3: Absolute path attempt
	t.Run("AbsolutePathBlocked", func(t *testing.T) {
		backupPath := filepath.Join(tempDir, "absolute-backup.tar.gz")
		if err := createTestBackupWithContent(backupPath, map[string]string{
			"jobs.db":     "valid content",
			"/etc/passwd": "malicious content",
		}); err != nil {
			t.Fatalf("failed to create test backup: %v", err)
		}

		err := restoreFromArchive(backupPath, dataDir, false)
		if err == nil {
			t.Error("restoreFromArchive should fail for archive with absolute path")
		}
	})

	// Test case 4: Valid files should restore successfully
	t.Run("ValidFilesRestored", func(t *testing.T) {
		backupPath := filepath.Join(tempDir, "valid-backup.tar.gz")
		testFiles := map[string]string{
			"jobs.db":                "test database content",
			"auth_vault.json":        `{"profiles": {}}`,
			"subdir/nested/file.txt": "nested content",
		}

		if err := createTestBackupWithContent(backupPath, testFiles); err != nil {
			t.Fatalf("failed to create test backup: %v", err)
		}

		dataDir := filepath.Join(tempDir, "valid-data")
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
	})
}
