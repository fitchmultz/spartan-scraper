// Tests for the crawl_site MCP tool.
// Verifies job creation with partial pipeline options and validates the tool's
// JSON schema definition.
//
// Does NOT handle:
// - Actual crawling execution or URL discovery
// - Concurrent job processing or depth limiting
// - Pipeline processor/transformer execution
//
// Invariants:
// - crawl_site accepts optional pipeline options
// - Partial pipeline options should default empty slices for unspecified fields
// - Schema must include preProcessors, postProcessors, transformers, incremental fields
package mcp

import (
	"context"
	"os"
	"testing"

	"spartan-scraper/internal/extract"
	"spartan-scraper/internal/fetch"
	"spartan-scraper/internal/pipeline"
)

func TestCrawlSiteWithPartialPipelineOptions(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	ctx := context.Background()

	job, err := srv.manager.CreateCrawlJob(
		ctx,
		"http://example.com",
		2,
		100,
		false,
		false,
		fetch.AuthOptions{},
		30,
		extract.ExtractOptions{},
		pipeline.Options{
			PreProcessors: []string{"only-prep"},
		},
		false,
	)
	if err != nil {
		t.Fatalf("CreateCrawlJob failed: %v", err)
	}

	pipelineOpts, _ := job.Params["pipeline"].(pipeline.Options)
	if len(pipelineOpts.PreProcessors) != 1 {
		t.Errorf("expected 1 preProcessor, got %d", len(pipelineOpts.PreProcessors))
	}
	if len(pipelineOpts.PostProcessors) != 0 {
		t.Errorf("expected 0 postProcessors, got %d", len(pipelineOpts.PostProcessors))
	}
}

func TestCrawlSiteSchema(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	tools := srv.toolsList()
	toolMap := make(map[string]tool)
	for _, t := range tools {
		toolMap[t.Name] = t
	}

	crawlTool, ok := toolMap["crawl_site"]
	if !ok {
		t.Fatal("crawl_site tool not found")
	}
	schema := crawlTool.InputSchema
	props, _ := schema["properties"].(map[string]interface{})
	for _, field := range []string{"preProcessors", "postProcessors", "transformers", "incremental"} {
		if _, ok := props[field]; !ok {
			t.Errorf("expected %s in properties", field)
		}
	}
}
