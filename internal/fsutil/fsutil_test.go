// Package fsutil provides tests for secure filesystem utilities.
//
// Tests cover:
// - Directory creation with secure permissions (EnsureDataDir, MkdirAllSecure)
// - File creation with secure permissions (WriteFileSecure, CreateSecure)
// - Permission enforcement on Unix systems
// - Nested directory creation
// - Existing file/directory handling
//
// Does NOT test:
// - Windows permission semantics (skipped on Windows)
// - Symbolic link handling
// - File locking or concurrent access
//
// Assumes:
// - DirMode and FileMode constants define secure permissions
// - Unix permission model (tests skip on Windows)
// - Parent directories exist or can be created
package fsutil

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEnsureDataDir_CreatesDirectoryWithCorrectPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping permission tests on Windows")
	}

	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")

	err := EnsureDataDir(dataDir)
	if err != nil {
		t.Fatalf("EnsureDataDir failed: %v", err)
	}

	info, err := os.Stat(dataDir)
	if err != nil {
		t.Fatalf("failed to stat data directory: %v", err)
	}

	if !info.IsDir() {
		t.Errorf("expected %s to be a directory", dataDir)
	}

	mode := info.Mode().Perm()
	if mode != DirMode {
		t.Errorf("expected directory permissions %o, got %o", DirMode, mode)
	}
}

func TestEnsureDataDir_UpdatesExistingDirectoryPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping permission tests on Windows")
	}

	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")

	// Create directory with permissive permissions
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	err := EnsureDataDir(dataDir)
	if err != nil {
		t.Fatalf("EnsureDataDir failed: %v", err)
	}

	info, err := os.Stat(dataDir)
	if err != nil {
		t.Fatalf("failed to stat data directory: %v", err)
	}

	mode := info.Mode().Perm()
	if mode != DirMode {
		t.Errorf("expected directory permissions %o, got %o", DirMode, mode)
	}
}

func TestEnsureDataDir_NestedPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping permission tests on Windows")
	}

	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "a", "b", "c")

	err := EnsureDataDir(dataDir)
	if err != nil {
		t.Fatalf("EnsureDataDir failed: %v", err)
	}

	info, err := os.Stat(dataDir)
	if err != nil {
		t.Fatalf("failed to stat data directory: %v", err)
	}

	if !info.IsDir() {
		t.Errorf("expected %s to be a directory", dataDir)
	}

	mode := info.Mode().Perm()
	if mode != DirMode {
		t.Errorf("expected directory permissions %o, got %o", DirMode, mode)
	}
}

func TestWriteFileSecure_CreatesFileWithCorrectPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping permission tests on Windows")
	}

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	data := []byte("test data")

	err := WriteFileSecure(filePath, data)
	if err != nil {
		t.Fatalf("WriteFileSecure failed: %v", err)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if string(content) != string(data) {
		t.Errorf("expected file content %q, got %q", data, content)
	}

	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}

	mode := info.Mode().Perm()
	if mode != FileMode {
		t.Errorf("expected file permissions %o, got %o", FileMode, mode)
	}
}

func TestCreateSecure_CreatesFileWithCorrectPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping permission tests on Windows")
	}

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")

	f, err := CreateSecure(filePath)
	if err != nil {
		t.Fatalf("CreateSecure failed: %v", err)
	}
	defer f.Close()

	data := []byte("test data")
	if _, err := f.Write(data); err != nil {
		t.Fatalf("failed to write to file: %v", err)
	}
	f.Close()

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if string(content) != string(data) {
		t.Errorf("expected file content %q, got %q", data, content)
	}

	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}

	mode := info.Mode().Perm()
	if mode != FileMode {
		t.Errorf("expected file permissions %o, got %o", FileMode, mode)
	}
}

func TestCreateSecure_TruncatesExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")

	// Create file with initial content
	if err := os.WriteFile(filePath, []byte("old content"), 0o644); err != nil {
		t.Fatalf("failed to create initial file: %v", err)
	}

	f, err := CreateSecure(filePath)
	if err != nil {
		t.Fatalf("CreateSecure failed: %v", err)
	}

	data := []byte("new content")
	if _, err := f.Write(data); err != nil {
		t.Fatalf("failed to write to file: %v", err)
	}
	f.Close()

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if string(content) != string(data) {
		t.Errorf("expected file content %q, got %q", data, content)
	}
}

func TestMkdirAllSecure_CreatesDirectoriesWithCorrectPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping permission tests on Windows")
	}

	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "a", "b", "c")

	err := MkdirAllSecure(nestedDir)
	if err != nil {
		t.Fatalf("MkdirAllSecure failed: %v", err)
	}

	info, err := os.Stat(nestedDir)
	if err != nil {
		t.Fatalf("failed to stat directory: %v", err)
	}

	if !info.IsDir() {
		t.Errorf("expected %s to be a directory", nestedDir)
	}

	mode := info.Mode().Perm()
	if mode != DirMode {
		t.Errorf("expected directory permissions %o, got %o", DirMode, mode)
	}

	// Check intermediate directories also have correct permissions
	for _, dir := range []string{
		filepath.Join(tmpDir, "a"),
		filepath.Join(tmpDir, "a", "b"),
	} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("failed to stat intermediate directory %s: %v", dir, err)
		}

		mode := info.Mode().Perm()
		if mode != DirMode {
			t.Errorf("expected intermediate directory %s permissions %o, got %o", dir, DirMode, mode)
		}
	}
}

func TestMkdirAllSecure_ExistingDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Directory already exists
	err := MkdirAllSecure(tmpDir)
	if err != nil {
		t.Fatalf("MkdirAllSecure failed on existing directory: %v", err)
	}
}

func TestIsDirEmpty(t *testing.T) {
	tempDir := t.TempDir()

	// Test 1: Non-existent directory should be considered empty
	t.Run("NonExistentDir", func(t *testing.T) {
		nonExistent := filepath.Join(tempDir, "does-not-exist")
		isEmpty, err := IsDirEmpty(nonExistent)
		if err != nil {
			t.Errorf("IsDirEmpty should not error for non-existent directory: %v", err)
		}
		if !isEmpty {
			t.Error("IsDirEmpty should return true for non-existent directory")
		}
	})

	// Test 2: Empty directory should return true
	t.Run("EmptyDir", func(t *testing.T) {
		emptyDir := filepath.Join(tempDir, "empty")
		if err := os.MkdirAll(emptyDir, 0755); err != nil {
			t.Fatalf("failed to create empty directory: %v", err)
		}

		isEmpty, err := IsDirEmpty(emptyDir)
		if err != nil {
			t.Errorf("IsDirEmpty should not error for empty directory: %v", err)
		}
		if !isEmpty {
			t.Error("IsDirEmpty should return true for empty directory")
		}
	})

	// Test 3: Directory with a file should return false
	t.Run("DirWithFile", func(t *testing.T) {
		dirWithFile := filepath.Join(tempDir, "with-file")
		if err := os.MkdirAll(dirWithFile, 0755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dirWithFile, "test.txt"), []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		isEmpty, err := IsDirEmpty(dirWithFile)
		if err != nil {
			t.Errorf("IsDirEmpty should not error: %v", err)
		}
		if isEmpty {
			t.Error("IsDirEmpty should return false for directory with file")
		}
	})

	// Test 4: Directory with subdirectory should return false
	t.Run("DirWithSubdir", func(t *testing.T) {
		dirWithSubdir := filepath.Join(tempDir, "with-subdir")
		if err := os.MkdirAll(filepath.Join(dirWithSubdir, "nested"), 0755); err != nil {
			t.Fatalf("failed to create nested directory: %v", err)
		}

		isEmpty, err := IsDirEmpty(dirWithSubdir)
		if err != nil {
			t.Errorf("IsDirEmpty should not error: %v", err)
		}
		if isEmpty {
			t.Error("IsDirEmpty should return false for directory with subdirectory")
		}
	})

	// Test 5: Directory with hidden file should return false
	t.Run("DirWithHiddenFile", func(t *testing.T) {
		dirWithHidden := filepath.Join(tempDir, "with-hidden")
		if err := os.MkdirAll(dirWithHidden, 0755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dirWithHidden, ".hidden"), []byte("secret"), 0644); err != nil {
			t.Fatalf("failed to create hidden file: %v", err)
		}

		isEmpty, err := IsDirEmpty(dirWithHidden)
		if err != nil {
			t.Errorf("IsDirEmpty should not error: %v", err)
		}
		if isEmpty {
			t.Error("IsDirEmpty should return false for directory with hidden file")
		}
	})
}

func TestWriteFileAtomic(t *testing.T) {
	t.Run("creates new file", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		data := []byte("atomic test data")

		err := WriteFileAtomic(filePath, data, 0o644)
		if err != nil {
			t.Fatalf("WriteFileAtomic failed: %v", err)
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		if string(content) != string(data) {
			t.Errorf("expected content %q, got %q", data, content)
		}
	})

	t.Run("replaces existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")

		// Create initial file
		if err := os.WriteFile(filePath, []byte("old data"), 0o644); err != nil {
			t.Fatalf("failed to create initial file: %v", err)
		}

		newData := []byte("new atomic data")
		err := WriteFileAtomic(filePath, newData, 0o644)
		if err != nil {
			t.Fatalf("WriteFileAtomic failed: %v", err)
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		if string(content) != string(newData) {
			t.Errorf("expected content %q, got %q", newData, content)
		}
	})

	t.Run("creates parent directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "a", "b", "c", "test.txt")
		data := []byte("nested data")

		err := WriteFileAtomic(filePath, data, 0o644)
		if err != nil {
			t.Fatalf("WriteFileAtomic failed: %v", err)
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		if string(content) != string(data) {
			t.Errorf("expected content %q, got %q", data, content)
		}
	})

	t.Run("sets correct permissions", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("skipping permission tests on Windows")
		}

		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		data := []byte("permission test")

		err := WriteFileAtomic(filePath, data, 0o640)
		if err != nil {
			t.Fatalf("WriteFileAtomic failed: %v", err)
		}

		info, err := os.Stat(filePath)
		if err != nil {
			t.Fatalf("failed to stat file: %v", err)
		}

		mode := info.Mode().Perm()
		if mode != 0o640 {
			t.Errorf("expected permissions %o, got %o", 0o640, mode)
		}
	})

	t.Run("no temp file left behind on success", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		data := []byte("cleanup test")

		err := WriteFileAtomic(filePath, data, 0o644)
		if err != nil {
			t.Fatalf("WriteFileAtomic failed: %v", err)
		}

		// Check no .tmp-* files exist
		entries, err := os.ReadDir(tmpDir)
		if err != nil {
			t.Fatalf("failed to read directory: %v", err)
		}

		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), ".tmp-") {
				t.Errorf("temp file not cleaned up: %s", entry.Name())
			}
		}
	})

	t.Run("no partial file on write failure", func(t *testing.T) {
		// This is hard to test directly, but we can verify the target file
		// doesn't exist if we use an unwritable directory
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")

		// Make directory unwritable
		if err := os.Chmod(tmpDir, 0o555); err != nil {
			t.Fatalf("failed to chmod directory: %v", err)
		}
		defer os.Chmod(tmpDir, 0o755) // Restore permissions for cleanup

		data := []byte("should not be written")
		err := WriteFileAtomic(filePath, data, 0o644)
		if err == nil {
			t.Fatal("expected error for unwritable directory")
		}

		// File should not exist
		_, statErr := os.Stat(filePath)
		if !os.IsNotExist(statErr) {
			t.Error("file should not exist after failed write")
		}
	})
}
