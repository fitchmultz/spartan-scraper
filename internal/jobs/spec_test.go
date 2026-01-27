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

	t.Run("Zero timeout fails validation", func(t *testing.T) {
		spec := JobSpec{
			Kind:           model.KindScrape,
			URL:            "https://example.com",
			TimeoutSeconds: 0,
		}
		if err := spec.Validate(); err == nil {
			t.Error("expected error for scrape spec with zero timeout")
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
