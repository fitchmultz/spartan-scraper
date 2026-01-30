// Package exporter provides database export integration tests.
//
// These tests use testcontainers to spin up real PostgreSQL, MySQL, and MongoDB
// instances for testing. They are skipped in short mode (CI) or when Docker is
// not available.
//
// This file does NOT test:
// - File-based export formats (see other test files)
// - Connection pool behavior under high concurrency
// - Schema migration scenarios for existing tables
// - Database-specific SQL dialect edge cases
package exporter

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// dockerAvailable checks if Docker is available for testcontainers.
// Testcontainers requires a properly configured Docker environment.
func dockerAvailable() bool {
	// Check if TESTCONTAINERS_ENABLED is explicitly set
	if os.Getenv("TESTCONTAINERS_ENABLED") != "1" {
		return false
	}
	cmd := exec.Command("docker", "info")
	err := cmd.Run()
	return err == nil
}

// TestExportPostgres tests PostgreSQL export functionality.
func TestExportPostgres(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if !dockerAvailable() {
		t.Skip("skipping integration test: Docker not available")
	}

	ctx := context.Background()

	// Start PostgreSQL container
	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
	)
	require.NoError(t, err)
	defer container.Terminate(ctx)

	connStr, err := container.ConnectionString(ctx)
	require.NoError(t, err)

	t.Run("scrape", func(t *testing.T) {
		job := model.Job{Kind: model.KindScrape}
		data := `{"url":"http://example.com","status":200,"title":"Test Page","text":"content","metadata":{"description":"desc"},"normalized":{}}`

		cfg := DatabaseExportConfig{
			ConnectionString: connStr,
			Table:            "test_scrape",
		}

		err := exportPostgresStream(job, strings.NewReader(data), cfg)
		require.NoError(t, err)
	})

	t.Run("crawl", func(t *testing.T) {
		job := model.Job{Kind: model.KindCrawl}
		data := `{"url":"http://example.com","status":200,"title":"Test","text":"content","normalized":{}}
{"url":"http://example2.com","status":200,"title":"Test2","text":"content2","normalized":{}}`

		cfg := DatabaseExportConfig{
			ConnectionString: connStr,
			Table:            "test_crawl",
		}

		err := exportPostgresStream(job, strings.NewReader(data), cfg)
		require.NoError(t, err)
	})

	t.Run("crawl_upsert", func(t *testing.T) {
		job := model.Job{Kind: model.KindCrawl}
		data := `{"url":"http://example.com","status":200,"title":"Test","text":"content","normalized":{}}`

		cfg := DatabaseExportConfig{
			ConnectionString: connStr,
			Table:            "test_crawl_upsert",
			Mode:             "upsert",
			UpsertKey:        "url",
		}

		// First insert
		err := exportPostgresStream(job, strings.NewReader(data), cfg)
		require.NoError(t, err)

		// Second insert (should update)
		err = exportPostgresStream(job, strings.NewReader(data), cfg)
		require.NoError(t, err)
	})

	t.Run("research", func(t *testing.T) {
		job := model.Job{Kind: model.KindResearch}
		data := `{"query":"test query","summary":"test summary","confidence":0.95,"evidence":[{"url":"http://example.com","title":"Test","snippet":"snippet","score":0.9,"confidence":0.8,"clusterId":"c1","citationUrl":"http://example.com"}],"clusters":[],"citations":[]}`

		cfg := DatabaseExportConfig{
			ConnectionString: connStr,
			Table:            "test_research",
		}

		err := exportPostgresStream(job, strings.NewReader(data), cfg)
		require.NoError(t, err)
	})
}

// TestExportMySQL tests MySQL export functionality.
func TestExportMySQL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if !dockerAvailable() {
		t.Skip("skipping integration test: Docker not available")
	}

	ctx := context.Background()

	// Start MySQL container
	container, err := mysql.Run(ctx, "mysql:8.0",
		mysql.WithDatabase("test"),
		mysql.WithUsername("test"),
		mysql.WithPassword("test"),
	)
	require.NoError(t, err)
	defer container.Terminate(ctx)

	connStr, err := container.ConnectionString(ctx, "parseTime=true")
	require.NoError(t, err)

	t.Run("scrape", func(t *testing.T) {
		job := model.Job{Kind: model.KindScrape}
		data := `{"url":"http://example.com","status":200,"title":"Test Page","text":"content","metadata":{"description":"desc"},"normalized":{}}`

		cfg := DatabaseExportConfig{
			ConnectionString: connStr,
			Table:            "test_scrape",
		}

		err := exportMySQLStream(job, strings.NewReader(data), cfg)
		require.NoError(t, err)
	})

	t.Run("crawl", func(t *testing.T) {
		job := model.Job{Kind: model.KindCrawl}
		data := `{"url":"http://example.com","status":200,"title":"Test","text":"content","normalized":{}}
{"url":"http://example2.com","status":200,"title":"Test2","text":"content2","normalized":{}}`

		cfg := DatabaseExportConfig{
			ConnectionString: connStr,
			Table:            "test_crawl",
		}

		err := exportMySQLStream(job, strings.NewReader(data), cfg)
		require.NoError(t, err)
	})

	t.Run("crawl_upsert", func(t *testing.T) {
		job := model.Job{Kind: model.KindCrawl}
		data := `{"url":"http://example.com","status":200,"title":"Test","text":"content","normalized":{}}`

		cfg := DatabaseExportConfig{
			ConnectionString: connStr,
			Table:            "test_crawl_upsert",
			Mode:             "upsert",
			UpsertKey:        "url",
		}

		// First insert
		err := exportMySQLStream(job, strings.NewReader(data), cfg)
		require.NoError(t, err)

		// Second insert (should update)
		err = exportMySQLStream(job, strings.NewReader(data), cfg)
		require.NoError(t, err)
	})

	t.Run("research", func(t *testing.T) {
		job := model.Job{Kind: model.KindResearch}
		data := `{"query":"test query","summary":"test summary","confidence":0.95,"evidence":[{"url":"http://example.com","title":"Test","snippet":"snippet","score":0.9,"confidence":0.8,"clusterId":"c1","citationUrl":"http://example.com"}],"clusters":[],"citations":[]}`

		cfg := DatabaseExportConfig{
			ConnectionString: connStr,
			Table:            "test_research",
		}

		err := exportMySQLStream(job, strings.NewReader(data), cfg)
		require.NoError(t, err)
	})
}

// TestExportMongoDB tests MongoDB export functionality.
func TestExportMongoDB(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if !dockerAvailable() {
		t.Skip("skipping integration test: Docker not available")
	}

	ctx := context.Background()

	// Start MongoDB container
	container, err := mongodb.Run(ctx, "mongo:7")
	require.NoError(t, err)
	defer container.Terminate(ctx)

	connStr, err := container.ConnectionString(ctx)
	require.NoError(t, err)

	t.Run("scrape", func(t *testing.T) {
		job := model.Job{Kind: model.KindScrape}
		data := `{"url":"http://example.com","status":200,"title":"Test Page","text":"content","metadata":{"description":"desc"},"normalized":{}}`

		cfg := DatabaseExportConfig{
			ConnectionString: connStr,
			Table:            "test_scrape",
			Database:         "test",
		}

		err := exportMongoDBStream(job, strings.NewReader(data), cfg)
		require.NoError(t, err)
	})

	t.Run("crawl", func(t *testing.T) {
		job := model.Job{Kind: model.KindCrawl}
		data := `{"url":"http://example.com","status":200,"title":"Test","text":"content","normalized":{}}
{"url":"http://example2.com","status":200,"title":"Test2","text":"content2","normalized":{}}`

		cfg := DatabaseExportConfig{
			ConnectionString: connStr,
			Table:            "test_crawl",
			Database:         "test",
		}

		err := exportMongoDBStream(job, strings.NewReader(data), cfg)
		require.NoError(t, err)
	})

	t.Run("crawl_upsert", func(t *testing.T) {
		job := model.Job{Kind: model.KindCrawl}
		data := `{"url":"http://example.com","status":200,"title":"Test","text":"content","normalized":{}}`

		cfg := DatabaseExportConfig{
			ConnectionString: connStr,
			Table:            "test_crawl_upsert",
			Database:         "test",
			Mode:             "upsert",
			UpsertKey:        "url",
		}

		// First insert
		err := exportMongoDBStream(job, strings.NewReader(data), cfg)
		require.NoError(t, err)

		// Second insert (should update)
		err = exportMongoDBStream(job, strings.NewReader(data), cfg)
		require.NoError(t, err)
	})

	t.Run("research", func(t *testing.T) {
		job := model.Job{Kind: model.KindResearch}
		data := `{"query":"test query","summary":"test summary","confidence":0.95,"evidence":[{"url":"http://example.com","title":"Test","snippet":"snippet","score":0.9,"confidence":0.8,"clusterId":"c1","citationUrl":"http://example.com"}],"clusters":[],"citations":[]}`

		cfg := DatabaseExportConfig{
			ConnectionString: connStr,
			Table:            "test_research",
			Database:         "test",
		}

		err := exportMongoDBStream(job, strings.NewReader(data), cfg)
		require.NoError(t, err)
	})
}

// TestDatabaseConfigValidation tests configuration validation.
func TestDatabaseConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		format    string
		cfg       DatabaseExportConfig
		wantError bool
	}{
		{
			name:      "postgres without connection string",
			format:    "postgres",
			cfg:       DatabaseExportConfig{},
			wantError: true,
		},
		{
			name:   "postgres with connection string",
			format: "postgres",
			cfg: DatabaseExportConfig{
				ConnectionString: "postgres://user:pass@localhost/db",
			},
			wantError: false,
		},
		{
			name:      "mysql without connection string",
			format:    "mysql",
			cfg:       DatabaseExportConfig{},
			wantError: true,
		},
		{
			name:   "mysql with connection string",
			format: "mysql",
			cfg: DatabaseExportConfig{
				ConnectionString: "user:pass@tcp(localhost:3306)/db",
			},
			wantError: false,
		},
		{
			name:      "mongodb without connection string",
			format:    "mongodb",
			cfg:       DatabaseExportConfig{},
			wantError: true,
		},
		{
			name:   "mongodb with connection string",
			format: "mongodb",
			cfg: DatabaseExportConfig{
				ConnectionString: "mongodb://localhost:27017/db",
			},
			wantError: false,
		},
		{
			name:      "non-database format",
			format:    "json",
			cfg:       DatabaseExportConfig{},
			wantError: false,
		},
		{
			name:   "upsert without key",
			format: "postgres",
			cfg: DatabaseExportConfig{
				ConnectionString: "postgres://user:pass@localhost/db",
				Mode:             "upsert",
			},
			wantError: true,
		},
		{
			name:   "upsert with key",
			format: "postgres",
			cfg: DatabaseExportConfig{
				ConnectionString: "postgres://user:pass@localhost/db",
				Mode:             "upsert",
				UpsertKey:        "url",
			},
			wantError: false,
		},
		{
			name:   "invalid mode",
			format: "postgres",
			cfg: DatabaseExportConfig{
				ConnectionString: "postgres://user:pass@localhost/db",
				Mode:             "invalid",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDatabaseConfig(tt.format, tt.cfg)
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestIsDatabaseFormat tests the format checker.
func TestIsDatabaseFormat(t *testing.T) {
	tests := []struct {
		format string
		want   bool
	}{
		{"postgres", true},
		{"mysql", true},
		{"mongodb", true},
		{"json", false},
		{"jsonl", false},
		{"csv", false},
		{"xlsx", false},
		{"parquet", false},
		{"har", false},
		{"md", false},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			got := IsDatabaseFormat(tt.format)
			require.Equal(t, tt.want, got)
		})
	}
}

// TestDefaultTableName tests table name generation.
func TestDefaultTableName(t *testing.T) {
	now := time.Now()
	expectedPrefix := now.Format("20060102")

	tests := []struct {
		job  model.Job
		want string
	}{
		{model.Job{Kind: model.KindScrape}, "scrape_"},
		{model.Job{Kind: model.KindCrawl}, "crawl_"},
		{model.Job{Kind: model.KindResearch}, "research_"},
	}

	for _, tt := range tests {
		t.Run(string(tt.job.Kind), func(t *testing.T) {
			got := defaultTableName(tt.job)
			require.True(t, strings.HasPrefix(got, tt.want))
			require.True(t, strings.Contains(got, expectedPrefix))
		})
	}
}

// TestDefaultCollectionName tests collection name generation.
func TestDefaultCollectionName(t *testing.T) {
	tests := []struct {
		job  model.Job
		want string
	}{
		{model.Job{Kind: model.KindScrape}, "scrape"},
		{model.Job{Kind: model.KindCrawl}, "crawl"},
		{model.Job{Kind: model.KindResearch}, "research"},
	}

	for _, tt := range tests {
		t.Run(string(tt.job.Kind), func(t *testing.T) {
			got := defaultCollectionName(tt.job)
			require.Equal(t, tt.want, got)
		})
	}
}

// TestResolveConnectionString tests connection string resolution.
func TestResolveConnectionString(t *testing.T) {
	t.Setenv("SPARTAN_POSTGRES_URL", "postgres://env-postgres")
	t.Setenv("SPARTAN_MYSQL_URL", "mysql://env-mysql")
	t.Setenv("SPARTAN_MONGODB_URL", "mongodb://env-mongodb")

	tests := []struct {
		name     string
		dbType   string
		cfgValue string
		want     string
	}{
		{
			name:     "postgres from config",
			dbType:   "postgres",
			cfgValue: "postgres://config",
			want:     "postgres://config",
		},
		{
			name:     "postgres from env",
			dbType:   "postgres",
			cfgValue: "",
			want:     "postgres://env-postgres",
		},
		{
			name:     "mysql from config",
			dbType:   "mysql",
			cfgValue: "mysql://config",
			want:     "mysql://config",
		},
		{
			name:     "mysql from env",
			dbType:   "mysql",
			cfgValue: "",
			want:     "mysql://env-mysql",
		},
		{
			name:     "mongodb from config",
			dbType:   "mongodb",
			cfgValue: "mongodb://config",
			want:     "mongodb://config",
		},
		{
			name:     "mongodb from env",
			dbType:   "mongodb",
			cfgValue: "",
			want:     "mongodb://env-mongodb",
		},
		{
			name:     "unknown db type",
			dbType:   "unknown",
			cfgValue: "",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveConnectionString(tt.dbType, tt.cfgValue)
			require.Equal(t, tt.want, got)
		})
	}
}

// TestBuildPostgresCreateTable tests PostgreSQL table creation SQL.
func TestBuildPostgresCreateTable(t *testing.T) {
	schema := tableSchema{
		Columns: []columnDef{
			{Name: "id", Type: "INTEGER", Nullable: false},
			{Name: "name", Type: "TEXT", Nullable: true},
		},
	}

	sql := buildPostgresCreateTable("test_table", schema)
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS")
	require.Contains(t, sql, "test_table")
	// pgx.Identifier.Sanitize() quotes identifiers
	require.Contains(t, sql, `"id" INTEGER NOT NULL`)
	require.Contains(t, sql, `"name" TEXT NULL`)
}

// TestBuildMySQLCreateTable tests MySQL table creation SQL.
func TestBuildMySQLCreateTable(t *testing.T) {
	schema := tableSchema{
		Columns: []columnDef{
			{Name: "id", Type: "INTEGER", Nullable: false},
			{Name: "name", Type: "TEXT", Nullable: true},
		},
	}

	sql := buildMySQLCreateTable("test_table", schema)
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS")
	require.Contains(t, sql, "test_table")
	require.Contains(t, sql, "`id` INT NOT NULL")
	require.Contains(t, sql, "`name` LONGTEXT NULL")
}

// TestBuildPostgresInsertQuery tests PostgreSQL insert query building.
func TestBuildPostgresInsertQuery(t *testing.T) {
	columns := []string{"url", "status", "title"}

	t.Run("insert", func(t *testing.T) {
		cfg := DatabaseExportConfig{Mode: "insert"}
		query := buildPostgresInsertQuery("test", cfg, columns)
		require.Contains(t, query, "INSERT INTO")
		require.Contains(t, query, "test")
		// pgx.Identifier.Sanitize() quotes identifiers
		require.Contains(t, query, `"url", "status", "title"`)
		require.Contains(t, query, "$1, $2, $3")
		require.NotContains(t, query, "ON CONFLICT")
	})

	t.Run("upsert", func(t *testing.T) {
		cfg := DatabaseExportConfig{Mode: "upsert", UpsertKey: "url"}
		query := buildPostgresInsertQuery("test", cfg, columns)
		require.Contains(t, query, "INSERT INTO")
		require.Contains(t, query, "ON CONFLICT")
		require.Contains(t, query, "DO UPDATE SET")
	})
}

// TestBuildMySQLInsertQuery tests MySQL insert query building.
func TestBuildMySQLInsertQuery(t *testing.T) {
	columns := []string{"url", "status", "title"}

	t.Run("insert", func(t *testing.T) {
		cfg := DatabaseExportConfig{Mode: "insert"}
		query := buildMySQLInsertQuery("test", cfg, columns)
		require.Contains(t, query, "INSERT INTO")
		require.Contains(t, query, "test")
		require.Contains(t, query, "`url`, `status`, `title`")
		require.Contains(t, query, "?, ?, ?")
		require.NotContains(t, query, "ON DUPLICATE KEY UPDATE")
	})

	t.Run("upsert", func(t *testing.T) {
		cfg := DatabaseExportConfig{Mode: "upsert", UpsertKey: "url"}
		query := buildMySQLInsertQuery("test", cfg, columns)
		require.Contains(t, query, "INSERT INTO")
		require.Contains(t, query, "ON DUPLICATE KEY UPDATE")
	})
}
