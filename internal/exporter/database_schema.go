// Package exporter provides database export schema definitions.
//
// This file defines the schema types and inference functions for database
// table creation across PostgreSQL, MySQL, and other SQL databases.
//
// This file does NOT handle:
// - Actual database connections or queries
// - Data insertion or batch operations
// - Database-specific type mappings (see database_helpers.go)
package exporter

import (
	"time"
)

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
