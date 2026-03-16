// Package jobs tests the unified job specification and validation logic.
package jobs

import (
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

func TestJobSpec_Validate_Scrape(t *testing.T) {
	t.Run("Valid scrape spec passes validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindScrape,
			URL:            "https://example.com",
			Headless:       true,
			UsePlaywright:  false,
			Auth:           fetch.AuthOptions{},
			TimeoutSeconds: 30,
			Extract:        extract.ExtractOptions{},
			Pipeline:       pipeline.Options{},
			Incremental:    false,
		}
		if err := spec.Validate(); err != nil {
			t.Errorf("valid scrape spec failed validation: %v", err)
		}
	})

	t.Run("Empty URL fails validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindScrape,
			URL:            "",
			TimeoutSeconds: 30,
		}
		if err := spec.Validate(); err == nil {
			t.Error("expected error for scrape spec with empty URL")
		}
	})

	t.Run("Zero timeout means use default, passes validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindScrape,
			URL:            "https://example.com",
			TimeoutSeconds: 0,
		}
		if err := spec.Validate(); err != nil {
			t.Errorf("zero timeout should be valid (means use default): %v", err)
		}
	})

	t.Run("Negative timeout fails validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindScrape,
			URL:            "https://example.com",
			TimeoutSeconds: -5,
		}
		if err := spec.Validate(); err == nil {
			t.Error("expected error for negative timeout")
		}
	})

	t.Run("Timeout below range (4) fails validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindScrape,
			URL:            "https://example.com",
			TimeoutSeconds: 4,
		}
		if err := spec.Validate(); err == nil {
			t.Error("expected error for timeout below 5")
		}
	})

	t.Run("Timeout at minimum valid (5) passes validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindScrape,
			URL:            "https://example.com",
			TimeoutSeconds: 5,
		}
		if err := spec.Validate(); err != nil {
			t.Errorf("timeout 5 should be valid: %v", err)
		}
	})

	t.Run("Timeout at maximum valid (300) passes validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindScrape,
			URL:            "https://example.com",
			TimeoutSeconds: 300,
		}
		if err := spec.Validate(); err != nil {
			t.Errorf("timeout 300 should be valid: %v", err)
		}
	})

	t.Run("Timeout above range (301) fails validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindScrape,
			URL:            "https://example.com",
			TimeoutSeconds: 301,
		}
		if err := spec.Validate(); err == nil {
			t.Error("expected error for timeout above 300")
		}
	})

	t.Run("Invalid URL (missing scheme) fails validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindScrape,
			URL:            "example.com",
			TimeoutSeconds: 30,
		}
		if err := spec.Validate(); err == nil {
			t.Error("expected error for URL without scheme")
		}
	})

	t.Run("Invalid URL (missing host) fails validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindScrape,
			URL:            "http://",
			TimeoutSeconds: 30,
		}
		if err := spec.Validate(); err == nil {
			t.Error("expected error for URL without host")
		}
	})

	t.Run("Invalid URL scheme (ftp) fails validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindScrape,
			URL:            "ftp://example.com",
			TimeoutSeconds: 30,
		}
		if err := spec.Validate(); err == nil {
			t.Error("expected error for URL with ftp scheme")
		}
	})

	t.Run("Valid http URL passes validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindScrape,
			URL:            "http://example.com",
			TimeoutSeconds: 30,
		}
		if err := spec.Validate(); err != nil {
			t.Errorf("http URL should be valid: %v", err)
		}
	})

	t.Run("Valid https URL passes validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindScrape,
			URL:            "https://example.com",
			TimeoutSeconds: 30,
		}
		if err := spec.Validate(); err != nil {
			t.Errorf("https URL should be valid: %v", err)
		}
	})
}

func TestJobSpec_Validate_WebhookURL(t *testing.T) {
	t.Run("Valid webhook URL passes validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindScrape,
			URL:            "https://example.com",
			TimeoutSeconds: 30,
			WebhookURL:     "https://hooks.example.com/job",
		}
		if err := spec.Validate(); err != nil {
			t.Fatalf("expected valid webhook URL, got %v", err)
		}
	})

	t.Run("Invalid webhook URL scheme fails validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindScrape,
			URL:            "https://example.com",
			TimeoutSeconds: 30,
			WebhookURL:     "ftp://hooks.example.com/job",
		}
		if err := spec.Validate(); err == nil {
			t.Fatal("expected invalid webhook URL to fail validation")
		}
	})

	t.Run("Webhook secret without URL fails validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindScrape,
			URL:            "https://example.com",
			TimeoutSeconds: 30,
			WebhookSecret:  "top-secret",
		}
		if err := spec.Validate(); err == nil {
			t.Fatal("expected missing webhook URL to fail validation when webhook fields are present")
		}
	})
}

func TestJobSpec_Validate_Crawl(t *testing.T) {
	t.Run("Valid crawl spec passes validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindCrawl,
			URL:            "https://example.com",
			MaxDepth:       2,
			MaxPages:       100,
			Headless:       true,
			UsePlaywright:  false,
			Auth:           fetch.AuthOptions{},
			TimeoutSeconds: 30,
			Extract:        extract.ExtractOptions{},
			Pipeline:       pipeline.Options{},
			Incremental:    false,
		}
		if err := spec.Validate(); err != nil {
			t.Errorf("valid crawl spec failed validation: %v", err)
		}
	})

	t.Run("Empty URL fails validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindCrawl,
			MaxDepth:       2,
			MaxPages:       100,
			TimeoutSeconds: 30,
		}
		if err := spec.Validate(); err == nil {
			t.Error("expected error for crawl spec with empty URL")
		}
	})

	t.Run("Zero MaxDepth means default, passes validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindCrawl,
			URL:            "https://example.com",
			MaxDepth:       0,
			MaxPages:       100,
			TimeoutSeconds: 30,
		}
		if err := spec.Validate(); err != nil {
			t.Errorf("zero MaxDepth should be valid (means use default): %v", err)
		}
	})

	t.Run("Negative MaxDepth fails validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindCrawl,
			URL:            "https://example.com",
			MaxDepth:       -1,
			MaxPages:       100,
			TimeoutSeconds: 30,
		}
		if err := spec.Validate(); err == nil {
			t.Error("expected error for negative MaxDepth")
		}
	})

	t.Run("MaxDepth at minimum valid (1) passes validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindCrawl,
			URL:            "https://example.com",
			MaxDepth:       1,
			MaxPages:       100,
			TimeoutSeconds: 30,
		}
		if err := spec.Validate(); err != nil {
			t.Errorf("MaxDepth 1 should be valid: %v", err)
		}
	})

	t.Run("MaxDepth at maximum valid (10) passes validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindCrawl,
			URL:            "https://example.com",
			MaxDepth:       10,
			MaxPages:       100,
			TimeoutSeconds: 30,
		}
		if err := spec.Validate(); err != nil {
			t.Errorf("MaxDepth 10 should be valid: %v", err)
		}
	})

	t.Run("MaxDepth above range (11) fails validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindCrawl,
			URL:            "https://example.com",
			MaxDepth:       11,
			MaxPages:       100,
			TimeoutSeconds: 30,
		}
		if err := spec.Validate(); err == nil {
			t.Error("expected error for MaxDepth above 10")
		}
	})

	t.Run("Zero MaxPages means default, passes validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindCrawl,
			URL:            "https://example.com",
			MaxDepth:       2,
			MaxPages:       0,
			TimeoutSeconds: 30,
		}
		if err := spec.Validate(); err != nil {
			t.Errorf("zero MaxPages should be valid (means use default): %v", err)
		}
	})

	t.Run("Negative MaxPages fails validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindCrawl,
			URL:            "https://example.com",
			MaxDepth:       2,
			MaxPages:       -1,
			TimeoutSeconds: 30,
		}
		if err := spec.Validate(); err == nil {
			t.Error("expected error for negative MaxPages")
		}
	})

	t.Run("MaxPages at minimum valid (1) passes validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindCrawl,
			URL:            "https://example.com",
			MaxDepth:       2,
			MaxPages:       1,
			TimeoutSeconds: 30,
		}
		if err := spec.Validate(); err != nil {
			t.Errorf("MaxPages 1 should be valid: %v", err)
		}
	})

	t.Run("MaxPages at maximum valid (10000) passes validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindCrawl,
			URL:            "https://example.com",
			MaxDepth:       2,
			MaxPages:       10000,
			TimeoutSeconds: 30,
		}
		if err := spec.Validate(); err != nil {
			t.Errorf("MaxPages 10000 should be valid: %v", err)
		}
	})

	t.Run("MaxPages above range (10001) fails validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindCrawl,
			URL:            "https://example.com",
			MaxDepth:       2,
			MaxPages:       10001,
			TimeoutSeconds: 30,
		}
		if err := spec.Validate(); err == nil {
			t.Error("expected error for MaxPages above 10000")
		}
	})

	t.Run("Crawl with zero timeout passes validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindCrawl,
			URL:            "https://example.com",
			MaxDepth:       2,
			MaxPages:       100,
			TimeoutSeconds: 0,
		}
		if err := spec.Validate(); err != nil {
			t.Errorf("zero timeout should be valid for crawl: %v", err)
		}
	})
}

func TestJobSpec_Validate_Research(t *testing.T) {
	t.Run("Valid research spec passes validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindResearch,
			Query:          "test query",
			URLs:           []string{"https://example.com"},
			MaxDepth:       2,
			MaxPages:       100,
			Headless:       true,
			UsePlaywright:  false,
			Auth:           fetch.AuthOptions{},
			TimeoutSeconds: 30,
			Extract:        extract.ExtractOptions{},
			Pipeline:       pipeline.Options{},
			Incremental:    false,
		}
		if err := spec.Validate(); err != nil {
			t.Errorf("valid research spec failed validation: %v", err)
		}
	})

	t.Run("Empty query fails validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindResearch,
			URLs:           []string{"https://example.com"},
			MaxDepth:       2,
			MaxPages:       100,
			TimeoutSeconds: 30,
		}
		if err := spec.Validate(); err == nil {
			t.Error("expected error for research spec with empty query")
		}
	})

	t.Run("Empty URLs fails validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindResearch,
			Query:          "test query",
			MaxDepth:       2,
			MaxPages:       100,
			TimeoutSeconds: 30,
		}
		if err := spec.Validate(); err == nil {
			t.Error("expected error for research spec with empty URLs")
		}
	})

	t.Run("Nil URLs fails validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindResearch,
			Query:          "test query",
			URLs:           nil,
			MaxDepth:       2,
			MaxPages:       100,
			TimeoutSeconds: 30,
		}
		if err := spec.Validate(); err == nil {
			t.Error("expected error for research spec with nil URLs")
		}
	})

	t.Run("Invalid URL in list fails validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindResearch,
			Query:          "test query",
			URLs:           []string{"https://example.com", "invalid-url"},
			MaxDepth:       2,
			MaxPages:       100,
			TimeoutSeconds: 30,
		}
		if err := spec.Validate(); err == nil {
			t.Error("expected error for research spec with invalid URL in list")
		}
	})

	t.Run("Zero MaxDepth means default, passes validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindResearch,
			Query:          "test query",
			URLs:           []string{"https://example.com"},
			MaxDepth:       0,
			MaxPages:       100,
			TimeoutSeconds: 30,
		}
		if err := spec.Validate(); err != nil {
			t.Errorf("zero MaxDepth should be valid for research: %v", err)
		}
	})

	t.Run("Negative MaxDepth fails validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindResearch,
			Query:          "test query",
			URLs:           []string{"https://example.com"},
			MaxDepth:       -1,
			MaxPages:       100,
			TimeoutSeconds: 30,
		}
		if err := spec.Validate(); err == nil {
			t.Error("expected error for negative MaxDepth in research")
		}
	})

	t.Run("MaxDepth above range (11) fails validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindResearch,
			Query:          "test query",
			URLs:           []string{"https://example.com"},
			MaxDepth:       11,
			MaxPages:       100,
			TimeoutSeconds: 30,
		}
		if err := spec.Validate(); err == nil {
			t.Error("expected error for MaxDepth above 10 in research")
		}
	})

	t.Run("Zero MaxPages means default, passes validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindResearch,
			Query:          "test query",
			URLs:           []string{"https://example.com"},
			MaxDepth:       2,
			MaxPages:       0,
			TimeoutSeconds: 30,
		}
		if err := spec.Validate(); err != nil {
			t.Errorf("zero MaxPages should be valid for research: %v", err)
		}
	})

	t.Run("Negative MaxPages fails validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindResearch,
			Query:          "test query",
			URLs:           []string{"https://example.com"},
			MaxDepth:       2,
			MaxPages:       -1,
			TimeoutSeconds: 30,
		}
		if err := spec.Validate(); err == nil {
			t.Error("expected error for negative MaxPages in research")
		}
	})

	t.Run("MaxPages above range (10001) fails validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindResearch,
			Query:          "test query",
			URLs:           []string{"https://example.com"},
			MaxDepth:       2,
			MaxPages:       10001,
			TimeoutSeconds: 30,
		}
		if err := spec.Validate(); err == nil {
			t.Error("expected error for MaxPages above 10000 in research")
		}
	})

	t.Run("Research with zero timeout passes validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindResearch,
			Query:          "test query",
			URLs:           []string{"https://example.com"},
			MaxDepth:       2,
			MaxPages:       100,
			TimeoutSeconds: 0,
		}
		if err := spec.Validate(); err != nil {
			t.Errorf("zero timeout should be valid for research: %v", err)
		}
	})
}

func TestJobSpec_Validate_UnknownKind(t *testing.T) {
	spec := JobSpec{
		Kind:           "unknown",
		TimeoutSeconds: 30,
	}
	if err := spec.Validate(); err == nil {
		t.Error("expected error for spec with unknown kind")
	}
}

func TestNewScrapeSpec(t *testing.T) {
	spec := NewScrapeSpec("https://example.com")
	if spec.Kind != model.KindScrape {
		t.Errorf("expected KindScrape, got %s", spec.Kind)
	}
	if spec.URL != "https://example.com" {
		t.Errorf("expected URL 'https://example.com', got '%s'", spec.URL)
	}
}

func TestNewCrawlSpec(t *testing.T) {
	spec := NewCrawlSpec("https://example.com", 3, 200)
	if spec.Kind != model.KindCrawl {
		t.Errorf("expected KindCrawl, got %s", spec.Kind)
	}
	if spec.URL != "https://example.com" {
		t.Errorf("expected URL 'https://example.com', got '%s'", spec.URL)
	}
	if spec.MaxDepth != 3 {
		t.Errorf("expected MaxDepth 3, got %d", spec.MaxDepth)
	}
	if spec.MaxPages != 200 {
		t.Errorf("expected MaxPages 200, got %d", spec.MaxPages)
	}
}

func TestNewResearchSpec(t *testing.T) {
	urls := []string{"https://example.com", "https://example.org"}
	spec := NewResearchSpec("test query", urls, 2, 150)
	if spec.Kind != model.KindResearch {
		t.Errorf("expected KindResearch, got %s", spec.Kind)
	}
	if spec.Query != "test query" {
		t.Errorf("expected Query 'test query', got '%s'", spec.Query)
	}
	if len(spec.URLs) != 2 {
		t.Errorf("expected 2 URLs, got %d", len(spec.URLs))
	}
	if spec.MaxDepth != 2 {
		t.Errorf("expected MaxDepth 2, got %d", spec.MaxDepth)
	}
	if spec.MaxPages != 150 {
		t.Errorf("expected MaxPages 150, got %d", spec.MaxPages)
	}
}
