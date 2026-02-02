// Package dedup provides tests for the content deduplication functionality.
package dedup

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	return db
}

func TestSQLiteIndex_Index(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	idx, err := NewSQLiteIndex(db)
	if err != nil {
		t.Fatalf("failed to create index: %v", err)
	}
	defer idx.Close()

	ctx := context.Background()

	tests := []struct {
		name    string
		jobID   string
		url     string
		simhash uint64
		wantErr bool
	}{
		{
			name:    "valid entry",
			jobID:   "job-1",
			url:     "https://example.com/page1",
			simhash: 12345,
			wantErr: false,
		},
		{
			name:    "empty job_id",
			jobID:   "",
			url:     "https://example.com/page1",
			simhash: 12345,
			wantErr: true,
		},
		{
			name:    "empty url",
			jobID:   "job-1",
			url:     "",
			simhash: 12345,
			wantErr: true,
		},
		{
			name:    "duplicate entry updates",
			jobID:   "job-1",
			url:     "https://example.com/page1",
			simhash: 99999,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := idx.Index(ctx, tt.jobID, tt.url, tt.simhash)
			if (err != nil) != tt.wantErr {
				t.Errorf("Index() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSQLiteIndex_FindDuplicates(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	idx, err := NewSQLiteIndex(db)
	if err != nil {
		t.Fatalf("failed to create index: %v", err)
	}
	defer idx.Close()

	ctx := context.Background()

	// Index some test content
	testData := []struct {
		jobID   string
		url     string
		simhash uint64
	}{
		{"job-1", "https://example.com/page1", 0x1234567890ABCDEF},
		{"job-1", "https://example.com/page2", 0x1234567890ABCDEE}, // 1 bit different
		{"job-2", "https://example.com/page1", 0x1234567890ABCDEF}, // exact duplicate
		{"job-2", "https://example.com/page3", 0x1234567890ABCCEF}, // 2 bits different
		{"job-3", "https://other.com/page", 0xFFFFFFFFFFFFFFF0},    // completely different
	}

	for _, td := range testData {
		if err := idx.Index(ctx, td.jobID, td.url, td.simhash); err != nil {
			t.Fatalf("failed to index: %v", err)
		}
	}

	tests := []struct {
		name          string
		targetHash    uint64
		threshold     int
		wantCount     int
		wantExactURLs []string
	}{
		{
			name:          "exact match threshold 0",
			targetHash:    0x1234567890ABCDEF,
			threshold:     0,
			wantCount:     2,
			wantExactURLs: []string{"https://example.com/page1"},
		},
		{
			name:       "near duplicate threshold 3",
			targetHash: 0x1234567890ABCDEF,
			threshold:  3,
			wantCount:  4, // all 4 similar pages
		},
		{
			name:       "no matches for different content",
			targetHash: 0x0000000000000000,
			threshold:  10,
			wantCount:  0,
		},
		{
			name:       "default threshold",
			targetHash: 0x1234567890ABCDEF,
			threshold:  -1, // should use default
			wantCount:  4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches, err := idx.FindDuplicates(ctx, tt.targetHash, tt.threshold)
			if err != nil {
				t.Fatalf("FindDuplicates() error = %v", err)
			}
			if len(matches) != tt.wantCount {
				t.Errorf("FindDuplicates() got %d matches, want %d", len(matches), tt.wantCount)
			}
			if len(tt.wantExactURLs) > 0 {
				for _, url := range tt.wantExactURLs {
					found := false
					for _, m := range matches {
						if m.URL == url {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("FindDuplicates() missing expected URL: %s", url)
					}
				}
			}
		})
	}
}

func TestSQLiteIndex_GetContentHistory(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	idx, err := NewSQLiteIndex(db)
	if err != nil {
		t.Fatalf("failed to create index: %v", err)
	}
	defer idx.Close()

	ctx := context.Background()

	// Index same URL across multiple jobs
	url := "https://example.com/page1"
	testData := []struct {
		jobID   string
		simhash uint64
	}{
		{"job-1", 0x1111111111111111},
		{"job-2", 0x2222222222222222},
		{"job-3", 0x3333333333333333},
	}

	for _, td := range testData {
		if err := idx.Index(ctx, td.jobID, url, td.simhash); err != nil {
			t.Fatalf("failed to index: %v", err)
		}
		time.Sleep(50 * time.Millisecond) // Ensure different timestamps
	}

	tests := []struct {
		name       string
		url        string
		wantCount  int
		wantJobIDs []string // set of expected job IDs (order not guaranteed)
		wantErr    bool
	}{
		{
			name:       "existing url",
			url:        url,
			wantCount:  3,
			wantJobIDs: []string{"job-1", "job-2", "job-3"},
			wantErr:    false,
		},
		{
			name:      "non-existent url",
			url:       "https://example.com/nonexistent",
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:    "empty url",
			url:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries, err := idx.GetContentHistory(ctx, tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetContentHistory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(entries) != tt.wantCount {
				t.Errorf("GetContentHistory() got %d entries, want %d", len(entries), tt.wantCount)
			}
			if len(tt.wantJobIDs) > 0 {
				// Check that all expected job IDs are present (order not guaranteed)
				gotJobIDs := make(map[string]bool)
				for _, entry := range entries {
					gotJobIDs[entry.JobID] = true
				}
				for _, wantJobID := range tt.wantJobIDs {
					if !gotJobIDs[wantJobID] {
						t.Errorf("GetContentHistory() missing expected jobID: %s", wantJobID)
					}
				}
			}
		})
	}
}

func TestSQLiteIndex_DeleteJobEntries(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	idx, err := NewSQLiteIndex(db)
	if err != nil {
		t.Fatalf("failed to create index: %v", err)
	}
	defer idx.Close()

	ctx := context.Background()

	// Index some test content
	testData := []struct {
		jobID   string
		url     string
		simhash uint64
	}{
		{"job-1", "https://example.com/page1", 0x1111},
		{"job-1", "https://example.com/page2", 0x2222},
		{"job-2", "https://example.com/page3", 0x3333},
	}

	for _, td := range testData {
		if err := idx.Index(ctx, td.jobID, td.url, td.simhash); err != nil {
			t.Fatalf("failed to index: %v", err)
		}
	}

	tests := []struct {
		name          string
		jobID         string
		wantDeleted   int64
		wantErr       bool
		wantRemaining int64
	}{
		{
			name:          "delete existing job",
			jobID:         "job-1",
			wantDeleted:   2,
			wantErr:       false,
			wantRemaining: 1,
		},
		{
			name:        "delete non-existent job",
			jobID:       "job-999",
			wantDeleted: 0,
			wantErr:     false,
		},
		{
			name:    "empty job id",
			jobID:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deleted, err := idx.DeleteJobEntries(ctx, tt.jobID)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteJobEntries() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if deleted != tt.wantDeleted {
				t.Errorf("DeleteJobEntries() deleted = %d, want %d", deleted, tt.wantDeleted)
			}
		})
	}
}

func TestSQLiteIndex_Stats(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	idx, err := NewSQLiteIndex(db)
	if err != nil {
		t.Fatalf("failed to create index: %v", err)
	}
	defer idx.Close()

	ctx := context.Background()

	// Initially empty
	stats, err := idx.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if stats.TotalIndexed != 0 {
		t.Errorf("Stats() TotalIndexed = %d, want 0", stats.TotalIndexed)
	}

	// Index some test content
	testData := []struct {
		jobID   string
		url     string
		simhash uint64
	}{
		{"job-1", "https://example.com/page1", 0x1111},
		{"job-1", "https://example.com/page2", 0x2222},
		{"job-2", "https://example.com/page1", 0x3333}, // same URL, different job
		{"job-2", "https://example.com/page3", 0x4444},
	}

	for _, td := range testData {
		if err := idx.Index(ctx, td.jobID, td.url, td.simhash); err != nil {
			t.Fatalf("failed to index: %v", err)
		}
	}

	stats, err = idx.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}

	if stats.TotalIndexed != 4 {
		t.Errorf("Stats() TotalIndexed = %d, want 4", stats.TotalIndexed)
	}
	if stats.UniqueURLs != 3 {
		t.Errorf("Stats() UniqueURLs = %d, want 3", stats.UniqueURLs)
	}
	if stats.UniqueJobs != 2 {
		t.Errorf("Stats() UniqueJobs = %d, want 2", stats.UniqueJobs)
	}
	// DuplicatePairs counts URLs appearing in multiple jobs
	if stats.DuplicatePairs != 1 {
		t.Errorf("Stats() DuplicatePairs = %d, want 1", stats.DuplicatePairs)
	}
}

func TestSQLiteIndex_FindDuplicatesByURL(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	idx, err := NewSQLiteIndex(db)
	if err != nil {
		t.Fatalf("failed to create index: %v", err)
	}
	defer idx.Close()

	ctx := context.Background()

	// Index some test content with near-duplicates
	testData := []struct {
		jobID   string
		url     string
		simhash uint64
	}{
		{"job-1", "https://example.com/page1", 0x1234567890ABCDEF},
		{"job-2", "https://example.com/page1", 0x1234567890ABCDEF}, // exact duplicate
		{"job-2", "https://example.com/page2", 0x1234567890ABCDEE}, // 1 bit different
	}

	for _, td := range testData {
		if err := idx.Index(ctx, td.jobID, td.url, td.simhash); err != nil {
			t.Fatalf("failed to index: %v", err)
		}
	}

	matches, err := idx.FindDuplicatesByURL(ctx, "https://example.com/page1", 0)
	if err != nil {
		t.Fatalf("FindDuplicatesByURL() error = %v", err)
	}

	// Should find both entries (job-1 and job-2) since they both have the same hash for page1
	// Note: FindDuplicatesByURL returns ALL matches including the queried URL
	if len(matches) != 2 {
		t.Errorf("FindDuplicatesByURL() got %d matches, want 2", len(matches))
	}
}

func TestThresholdConstants(t *testing.T) {
	if ThresholdExact != 0 {
		t.Errorf("ThresholdExact = %d, want 0", ThresholdExact)
	}
	if ThresholdNear != 3 {
		t.Errorf("ThresholdNear = %d, want 3", ThresholdNear)
	}
	if ThresholdSimilar != 8 {
		t.Errorf("ThresholdSimilar = %d, want 8", ThresholdSimilar)
	}
}

func TestDuplicateMatch_String(t *testing.T) {
	m := DuplicateMatch{
		JobID:    "job-1",
		URL:      "https://example.com/page",
		Distance: 2,
	}
	s := m.String()
	if s == "" {
		t.Error("DuplicateMatch.String() returned empty string")
	}
}

func TestContentEntry_String(t *testing.T) {
	e := ContentEntry{
		JobID:   "job-1",
		SimHash: 12345,
	}
	s := e.String()
	if s == "" {
		t.Error("ContentEntry.String() returned empty string")
	}
}
