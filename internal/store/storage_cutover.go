// Package store provides store functionality for Spartan Scraper.
//
// Purpose:
// - Implement storage cutover support for package store.
//
// Responsibilities:
// - Define the file-local types, functions, and helpers that belong to this package concern.
//
// Scope:
// - Package-internal behavior owned by this file; broader orchestration stays in adjacent package files.
//
// Usage:
// - Used by other files in package `store` and any exported callers that depend on this package.
//
// Invariants/Assumptions:
// - This file should preserve the package contract and rely on surrounding package configuration as the source of truth.

package store

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"

	_ "modernc.org/sqlite"
)

type DataDirStatus string

const (
	DataDirStatusMissing     DataDirStatus = "missing"
	DataDirStatusCurrent     DataDirStatus = "current"
	DataDirStatusLegacy      DataDirStatus = "legacy"
	DataDirStatusUnsupported DataDirStatus = "unsupported"
)

type DataDirInspection struct {
	DataDir       string
	DBPath        string
	Status        DataDirStatus
	SchemaVersion string
}

func InspectDataDir(dataDir string) (DataDirInspection, error) {
	report := DataDirInspection{
		DataDir: dataDir,
		DBPath:  filepath.Join(dataDir, "jobs.db"),
		Status:  DataDirStatusMissing,
	}

	_, err := os.Stat(report.DBPath)
	if err != nil {
		if os.IsNotExist(err) {
			return report, nil
		}
		return DataDirInspection{}, apperrors.Wrap(apperrors.KindInternal, "failed to inspect jobs database", err)
	}

	dsn := fmt.Sprintf("file:%s?mode=ro", url.PathEscape(report.DBPath))
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return DataDirInspection{}, apperrors.Wrap(apperrors.KindInternal, "failed to inspect jobs database", err)
	}
	defer db.Close()

	var metadataTable string
	err = db.QueryRow(`select name from sqlite_master where type = 'table' and name = 'store_metadata'`).Scan(&metadataTable)
	switch {
	case err == sql.ErrNoRows:
		report.Status = DataDirStatusLegacy
		return report, nil
	case err != nil:
		return DataDirInspection{}, apperrors.Wrap(apperrors.KindInternal, "failed to inspect store metadata", err)
	}

	var schemaVersion string
	err = db.QueryRow(`select value from store_metadata where key = 'storage_schema'`).Scan(&schemaVersion)
	switch {
	case err == sql.ErrNoRows:
		report.Status = DataDirStatusLegacy
		return report, nil
	case err != nil:
		return DataDirInspection{}, apperrors.Wrap(apperrors.KindInternal, "failed to read store schema version", err)
	}

	report.SchemaVersion = schemaVersion
	if schemaVersion == balanced10StorageSchemaVersion {
		report.Status = DataDirStatusCurrent
		return report, nil
	}

	report.Status = DataDirStatusUnsupported
	return report, nil
}
