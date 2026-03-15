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

	for _, field := range []string{"method", "body", "contentType", "headless", "playwright", "timeoutSeconds", "authProfile", "auth", "extract", "pipeline", "incremental", "webhook", "screenshot", "device", "networkIntercept"} {
		if _, ok := props[field]; !ok {
			t.Errorf("expected %s in properties", field)
		}
		if requiredSet[field] {
			t.Errorf("expected %s to be optional, but it's in required", field)
		}
	}

	authType, ok := props["auth"].(map[string]string)
	if !ok || authType["type"] != "object" {
		t.Error("auth should be object type")
	}
	extractType, ok := props["extract"].(map[string]string)
	if !ok || extractType["type"] != "object" {
		t.Error("extract should be object type")
	}
	pipelineType, ok := props["pipeline"].(map[string]string)
	if !ok || pipelineType["type"] != "object" {
		t.Error("pipeline should be object type")
	}
	screenshotType, ok := props["screenshot"].(map[string]string)
	if !ok || screenshotType["type"] != "object" {
		t.Error("screenshot should be object type")
	}
	deviceType, ok := props["device"].(map[string]string)
	if !ok || deviceType["type"] != "object" {
		t.Error("device should be object type")
	}
	networkInterceptType, ok := props["networkIntercept"].(map[string]string)
	if !ok || networkInterceptType["type"] != "object" {
		t.Error("networkIntercept should be object type")
	}
	incrementalType, ok := props["incremental"].(map[string]string)
	if !ok || incrementalType["type"] != "boolean" {
		t.Error("incremental should be boolean type")
	}
}
