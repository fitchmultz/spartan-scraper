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
			name: "valid scrape schedule",
			schedule: Schedule{
				Kind:            model.KindScrape,
				IntervalSeconds: 60,
				Params: map[string]interface{}{
					"url":     "https://example.com",
					"timeout": 30,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid scrape schedule - invalid URL",
			schedule: Schedule{
				Kind:            model.KindScrape,
				IntervalSeconds: 60,
				Params: map[string]interface{}{
					"url": "ftp://example.com",
				},
			},
			wantErr:     true,
			errContains: "invalid scrape schedule",
		},
		{
			name: "invalid scrape schedule - timeout too low",
			schedule: Schedule{
				Kind:            model.KindScrape,
				IntervalSeconds: 60,
				Params: map[string]interface{}{
					"url":     "https://example.com",
					"timeout": 4,
				},
			},
			wantErr:     true,
			errContains: "invalid scrape schedule",
		},
		{
			name: "valid crawl schedule",
			schedule: Schedule{
				Kind:            model.KindCrawl,
				IntervalSeconds: 60,
				Params: map[string]interface{}{
					"url":      "https://example.com",
					"maxDepth": 3,
					"maxPages": 100,
					"timeout":  30,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid crawl schedule - maxDepth too high",
			schedule: Schedule{
				Kind:            model.KindCrawl,
				IntervalSeconds: 60,
				Params: map[string]interface{}{
					"url":      "https://example.com",
					"maxDepth": 11,
					"maxPages": 100,
				},
			},
			wantErr:     true,
			errContains: "invalid crawl schedule",
		},
		{
			name: "invalid crawl schedule - maxPages too high",
			schedule: Schedule{
				Kind:            model.KindCrawl,
				IntervalSeconds: 60,
				Params: map[string]interface{}{
					"url":      "https://example.com",
					"maxDepth": 3,
					"maxPages": 10001,
				},
			},
			wantErr:     true,
			errContains: "invalid crawl schedule",
		},
		{
			name: "valid research schedule",
			schedule: Schedule{
				Kind:            model.KindResearch,
				IntervalSeconds: 60,
				Params: map[string]interface{}{
					"query": "test query",
					"urls":  []string{"https://example.com", "https://example.org"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid research schedule - empty query",
			schedule: Schedule{
				Kind:            model.KindResearch,
				IntervalSeconds: 60,
				Params: map[string]interface{}{
					"query": "",
					"urls":  []string{"https://example.com"},
				},
			},
			wantErr:     true,
			errContains: "invalid research schedule",
		},
		{
			name: "invalid research schedule - invalid URL in list",
			schedule: Schedule{
				Kind:            model.KindResearch,
				IntervalSeconds: 60,
				Params: map[string]interface{}{
					"query": "test query",
					"urls":  []string{"https://example.com", "ftp://example.org"},
				},
			},
			wantErr:     true,
			errContains: "invalid research schedule",
		},
		{
			name: "unknown schedule kind",
			schedule: Schedule{
				Kind:            "unknown",
				IntervalSeconds: 60,
				Params:          map[string]interface{}{},
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
				schedule := Schedule{
					Kind:            model.KindScrape,
					IntervalSeconds: 60,
					Params:          map[string]interface{}{"url": "ftp://invalid.com"},
				}
				_, err := Add(dataDir, schedule)
				return err
			},
			wantKind: apperrors.KindValidation,
		},
		{
			name: "invalid crawl validation returns KindValidation",
			testFunc: func(t *testing.T) error {
				dataDir := t.TempDir()
				schedule := Schedule{
					Kind:            model.KindCrawl,
					IntervalSeconds: 60,
					Params: map[string]interface{}{
						"url":      "https://example.com",
						"maxDepth": 11,
					},
				}
				_, err := Add(dataDir, schedule)
				return err
			},
			wantKind: apperrors.KindValidation,
		},
		{
			name: "invalid research validation returns KindValidation",
			testFunc: func(t *testing.T) error {
				dataDir := t.TempDir()
				schedule := Schedule{
					Kind:            model.KindResearch,
					IntervalSeconds: 60,
					Params:          map[string]interface{}{"query": ""},
				}
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
				schedule := Schedule{
					ID:              "test-id",
					Kind:            model.KindScrape,
					IntervalSeconds: 60,
					Params: map[string]interface{}{
						"url":         "https://example.com",
						"authProfile": "non-existent-profile",
					},
				}
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
