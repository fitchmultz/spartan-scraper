// Package mcp provides tests for schema validation and helper functions.
// Tests cover shared MCP argument decoding and tool schema definitions.
// Does NOT test tool execution behavior, server lifecycle, or job management operations.
package mcp

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/paramdecode"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

func TestGetPipelineOptions(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]interface{}
		expected pipeline.Options
	}{
		{
			name:     "nil args returns empty",
			args:     nil,
			expected: pipeline.Options{},
		},
		{
			name: "all fields populated",
			args: map[string]interface{}{
				"preProcessors":  []interface{}{"prep1", "prep2"},
				"postProcessors": []interface{}{"post1"},
				"transformers":   []interface{}{"trans1"},
			},
			expected: pipeline.Options{
				PreProcessors:  []string{"prep1", "prep2"},
				PostProcessors: []string{"post1"},
				Transformers:   []string{"trans1"},
			},
		},
		{
			name: "partial fields",
			args: map[string]interface{}{
				"preProcessors": []interface{}{"only-prep"},
			},
			expected: pipeline.Options{
				PreProcessors: []string{"only-prep"},
			},
		},
		{
			name:     "missing keys return empty slices",
			args:     map[string]interface{}{},
			expected: pipeline.Options{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPipelineOptions(tt.args)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("got %+v, want %+v", result, tt.expected)
			}
		})
	}
}

func TestMCPArgumentDecodingMatchesSharedSemantics(t *testing.T) {
	args := map[string]interface{}{
		"playwright":     false,
		"timeoutSeconds": 45.0,
		"urls":           []interface{}{"https://example.com", "https://example.com/docs"},
	}

	if got := paramdecode.BoolDefault(args, "playwright", true); got {
		t.Fatalf("expected explicit false to win over fallback")
	}
	if got := paramdecode.PositiveInt(args, "timeoutSeconds", 5); got != 45 {
		t.Fatalf("expected timeout 45, got %d", got)
	}
	if got := paramdecode.StringSlice(args, "urls"); !reflect.DeepEqual(got, []string{"https://example.com", "https://example.com/docs"}) {
		t.Fatalf("unexpected urls: %#v", got)
	}
}

func TestToolSchemas(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	tools := srv.toolsList()
	toolMap := make(map[string]tool)
	for _, t := range tools {
		toolMap[t.Name] = t
	}

	t.Run("all tools have required schema fields", func(t *testing.T) {
		for name, tool := range toolMap {
			schema := tool.InputSchema
			if schema == nil {
				t.Errorf("tool %s has nil schema", name)
				continue
			}

			if tool.Name == "" {
				t.Errorf("tool has empty name")
			}
			if tool.Description == "" {
				t.Errorf("tool %s has empty description", name)
			}
		}
	})
}

func TestSchemaDeterministicOutput(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	// Collect JSON outputs from multiple calls
	outputs := make([]string, 10)
	for i := 0; i < len(outputs); i++ {
		tools := srv.toolsList()
		data, err := json.Marshal(tools)
		if err != nil {
			t.Fatalf("failed to marshal tools: %v", err)
		}
		outputs[i] = string(data)
	}

	// All outputs must be identical
	for i := 1; i < len(outputs); i++ {
		if outputs[i] != outputs[0] {
			t.Errorf("output %d differs from first output\nfirst:\n%s\n%d:\n%s",
				i, outputs[0], i, outputs[i])
		}
	}
}
