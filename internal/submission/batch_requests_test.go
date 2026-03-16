// Package submission verifies canonical batch request validation and conversion.
//
// Purpose:
//   - Prove batch scrape, crawl, and research requests reuse the same canonical
//     request-to-spec conversion path as single-job submissions.
//
// Responsibilities:
// - Assert batch scrape specs inherit single-job defaults like GET.
// - Assert research batches collapse submitted URLs into one research job spec.
// - Assert live batch conversion resolves auth/env defaults when requested.
//
// Scope:
// - Submission-layer batch validation and conversion only.
//
// Usage:
// - Run with `go test ./internal/submission`.
//
// Invariants/Assumptions:
// - Batch conversion delegates to the single-job converters rather than rebuilding specs separately.
package submission

import (
	"net/http"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestJobSpecsFromBatchScrapeRequestDefaultsMethodToGET(t *testing.T) {
	specs, err := JobSpecsFromBatchScrapeRequest(config.Config{}, BatchDefaults{
		Defaults: Defaults{DefaultTimeoutSeconds: 30, DefaultUsePlaywright: false, ResolveAuth: false},
	}, BatchScrapeRequest{
		Jobs: []BatchJobRequest{{URL: "https://example.com/articles/1"}},
	})
	if err != nil {
		t.Fatalf("JobSpecsFromBatchScrapeRequest() error = %v", err)
	}
	if len(specs) != 1 {
		t.Fatalf("expected 1 spec, got %d", len(specs))
	}
	if specs[0].Method != http.MethodGet {
		t.Fatalf("expected method %q, got %q", http.MethodGet, specs[0].Method)
	}
}

func TestJobSpecsFromBatchResearchRequestCreatesSingleResearchSpec(t *testing.T) {
	specs, err := JobSpecsFromBatchResearchRequest(config.Config{}, BatchDefaults{
		Defaults: Defaults{DefaultTimeoutSeconds: 30, DefaultUsePlaywright: false, ResolveAuth: false},
	}, BatchResearchRequest{
		Query: "recent announcements",
		Jobs: []BatchJobRequest{
			{URL: "https://example.com/one"},
			{URL: "https://example.com/two"},
		},
		Agentic: &model.ResearchAgenticConfig{Enabled: true, MaxRounds: 2, MaxFollowUpURLs: 4},
	})
	if err != nil {
		t.Fatalf("JobSpecsFromBatchResearchRequest() error = %v", err)
	}
	if len(specs) != 1 {
		t.Fatalf("expected 1 research spec, got %d", len(specs))
	}
	if specs[0].Kind != model.KindResearch {
		t.Fatalf("expected research kind, got %s", specs[0].Kind)
	}
	if len(specs[0].URLs) != 2 {
		t.Fatalf("expected 2 research URLs, got %d", len(specs[0].URLs))
	}
	if specs[0].Agentic == nil || !specs[0].Agentic.Enabled {
		t.Fatalf("expected agentic config to be preserved, got %#v", specs[0].Agentic)
	}
}

func TestJobSpecsFromBatchScrapeRequestResolvesEnvAuthWhenEnabled(t *testing.T) {
	cfg := config.Config{
		DataDir: t.TempDir(),
		AuthOverrides: config.EnvOverrides{
			Headers: map[string]string{"X-Batch-Env": "present"},
		},
	}

	specs, err := JobSpecsFromBatchScrapeRequest(cfg, BatchDefaults{
		Defaults: Defaults{DefaultTimeoutSeconds: 30, DefaultUsePlaywright: false, ResolveAuth: true},
	}, BatchScrapeRequest{
		Jobs: []BatchJobRequest{{URL: "https://example.com/articles/1"}},
	})
	if err != nil {
		t.Fatalf("JobSpecsFromBatchScrapeRequest() error = %v", err)
	}
	if len(specs) != 1 {
		t.Fatalf("expected 1 spec, got %d", len(specs))
	}
	if got := specs[0].Auth.Headers["X-Batch-Env"]; got != "present" {
		t.Fatalf("expected env auth header to be resolved, got %#v", specs[0].Auth.Headers)
	}
}
