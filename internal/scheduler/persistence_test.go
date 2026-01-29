package scheduler

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/auth"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestExtractConfigPersistence(t *testing.T) {
	dataDir := t.TempDir()

	extractConfigPath := filepath.Join(dataDir, "extract.json")
	extractConfig := `{
		"name": "test-template",
		"selectors": [
			{
				"name": "title",
				"selector": "h1",
				"attr": "text"
			}
		]
	}`
	if err := os.WriteFile(extractConfigPath, []byte(extractConfig), 0o644); err != nil {
		t.Fatalf("failed to write extract config: %v", err)
	}

	schedule := Schedule{
		Kind:            model.KindScrape,
		IntervalSeconds: 60,
		Params: map[string]interface{}{
			"url":             "https://example.com",
			"extractTemplate": "product",
			"extractConfig":   extractConfigPath,
			"extractValidate": true,
		},
	}

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

	extractOpts := loadExtract(loaded[0].Params)
	if extractOpts.Template != "product" {
		t.Errorf("expected extractTemplate 'product', got %s", extractOpts.Template)
	}
	if !extractOpts.Validate {
		t.Error("expected extractValidate true")
	}
	if extractOpts.Inline == nil {
		t.Error("expected inline template to be loaded from extractConfig")
	}
	if extractOpts.Inline.Name != "test-template" {
		t.Errorf("expected inline template name 'test-template', got %s", extractOpts.Inline.Name)
	}
	if len(extractOpts.Inline.Selectors) != 1 {
		t.Errorf("expected 1 selector, got %d", len(extractOpts.Inline.Selectors))
	}
}

func TestIncrementalModePersistence(t *testing.T) {
	testCases := []struct {
		name       string
		kind       model.Kind
		params     map[string]interface{}
		wantResult bool
	}{
		{
			name: "scrape with incremental enabled",
			kind: model.KindScrape,
			params: map[string]interface{}{
				"url":         "https://example.com",
				"incremental": true,
			},
			wantResult: true,
		},
		{
			name: "crawl with incremental enabled",
			kind: model.KindCrawl,
			params: map[string]interface{}{
				"url":         "https://example.com",
				"incremental": true,
			},
			wantResult: true,
		},
		{
			name: "scrape with incremental disabled",
			kind: model.KindScrape,
			params: map[string]interface{}{
				"url":         "https://example.com",
				"incremental": false,
			},
			wantResult: false,
		},
		{
			name: "scrape without incremental flag",
			kind: model.KindScrape,
			params: map[string]interface{}{
				"url": "https://example.com",
			},
			wantResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testDataDir := t.TempDir()

			schedule := Schedule{
				Kind:            tc.kind,
				IntervalSeconds: 60,
				Params:          tc.params,
			}

			if _, err := Add(testDataDir, schedule); err != nil {
				t.Fatalf("Add failed: %v", err)
			}

			loaded, err := LoadAll(testDataDir)
			if err != nil {
				t.Fatalf("LoadAll failed: %v", err)
			}
			if len(loaded) != 1 {
				t.Fatalf("expected 1 schedule, got %d", len(loaded))
			}

			got := boolParam(loaded[0].Params, "incremental")
			if got != tc.wantResult {
				t.Errorf("incremental = %v, want %v", got, tc.wantResult)
			}
		})
	}
}

func TestAuthOverridePersistence(t *testing.T) {
	testCases := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name: "auth with headers",
			params: map[string]interface{}{
				"url": "https://example.com",
				"headers": []auth.HeaderKV{
					{Key: "X-API-Key", Value: "secret"},
					{Key: "Authorization", Value: "Bearer token"},
				},
			},
			wantErr: false,
		},
		{
			name: "auth with cookies",
			params: map[string]interface{}{
				"url": "https://example.com",
				"cookies": []auth.Cookie{
					{Name: "session", Value: "abc123"},
				},
			},
			wantErr: false,
		},
		{
			name: "auth with basic token",
			params: map[string]interface{}{
				"url":       "https://example.com",
				"authBasic": "user:pass",
			},
			wantErr: false,
		},
		{
			name: "auth with bearer tokens",
			params: map[string]interface{}{
				"url":       "https://example.com",
				"tokenKind": "bearer",
				"tokens":    []string{"token1", "token2"},
			},
			wantErr: false,
		},
		{
			name: "auth with login flow",
			params: map[string]interface{}{
				"url":                 "https://example.com",
				"loginURL":            "https://example.com/login",
				"loginUserSelector":   "#username",
				"loginPassSelector":   "#password",
				"loginSubmitSelector": "button[type=submit]",
				"loginUser":           "test@example.com",
				"loginPass":           "password",
			},
			wantErr: false,
		},
		{
			name: "auth with profile and overrides",
			params: map[string]interface{}{
				"url":         "https://example.com",
				"authProfile": "base",
				"headers": []auth.HeaderKV{
					{Key: "X-Custom", Value: "value"},
				},
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testDataDir := t.TempDir()

			if _, hasProfile := tc.params["authProfile"]; hasProfile {
				profile := auth.Profile{
					Name: "base",
				}
				if err := auth.UpsertProfile(testDataDir, profile); err != nil {
					t.Fatalf("failed to create profile: %v", err)
				}
			}

			schedule := Schedule{
				Kind:            model.KindScrape,
				IntervalSeconds: 60,
				Params:          tc.params,
			}

			_, err := Add(testDataDir, schedule)
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

			input := loadAuthOverrides(loaded[0].Params)

			if headers, hasHeaders := tc.params["headers"]; hasHeaders {
				if len(input.Headers) != len(headers.([]auth.HeaderKV)) {
					t.Errorf("expected %d headers, got %d", len(headers.([]auth.HeaderKV)), len(input.Headers))
				}
			}

			if cookies, hasCookies := tc.params["cookies"]; hasCookies {
				if len(input.Cookies) != len(cookies.([]auth.Cookie)) {
					t.Errorf("expected %d cookies, got %d", len(cookies.([]auth.Cookie)), len(input.Cookies))
				}
			}

			if tokens, hasTokens := tc.params["tokens"]; hasTokens {
				if len(input.Tokens) != len(tokens.([]string)) {
					t.Errorf("expected %d tokens, got %d", len(tokens.([]string)), len(input.Tokens))
				}
			}

			if _, hasBasic := tc.params["authBasic"]; hasBasic {
				if len(input.Tokens) == 0 {
					t.Error("expected basic auth token")
				}
			}

			if loginURL, hasLogin := tc.params["loginURL"]; hasLogin {
				if input.Login == nil {
					t.Error("expected login flow")
				}
				if input.Login.URL != loginURL {
					t.Errorf("expected login URL %s, got %s", loginURL, input.Login.URL)
				}
			}
		})
	}
}
