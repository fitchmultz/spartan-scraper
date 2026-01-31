// Package exporter provides database export helper functions.
//
// This file contains shared utilities used by all database backends
// including connection string resolution, table name generation, and
// SQL query builders.
//
// This file does NOT handle:
// - Actual database connections or driver-specific code
// - Data insertion or batch operations
// - Database-specific export implementations
package exporter

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/jackc/pgx/v5"
)

// resolveConnectionString gets connection string from config or env var.
func resolveConnectionString(dbType string, cfgValue string) string {
	if cfgValue != "" {
		return cfgValue
	}

	switch dbType {
	case "postgres":
		return os.Getenv("SPARTAN_POSTGRES_URL")
	case "mysql":
		return os.Getenv("SPARTAN_MYSQL_URL")
	case "mongodb":
		return os.Getenv("SPARTAN_MONGODB_URL")
	}
	return ""
}

// defaultTableName generates a default table name from job kind.
func defaultTableName(job model.Job) string {
	timestamp := time.Now().Format("20060102_150405")
	return fmt.Sprintf("%s_%s", job.Kind, timestamp)
}

// defaultCollectionName generates a default collection name from job kind.
func defaultCollectionName(job model.Job) string {
	return string(job.Kind)
}

// getPostgresType returns the PostgreSQL type for a column definition.
func getPostgresType(col columnDef) string {
	return col.Type
}

// getMySQLType returns the MySQL type for a column definition.
func getMySQLType(col columnDef) string {
	switch col.Type {
	case "TEXT":
		return "LONGTEXT"
	case "TIMESTAMPTZ":
		return "TIMESTAMP"
	case "INTEGER":
		return "INT"
	case "REAL":
		return "DOUBLE"
	default:
		return col.Type
	}
}

// buildPostgresCreateTable builds a CREATE TABLE statement for PostgreSQL.
func buildPostgresCreateTable(table string, schema tableSchema) string {
	var cols []string
	for _, col := range schema.Columns {
		colType := getPostgresType(col)
		nullConstraint := "NOT NULL"
		if col.Nullable {
			nullConstraint = "NULL"
		}
		cols = append(cols, fmt.Sprintf("%s %s %s", pgx.Identifier{col.Name}.Sanitize(), colType, nullConstraint))
	}

	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)",
		pgx.Identifier{table}.Sanitize(),
		strings.Join(cols, ", "))
}

// buildMySQLCreateTable builds a CREATE TABLE statement for MySQL.
func buildMySQLCreateTable(table string, schema tableSchema) string {
	var cols []string
	for _, col := range schema.Columns {
		colType := getMySQLType(col)
		nullConstraint := "NOT NULL"
		if col.Nullable {
			nullConstraint = "NULL"
		}
		cols = append(cols, fmt.Sprintf("`%s` %s %s", col.Name, colType, nullConstraint))
	}

	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (%s)",
		table,
		strings.Join(cols, ", "))
}

// buildPostgresInsertQuery builds an INSERT or UPSERT query for PostgreSQL.
func buildPostgresInsertQuery(table string, cfg DatabaseExportConfig, columns []string) string {
	quotedTable := pgx.Identifier{table}.Sanitize()
	var quotedCols []string
	var placeholders []string
	for i, col := range columns {
		quotedCols = append(quotedCols, pgx.Identifier{col}.Sanitize())
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		quotedTable,
		strings.Join(quotedCols, ", "),
		strings.Join(placeholders, ", "))

	if cfg.Mode == "upsert" && cfg.UpsertKey != "" {
		query += fmt.Sprintf(" ON CONFLICT (%s) DO UPDATE SET ", pgx.Identifier{cfg.UpsertKey}.Sanitize())
		var updates []string
		for _, col := range columns {
			if col != cfg.UpsertKey {
				updates = append(updates, fmt.Sprintf("%s = EXCLUDED.%s",
					pgx.Identifier{col}.Sanitize(),
					pgx.Identifier{col}.Sanitize()))
			}
		}
		query += strings.Join(updates, ", ")
	}

	return query
}

// buildMySQLInsertQuery builds an INSERT or UPSERT query for MySQL.
func buildMySQLInsertQuery(table string, cfg DatabaseExportConfig, columns []string) string {
	var quotedCols []string
	var placeholders []string
	for _, col := range columns {
		quotedCols = append(quotedCols, fmt.Sprintf("`%s`", col))
		placeholders = append(placeholders, "?")
	}

	query := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s)",
		table,
		strings.Join(quotedCols, ", "),
		strings.Join(placeholders, ", "))

	if cfg.Mode == "upsert" && cfg.UpsertKey != "" {
		query += " ON DUPLICATE KEY UPDATE "
		var updates []string
		for _, col := range columns {
			if col != cfg.UpsertKey {
				updates = append(updates, fmt.Sprintf("`%s` = VALUES(`%s`)", col, col))
			}
		}
		query += strings.Join(updates, ", ")
	}

	return query
}
