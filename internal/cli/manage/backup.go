// Package manage provides backup CLI command implementation.
//
// This file handles:
// - Creating timestamped backup archives of all data directory contents
// - Listing available backups
// - SQLite-safe backup with WAL checkpointing
//
// It does NOT handle:
// - Remote backup storage (S3, etc.) - local filesystem only
// - Backup encryption (archives are gzip compressed only)
// - Automatic scheduled backups (scheduler handles this)
//
// Invariants:
// - Backup archives use tar.gz format with preserved permissions
// - Database is checkpointed before backup for consistency
// - Backup files are created with 0600 permissions
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
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/fsutil"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

// backupFiles defines the files to include in a backup (relative to dataDir)
var backupFiles = []string{
	"jobs.db",
	"jobs.db-shm",
	"jobs.db-wal",
	"auth_vault.json",
	"render_profiles.json",
	"extract_templates.json",
	"pipeline_js.json",
}

// RunBackup handles the backup subcommand.
func RunBackup(ctx context.Context, cfg config.Config, args []string) int {
	if len(args) < 1 {
		printBackupHelp()
		return 1
	}

	switch args[0] {
	case "create":
		return runBackupCreate(ctx, cfg, args[1:])
	case "list":
		return runBackupList(ctx, cfg, args[1:])
	case "help", "--help", "-h":
		printBackupHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown backup subcommand: %s\n", args[0])
		printBackupHelp()
		return 1
	}
}

func runBackupCreate(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("backup create", flag.ExitOnError)
	outputDir := fs.String("output", ".", "Output directory for backup archive")
	outputDirShort := fs.String("o", ".", "Output directory for backup archive (shorthand)")
	excludeJobs := fs.Bool("exclude-jobs", false, "Exclude job results directories from backup")
	_ = fs.Parse(args)

	// Use -o if provided, otherwise use --output
	outDir := *outputDir
	if *outputDirShort != "." {
		outDir = *outputDirShort
	}

	// Validate output directory
	if err := fsutil.EnsureDataDir(outDir); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create output directory: %v\n", err)
		return 1
	}

	// Open store to checkpoint database
	st, err := store.Open(cfg.DataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open store: %v\n", err)
		return 1
	}

	// Checkpoint WAL before backup
	if err := st.Checkpoint(ctx); err != nil {
		st.Close()
		fmt.Fprintf(os.Stderr, "failed to checkpoint database: %v\n", err)
		return 1
	}
	st.Close()

	// Generate backup filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	backupName := fmt.Sprintf("spartan-backup-%s.tar.gz", timestamp)
	backupPath := filepath.Join(outDir, backupName)

	// Create backup archive
	if err := createBackupArchive(cfg.DataDir, backupPath, *excludeJobs); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create backup: %v\n", err)
		return 1
	}

	fmt.Printf("Backup created: %s\n", backupPath)
	return 0
}

func runBackupList(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("backup list", flag.ExitOnError)
	backupDir := fs.String("dir", ".", "Directory to search for backups")
	_ = fs.Parse(args)

	backups, err := findBackups(*backupDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to list backups: %v\n", err)
		return 1
	}

	if len(backups) == 0 {
		fmt.Println("No backups found")
		return 0
	}

	fmt.Printf("Found %d backup(s):\n\n", len(backups))
	fmt.Printf("%-40s %12s %s\n", "NAME", "SIZE", "MODIFIED")
	fmt.Println(strings.Repeat("-", 70))

	for _, b := range backups {
		fmt.Printf("%-40s %12s %s\n", b.Name, formatBytes(b.Size), b.ModTime.Format("2006-01-02 15:04:05"))
	}

	return 0
}

// backupInfo holds information about a backup file
type backupInfo struct {
	Name    string
	Path    string
	Size    int64
	ModTime time.Time
}

func findBackups(dir string) ([]backupInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to read backup directory", err)
	}

	var backups []backupInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "spartan-backup-") && strings.HasSuffix(name, ".tar.gz") {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			backups = append(backups, backupInfo{
				Name:    name,
				Path:    filepath.Join(dir, name),
				Size:    info.Size(),
				ModTime: info.ModTime(),
			})
		}
	}

	return backups, nil
}

func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func createBackupArchive(dataDir, outputPath string, excludeJobs bool) error {
	// Create output file with secure permissions
	outFile, err := fsutil.CreateSecure(outputPath)
	if err != nil {
		return apperrors.Wrap(apperrors.KindPermission, "failed to create backup file", err)
	}
	defer outFile.Close()

	// Create gzip writer
	gzWriter := gzip.NewWriter(outFile)
	defer gzWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Add core backup files
	for _, file := range backupFiles {
		filePath := filepath.Join(dataDir, file)
		if err := addFileToArchive(tarWriter, filePath, file); err != nil {
			// Skip files that don't exist (e.g., WAL files may not always be present)
			if os.IsNotExist(err) {
				continue
			}
			return apperrors.Wrap(apperrors.KindInternal, fmt.Sprintf("failed to add %s to backup", file), err)
		}
	}

	// Add jobs directory if not excluded
	if !excludeJobs {
		jobsDir := filepath.Join(dataDir, "jobs")
		if err := addDirToArchive(tarWriter, jobsDir, "jobs"); err != nil {
			// Don't fail if jobs directory doesn't exist
			if !os.IsNotExist(err) {
				return apperrors.Wrap(apperrors.KindInternal, "failed to add jobs directory to backup", err)
			}
		}
	}

	return nil
}

func addFileToArchive(tw *tar.Writer, filePath, archivePath string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = archivePath

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(tw, file)
	return err
}

func addDirToArchive(tw *tar.Writer, dirPath, archivePrefix string) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Compute archive path
		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return err
		}
		archivePath := filepath.Join(archivePrefix, relPath)

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = archivePath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// If it's a file, copy its contents
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			if _, err := io.Copy(tw, file); err != nil {
				return err
			}
		}

		return nil
	})
}

func printBackupHelp() {
	fmt.Print(`Usage: spartan backup <command> [options]

Commands:
  create              Create a backup archive of all data
  list                List available backup archives

Create Options:
  -o, --output=DIR    Output directory for backup (default: current directory)
  --exclude-jobs      Exclude job results directories from backup

List Options:
  --dir=DIR           Directory to search for backups (default: current directory)

Examples:
  spartan backup create
  spartan backup create -o /backups
  spartan backup create --exclude-jobs
  spartan backup list
  spartan backup list --dir /backups
`)
}
