// Package mcp provides tests for the scrape_page MCP tool.
// Tests cover job creation with pipeline options (preProcessors, postProcessors, transformers)
// and incremental mode, plus JSON schema validation.
// Does NOT test actual scraping execution, HTTP handling, or content fetching.
package mcp

import (
	"context"
	"os"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

func TestScrapePageWithPipelineAndIncremental(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	ctx := context.Background()

	job, err := srv.manager.CreateScrapeJob(
		ctx,
		"http://example.com",
		"GET",
		nil,
		"",
		false,
		false,
		fetch.AuthOptions{},
		30,
		extract.ExtractOptions{},
		pipeline.Options{
			PreProcessors:  []string{"prep1", "prep2"},
			PostProcessors: []string{"post1"},
			Transformers:   []string{"trans1", "trans2", "trans3"},
		},
		true,
		"",
		"",
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("CreateScrapeJob failed: %v", err)
	}

	if job.Params["pipeline"] == nil {
		t.Error("pipeline options not stored in job params")
	}
	pipelineOpts, ok := job.Params["pipeline"].(pipeline.Options)
	if !ok {
		t.Fatalf("pipeline params is not pipeline.Options type")
	}
	if len(pipelineOpts.PreProcessors) != 2 {
		t.Errorf("expected 2 preProcessors, got %d", len(pipelineOpts.PreProcessors))
	}
	if len(pipelineOpts.PostProcessors) != 1 {
		t.Errorf("expected 1 postProcessor, got %d", len(pipelineOpts.PostProcessors))
	}
	if len(pipelineOpts.Transformers) != 3 {
		t.Errorf("expected 3 transformers, got %d", len(pipelineOpts.Transformers))
	}

	inc, ok := job.Params["incremental"].(bool)
	if !ok || !inc {
		t.Error("incremental flag not stored correctly in job params")
	}
}

func TestScrapePageSchema(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	tools := srv.toolsList()
	toolMap := make(map[string]tool)
	for _, t := range tools {
		toolMap[t.Name] = t
	}

	scrapeTool, ok := toolMap["scrape_page"]
	if !ok {
		t.Fatal("scrape_page tool not found")
	}
	schema := scrapeTool.InputSchema
	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("properties not found in schema")
	}
	requiredFields := schema["required"]
	requiredSlice, _ := requiredFields.([]interface{})
	requiredSet := make(map[string]bool)
	for _, f := range requiredSlice {
		requiredSet[f.(string)] = true
	}

	for _, field := range []string{"preProcessors", "postProcessors", "transformers", "incremental"} {
		if _, ok := props[field]; !ok {
			t.Errorf("expected %s in properties", field)
		}
		if requiredSet[field] {
			t.Errorf("expected %s to be optional, but it's in required", field)
		}
	}

	preProcessorsType, ok := props["preProcessors"].(map[string]string)
	if !ok || preProcessorsType["type"] != "array" {
		t.Error("preProcessors should be array type")
	}
	postProcessorsType, ok := props["postProcessors"].(map[string]string)
	if !ok || postProcessorsType["type"] != "array" {
		t.Error("postProcessors should be array type")
	}
	transformersType, ok := props["transformers"].(map[string]string)
	if !ok || transformersType["type"] != "array" {
		t.Error("transformers should be array type")
	}
	incrementalType, ok := props["incremental"].(map[string]string)
	if !ok || incrementalType["type"] != "boolean" {
		t.Error("incremental should be boolean type")
	}
}
