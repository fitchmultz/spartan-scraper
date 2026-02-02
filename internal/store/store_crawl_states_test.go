// Package store provides tests for crawl state persistence operations.
// Tests cover UpsertCrawlState, GetCrawlState, ListCrawlStates, CountCrawlStates,
// and pagination options with default value handling.
// Does NOT test job CRUD or database migration logic.
package store

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestStoreCrawlState(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	state := model.CrawlState{
		URL:         "http://example.com",
		ETag:        "tag",
		LastScraped: time.Now(),
		Depth:       1,
		JobID:       "test-job",
	}

	if err := s.UpsertCrawlState(ctx, state); err != nil {
		t.Fatalf("UpsertCrawlState failed: %v", err)
	}

	got, err := s.GetCrawlState(ctx, "http://example.com")
	if err != nil {
		t.Fatalf("GetCrawlState failed: %v", err)
	}
	if got.URL != state.URL || got.ETag != state.ETag || got.Depth != state.Depth || got.JobID != state.JobID {
		t.Errorf("GetCrawlState returned unexpected state: %+v", got)
	}

	state.ETag = "new-tag"
	if err := s.UpsertCrawlState(ctx, state); err != nil {
		t.Fatalf("UpsertCrawlState (update) failed: %v", err)
	}

	got, _ = s.GetCrawlState(ctx, "http://example.com")
	if got.ETag != "new-tag" {
		t.Errorf("expected etag new-tag, got %s", got.ETag)
	}
}

func TestListCrawlStates(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	states := []model.CrawlState{
		{
			URL:          "https://example.com/page1",
			ETag:         "etag1",
			LastModified: "Mon, 01 Jan 2026 00:00:00 GMT",
			ContentHash:  "hash1",
			LastScraped:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			Depth:        1,
			JobID:        "job1",
		},
		{
			URL:          "https://example.com/page2",
			ETag:         "etag2",
			LastModified: "Tue, 02 Jan 2026 00:00:00 GMT",
			ContentHash:  "hash2",
			LastScraped:  time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
			Depth:        2,
			JobID:        "job2",
		},
	}

	for _, state := range states {
		err := s.UpsertCrawlState(ctx, state)
		if err != nil {
			t.Fatalf("failed to insert crawl state: %v", err)
		}
	}

	listed, err := s.ListCrawlStates(ctx, ListCrawlStatesOptions{})
	if err != nil {
		t.Fatalf("failed to list crawl states: %v", err)
	}

	if len(listed) != 2 {
		t.Errorf("expected 2 states, got %d", len(listed))
	}

	if listed[0].URL != "https://example.com/page2" {
		t.Errorf("expected page2 first, got %s", listed[0].URL)
	}
	if listed[0].Depth != 2 || listed[0].JobID != "job2" {
		t.Errorf("expected Depth 2 and JobID job2, got %d and %s", listed[0].Depth, listed[0].JobID)
	}
	if listed[1].Depth != 1 || listed[1].JobID != "job1" {
		t.Errorf("expected Depth 1 and JobID job1, got %d and %s", listed[1].Depth, listed[1].JobID)
	}
}

func TestListCrawlStatesPagination(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	for i := 1; i <= 3; i++ {
		state := model.CrawlState{
			URL:         fmt.Sprintf("https://example.com/page%d", i),
			ETag:        fmt.Sprintf("etag%d", i),
			ContentHash: fmt.Sprintf("hash%d", i),
			LastScraped: time.Date(2026, 1, i, 0, 0, 0, 0, time.UTC),
		}
		err := s.UpsertCrawlState(ctx, state)
		if err != nil {
			t.Fatalf("failed to insert crawl state: %v", err)
		}
	}

	listed, err := s.ListCrawlStates(ctx, ListCrawlStatesOptions{Limit: 2})
	if err != nil {
		t.Fatalf("failed to list crawl states: %v", err)
	}
	if len(listed) != 2 {
		t.Errorf("expected 2 states with limit, got %d", len(listed))
	}

	listed, err = s.ListCrawlStates(ctx, ListCrawlStatesOptions{Offset: 1})
	if err != nil {
		t.Fatalf("failed to list crawl states: %v", err)
	}
	if len(listed) != 2 {
		t.Errorf("expected 2 states with offset 1, got %d", len(listed))
	}
}

func TestListCrawlStatesOptionsDefaults(t *testing.T) {
	tests := []struct {
		name       string
		input      ListCrawlStatesOptions
		wantLimit  int
		wantOffset int
	}{
		{"zero values use defaults", ListCrawlStatesOptions{}, 100, 0},
		{"negative limit uses default", ListCrawlStatesOptions{Limit: -1}, 100, 0},
		{"negative offset uses zero", ListCrawlStatesOptions{Offset: -5}, 100, 0},
		{"max limit capped", ListCrawlStatesOptions{Limit: 2000}, 1000, 0},
		{"valid values preserved", ListCrawlStatesOptions{Limit: 50, Offset: 10}, 50, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.input.Defaults()
			if got.Limit != tt.wantLimit || got.Offset != tt.wantOffset {
				t.Errorf("Defaults() = {%d, %d}, want {%d, %d}",
					got.Limit, got.Offset, tt.wantLimit, tt.wantOffset)
			}
		})
	}
}

func TestStoreCountCrawlStates(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	_ = s.UpsertCrawlState(ctx, model.CrawlState{URL: "https://example.com/1", LastScraped: time.Now()})
	_ = s.UpsertCrawlState(ctx, model.CrawlState{URL: "https://example.com/2", LastScraped: time.Now()})
	_ = s.UpsertCrawlState(ctx, model.CrawlState{URL: "https://example.com/3", LastScraped: time.Now()})

	count, err := s.CountCrawlStates(ctx)
	if err != nil {
		t.Fatalf("CountCrawlStates failed: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 crawl states, got %d", count)
	}
}

func TestDeleteCrawlStatesOlderThan(t *testing.T) {
	dataDir := t.TempDir()
	st, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer st.Close()

	ctx := context.Background()
	now := time.Now()

	// Create crawl states with different ages
	oldState := model.CrawlState{
		URL:         "http://old.example.com",
		LastScraped: now.AddDate(0, 0, -100),
		ContentHash: "abc123",
	}
	newState := model.CrawlState{
		URL:         "http://new.example.com",
		LastScraped: now.AddDate(0, 0, -5),
		ContentHash: "def456",
	}

	if err := st.UpsertCrawlState(ctx, oldState); err != nil {
		t.Fatalf("UpsertCrawlState old failed: %v", err)
	}
	if err := st.UpsertCrawlState(ctx, newState); err != nil {
		t.Fatalf("UpsertCrawlState new failed: %v", err)
	}

	// Delete crawl states older than 60 days
	cutoff := now.AddDate(0, 0, -60)
	deleted, err := st.DeleteCrawlStatesOlderThan(ctx, cutoff)
	if err != nil {
		t.Fatalf("DeleteCrawlStatesOlderThan failed: %v", err)
	}

	if deleted != 1 {
		t.Errorf("expected 1 deleted crawl state, got %d", deleted)
	}

	// Verify remaining crawl state
	states, err := st.ListCrawlStates(ctx, ListCrawlStatesOptions{})
	if err != nil {
		t.Fatalf("ListCrawlStates failed: %v", err)
	}
	if len(states) != 1 {
		t.Errorf("expected 1 remaining crawl state, got %d", len(states))
	}
}
