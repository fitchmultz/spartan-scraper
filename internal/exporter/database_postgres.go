// Package exporter provides PostgreSQL database export implementation.
//
// This file contains PostgreSQL-specific export functions including
// connection handling, batch operations, and upsert support.
//
// This file does NOT handle:
// - MySQL or MongoDB exports (see database_mysql.go, database_mongodb.go)
// - Schema definitions (see database_schema.go)
// - SQL query building (see database_helpers.go)
package exporter

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/jackc/pgx/v5"
)

// exportPostgresStream exports job results to PostgreSQL.
func exportPostgresStream(job model.Job, r io.Reader, cfg DatabaseExportConfig) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	connStr := resolveConnectionString("postgres", cfg.ConnectionString)
	if connStr == "" {
		return apperrors.Validation("postgres connection string required (set SPARTAN_POSTGRES_URL or pass connectionString)")
	}

	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to connect to postgres", err)
	}
	defer conn.Close(ctx)

	table := cfg.Table
	if table == "" {
		table = defaultTableName(job)
	}

	switch job.Kind {
	case model.KindScrape:
		return exportScrapeToPostgres(ctx, conn, r, table, cfg)
	case model.KindCrawl:
		return exportCrawlToPostgres(ctx, conn, r, table, cfg)
	case model.KindResearch:
		return exportResearchToPostgres(ctx, conn, r, table, cfg)
	default:
		return apperrors.Internal("unknown job kind")
	}
}

// exportScrapeToPostgres exports a single scrape result to PostgreSQL.
func exportScrapeToPostgres(ctx context.Context, conn *pgx.Conn, r io.Reader, table string, cfg DatabaseExportConfig) error {
	item, err := parseSingleReader[ScrapeResult](r)
	if err != nil {
		return err
	}

	// Ensure table exists with inferred schema
	createSQL := buildPostgresCreateTable(table, scrapeResultSchema())
	_, err = conn.Exec(ctx, createSQL)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to create table", err)
	}

	// Build and execute insert/upsert
	query := buildPostgresInsertQuery(table, cfg, getScrapeResultColumns())
	args := scrapeResultToArgs(item)

	_, err = conn.Exec(ctx, query, args...)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to insert scrape result", err)
	}
	return nil
}

// exportCrawlToPostgres exports multiple crawl results using batch COPY.
func exportCrawlToPostgres(ctx context.Context, conn *pgx.Conn, r io.Reader, table string, cfg DatabaseExportConfig) error {
	rs, cleanup, err := ensureSeekable(r)
	if err != nil {
		return err
	}
	defer cleanup()

	// Ensure table exists
	createSQL := buildPostgresCreateTable(table, crawlResultSchema())
	_, err = conn.Exec(ctx, createSQL)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to create table", err)
	}

	// For upsert mode, we need to use INSERT instead of COPY
	if cfg.Mode == "upsert" && cfg.UpsertKey != "" {
		return exportCrawlToPostgresWithUpsert(ctx, conn, rs, table, cfg)
	}

	// Collect rows for COPY
	var rows [][]interface{}
	err = scanReader[CrawlResult](rs, func(item CrawlResult) error {
		rows = append(rows, crawlResultToArgs(item))
		return nil
	})
	if err != nil {
		return err
	}

	if len(rows) == 0 {
		return nil
	}

	// Execute COPY
	copyCount, err := conn.CopyFrom(ctx, pgx.Identifier{table}, getCrawlResultColumns(), pgx.CopyFromRows(rows))
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to copy crawl results", err)
	}

	if copyCount != int64(len(rows)) {
		return apperrors.Internal(fmt.Sprintf("copy mismatch: expected %d rows, got %d", len(rows), copyCount))
	}
	return nil
}

// exportCrawlToPostgresWithUpsert exports crawl results with upsert support.
func exportCrawlToPostgresWithUpsert(ctx context.Context, conn *pgx.Conn, rs io.ReadSeeker, table string, cfg DatabaseExportConfig) error {
	query := buildPostgresInsertQuery(table, cfg, getCrawlResultColumns())

	batchSize := 100
	var batch []CrawlResult

	err := scanReader[CrawlResult](rs, func(item CrawlResult) error {
		batch = append(batch, item)
		if len(batch) >= batchSize {
			if err := flushPostgresBatch(ctx, conn, query, batch); err != nil {
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
		return flushPostgresBatch(ctx, conn, query, batch)
	}
	return nil
}

// flushPostgresBatch executes a batch insert for PostgreSQL.
func flushPostgresBatch(ctx context.Context, conn *pgx.Conn, query string, batch []CrawlResult) error {
	for _, item := range batch {
		args := crawlResultToArgs(item)
		_, err := conn.Exec(ctx, query, args...)
		if err != nil {
			return apperrors.Wrap(apperrors.KindInternal, "failed to insert batch item", err)
		}
	}
	return nil
}

// exportResearchToPostgres exports research results to PostgreSQL.
func exportResearchToPostgres(ctx context.Context, conn *pgx.Conn, r io.Reader, table string, cfg DatabaseExportConfig) error {
	item, err := parseSingleReader[ResearchResult](r)
	if err != nil {
		return err
	}

	// Create main results table
	createSQL := buildPostgresCreateTable(table, researchResultSchema())
	_, err = conn.Exec(ctx, createSQL)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to create research table", err)
	}

	// Insert main result
	query := buildPostgresInsertQuery(table, cfg, getResearchResultColumns())
	_, err = conn.Exec(ctx, query, item.Query, item.Summary, item.Confidence, time.Now())
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to insert research result", err)
	}

	// Create evidence table
	evidenceTable := table + "_evidence"
	createEvidenceSQL := buildPostgresCreateTable(evidenceTable, researchEvidenceSchema())
	_, err = conn.Exec(ctx, createEvidenceSQL)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to create evidence table", err)
	}

	// Insert evidence
	evidenceQuery := buildPostgresInsertQuery(evidenceTable, cfg, getResearchEvidenceColumns())
	for _, ev := range item.Evidence {
		_, err = conn.Exec(ctx, evidenceQuery,
			ev.URL, ev.Title, ev.Snippet, ev.Score, ev.Confidence, ev.ClusterID, ev.CitationURL, time.Now())
		if err != nil {
			return apperrors.Wrap(apperrors.KindInternal, "failed to insert evidence", err)
		}
	}

	return nil
}
