// Package scheduler provides tests for schedule persistence of complex configurations.
// Tests cover extraction configs, incremental mode, and auth overrides.
// Does NOT test basic CRUD operations or schedule validation.
package scheduler

import (
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/auth"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestExtractConfigPersistence(t *testing.T) {
	dataDir := t.TempDir()

	schedule := testScrapeSchedule("https://example.com")
	spec := schedule.Spec.(model.ScrapeSpecV1)
	spec.Execution.Extract = extract.ExtractOptions{
		Template: "product",
		Validate: true,
		Inline: &extract.Template{
			Name: "test-template",
			Selectors: []extract.SelectorRule{
				{Name: "title", Selector: "h1", Attr: "text"},
			},
		},
	}
	schedule.Spec = spec

	if _, err := Add(dataDir, schedule); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	loaded, err := LoadAll(dataDir)
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 schedule, got %d", len(loaded))
	}

	extractOpts, err := executionSpecForSchedule(loaded[0])
	if err != nil {
		t.Fatalf("executionSpecForSchedule failed: %v", err)
	}
	if extractOpts.Extract.Template != "product" {
		t.Errorf("expected extractTemplate 'product', got %s", extractOpts.Extract.Template)
	}
	if !extractOpts.Extract.Validate {
		t.Error("expected extractValidate true")
	}
	if extractOpts.Extract.Inline == nil {
		t.Error("expected inline template to be loaded from extractConfig")
	}
	if extractOpts.Extract.Inline.Name != "test-template" {
		t.Errorf("expected inline template name 'test-template', got %s", extractOpts.Extract.Inline.Name)
	}
	if len(extractOpts.Extract.Inline.Selectors) != 1 {
		t.Errorf("expected 1 selector, got %d", len(extractOpts.Extract.Inline.Selectors))
	}
}

func TestIncrementalModePersistence(t *testing.T) {
	testCases := []struct {
		name       string
		kind       model.Kind
		schedule   Schedule
		wantResult bool
	}{
		{
			name: "scrape with incremental enabled",
			kind: model.KindScrape,
			schedule: func() Schedule {
				s := testScrapeSchedule("https://example.com")
				spec := s.Spec.(model.ScrapeSpecV1)
				spec.Incremental = true
				s.Spec = spec
				return s
			}(),
			wantResult: true,
		},
		{
			name: "crawl with incremental enabled",
			kind: model.KindCrawl,
			schedule: func() Schedule {
				s := testCrawlSchedule("https://example.com", 2, 200)
				spec := s.Spec.(model.CrawlSpecV1)
				spec.Incremental = true
				s.Spec = spec
				return s
			}(),
			wantResult: true,
		},
		{
			name:       "scrape with incremental disabled",
			kind:       model.KindScrape,
			schedule:   testScrapeSchedule("https://example.com"),
			wantResult: false,
		},
		{
			name:       "scrape without incremental flag",
			kind:       model.KindScrape,
			schedule:   testScrapeSchedule("https://example.com"),
			wantResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testDataDir := t.TempDir()

			if _, err := Add(testDataDir, tc.schedule); err != nil {
				t.Fatalf("Add failed: %v", err)
			}

			loaded, err := LoadAll(testDataDir)
			if err != nil {
				t.Fatalf("LoadAll failed: %v", err)
			}
			if len(loaded) != 1 {
				t.Fatalf("expected 1 schedule, got %d", len(loaded))
			}

			got := false
			switch spec := loaded[0].Spec.(type) {
			case model.ScrapeSpecV1:
				got = spec.Incremental
			case model.CrawlSpecV1:
				got = spec.Incremental
			default:
				t.Fatalf("unexpected schedule spec type %T", loaded[0].Spec)
			}
			if got != tc.wantResult {
				t.Errorf("incremental = %v, want %v", got, tc.wantResult)
			}
		})
	}
}

func TestAuthOverridePersistence(t *testing.T) {
	testCases := []struct {
		name     string
		schedule Schedule
		wantErr  bool
	}{
		{
			name: "auth with headers",
			schedule: func() Schedule {
				s := testScrapeSchedule("https://example.com")
				spec := s.Spec.(model.ScrapeSpecV1)
				spec.Execution.Auth.Headers = map[string]string{
					"X-API-Key":     "secret",
					"Authorization": "Bearer token",
				}
				s.Spec = spec
				return s
			}(),
			wantErr: false,
		},
		{
			name: "auth with cookies",
			schedule: func() Schedule {
				s := testScrapeSchedule("https://example.com")
				spec := s.Spec.(model.ScrapeSpecV1)
				spec.Execution.Auth.Cookies = []string{"session=abc123"}
				s.Spec = spec
				return s
			}(),
			wantErr: false,
		},
		{
			name: "auth with basic token",
			schedule: func() Schedule {
				s := testScrapeSchedule("https://example.com")
				spec := s.Spec.(model.ScrapeSpecV1)
				spec.Execution.Auth.Basic = "user:pass"
				s.Spec = spec
				return s
			}(),
			wantErr: false,
		},
		{
			name: "auth with bearer tokens",
			schedule: func() Schedule {
				s := testScrapeSchedule("https://example.com")
				spec := s.Spec.(model.ScrapeSpecV1)
				spec.Execution.Auth.Headers = map[string]string{"Authorization": "Bearer token1"}
				s.Spec = spec
				return s
			}(),
			wantErr: false,
		},
		{
			name: "auth with login flow",
			schedule: func() Schedule {
				s := testScrapeSchedule("https://example.com")
				spec := s.Spec.(model.ScrapeSpecV1)
				spec.Execution.Auth.LoginURL = "https://example.com/login"
				spec.Execution.Auth.LoginUserSelector = "#username"
				spec.Execution.Auth.LoginPassSelector = "#password"
				spec.Execution.Auth.LoginSubmitSelector = "button[type=submit]"
				spec.Execution.Auth.LoginUser = "test@example.com"
				spec.Execution.Auth.LoginPass = "password"
				s.Spec = spec
				return s
			}(),
			wantErr: false,
		},
		{
			name: "auth with profile and overrides",
			schedule: func() Schedule {
				s := testScrapeSchedule("https://example.com")
				spec := s.Spec.(model.ScrapeSpecV1)
				spec.Execution.AuthProfile = "base"
				spec.Execution.Auth.Headers = map[string]string{"X-Custom": "value"}
				s.Spec = spec
				return s
			}(),
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testDataDir := t.TempDir()

			if spec := tc.schedule.Spec.(model.ScrapeSpecV1); spec.Execution.AuthProfile != "" {
				profile := auth.Profile{
					Name: "base",
				}
				if err := auth.UpsertProfile(testDataDir, profile); err != nil {
					t.Fatalf("failed to create profile: %v", err)
				}
			}

			_, err := Add(testDataDir, tc.schedule)
			if (err != nil) != tc.wantErr {
				t.Errorf("Add() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if err != nil {
				return
			}

			loaded, err := LoadAll(testDataDir)
			if err != nil {
				t.Fatalf("LoadAll failed: %v", err)
			}
			if len(loaded) != 1 {
				t.Fatalf("expected 1 schedule, got %d", len(loaded))
			}

			exec, err := executionSpecForSchedule(loaded[0])
			if err != nil {
				t.Fatalf("executionSpecForSchedule failed: %v", err)
			}
			input := model.AuthOverridesFromExecution(exec)

			if len(exec.Auth.Headers) > 0 && len(input.Headers) != len(exec.Auth.Headers) {
				t.Errorf("expected %d headers, got %d", len(exec.Auth.Headers), len(input.Headers))
			}
			if len(exec.Auth.Cookies) > 0 && len(input.Cookies) != len(exec.Auth.Cookies) {
				t.Errorf("expected %d cookies, got %d", len(exec.Auth.Cookies), len(input.Cookies))
			}
			if exec.Auth.Basic != "" && len(input.Tokens) == 0 {
				t.Error("expected basic auth token")
			}
			if exec.Auth.LoginURL != "" {
				if input.Login == nil {
					t.Error("expected login flow")
				}
				if input.Login != nil && input.Login.URL != exec.Auth.LoginURL {
					t.Errorf("expected login URL %s, got %s", exec.Auth.LoginURL, input.Login.URL)
				}
			}
		})
	}
}
