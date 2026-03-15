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

func pipelineMapFromSpec(t *testing.T, jobSpec map[string]interface{}) map[string]interface{} {
	t.Helper()
	raw, ok := jobSpec["pipeline"]
	if !ok {
		t.Fatal("pipeline options not stored in job spec")
	}
	pipelineMap, ok := raw.(map[string]interface{})
	if !ok {
		t.Fatalf("pipeline spec is not a map, got %T", raw)
	}
	return pipelineMap
}

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

	pipelineMap := pipelineMapFromSpec(t, job.SpecMap())
	preProcessors, _ := pipelineMap["preProcessors"].([]interface{})
	postProcessors, _ := pipelineMap["postProcessors"].([]interface{})
	transformers, _ := pipelineMap["transformers"].([]interface{})
	if len(preProcessors) != 2 {
		t.Errorf("expected 2 preProcessors, got %d", len(preProcessors))
	}
	if len(postProcessors) != 1 {
		t.Errorf("expected 1 postProcessor, got %d", len(postProcessors))
	}
	if len(transformers) != 3 {
		t.Errorf("expected 3 transformers, got %d", len(transformers))
	}

	inc, ok := job.SpecMap()["incremental"].(bool)
	if !ok || !inc {
		t.Error("incremental flag not stored correctly in job spec")
	}
}

func TestResolveAuthForTool_RejectsConflictingProxyOverrides(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	_, err := resolveAuthForTool(srv.cfg, "https://example.com", "", fetch.AuthOptions{
		Proxy: &fetch.ProxyConfig{URL: "http://proxy.example:8080"},
		ProxyHints: &fetch.ProxySelectionHints{
			PreferredRegion: "us-east",
		},
	})
	if err == nil {
		t.Fatal("expected conflicting proxy overrides to fail")
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

	for _, field := range []string{"aiExtract", "aiMode", "aiPrompt", "aiSchema", "aiFields", "preProcessors", "postProcessors", "transformers", "incremental", "proxy", "proxyUsername", "proxyPassword", "proxyRegion", "proxyTags", "excludeProxyIds"} {
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
	aiExtractType, ok := props["aiExtract"].(map[string]string)
	if !ok || aiExtractType["type"] != "boolean" {
		t.Error("aiExtract should be boolean type")
	}
	aiModeType, ok := props["aiMode"].(map[string]string)
	if !ok || aiModeType["type"] != "string" {
		t.Error("aiMode should be string type")
	}
	aiSchemaType, ok := props["aiSchema"].(map[string]string)
	if !ok || aiSchemaType["type"] != "object" {
		t.Error("aiSchema should be object type")
	}
	aiFieldsType, ok := props["aiFields"].(map[string]string)
	if !ok || aiFieldsType["type"] != "array" {
		t.Error("aiFields should be array type")
	}
	incrementalType, ok := props["incremental"].(map[string]string)
	if !ok || incrementalType["type"] != "boolean" {
		t.Error("incremental should be boolean type")
	}
}
