// Package fsutil provides filesystem utilities for secure data directory management.
// It handles creation of directories with restricted permissions (0700) and
// provides helpers for writing files with 0600 permissions.
//
// This package does NOT handle platform-specific permission behavior on Windows.
// Callers should test accordingly on Windows platforms.
//
// Migration note: This package only applies secure permissions to NEW directories
// and files. Existing directories and files are NOT modified to avoid breaking
// running systems.
package fsutil

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

const (
	// DirMode is the permission mode for data directories (owner-only: rwx------)
	DirMode = 0o700
	// FileMode is the permission mode for secret-bearing files (owner-only: rw-------)
	FileMode = 0o600
)

// EnsureDataDir creates the data directory with 0700 permissions (owner-only).
// If the directory already exists, it ensures the permissions are at least 0700
// by applying 0700 to the existing directory.
// Returns an error if the directory cannot be created or permissions cannot be set.
func EnsureDataDir(path string) error {
	if err := os.MkdirAll(path, DirMode); err != nil {
		return fmt.Errorf("failed to create data directory %s: %w", path, err)
	}

	// On Unix systems, ensure the directory has the correct permissions
	// This handles the case where the directory already existed with different permissions
	if runtime.GOOS != "windows" {
		if err := os.Chmod(path, DirMode); err != nil {
			return fmt.Errorf("failed to set permissions on data directory %s: %w", path, err)
		}
	}

	return nil
}

// WriteFileSecure writes data to a file with 0600 permissions (owner-only read/write).
// The file is created if it doesn't exist, or truncated if it does.
// Returns an error if the file cannot be written.
func WriteFileSecure(path string, data []byte) error {
	if err := os.WriteFile(path, data, FileMode); err != nil {
		return fmt.Errorf("failed to write secure file %s: %w", path, err)
	}
	return nil
}

// CreateSecure creates a file with 0600 permissions and returns the file handle.
// This is equivalent to os.OpenFile with O_CREATE|O_WRONLY|O_TRUNC and 0600 perms.
// The caller is responsible for closing the file.
func CreateSecure(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, FileMode)
	if err != nil {
		return nil, fmt.Errorf("failed to create secure file %s: %w", path, err)
	}
	return f, nil
}

// MkdirAllSecure creates a directory path with 0700 permissions.
// It is equivalent to os.MkdirAll with 0700 perms.
func MkdirAllSecure(path string) error {
	if err := os.MkdirAll(path, DirMode); err != nil {
		return fmt.Errorf("failed to create secure directory %s: %w", path, err)
	}
	return nil
}

// IsDirEmpty returns true if the directory exists and contains no entries
// (excluding "." and ".."). Returns true if the directory doesn't exist
// (treated as empty for restore use case). Returns false if the directory
// contains any files or directories.
func IsDirEmpty(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil // Non-existent directory is considered "empty"
		}
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1) // Read just one entry
	if err == io.EOF {
		return true, nil // Directory is empty
	}
	if err != nil {
		return false, err
	}
	return false, nil // Directory has at least one entry
}

// WriteFileAtomic writes data to a file atomically using a temp file and rename.
// This ensures that readers never see a partially written file.
//
// The write process:
// 1. Create a temp file in the same directory as the target file
// 2. Write data to the temp file
// 3. Sync the temp file to disk (fsync)
// 4. Close the temp file
// 5. Rename temp file to target file (atomic on POSIX systems)
// 6. Best-effort sync of the parent directory (for durability)
//
// If any step fails, the temp file is cleaned up.
// The parent directory must exist or be creatable.
func WriteFileAtomic(path string, data []byte, mode os.FileMode) error {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, DirMode); err != nil {
		return fmt.Errorf("failed to create parent directory for %s: %w", path, err)
	}

	// Create temp file in same directory (for atomic rename)
	tempFile, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file for %s: %w", path, err)
	}
	tempPath := tempFile.Name()

	// Clean up temp file on any error
	cleanup := func() {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}

	// Write data
	if _, err := tempFile.Write(data); err != nil {
		cleanup()
		return fmt.Errorf("failed to write to temp file %s: %w", tempPath, err)
	}

	// Sync to disk for durability
	if err := tempFile.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("failed to sync temp file %s: %w", tempPath, err)
	}

	// Close the file before renaming
	if err := tempFile.Close(); err != nil {
		cleanup()
		return fmt.Errorf("failed to close temp file %s: %w", tempPath, err)
	}

	// Set the desired permissions on the temp file
	if err := os.Chmod(tempPath, mode); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to set permissions on temp file %s: %w", tempPath, err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to rename temp file to %s: %w", path, err)
	}

	// Best-effort sync of parent directory (not critical for atomicity, but helps durability)
	if runtime.GOOS != "windows" {
		dirFile, err := os.Open(dir)
		if err == nil {
			_ = dirFile.Sync()
			dirFile.Close()
		}
	}

	return nil
}
