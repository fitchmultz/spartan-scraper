// Package mcp provides integration tests for MCP tool execution.
// Tests cover handleToolCall routing, argument parsing, and job creation for scrape_page,
// crawl_site, and research tools with pipeline/incremental options.
// Does NOT test schema validation, server lifecycle, or job management operations.
package mcp

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestHandleToolCallWithPipelineAndIncremental(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	ctx := context.Background()

	t.Run("scrape_page with all pipeline options and incremental true", func(t *testing.T) {
		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name": "scrape_page",
				"arguments": map[string]interface{}{
					"url":            "https://example.com",
					"headless":       true,
					"playwright":     true,
					"timeoutSeconds": 30,
					"incremental":    true,
					"extract": map[string]interface{}{
						"ai": map[string]interface{}{
							"enabled": true,
							"mode":    "natural_language",
							"prompt":  "Extract the title and price",
							"fields":  []string{"title", "price"},
						},
					},
					"pipeline": map[string]interface{}{
						"preProcessors":  []string{"prep1", "prep2"},
						"postProcessors": []string{"post1"},
						"transformers":   []string{"trans1", "trans2"},
					},
					"auth": map[string]interface{}{
						"proxyHints": map[string]interface{}{
							"preferred_region": "us-east",
							"required_tags":    []string{"residential"},
						},
					},
					"screenshot": map[string]interface{}{
						"enabled":  true,
						"fullPage": true,
						"format":   "png",
					},
					"device": map[string]interface{}{
						"name": "iPhone 15",
					},
					"networkIntercept": map[string]interface{}{
						"enabled":             true,
						"urlPatterns":         []string{"**/api/**"},
						"resourceTypes":       []string{"xhr", "fetch"},
						"captureRequestBody":  true,
						"captureResponseBody": true,
						"maxBodySize":         1024,
						"maxEntries":          10,
					},
				},
			}),
		}

		_, err := srv.handleToolCall(ctx, base)
		if err != nil {
			t.Fatalf("handleToolCall failed: %v", err)
		}

		jobs, err := srv.store.List(ctx)
		if err != nil {
			t.Fatalf("failed to list jobs: %v", err)
		}
		if len(jobs) == 0 {
			t.Fatal("expected a job to be created")
		}

		job := jobs[0]
		pipelineMap, ok := job.SpecMap()["pipeline"].(map[string]interface{})
		if !ok {
			t.Fatal("pipeline params not found or wrong type")
		}
		extractMap, ok := job.SpecMap()["extract"].(map[string]interface{})
		if !ok {
			t.Fatal("extract params not found or wrong type")
		}
		aiMap, ok := extractMap["ai"].(map[string]interface{})
		if !ok {
			t.Fatal("AI extraction params not found or wrong type")
		}
		preProcessors, _ := pipelineMap["preProcessors"].([]interface{})
		postProcessors, _ := pipelineMap["postProcessors"].([]interface{})
		transformers, _ := pipelineMap["transformers"].([]interface{})
		inc, ok := job.SpecMap()["incremental"].(bool)
		if !ok || !inc {
			t.Errorf("incremental: got %v, want true", inc)
		}

		preProcessorsStr := make([]string, len(preProcessors))
		for i, v := range preProcessors {
			preProcessorsStr[i] = v.(string)
		}
		postProcessorsStr := make([]string, len(postProcessors))
		for i, v := range postProcessors {
			postProcessorsStr[i] = v.(string)
		}
		transformersStr := make([]string, len(transformers))
		for i, v := range transformers {
			transformersStr[i] = v.(string)
		}

		if !reflect.DeepEqual(preProcessorsStr, []string{"prep1", "prep2"}) {
			t.Errorf("preProcessors: got %+v, want [prep1 prep2]", preProcessorsStr)
		}
		if !reflect.DeepEqual(postProcessorsStr, []string{"post1"}) {
			t.Errorf("postProcessors: got %+v, want [post1]", postProcessorsStr)
		}
		if !reflect.DeepEqual(transformersStr, []string{"trans1", "trans2"}) {
			t.Errorf("transformers: got %+v, want [trans1 trans2]", transformersStr)
		}
		if enabled, _ := aiMap["enabled"].(bool); !enabled {
			t.Errorf("ai.enabled: got %v, want true", enabled)
		}
		if mode, _ := aiMap["mode"].(string); mode != "natural_language" {
			t.Errorf("ai.mode: got %q, want natural_language", mode)
		}
		if prompt, _ := aiMap["prompt"].(string); prompt != "Extract the title and price" {
			t.Errorf("ai.prompt: got %q", prompt)
		}
		if screenshotMap, ok := job.SpecMap()["screenshot"].(map[string]interface{}); !ok {
			t.Fatal("screenshot config not found or wrong type")
		} else if enabled, _ := screenshotMap["enabled"].(bool); !enabled {
			t.Errorf("screenshot.enabled: got %v, want true", enabled)
		}
		if deviceMap, ok := job.SpecMap()["device"].(map[string]interface{}); !ok {
			t.Fatal("device config not found or wrong type")
		} else if name, _ := deviceMap["name"].(string); name != "iPhone 15" {
			t.Errorf("device.name: got %q, want iPhone 15", name)
		}
		if interceptMap, ok := job.SpecMap()["networkIntercept"].(map[string]interface{}); !ok {
			t.Fatal("networkIntercept config not found or wrong type")
		} else if enabled, _ := interceptMap["enabled"].(bool); !enabled {
			t.Errorf("networkIntercept.enabled: got %v, want true", enabled)
		}
		authMap, ok := job.SpecMap()["auth"].(map[string]interface{})
		if !ok {
			t.Fatal("auth config not found or wrong type")
		}
		proxyHintsMap, ok := authMap["proxyHints"].(map[string]interface{})
		if !ok {
			t.Fatal("auth.proxyHints not found or wrong type")
		}
		if preferredRegion, _ := proxyHintsMap["preferred_region"].(string); preferredRegion != "us-east" {
			t.Errorf("auth.proxyHints.preferred_region: got %q, want us-east", preferredRegion)
		}
	})

	t.Run("crawl_site with partial pipeline options", func(t *testing.T) {
		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name": "crawl_site",
				"arguments": map[string]interface{}{
					"url":      "https://example.com",
					"maxDepth": 2,
					"maxPages": 10,
					"extract": map[string]interface{}{
						"ai": map[string]interface{}{
							"enabled": true,
							"mode":    "schema_guided",
							"schema": map[string]interface{}{
								"title": "Example",
								"price": "$19.99",
							},
							"fields": []string{"title", "price"},
						},
					},
					"pipeline": map[string]interface{}{
						"preProcessors": []string{"only-prep"},
					},
					"incremental": false,
				},
			}),
		}

		_, err := srv.handleToolCall(ctx, base)
		if err != nil {
			t.Fatalf("handleToolCall failed: %v", err)
		}

		jobs, err := srv.store.List(ctx)
		if err != nil {
			t.Fatalf("failed to list jobs: %v", err)
		}
		job := jobs[0]
		pipelineMap, _ := job.SpecMap()["pipeline"].(map[string]interface{})
		extractMap, _ := job.SpecMap()["extract"].(map[string]interface{})
		aiMap, _ := extractMap["ai"].(map[string]interface{})
		preProcessors, _ := pipelineMap["preProcessors"].([]interface{})
		postProcessors, _ := pipelineMap["postProcessors"].([]interface{})
		transformers, _ := pipelineMap["transformers"].([]interface{})
		inc, _ := job.SpecMap()["incremental"].(bool)
		if inc {
			t.Error("incremental: got true, want false")
		}

		preProcessorsStr := make([]string, len(preProcessors))
		for i, v := range preProcessors {
			preProcessorsStr[i] = v.(string)
		}

		if !reflect.DeepEqual(preProcessorsStr, []string{"only-prep"}) {
			t.Errorf("preProcessors: got %+v, want [only-prep]", preProcessorsStr)
		}
		if len(postProcessors) != 0 {
			t.Errorf("postProcessors: got %+v, want empty", postProcessors)
		}
		if len(transformers) != 0 {
			t.Errorf("transformers: got %+v, want empty", transformers)
		}
		if enabled, _ := aiMap["enabled"].(bool); !enabled {
			t.Errorf("ai.enabled: got %v, want true", enabled)
		}
		if mode, _ := aiMap["mode"].(string); mode != "schema_guided" {
			t.Errorf("ai.mode: got %q, want schema_guided", mode)
		}
		schema, _ := aiMap["schema"].(map[string]interface{})
		if title, _ := schema["title"].(string); title != "Example" {
			t.Errorf("ai.schema.title: got %q", title)
		}
	})

	t.Run("crawl_site rejects schema_guided AI without aiSchema", func(t *testing.T) {
		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name": "crawl_site",
				"arguments": map[string]interface{}{
					"url":      "https://example.com",
					"maxDepth": 2,
					"maxPages": 10,
					"extract": map[string]interface{}{
						"ai": map[string]interface{}{
							"enabled": true,
							"mode":    "schema_guided",
						},
					},
				},
			}),
		}

		_, err := srv.handleToolCall(ctx, base)
		if err == nil {
			t.Fatal("expected validation error")
		}
		if err.Error() != "extract.ai.schema is required when extract.ai.mode is schema_guided" {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("research stores AI extraction options", func(t *testing.T) {
		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name": "research",
				"arguments": map[string]interface{}{
					"query": "pricing model",
					"urls":  []string{"https://example.com"},
					"extract": map[string]interface{}{
						"ai": map[string]interface{}{
							"enabled": true,
							"mode":    "natural_language",
							"prompt":  "Extract the pricing model and contract terms",
							"fields":  []string{"pricing_model", "contract_terms"},
						},
					},
					"pipeline": map[string]interface{}{
						"preProcessors": []string{"prep1"},
					},
				},
			}),
		}

		_, err := srv.handleToolCall(ctx, base)
		if err != nil {
			t.Fatalf("handleToolCall failed: %v", err)
		}

		jobs, err := srv.store.List(ctx)
		if err != nil {
			t.Fatalf("failed to list jobs: %v", err)
		}
		job := jobs[0]
		extractMap, _ := job.SpecMap()["extract"].(map[string]interface{})
		aiMap, _ := extractMap["ai"].(map[string]interface{})
		if enabled, _ := aiMap["enabled"].(bool); !enabled {
			t.Errorf("ai.enabled: got %v, want true", enabled)
		}
		if mode, _ := aiMap["mode"].(string); mode != "natural_language" {
			t.Errorf("ai.mode: got %q, want natural_language", mode)
		}
		if prompt, _ := aiMap["prompt"].(string); prompt != "Extract the pricing model and contract terms" {
			t.Errorf("ai.prompt: got %q", prompt)
		}
	})

	t.Run("research stores agentic options", func(t *testing.T) {
		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name": "research",
				"arguments": map[string]interface{}{
					"query": "pricing model",
					"urls":  []string{"https://example.com"},
					"agentic": map[string]interface{}{
						"enabled":         true,
						"instructions":    "Prioritize pricing and support commitments",
						"maxRounds":       2,
						"maxFollowUpUrls": 4,
					},
				},
			}),
		}

		_, err := srv.handleToolCall(ctx, base)
		if err != nil {
			t.Fatalf("handleToolCall failed: %v", err)
		}

		jobs, err := srv.store.List(ctx)
		if err != nil {
			t.Fatalf("failed to list jobs: %v", err)
		}
		job := jobs[0]
		agenticMap, _ := job.SpecMap()["agentic"].(map[string]interface{})
		if enabled, _ := agenticMap["enabled"].(bool); !enabled {
			t.Errorf("agentic.enabled: got %v, want true", enabled)
		}
		if instructions, _ := agenticMap["instructions"].(string); instructions != "Prioritize pricing and support commitments" {
			t.Errorf("agentic.instructions: got %q", instructions)
		}
	})

	t.Run("research rejects schema_guided AI without aiSchema", func(t *testing.T) {
		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name": "research",
				"arguments": map[string]interface{}{
					"query": "pricing model",
					"urls":  []string{"https://example.com"},
					"extract": map[string]interface{}{
						"ai": map[string]interface{}{
							"enabled": true,
							"mode":    "schema_guided",
						},
					},
				},
			}),
		}

		_, err := srv.handleToolCall(ctx, base)
		if err == nil {
			t.Fatal("expected validation error")
		}
		if err.Error() != "extract.ai.schema is required when extract.ai.mode is schema_guided" {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("research with empty pipeline options (default behavior)", func(t *testing.T) {
		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name": "research",
				"arguments": map[string]interface{}{
					"query": "test",
					"urls":  []string{"https://example.com"},
				},
			}),
		}

		_, err := srv.handleToolCall(ctx, base)
		if err != nil {
			t.Fatalf("handleToolCall failed: %v", err)
		}

		jobs, err := srv.store.List(ctx)
		if err != nil {
			t.Fatalf("failed to list jobs: %v", err)
		}
		job := jobs[0]
		pipelineMap, _ := job.SpecMap()["pipeline"].(map[string]interface{})
		preProcessors, _ := pipelineMap["preProcessors"].([]interface{})
		postProcessors, _ := pipelineMap["postProcessors"].([]interface{})
		transformers, _ := pipelineMap["transformers"].([]interface{})
		inc, _ := job.SpecMap()["incremental"].(bool)
		if inc {
			t.Error("incremental: got true, want false")
		}

		if len(preProcessors) != 0 || len(postProcessors) != 0 || len(transformers) != 0 {
			t.Error("expected all pipeline slices to be empty")
		}
	})
}
