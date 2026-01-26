// Tests for schema validation and helper functions.
// Tests verify the getPipelineOptions helper function and ensure all tools
// have valid schema definitions with required fields (name, description, inputSchema).
//
// Does NOT handle:
// - Tool execution behavior
// - Server lifecycle
// - Job management operations
//
// Invariants:
// - getPipelineOptions must handle nil args by returning empty pipeline.Options
// - getPipelineOptions must extract preProcessors, postProcessors, transformers from args
// - All tools must have non-empty name, description, and inputSchema fields
// - Schema helper (schema()) must create valid JSON Schema with type, properties, required
package mcp

import (
	"os"
	"reflect"
	"testing"

	"spartan-scraper/internal/pipeline"
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
