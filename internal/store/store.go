// Package store provides SQLite-backed persistent storage for jobs and crawl states.
// It handles job CRUD operations, status tracking, and incremental crawling state.
// It does NOT handle job execution or scheduling.
package store

import (
	"database/sql"
)

type Store struct {
	db      *sql.DB
	dataDir string

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
