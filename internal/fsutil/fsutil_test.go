package fsutil

import (
	"os"
	"path/filepath"
	"runtime"
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
