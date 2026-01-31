// Package exporter provides MySQL database export implementation.
//
// This file contains MySQL-specific export functions including
// connection handling, batch operations with transactions, and upsert support.
//
// This file does NOT handle:
// - PostgreSQL or MongoDB exports (see database_postgres.go, database_mongodb.go)
// - Schema definitions (see database_schema.go)
// - SQL query building (see database_helpers.go)
package exporter

import (
	"context"
	"database/sql"
	"io"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	_ "github.com/go-sql-driver/mysql"
)

// exportMySQLStream exports job results to MySQL.
func exportMySQLStream(job model.Job, r io.Reader, cfg DatabaseExportConfig) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	connStr := resolveConnectionString("mysql", cfg.ConnectionString)
	if connStr == "" {
		return apperrors.Validation("mysql connection string required (set SPARTAN_MYSQL_URL or pass connectionString)")
	}

	db, err := sql.Open("mysql", connStr)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to open mysql connection", err)
	}
	defer db.Close()

	// Test connection
	if err := db.PingContext(ctx); err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to ping mysql", err)
	}

	table := cfg.Table
	if table == "" {
		table = defaultTableName(job)
	}

	switch job.Kind {
	case model.KindScrape:
		return exportScrapeToMySQL(ctx, db, r, table, cfg)
	case model.KindCrawl:
		return exportCrawlToMySQL(ctx, db, r, table, cfg)
	case model.KindResearch:
		return exportResearchToMySQL(ctx, db, r, table, cfg)
	default:
		return apperrors.Internal("unknown job kind")
	}
}

// exportScrapeToMySQL exports a single scrape result to MySQL.
func exportScrapeToMySQL(ctx context.Context, db *sql.DB, r io.Reader, table string, cfg DatabaseExportConfig) error {
	item, err := parseSingleReader[ScrapeResult](r)
	if err != nil {
		return err
	}

	// Ensure table exists
	createSQL := buildMySQLCreateTable(table, scrapeResultSchema())
	_, err = db.ExecContext(ctx, createSQL)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to create mysql table", err)
	}

	// Build and execute insert/upsert
	query := buildMySQLInsertQuery(table, cfg, getScrapeResultColumns())
	args := scrapeResultToMySQLArgs(item)

	_, err = db.ExecContext(ctx, query, args...)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to insert scrape result to mysql", err)
	}
	return nil
}

// exportCrawlToMySQL uses batch INSERT with prepared statements.
func exportCrawlToMySQL(ctx context.Context, db *sql.DB, r io.Reader, table string, cfg DatabaseExportConfig) error {
	rs, cleanup, err := ensureSeekable(r)
	if err != nil {
		return err
	}
	defer cleanup()

	// Ensure table exists
	createSQL := buildMySQLCreateTable(table, crawlResultSchema())
	_, err = db.ExecContext(ctx, createSQL)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to create mysql table", err)
	}

	// Process in batches
	const batchSize = 100
	query := buildMySQLInsertQuery(table, cfg, getCrawlResultColumns())

	var batch []CrawlResult
	err = scanReader[CrawlResult](rs, func(item CrawlResult) error {
		batch = append(batch, item)
		if len(batch) >= batchSize {
			if err := flushMySQLBatch(ctx, db, query, batch); err != nil {
				return err
			}
			batch = batch[:0]
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Flush remaining
	if len(batch) > 0 {
		return flushMySQLBatch(ctx, db, query, batch)
	}
	return nil
}

// flushMySQLBatch executes a batch insert for MySQL.
func flushMySQLBatch(ctx context.Context, db *sql.DB, query string, batch []CrawlResult) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to begin mysql transaction", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare mysql statement", err)
	}
	defer stmt.Close()

	for _, item := range batch {
		args := crawlResultToMySQLArgs(item)
		_, err := stmt.ExecContext(ctx, args...)
		if err != nil {
			return apperrors.Wrap(apperrors.KindInternal, "failed to insert mysql batch item", err)
		}
	}

	return tx.Commit()
}

// exportResearchToMySQL exports research results to MySQL.
func exportResearchToMySQL(ctx context.Context, db *sql.DB, r io.Reader, table string, cfg DatabaseExportConfig) error {
	item, err := parseSingleReader[ResearchResult](r)
	if err != nil {
		return err
	}

	// Create main results table
	createSQL := buildMySQLCreateTable(table, researchResultSchema())
	_, err = db.ExecContext(ctx, createSQL)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to create mysql research table", err)
	}

	// Insert main result
	query := buildMySQLInsertQuery(table, cfg, getResearchResultColumns())
	_, err = db.ExecContext(ctx, query, item.Query, item.Summary, item.Confidence, time.Now())
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to insert research result to mysql", err)
	}

	// Create evidence table
	evidenceTable := table + "_evidence"
	createEvidenceSQL := buildMySQLCreateTable(evidenceTable, researchEvidenceSchema())
	_, err = db.ExecContext(ctx, createEvidenceSQL)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to create mysql evidence table", err)
	}

	// Insert evidence
	evidenceQuery := buildMySQLInsertQuery(evidenceTable, cfg, getResearchEvidenceColumns())
	for _, ev := range item.Evidence {
		_, err = db.ExecContext(ctx, evidenceQuery,
			ev.URL, ev.Title, ev.Snippet, ev.Score, ev.Confidence, ev.ClusterID, ev.CitationURL, time.Now())
		if err != nil {
			return apperrors.Wrap(apperrors.KindInternal, "failed to insert mysql evidence", err)
		}
	}

	return nil
}
