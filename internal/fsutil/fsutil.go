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
	"os"
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
