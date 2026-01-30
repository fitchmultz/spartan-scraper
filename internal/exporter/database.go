// Package exporter provides database export implementations.
//
// This file implements export to PostgreSQL, MySQL, and MongoDB:
// - Connection management with context timeouts
// - Schema inference from JSON structure
// - Batch inserts for performance
// - Upsert support for idempotent exports
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
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
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

// tableSchema defines column definitions for inferred schemas.
type tableSchema struct {
	Columns []columnDef
}

// columnDef defines a single column in a table.
type columnDef struct {
	Name     string
	Type     string
	Nullable bool
}

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

// scrapeResultSchema returns the schema for scrape results.
func scrapeResultSchema() tableSchema {
	return tableSchema{
		Columns: []columnDef{
			{Name: "url", Type: "TEXT", Nullable: false},
			{Name: "status", Type: "INTEGER", Nullable: false},
			{Name: "title", Type: "TEXT", Nullable: true},
			{Name: "text", Type: "TEXT", Nullable: true},
			{Name: "description", Type: "TEXT", Nullable: true},
			{Name: "extracted_at", Type: "TIMESTAMPTZ", Nullable: false},
		},
	}
}

// crawlResultSchema returns the schema for crawl results.
func crawlResultSchema() tableSchema {
	return tableSchema{
		Columns: []columnDef{
			{Name: "url", Type: "TEXT", Nullable: false},
			{Name: "status", Type: "INTEGER", Nullable: false},
			{Name: "title", Type: "TEXT", Nullable: true},
			{Name: "text", Type: "TEXT", Nullable: true},
			{Name: "extracted_at", Type: "TIMESTAMPTZ", Nullable: false},
		},
	}
}

// researchResultSchema returns the schema for research results.
func researchResultSchema() tableSchema {
	return tableSchema{
		Columns: []columnDef{
			{Name: "query", Type: "TEXT", Nullable: false},
			{Name: "summary", Type: "TEXT", Nullable: true},
			{Name: "confidence", Type: "REAL", Nullable: true},
			{Name: "extracted_at", Type: "TIMESTAMPTZ", Nullable: false},
		},
	}
}

// researchEvidenceSchema returns the schema for research evidence.
func researchEvidenceSchema() tableSchema {
	return tableSchema{
		Columns: []columnDef{
			{Name: "url", Type: "TEXT", Nullable: false},
			{Name: "title", Type: "TEXT", Nullable: true},
			{Name: "snippet", Type: "TEXT", Nullable: true},
			{Name: "score", Type: "REAL", Nullable: true},
			{Name: "confidence", Type: "REAL", Nullable: true},
			{Name: "cluster_id", Type: "TEXT", Nullable: true},
			{Name: "citation_url", Type: "TEXT", Nullable: true},
			{Name: "extracted_at", Type: "TIMESTAMPTZ", Nullable: false},
		},
	}
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

// getScrapeResultColumns returns column names for scrape results.
func getScrapeResultColumns() []string {
	return []string{"url", "status", "title", "text", "description", "extracted_at"}
}

// getCrawlResultColumns returns column names for crawl results.
func getCrawlResultColumns() []string {
	return []string{"url", "status", "title", "text", "extracted_at"}
}

// getResearchResultColumns returns column names for research results.
func getResearchResultColumns() []string {
	return []string{"query", "summary", "confidence", "extracted_at"}
}

// getResearchEvidenceColumns returns column names for research evidence.
func getResearchEvidenceColumns() []string {
	return []string{"url", "title", "snippet", "score", "confidence", "cluster_id", "citation_url", "extracted_at"}
}

// scrapeResultToArgs converts a ScrapeResult to PostgreSQL arguments.
func scrapeResultToArgs(item ScrapeResult) []interface{} {
	title := item.Title
	if item.Normalized.Title != "" {
		title = item.Normalized.Title
	}
	desc := item.Metadata.Description
	if item.Normalized.Description != "" {
		desc = item.Normalized.Description
	}
	return []interface{}{
		item.URL,
		item.Status,
		title,
		item.Text,
		desc,
		time.Now(),
	}
}

// crawlResultToArgs converts a CrawlResult to PostgreSQL arguments.
func crawlResultToArgs(item CrawlResult) []interface{} {
	title := item.Title
	if item.Normalized.Title != "" {
		title = item.Normalized.Title
	}
	return []interface{}{
		item.URL,
		item.Status,
		title,
		item.Text,
		time.Now(),
	}
}

// scrapeResultToMySQLArgs converts a ScrapeResult to MySQL arguments.
func scrapeResultToMySQLArgs(item ScrapeResult) []interface{} {
	title := item.Title
	if item.Normalized.Title != "" {
		title = item.Normalized.Title
	}
	desc := item.Metadata.Description
	if item.Normalized.Description != "" {
		desc = item.Normalized.Description
	}
	return []interface{}{
		item.URL,
		item.Status,
		title,
		item.Text,
		desc,
		time.Now(),
	}
}

// crawlResultToMySQLArgs converts a CrawlResult to MySQL arguments.
func crawlResultToMySQLArgs(item CrawlResult) []interface{} {
	title := item.Title
	if item.Normalized.Title != "" {
		title = item.Normalized.Title
	}
	return []interface{}{
		item.URL,
		item.Status,
		title,
		item.Text,
		time.Now(),
	}
}

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

// exportMongoDBStream exports job results to MongoDB.
func exportMongoDBStream(job model.Job, r io.Reader, cfg DatabaseExportConfig) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	connStr := resolveConnectionString("mongodb", cfg.ConnectionString)
	if connStr == "" {
		return apperrors.Validation("mongodb connection string required (set SPARTAN_MONGODB_URL or pass connectionString)")
	}

	client, err := mongo.Connect(options.Client().ApplyURI(connStr))
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to connect to mongodb", err)
	}
	defer client.Disconnect(ctx)

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to ping mongodb", err)
	}

	dbName := cfg.Database
	if dbName == "" {
		dbName = "spartan"
	}

	collName := cfg.Table
	if collName == "" {
		collName = defaultCollectionName(job)
	}

	coll := client.Database(dbName).Collection(collName)

	switch job.Kind {
	case model.KindScrape:
		return exportScrapeToMongoDB(ctx, coll, r, cfg)
	case model.KindCrawl:
		return exportCrawlToMongoDB(ctx, coll, r, cfg)
	case model.KindResearch:
		return exportResearchToMongoDB(ctx, coll, r, cfg)
	default:
		return apperrors.Internal("unknown job kind")
	}
}

// exportScrapeToMongoDB exports a single scrape result to MongoDB.
func exportScrapeToMongoDB(ctx context.Context, coll *mongo.Collection, r io.Reader, cfg DatabaseExportConfig) error {
	item, err := parseSingleReader[ScrapeResult](r)
	if err != nil {
		return err
	}

	doc := bson.M{
		"url":          item.URL,
		"status":       item.Status,
		"title":        safe(item.Normalized.Title, item.Title),
		"text":         item.Text,
		"description":  safe(item.Normalized.Description, item.Metadata.Description),
		"extracted_at": time.Now(),
	}

	if cfg.Mode == "upsert" && cfg.UpsertKey != "" {
		filter := bson.M{cfg.UpsertKey: item.URL}
		update := bson.M{"$set": doc}
		opts := options.UpdateOne().SetUpsert(true)
		_, err = coll.UpdateOne(ctx, filter, update, opts)
	} else {
		_, err = coll.InsertOne(ctx, doc)
	}

	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to insert scrape result to mongodb", err)
	}
	return nil
}

// exportCrawlToMongoDB uses InsertMany for batch document insertion.
func exportCrawlToMongoDB(ctx context.Context, coll *mongo.Collection, r io.Reader, cfg DatabaseExportConfig) error {
	rs, cleanup, err := ensureSeekable(r)
	if err != nil {
		return err
	}
	defer cleanup()

	const batchSize = 100
	var docs []interface{}

	err = scanReader[CrawlResult](rs, func(item CrawlResult) error {
		doc := bson.M{
			"url":          item.URL,
			"status":       item.Status,
			"title":        safe(item.Normalized.Title, item.Title),
			"text":         item.Text,
			"extracted_at": time.Now(),
		}
		docs = append(docs, doc)

		if len(docs) >= batchSize {
			if err := flushMongoDBBatch(ctx, coll, docs, cfg); err != nil {
				return err
			}
			docs = docs[:0]
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Insert remaining
	if len(docs) > 0 {
		return flushMongoDBBatch(ctx, coll, docs, cfg)
	}
	return nil
}

// flushMongoDBBatch executes a batch insert for MongoDB.
func flushMongoDBBatch(ctx context.Context, coll *mongo.Collection, docs []interface{}, cfg DatabaseExportConfig) error {
	if cfg.Mode == "upsert" && cfg.UpsertKey != "" {
		// Use bulk write for upsert
		var models []mongo.WriteModel
		for _, doc := range docs {
			docMap, ok := doc.(bson.M)
			if !ok {
				continue
			}
			filter := bson.M{cfg.UpsertKey: docMap[cfg.UpsertKey]}
			update := bson.M{"$set": docMap}
			models = append(models, mongo.NewUpdateOneModel().
				SetFilter(filter).
				SetUpdate(update).
				SetUpsert(true))
		}
		if len(models) > 0 {
			opts := options.BulkWrite().SetOrdered(false)
			_, err := coll.BulkWrite(ctx, models, opts)
			if err != nil {
				return apperrors.Wrap(apperrors.KindInternal, "failed to bulk write mongodb documents", err)
			}
		}
	} else {
		opts := options.InsertMany().SetOrdered(false)
		_, err := coll.InsertMany(ctx, docs, opts)
		if err != nil {
			return apperrors.Wrap(apperrors.KindInternal, "failed to insert mongodb documents", err)
		}
	}
	return nil
}

// exportResearchToMongoDB exports research results to MongoDB.
func exportResearchToMongoDB(ctx context.Context, coll *mongo.Collection, r io.Reader, cfg DatabaseExportConfig) error {
	item, err := parseSingleReader[ResearchResult](r)
	if err != nil {
		return err
	}

	// Convert evidence to BSON
	var evidence []bson.M
	for _, ev := range item.Evidence {
		evidence = append(evidence, bson.M{
			"url":          ev.URL,
			"title":        ev.Title,
			"snippet":      ev.Snippet,
			"score":        ev.Score,
			"confidence":   ev.Confidence,
			"cluster_id":   ev.ClusterID,
			"citation_url": ev.CitationURL,
		})
	}

	doc := bson.M{
		"query":        item.Query,
		"summary":      item.Summary,
		"confidence":   item.Confidence,
		"evidence":     evidence,
		"extracted_at": time.Now(),
	}

	if cfg.Mode == "upsert" && cfg.UpsertKey != "" {
		filter := bson.M{cfg.UpsertKey: item.Query}
		update := bson.M{"$set": doc}
		opts := options.UpdateOne().SetUpsert(true)
		_, err = coll.UpdateOne(ctx, filter, update, opts)
	} else {
		_, err = coll.InsertOne(ctx, doc)
	}

	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to insert research result to mongodb", err)
	}
	return nil
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
