// Package exporter provides database export implementations.
//
// This file defines the public API for database exports including
// PostgreSQL, MySQL, and MongoDB. The actual implementations are in
// separate files organized by responsibility.
//
// Environment variables for connection (as fallback):
// - SPARTAN_POSTGRES_URL
// - SPARTAN_MYSQL_URL
// - SPARTAN_MONGODB_URL
//
// This file does NOT handle:
// - Connection pooling (creates new connections per export)
// - Schema migrations for existing tables
// - Database-specific authentication methods (uses connection strings)
// - Complex transaction coordination across multiple tables
package exporter

import (
	"fmt"
	"io"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

// DatabaseExportConfig holds connection and export options for database targets.
type DatabaseExportConfig struct {
	// Connection string (DSN) - can also be set via env var
	ConnectionString string `json:"connectionString,omitempty"`

	// Target table/collection name
	Table string `json:"table,omitempty"`

	// Mode: "insert" (default) or "upsert"
	Mode string `json:"mode,omitempty"`

	// For upsert mode: column(s) to use as unique key
	UpsertKey string `json:"upsertKey,omitempty"`

	// Schema for PostgreSQL (optional, defaults to public)
	Schema string `json:"schema,omitempty"`

	// Database name for MongoDB (optional, parsed from connection string if not set)
	Database string `json:"database,omitempty"`
}

// ExportToDatabase exports job results directly to a database.
// This is a convenience function that wraps ExportStream for database targets.
func ExportToDatabase(job model.Job, r io.Reader, format string, cfg DatabaseExportConfig) error {
	switch format {
	case "postgres":
		return exportPostgresStream(job, r, cfg)
	case "mysql":
		return exportMySQLStream(job, r, cfg)
	case "mongodb":
		return exportMongoDBStream(job, r, cfg)
	default:
		return apperrors.Validation(fmt.Sprintf("unsupported database format: %s", format))
	}
}

// IsDatabaseFormat returns true if the format is a database export format.
func IsDatabaseFormat(format string) bool {
	switch format {
	case "postgres", "mysql", "mongodb":
		return true
	}
	return false
}

// ValidateDatabaseConfig validates the database export configuration.
func ValidateDatabaseConfig(format string, cfg DatabaseExportConfig) error {
	if !IsDatabaseFormat(format) {
		return nil // Not a database format, no validation needed
	}

	connStr := resolveConnectionString(format, cfg.ConnectionString)
	if connStr == "" {
		return apperrors.Validation(fmt.Sprintf("%s connection string required (set SPARTAN_%s_URL or pass connectionString)",
			format, strings.ToUpper(format)))
	}

	if cfg.Mode != "" && cfg.Mode != "insert" && cfg.Mode != "upsert" {
		return apperrors.Validation("mode must be 'insert' or 'upsert'")
	}

	if cfg.Mode == "upsert" && cfg.UpsertKey == "" {
		return apperrors.Validation("upsertKey is required when mode is 'upsert'")
	}

	return nil
}
