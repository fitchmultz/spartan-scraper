// Package mcp provides tests for the crawl_site MCP tool.
// Tests cover job creation with partial pipeline options and JSON schema validation.
// Does NOT test actual crawling execution, URL discovery, or depth limiting.
package mcp

import (
	"context"
	"os"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
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
		"",
		"",
		false,
		"",
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("CreateCrawlJob failed: %v", err)
	}

	pipelineMap := pipelineMapFromSpec(t, job.SpecMap())
	preProcessors, _ := pipelineMap["preProcessors"].([]interface{})
	postProcessors, _ := pipelineMap["postProcessors"].([]interface{})
	if len(preProcessors) != 1 {
		t.Errorf("expected 1 preProcessor, got %d", len(preProcessors))
	}
	if len(postProcessors) != 0 {
		t.Errorf("expected 0 postProcessors, got %d", len(postProcessors))
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
