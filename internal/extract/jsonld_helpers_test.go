// Package extract provides tests for JSON-LD helper functions.
// Tests cover getPath navigation (simple, nested, through arrays) and extractStrings type conversion.
// Does NOT test JSON-LD extraction or matching logic.
package extract

import (
	"testing"
)

func TestGetPathSimple(t *testing.T) {
	tests := []struct {
		name     string
		obj      map[string]any
		path     string
		expected any
	}{
		{
			name:     "single part path",
			obj:      map[string]any{"name": "John Doe"},
			path:     "name",
			expected: "John Doe",
		},
		{
			name:     "two part path",
			obj:      map[string]any{"author": map[string]any{"name": "Jane Doe"}},
			path:     "author.name",
			expected: "Jane Doe",
		},
		{
			name:     "three part path",
			obj:      map[string]any{"data": map[string]any{"nested": map[string]any{"value": "deep"}}},
			path:     "data.nested.value",
			expected: "deep",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPath(tt.obj, tt.path)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGetPathThroughArray(t *testing.T) {
	tests := []struct {
		name     string
		obj      map[string]any
		path     string
		expected any
	}{
		{
			name: "path traversing array",
			obj: map[string]any{
				"authors": []any{
					map[string]any{"name": "Author 1"},
					map[string]any{"name": "Author 2"},
					map[string]any{"name": "Author 3"},
				},
			},
			path:     "authors.name",
			expected: []any{"Author 1", "Author 2", "Author 3"},
		},
		{
			name: "array part not found",
			obj: map[string]any{
				"authors": []any{
					map[string]any{"id": "1"},
					map[string]any{"id": "2"},
				},
			},
			path:     "authors.name",
			expected: nil,
		},
		{
			name: "nested array path",
			obj: map[string]any{
				"articles": []any{
					map[string]any{"author": map[string]any{"name": "John"}},
					map[string]any{"author": map[string]any{"name": "Jane"}},
				},
			},
			path:     "articles.author.name",
			expected: []any{"John", "Jane"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPath(tt.obj, tt.path)

			if result == nil {
				if tt.expected != nil {
					t.Errorf("expected %v, got nil", tt.expected)
				}
				return
			}

			resultSlice, ok := result.([]any)
			if !ok {
				t.Errorf("expected result to be slice, got %T", result)
				return
			}

			expectedSlice, ok := tt.expected.([]any)
			if !ok {
				t.Errorf("expected expected to be slice, got %T", tt.expected)
				return
			}

			if len(resultSlice) != len(expectedSlice) {
				t.Fatalf("expected %d values, got %d", len(expectedSlice), len(resultSlice))
			}

			for i, val := range resultSlice {
				if val != expectedSlice[i] {
					t.Errorf("value %d: expected %v, got %v", i, expectedSlice[i], val)
				}
			}
		})
	}
}

func TestGetPathNotFound(t *testing.T) {
	tests := []struct {
		name string
		obj  map[string]any
		path string
	}{
		{
			name: "key not found",
			obj:  map[string]any{"name": "John"},
			path: "age",
		},
		{
			name: "intermediate key not found",
			obj:  map[string]any{"author": map[string]any{"name": "John"}},
			path: "author.age",
		},
		{
			name: "intermediate nil value",
			obj:  map[string]any{"author": nil},
			path: "author.name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPath(tt.obj, tt.path)

			if result != nil {
				t.Errorf("expected nil, got %v", result)
			}
		})
	}
}

func TestGetPathEmptyPath(t *testing.T) {
	obj := map[string]any{
		"name": "John Doe",
		"age":  30,
	}

	result := getPath(obj, "")

	if result != nil {
		t.Errorf("expected nil for empty path, got %v", result)
	}
}

func TestExtractStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected []string
	}{
		{
			name:     "string value",
			input:    "hello",
			expected: []string{"hello"},
		},
		{
			name:     "float64 value",
			input:    123.45,
			expected: []string{"123.45"},
		},
		{
			name:     "int value",
			input:    42,
			expected: []string{"42"},
		},
		{
			name:     "bool value true",
			input:    true,
			expected: []string{"true"},
		},
		{
			name:     "bool value false",
			input:    false,
			expected: []string{"false"},
		},
		{
			name:     "array of strings",
			input:    []any{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "nested array of strings",
			input:    []any{[]any{"a", "b"}, []any{"c", "d"}},
			expected: []string{"a", "b", "c", "d"},
		},
		{
			name:     "mixed nested array",
			input:    []any{"text", 123, true, []any{"nested"}},
			expected: []string{"text", "123", "true", "nested"},
		},
		{
			name:     "empty array",
			input:    []any{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result []string
			extractStrings(tt.input, &result)

			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d strings, got %d", len(tt.expected), len(result))
			}

			for i, str := range result {
				if str != tt.expected[i] {
					t.Errorf("string %d: expected %q, got %q", i, tt.expected[i], str)
				}
			}
		})
	}
}
