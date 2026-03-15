package jobs

import (
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

func TestDecodeExecutionInputsPreserveBrowserExecutionOptions(t *testing.T) {
	manager, _, cleanup := setupTestManager(t)
	defer cleanup()

	device := fetch.GetDevicePreset("iphone15")
	if device == nil {
		t.Fatal("missing iphone15 device preset")
	}
	screenshot := &fetch.ScreenshotConfig{
		Enabled:  true,
		FullPage: false,
		Format:   fetch.ScreenshotFormatJPEG,
		Quality:  85,
		Width:    900,
		Height:   700,
	}
	intercept := &fetch.NetworkInterceptConfig{
		Enabled:             true,
		URLPatterns:         []string{"**/api/**"},
		ResourceTypes:       []fetch.InterceptedResourceType{fetch.ResourceTypeXHR, fetch.ResourceTypeFetch},
		CaptureRequestBody:  false,
		CaptureResponseBody: true,
		MaxBodySize:         2048,
		MaxEntries:          25,
	}
	auth := fetch.AuthOptions{Headers: map[string]string{"X-Test-Header": "parity"}}
	extractOpts := extract.ExtractOptions{Template: "parity"}
	pipelineOpts := pipeline.Options{Transformers: []string{"json-clean"}}

	t.Run("scrape", func(t *testing.T) {
		job := model.Job{
			Kind: model.KindScrape,
			Spec: model.ScrapeSpecV1{
				Version: model.JobSpecVersion1,
				URL:     "https://example.com",
				Execution: model.ExecutionSpec{
					RequestID:        "req-scrape",
					TimeoutSeconds:   30,
					Auth:             auth,
					Extract:          extractOpts,
					Pipeline:         pipelineOpts,
					Screenshot:       screenshot,
					Device:           device,
					NetworkIntercept: intercept,
				},
			},
		}
		input, err := decodeScrapeExecutionInput(job, manager)
		if err != nil {
			t.Fatalf("decodeScrapeExecutionInput() failed: %v", err)
		}
		assertExecutionConfigParity(t, input.Config, screenshot, device, intercept)
	})

	t.Run("crawl", func(t *testing.T) {
		job := model.Job{
			Kind: model.KindCrawl,
			Spec: model.CrawlSpecV1{
				Version:  model.JobSpecVersion1,
				URL:      "https://example.com",
				MaxDepth: 1,
				MaxPages: 2,
				Execution: model.ExecutionSpec{
					RequestID:        "req-crawl",
					TimeoutSeconds:   30,
					Auth:             auth,
					Extract:          extractOpts,
					Pipeline:         pipelineOpts,
					Screenshot:       screenshot,
					Device:           device,
					NetworkIntercept: intercept,
				},
			},
		}
		input, err := decodeCrawlExecutionInput(job, manager)
		if err != nil {
			t.Fatalf("decodeCrawlExecutionInput() failed: %v", err)
		}
		assertExecutionConfigParity(t, input.Config, screenshot, device, intercept)
	})

	t.Run("research", func(t *testing.T) {
		job := model.Job{
			Kind: model.KindResearch,
			Spec: model.ResearchSpecV1{
				Version:  model.JobSpecVersion1,
				Query:    "pricing model",
				URLs:     []string{"https://example.com"},
				MaxDepth: 0,
				MaxPages: 2,
				Execution: model.ExecutionSpec{
					RequestID:        "req-research",
					TimeoutSeconds:   30,
					Auth:             auth,
					Extract:          extractOpts,
					Pipeline:         pipelineOpts,
					Screenshot:       screenshot,
					Device:           device,
					NetworkIntercept: intercept,
				},
			},
		}
		input, err := decodeResearchExecutionInput(job, manager)
		if err != nil {
			t.Fatalf("decodeResearchExecutionInput() failed: %v", err)
		}
		assertExecutionConfigParity(t, input.Config, screenshot, device, intercept)
	})
}

func assertExecutionConfigParity(t *testing.T, cfg executionConfig, screenshot *fetch.ScreenshotConfig, device *fetch.DeviceEmulation, intercept *fetch.NetworkInterceptConfig) {
	t.Helper()
	if cfg.Screenshot == nil || !cfg.Screenshot.Enabled {
		t.Fatalf("missing screenshot config: %#v", cfg.Screenshot)
	}
	if cfg.Screenshot.Format != screenshot.Format || cfg.Screenshot.Quality != screenshot.Quality {
		t.Fatalf("unexpected screenshot config: %#v", cfg.Screenshot)
	}
	if cfg.Device == nil || cfg.Device.Name != device.Name {
		t.Fatalf("unexpected device config: %#v", cfg.Device)
	}
	if cfg.NetworkIntercept == nil || !cfg.NetworkIntercept.Enabled {
		t.Fatalf("missing network intercept config: %#v", cfg.NetworkIntercept)
	}
	if cfg.NetworkIntercept.MaxEntries != intercept.MaxEntries || cfg.NetworkIntercept.MaxBodySize != intercept.MaxBodySize {
		t.Fatalf("unexpected network intercept config: %#v", cfg.NetworkIntercept)
	}
}
