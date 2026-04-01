// Package manage provides manage functionality for Spartan Scraper.
//
// Purpose:
// - Implement reset data support for package manage.
//
// Responsibilities:
// - Define the file-local types, functions, and helpers that belong to this package concern.
//
// Scope:
// - Package-internal behavior owned by this file; broader orchestration stays in adjacent package files.
//
// Usage:
// - Used by other files in package `manage` and any exported callers that depend on this package.
//
// Invariants/Assumptions:
// - This file should preserve the package contract and rely on surrounding package configuration as the source of truth.

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
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/fsutil"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

const defaultResetBackupDir = "output/cutover"

func RunResetData(_ context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("reset-data", flag.ExitOnError)
	backupDir := fs.String("backup-dir", defaultResetBackupDir, "Directory for the legacy data archive")
	force := fs.Bool("force", false, "Reset even when the data directory already matches the current schema")
	_ = fs.Parse(args)

	if isDataDirInUse(cfg.DataDir) {
		fmt.Fprintf(os.Stderr, "error: data directory appears to be in use by a running server: %s\n", cfg.DataDir)
		fmt.Fprintln(os.Stderr, "Stop the server before resetting persisted data.")
		return 1
	}

	inspection, err := store.InspectDataDir(cfg.DataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to inspect data directory: %v\n", err)
		return 1
	}

	isEmpty, err := fsutil.IsDirEmpty(cfg.DataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to inspect data directory: %v\n", err)
		return 1
	}

	if inspection.Status == store.DataDirStatusCurrent && !*force {
		fmt.Fprintf(os.Stderr, "refusing to reset current Balanced 1.0 data at %s without --force\n", cfg.DataDir)
		return 1
	}

	if inspection.Status == store.DataDirStatusMissing && !isEmpty && !*force {
		fmt.Fprintf(os.Stderr, "refusing to remove non-empty data directory without --force: %s\n", cfg.DataDir)
		return 1
	}

	backupPath := ""
	if !isEmpty {
		if err := fsutil.EnsureDataDir(*backupDir); err != nil {
			fmt.Fprintf(os.Stderr, "failed to prepare backup directory: %v\n", err)
			return 1
		}
		backupPath = filepath.Join(*backupDir, fmt.Sprintf("spartan-cutover-%s.tar.gz", time.Now().Format("20060102-150405")))
		if err := archiveEntireDataDir(cfg.DataDir, backupPath); err != nil {
			fmt.Fprintf(os.Stderr, "failed to archive data directory: %v\n", err)
			return 1
		}
	}

	if err := os.RemoveAll(cfg.DataDir); err != nil {
		fmt.Fprintf(os.Stderr, "failed to remove data directory: %v\n", err)
		return 1
	}
	if err := fsutil.EnsureDataDir(cfg.DataDir); err != nil {
		fmt.Fprintf(os.Stderr, "failed to recreate data directory: %v\n", err)
		return 1
	}

	if backupPath != "" {
		fmt.Printf("Archived %s to %s\n", cfg.DataDir, backupPath)
	} else {
		fmt.Printf("Data directory %s was already empty\n", cfg.DataDir)
	}
	fmt.Printf("Reset %s for a fresh Balanced 1.0 startup\n", cfg.DataDir)
	return 0
}

func archiveEntireDataDir(dataDir, outputPath string) error {
	outFile, err := fsutil.CreateSecure(outputPath)
	if err != nil {
		return apperrors.Wrap(apperrors.KindPermission, "failed to create backup file", err)
	}
	defer outFile.Close()

	gzWriter := gzip.NewWriter(outFile)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	return filepath.Walk(dataDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(dataDir, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relPath)

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(tarWriter, file)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}
