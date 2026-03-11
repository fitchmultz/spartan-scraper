// Package store provides performance and index verification tests.
//
// Tests cover:
// - Index usage verification for common query patterns
// - Benchmarks for job listing operations (ListByStatus, ListOpts)
// - Benchmarks for crawl state listing (ListCrawlStates)
//
// Does NOT test:
// - Functional correctness of store operations (see other store_*_test.go files)
// - Database migrations
// - Error handling paths
//
// Assumes:
// - SQLite is the underlying database
// - EXPLAIN QUERY PLAN output format is stable
// - Benchmarks run with sufficient data volume (10,000 rows)
package store

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestStore_IndexUsage(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	tests := []struct {
		name          string
		query         string
		args          []interface{}
		expectedIndex string
	}{
		{
			name:          "ListByStatus uses idx_jobs_status_created",
			query:         "select id, kind, status, created_at, updated_at, spec_version, spec_json, result_path, error, started_at, finished_at, selected_engine from jobs where status = ? order by created_at desc limit ? offset ?",
			args:          []interface{}{model.StatusQueued, 100, 0},
			expectedIndex: "idx_jobs_status_created",
		},
		{
			name:          "ListOpts uses idx_jobs_created",
			query:         "select id, kind, status, created_at, updated_at, spec_version, spec_json, result_path, error, started_at, finished_at, selected_engine from jobs order by created_at desc limit ? offset ?",
			args:          []interface{}{100, 0},
			expectedIndex: "idx_jobs_created",
		},
		{
			name:          "ListCrawlStates uses idx_crawl_states_last_scraped",
			query:         "select url, etag, last_modified, content_hash, last_scraped from crawl_states order by last_scraped desc limit ? offset ?",
			args:          []interface{}{100, 0},
			expectedIndex: "idx_crawl_states_last_scraped",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			explainQuery := "EXPLAIN QUERY PLAN " + tt.query
			rows, err := s.db.QueryContext(ctx, explainQuery, tt.args...)
			if err != nil {
				t.Fatalf("Explain failed: %v", err)
			}
			defer rows.Close()

			var found bool
			for rows.Next() {
				var id, parent, notused int
				var detail string
				if err := rows.Scan(&id, &parent, &notused, &detail); err != nil {
					t.Fatalf("Scan failed: %v", err)
				}
				if strings.Contains(detail, "USING INDEX "+tt.expectedIndex) || strings.Contains(detail, "USING COVERING INDEX "+tt.expectedIndex) {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("expected query plan to use index %s", tt.expectedIndex)
			}
		})
	}
}

func BenchmarkStore_ListByStatus(b *testing.B) {
	dataDir := b.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		b.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	count := 10000
	for i := 0; i < count; i++ {
		status := model.StatusQueued
		if i%2 == 0 {
			status = model.StatusSucceeded
		}
		job := model.Job{
			ID:        fmt.Sprintf("j%d", i),
			Kind:      model.KindScrape,
			Status:    status,
			CreatedAt: time.Now().Add(time.Duration(-i) * time.Minute),
			UpdatedAt: time.Now(),
		}
		if err := s.Create(ctx, job); err != nil {
			b.Fatalf("Create failed: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := s.ListByStatus(ctx, model.StatusQueued, ListByStatusOptions{Limit: 100})
		if err != nil {
			b.Fatalf("ListByStatus failed: %v", err)
		}
	}
}

func BenchmarkStore_ListOpts(b *testing.B) {
	dataDir := b.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		b.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	count := 10000
	for i := 0; i < count; i++ {
		job := model.Job{
			ID:        fmt.Sprintf("j%d", i),
			Kind:      model.KindScrape,
			Status:    model.StatusQueued,
			CreatedAt: time.Now().Add(time.Duration(-i) * time.Minute),
			UpdatedAt: time.Now(),
		}
		if err := s.Create(ctx, job); err != nil {
			b.Fatalf("Create failed: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := s.ListOpts(ctx, ListOptions{Limit: 100})
		if err != nil {
			b.Fatalf("ListOpts failed: %v", err)
		}
	}
}

func BenchmarkStore_ListCrawlStates(b *testing.B) {
	dataDir := b.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		b.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	count := 10000
	for i := 0; i < count; i++ {
		state := model.CrawlState{
			URL:         fmt.Sprintf("https://example.com/page%d", i),
			LastScraped: time.Now().Add(time.Duration(-i) * time.Minute),
		}
		if err := s.UpsertCrawlState(ctx, state); err != nil {
			b.Fatalf("UpsertCrawlState failed: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := s.ListCrawlStates(ctx, ListCrawlStatesOptions{Limit: 100})
		if err != nil {
			b.Fatalf("ListCrawlStates failed: %v", err)
		}
	}
}
