// Package store provides store functionality for Spartan Scraper.
//
// Purpose:
// - Implement store support for package store.
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

/*
Purpose: Define the long-lived SQLite store handle shared across storage subdomains.
Responsibilities: Own the database connection, prepared statements, and per-store auxiliary resources such as the dedup content index.
Scope: Store instance structure only; CRUD, migrations, and feature-specific behavior live in sibling files.
Usage: Construct via `Open(...)` and pass the returned `*Store` to API, runtime, scheduler, and retention layers.
Invariants/Assumptions: Each Store owns exactly one database handle and any lazily initialized helpers attached to that handle.
*/
package store

import (
	"database/sql"
	"sync"

	"github.com/fitchmultz/spartan-scraper/internal/dedup"
)

type Store struct {
	db      *sql.DB
	dataDir string

	contentIndexOnce sync.Once
	contentIndex     dedup.ContentIndex
	contentIndexErr  error

	insertJobStmt            *sql.Stmt
	updateJobStatusStmt      *sql.Stmt
	getJobStmt               *sql.Stmt
	getCrawlStateStmt        *sql.Stmt
	upsertCrawlStateStmt     *sql.Stmt
	deleteCrawlStateStmt     *sql.Stmt
	deleteAllCrawlStatesStmt *sql.Stmt

	// Chain statements
	stmtCreateChain    *sql.Stmt
	stmtGetChain       *sql.Stmt
	stmtGetChainByName *sql.Stmt
	stmtUpdateChain    *sql.Stmt
	stmtDeleteChain    *sql.Stmt
	stmtListChains     *sql.Stmt

	// Dependency statements
	stmtGetJobsByDependencyStatus *sql.Stmt
	stmtUpdateDependencyStatus    *sql.Stmt
	stmtGetDependentJobs          *sql.Stmt

	// Analytics statements
	stmtRecordHourlyMetrics *sql.Stmt
	stmtGetHourlyMetrics    *sql.Stmt
	stmtRecordHostMetrics   *sql.Stmt
	stmtGetHostMetrics      *sql.Stmt
	stmtGetTopHosts         *sql.Stmt
	stmtRecordDailyMetrics  *sql.Stmt
	stmtGetDailyMetrics     *sql.Stmt
	stmtRecordJobTrend      *sql.Stmt
	stmtGetJobTrends        *sql.Stmt
	stmtPurgeOldAnalytics   *sql.Stmt
}
