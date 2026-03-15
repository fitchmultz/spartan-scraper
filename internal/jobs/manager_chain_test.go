// Package jobs provides tests for chain submission using the shared operator-facing
// request model.
//
// Purpose:
//   - Verify chain jobs are created from request payloads via the same conversion path
//     as live job submissions.
//
// Responsibilities:
// - Assert dependency metadata is persisted correctly.
// - Assert only ready chain roots are enqueued immediately.
// - Assert persisted typed specs keep execution defaults after request conversion.
//
// Scope:
// - Chain submission behavior only.
//
// Usage:
// - Run with `go test ./internal/jobs`.
//
// Invariants/Assumptions:
// - Chain nodes now store operator-facing request payloads, not typed specs.
// - SubmitChain callers provide the shared request-to-JobSpec resolver.
package jobs

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestSubmitChainBuildsJobsFromOperatorRequests(t *testing.T) {
	manager, st, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()
	chain, err := manager.CreateChain(ctx, "request-backed-chain", "", model.ChainDefinition{
		Nodes: []model.ChainNode{
			{ID: "root", Kind: model.KindScrape, Request: json.RawMessage(`{"url":"https://example.com/root"}`)},
			{ID: "child", Kind: model.KindCrawl, Request: json.RawMessage(`{"url":"https://example.com/root","maxDepth":1,"maxPages":5}`)},
		},
		Edges: []model.ChainEdge{{From: "root", To: "child"}},
	})
	if err != nil {
		t.Fatalf("CreateChain() failed: %v", err)
	}

	createdJobs, err := manager.SubmitChain(ctx, chain.ID, nil, func(kind model.Kind, raw json.RawMessage) (JobSpec, error) {
		switch kind {
		case model.KindScrape:
			return JobSpec{
				Kind:           model.KindScrape,
				URL:            "https://example.com/root",
				TimeoutSeconds: manager.DefaultTimeoutSeconds(),
			}, nil
		case model.KindCrawl:
			return JobSpec{
				Kind:           model.KindCrawl,
				URL:            "https://example.com/root",
				MaxDepth:       1,
				MaxPages:       5,
				TimeoutSeconds: manager.DefaultTimeoutSeconds(),
			}, nil
		default:
			return JobSpec{}, nil
		}
	})
	if err != nil {
		t.Fatalf("SubmitChain() failed: %v", err)
	}
	if len(createdJobs) != 2 {
		t.Fatalf("expected 2 chain jobs, got %d", len(createdJobs))
	}

	root := createdJobs[0]
	child := createdJobs[1]
	if root.ChainID != chain.ID || child.ChainID != chain.ID {
		t.Fatalf("expected chain IDs to be persisted on both jobs")
	}
	if root.DependencyStatus != model.DependencyStatusReady {
		t.Fatalf("expected root dependency status ready, got %s", root.DependencyStatus)
	}
	if child.DependencyStatus != model.DependencyStatusPending {
		t.Fatalf("expected child dependency status pending, got %s", child.DependencyStatus)
	}
	if len(child.DependsOn) != 1 || child.DependsOn[0] != root.ID {
		t.Fatalf("expected child to depend on root job %s, got %#v", root.ID, child.DependsOn)
	}
	if gotQueued := len(manager.queue); gotQueued != 1 {
		t.Fatalf("expected only the ready root job to be enqueued, got %d queued jobs", gotQueued)
	}

	persistedRoot, err := st.Get(ctx, root.ID)
	if err != nil {
		t.Fatalf("failed to load persisted root job: %v", err)
	}
	rootSpec, ok := persistedRoot.Spec.(model.ScrapeSpecV1)
	if !ok {
		t.Fatalf("expected persisted scrape spec, got %T", persistedRoot.Spec)
	}
	if rootSpec.Execution.TimeoutSeconds != manager.DefaultTimeoutSeconds() {
		t.Fatalf("expected default timeout %d, got %d", manager.DefaultTimeoutSeconds(), rootSpec.Execution.TimeoutSeconds)
	}

	persistedChild, err := st.Get(ctx, child.ID)
	if err != nil {
		t.Fatalf("failed to load persisted child job: %v", err)
	}
	childSpec, ok := persistedChild.Spec.(model.CrawlSpecV1)
	if !ok {
		t.Fatalf("expected persisted crawl spec, got %T", persistedChild.Spec)
	}
	if childSpec.Execution.TimeoutSeconds != manager.DefaultTimeoutSeconds() {
		t.Fatalf("expected default timeout %d, got %d", manager.DefaultTimeoutSeconds(), childSpec.Execution.TimeoutSeconds)
	}
	if childSpec.MaxDepth != 1 || childSpec.MaxPages != 5 {
		t.Fatalf("unexpected child crawl bounds: depth=%d pages=%d", childSpec.MaxDepth, childSpec.MaxPages)
	}
}
