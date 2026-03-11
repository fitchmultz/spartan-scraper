// Package scheduler provides tests for schedule validation.
// Tests cover validation for scrape, crawl, and research schedule kinds.
// Does NOT test storage operations or job execution.
package scheduler

import (
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestScheduleValidation(t *testing.T) {
	tests := []struct {
		name        string
		schedule    Schedule
		wantErr     bool
		errContains string
	}{
		{
			name:     "valid scrape schedule",
			schedule: testScrapeSchedule("https://example.com"),
			wantErr:  false,
		},
		{
			name:        "invalid scrape schedule - invalid URL",
			schedule:    testScrapeSchedule("ftp://example.com"),
			wantErr:     true,
			errContains: "invalid scrape schedule",
		},
		{
			name: "invalid scrape schedule - timeout too low",
			schedule: func() Schedule {
				s := testScrapeSchedule("https://example.com")
				spec := s.Spec.(model.ScrapeSpecV1)
				spec.Execution.TimeoutSeconds = 4
				s.Spec = spec
				return s
			}(),
			wantErr:     true,
			errContains: "invalid scrape schedule",
		},
		{
			name:     "valid crawl schedule",
			schedule: testCrawlSchedule("https://example.com", 3, 100),
			wantErr:  false,
		},
		{
			name:        "invalid crawl schedule - maxDepth too high",
			schedule:    testCrawlSchedule("https://example.com", 11, 100),
			wantErr:     true,
			errContains: "invalid crawl schedule",
		},
		{
			name:        "invalid crawl schedule - maxPages too high",
			schedule:    testCrawlSchedule("https://example.com", 3, 10001),
			wantErr:     true,
			errContains: "invalid crawl schedule",
		},
		{
			name:     "valid research schedule",
			schedule: testResearchSchedule("test query", []string{"https://example.com", "https://example.org"}, 2, 200),
			wantErr:  false,
		},
		{
			name:        "invalid research schedule - empty query",
			schedule:    testResearchSchedule("", []string{"https://example.com"}, 2, 200),
			wantErr:     true,
			errContains: "invalid research schedule",
		},
		{
			name:        "invalid research schedule - invalid URL in list",
			schedule:    testResearchSchedule("test query", []string{"https://example.com", "ftp://example.org"}, 2, 200),
			wantErr:     true,
			errContains: "invalid research schedule",
		},
		{
			name: "unknown schedule kind",
			schedule: Schedule{
				Kind:            "unknown",
				IntervalSeconds: 60,
				SpecVersion:     model.JobSpecVersion1,
				Spec:            map[string]any{},
			},
			wantErr:     true,
			errContains: "unknown schedule kind",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDataDir := t.TempDir()
			_, err := Add(testDataDir, tt.schedule)
			if (err != nil) != tt.wantErr {
				t.Errorf("Add() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Error = %v, want containing %v", err.Error(), tt.errContains)
				}
			}

			if tt.wantErr {
				if !apperrors.IsKind(err, apperrors.KindValidation) {
					t.Errorf("error kind = %v, want %v", apperrors.KindOf(err), apperrors.KindValidation)
				}
				list, _ := List(testDataDir)
				for _, s := range list {
					if s.Kind == tt.schedule.Kind {
						t.Errorf("Invalid schedule should not have been added")
					}
				}
			}
		})
	}
}

func TestSchedulerErrorKinds(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(*testing.T) error
		wantKind apperrors.Kind
	}{
		{
			name: "invalid scrape validation returns KindValidation",
			testFunc: func(t *testing.T) error {
				dataDir := t.TempDir()
				schedule := testScrapeSchedule("ftp://invalid.com")
				_, err := Add(dataDir, schedule)
				return err
			},
			wantKind: apperrors.KindValidation,
		},
		{
			name: "invalid crawl validation returns KindValidation",
			testFunc: func(t *testing.T) error {
				dataDir := t.TempDir()
				schedule := testCrawlSchedule("https://example.com", 11, 200)
				_, err := Add(dataDir, schedule)
				return err
			},
			wantKind: apperrors.KindValidation,
		},
		{
			name: "invalid research validation returns KindValidation",
			testFunc: func(t *testing.T) error {
				dataDir := t.TempDir()
				schedule := testResearchSchedule("", []string{"https://example.com"}, 2, 200)
				_, err := Add(dataDir, schedule)
				return err
			},
			wantKind: apperrors.KindValidation,
		},
		{
			name: "invalid auth resolution returns KindInternal",
			testFunc: func(t *testing.T) error {
				dataDir := t.TempDir()
				manager, _, cleanup := setupTestManager(t)
				defer cleanup()
				schedule := testScrapeSchedule("https://example.com")
				schedule.ID = "test-id"
				spec := schedule.Spec.(model.ScrapeSpecV1)
				spec.Execution.AuthProfile = "non-existent-profile"
				schedule.Spec = spec
				return enqueue(t.Context(), manager, dataDir, schedule)
			},
			wantKind: apperrors.KindInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.testFunc(t)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !apperrors.IsKind(err, tt.wantKind) {
				t.Errorf("error kind = %v, want %v", apperrors.KindOf(err), tt.wantKind)
			}
		})
	}
}
