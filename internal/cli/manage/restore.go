// Package manage provides restore CLI command implementation.
//
// This file handles:
// - Restoring data from backup archives
// - Validating backup integrity before restore
// - Dry-run mode for previewing restore operations
// - Checking if server is running before restore
//
// It does NOT handle:
// - Remote backup restoration (S3, etc.) - local filesystem only
// - Backup decryption (archives are gzip compressed only)
// - Selective restoration (all-or-nothing per archive)
//
// Invariants:
// - Backup archives are validated before any restoration
// - Server must not be running (or --force must be used)
// - Original permissions are preserved during restore
// - Dry-run mode previews changes without modifying data
package manage

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/fsutil"
)

// expectedBackupFiles are the files we expect to find in a valid backup
var expectedBackupFiles = []string{
	"jobs.db",
}

// RunRestore handles the restore subcommand.
func RunRestore(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("restore", flag.ExitOnError)
	from := fs.String("from", "", "Path to backup archive (required)")
	force := fs.Bool("force", false, "Restore even if data directory is not empty")
	dryRun := fs.Bool("dry-run", false, "Preview what would be restored without making changes")
	_ = fs.Parse(args)

	if *from == "" {
		fmt.Fprintln(os.Stderr, "error: --from is required")
		printRestoreHelp()
		return 1
	}

	// Validate backup archive exists
	if _, err := os.Stat(*from); err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "backup file not found: %s\n", *from)
			return 1
		}
		fmt.Fprintf(os.Stderr, "failed to access backup file: %v\n", err)
		return 1
	}

	// Validate backup integrity
	if err := validateBackup(*from); err != nil {
		fmt.Fprintf(os.Stderr, "backup validation failed: %v\n", err)
		return 1
	}

	// Check if server might be running
	if !*dryRun && isDataDirInUse(cfg.DataDir) && !*force {
		fmt.Fprintln(os.Stderr, "error: data directory appears to be in use by a running server")
		fmt.Fprintln(os.Stderr, "Use --force to restore anyway (may corrupt running server data)")
		return 1
	}

	// Perform restore (or dry-run)
	if err := restoreFromArchive(*from, cfg.DataDir, *dryRun); err != nil {
		fmt.Fprintf(os.Stderr, "restore failed: %v\n", err)
		return 1
	}

	if *dryRun {
		fmt.Println("\n(DRY RUN - no changes were made)")
	} else {
		fmt.Printf("\nRestore completed successfully from: %s\n", *from)
	}

	return 0
}

func validateBackup(archivePath string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return apperrors.Wrap(apperrors.KindPermission, "failed to open backup archive", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return apperrors.Wrap(apperrors.KindValidation, "invalid gzip archive", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	foundFiles := make(map[string]bool)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return apperrors.Wrap(apperrors.KindValidation, "corrupted archive", err)
		}

		// Check for suspicious paths (path traversal)
		if containsPathTraversal(header.Name) {
			return apperrors.Validation(fmt.Sprintf("suspicious path in archive: %s", header.Name))
		}

		foundFiles[header.Name] = true
	}

	// Check for required files
	for _, required := range expectedBackupFiles {
		if !foundFiles[required] {
			return apperrors.Validation(fmt.Sprintf("backup missing required file: %s", required))
		}
	}

	return nil
}

func containsPathTraversal(path string) bool {
	// Check for path traversal attempts.
	// TAR format always uses '/' as path separator per POSIX standard,
	// so we primarily split on '/' to detect traversal consistently.
	parts := strings.Split(path, "/")
	for _, part := range parts {
		if part == ".." {
			return true
		}
	}

	// Also check for '..' in backslash-separated paths (defense in depth).
	// While TAR uses '/', we also check for Windows-style paths to handle
	// any edge cases or malicious archives that might use backslashes.
	parts = strings.Split(path, "\\")
	for _, part := range parts {
		if part == ".." {
			return true
		}
	}

	// Check for absolute paths (Unix '/path' and Windows '\\server\share' or 'C:\path').
	// Note: We check for ':' to catch Windows drive letters like "C:".
	return strings.HasPrefix(path, "/") || strings.HasPrefix(path, "\\") || strings.Contains(path, ":")
}

func isDataDirInUse(dataDir string) bool {
	// Try to detect if server is running by checking for lock file
	// or attempting to open the database exclusively
	lockFile := filepath.Join(dataDir, ".server-lock")
	if _, err := os.Stat(lockFile); err == nil {
		return true
	}

	// Additional check: try to open a test file exclusively
	testFile := filepath.Join(dataDir, ".write-test-restore")
	f, err := os.OpenFile(testFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, fsutil.FileMode)
	if err != nil {
		// If we can't create exclusively, something might be using the directory
		return true
	}
	f.Close()
	_ = os.Remove(testFile)

	return false
}

func restoreFromArchive(archivePath, dataDir string, dryRun bool) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return apperrors.Wrap(apperrors.KindPermission, "failed to open backup archive", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return apperrors.Wrap(apperrors.KindValidation, "invalid gzip archive", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	// First pass: collect what would be restored
	var filesToRestore []tar.Header
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return apperrors.Wrap(apperrors.KindValidation, "corrupted archive", err)
		}
		filesToRestore = append(filesToRestore, *header)
	}

	if dryRun {
		fmt.Println("Files that would be restored:")
		for _, h := range filesToRestore {
			fmt.Printf("  %s (%s)\n", h.Name, formatBytes(h.Size))
		}
		return nil
	}

	// Ensure data directory exists
	if err := fsutil.EnsureDataDir(dataDir); err != nil {
		return apperrors.Wrap(apperrors.KindPermission, "failed to create data directory", err)
	}

	// Reset to beginning of archive for actual extraction
	if _, err := file.Seek(0, 0); err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to reset archive position", err)
	}

	gzReader2, err := gzip.NewReader(file)
	if err != nil {
		return apperrors.Wrap(apperrors.KindValidation, "invalid gzip archive", err)
	}
	defer gzReader2.Close()

	tarReader2 := tar.NewReader(gzReader2)

	// Pre-compute absolute dataDir path for containment checks
	absDataDir, err := filepath.Abs(dataDir)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to resolve data directory", err)
	}

	restoredCount := 0
	for {
		header, err := tarReader2.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return apperrors.Wrap(apperrors.KindValidation, "corrupted archive", err)
		}

		// Re-validate path traversal before extraction (defense in depth).
		// This catches any bypass attempts that might have slipped past validateBackup.
		if containsPathTraversal(header.Name) {
			return apperrors.Validation(fmt.Sprintf("suspicious path in archive: %s", header.Name))
		}

		targetPath := filepath.Join(dataDir, header.Name)

		// Verify the resolved path stays within dataDir (prevents symlink attacks
		// and path normalization issues like 'foo/bar/../../../etc/passwd').
		absTargetPath, err := filepath.Abs(targetPath)
		if err != nil {
			return apperrors.Wrap(apperrors.KindInternal, fmt.Sprintf("failed to resolve target path for %s", header.Name), err)
		}
		relPath, err := filepath.Rel(absDataDir, absTargetPath)
		if err != nil {
			return apperrors.Wrap(apperrors.KindInternal, fmt.Sprintf("failed to compute relative path for %s", header.Name), err)
		}
		if strings.HasPrefix(relPath, "..") {
			return apperrors.Validation(fmt.Sprintf("path escapes data directory: %s", header.Name))
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := fsutil.MkdirAllSecure(targetPath); err != nil {
				return apperrors.Wrap(apperrors.KindPermission, fmt.Sprintf("failed to create directory %s", header.Name), err)
			}
			restoredCount++

		case tar.TypeReg:
			// Ensure parent directory exists
			parentDir := filepath.Dir(targetPath)
			if err := fsutil.MkdirAllSecure(parentDir); err != nil {
				return apperrors.Wrap(apperrors.KindPermission, fmt.Sprintf("failed to create parent directory for %s", header.Name), err)
			}

			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return apperrors.Wrap(apperrors.KindPermission, fmt.Sprintf("failed to create file %s", header.Name), err)
			}

			if _, err := io.Copy(outFile, tarReader2); err != nil {
				outFile.Close()
				return apperrors.Wrap(apperrors.KindInternal, fmt.Sprintf("failed to extract file %s", header.Name), err)
			}
			outFile.Close()
			restoredCount++

		default:
			// Skip other file types (symlinks, etc.)
			continue
		}
	}

	fmt.Printf("Restored %d file(s)/director(ies)\n", restoredCount)
	return nil
}

func printRestoreHelp() {
	fmt.Print(`Usage: spartan restore --from <backup-file> [options]

Options:
  --from=PATH         Path to backup archive (required)
  --force             Restore even if data directory appears to be in use
  --dry-run           Preview what would be restored without making changes

Examples:
  spartan restore --from spartan-backup-20240115-120000.tar.gz
  spartan restore --from /backups/spartan-backup-20240115-120000.tar.gz --dry-run
  spartan restore --from backup.tar.gz --force

Security Notes:
  - Backup archives are validated before restoration
  - Path traversal attempts in archives are rejected
  - Original file permissions are preserved
  - The server should not be running during restore
`)
}
