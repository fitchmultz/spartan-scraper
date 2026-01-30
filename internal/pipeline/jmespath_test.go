// Package pipeline provides tests for JMESPath transformer.
// Tests cover expression compilation, transformation, and error handling.
// Does NOT test the full pipeline integration.
package pipeline

import (
	"testing"
)

func TestNewJMESPathTransformer(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		wantErr    bool
	}{
		{
			name:       "valid expression",
			expression: "[].{url: url, title: title}",
			wantErr:    false,
		},
		{
			name:       "valid filter expression",
			expression: "[?status=='success']",
			wantErr:    false,
		},
		{
			name:       "invalid expression",
			expression: "[?invalid",
			wantErr:    true,
		},
		{
			name:       "empty expression",
			expression: "",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transformer, err := NewJMESPathTransformer(WithJMESPathExpression(tt.expression))
			if (err != nil) != tt.wantErr {
				t.Errorf("NewJMESPathTransformer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if transformer.Name() != "jmespath" {
				t.Errorf("Expected name 'jmespath', got %s", transformer.Name())
			}
			if transformer.Priority() != 50 {
				t.Errorf("Expected priority 50, got %d", transformer.Priority())
			}
		})
	}
}

func TestJMESPathTransformer_Enabled(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		opts       Options
		want       bool
	}{
		{
			name:       "enabled with instance expression",
			expression: "[].url",
			opts:       Options{},
			want:       true,
		},
		{
			name:       "enabled with options expression",
			expression: "",
			opts:       Options{JMESPath: "[].url"},
			want:       true,
		},
		{
			name:       "disabled when no expression",
			expression: "",
			opts:       Options{},
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transformer, err := NewJMESPathTransformer(WithJMESPathExpression(tt.expression))
			if err != nil {
				t.Fatalf("Failed to create transformer: %v", err)
			}

			if got := transformer.Enabled(Target{}, tt.opts); got != tt.want {
				t.Errorf("Enabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJMESPathTransformer_Transform(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		input      any
		wantErr    bool
		checkFn    func(t *testing.T, result any)
	}{
		{
			name:       "nil structured data",
			expression: "[].url",
			input:      nil,
			wantErr:    false,
			checkFn: func(t *testing.T, result any) {
				if result != nil {
					t.Errorf("Expected nil, got %v", result)
				}
			},
		},
		{
			name:       "project fields",
			expression: "[].{url: url, title: title}",
			input: []any{
				map[string]any{"url": "http://example.com", "title": "Example", "extra": "ignored"},
				map[string]any{"url": "http://test.com", "title": "Test", "extra": "ignored"},
			},
			wantErr: false,
			checkFn: func(t *testing.T, result any) {
				results, ok := result.([]any)
				if !ok {
					t.Fatalf("Expected []any, got %T", result)
				}
				if len(results) != 2 {
					t.Fatalf("Expected 2 results, got %d", len(results))
				}
				first, ok := results[0].(map[string]any)
				if !ok {
					t.Fatalf("Expected map[string]any, got %T", results[0])
				}
				if first["url"] != "http://example.com" {
					t.Errorf("Expected url 'http://example.com', got %v", first["url"])
				}
				if first["title"] != "Example" {
					t.Errorf("Expected title 'Example', got %v", first["title"])
				}
				if _, exists := first["extra"]; exists {
					t.Error("Expected 'extra' field to be filtered out")
				}
			},
		},
		{
			name:       "filter by status",
			expression: "[?status=='success']",
			input: []any{
				map[string]any{"url": "http://example.com", "status": "success"},
				map[string]any{"url": "http://test.com", "status": "error"},
				map[string]any{"url": "http://other.com", "status": "success"},
			},
			wantErr: false,
			checkFn: func(t *testing.T, result any) {
				results, ok := result.([]any)
				if !ok {
					t.Fatalf("Expected []any, got %T", result)
				}
				if len(results) != 2 {
					t.Fatalf("Expected 2 results, got %d", len(results))
				}
			},
		},
		{
			name:       "slice operator",
			expression: "[0:2]",
			input: []any{
				map[string]any{"id": 1},
				map[string]any{"id": 2},
				map[string]any{"id": 3},
				map[string]any{"id": 4},
			},
			wantErr: false,
			checkFn: func(t *testing.T, result any) {
				results, ok := result.([]any)
				if !ok {
					t.Fatalf("Expected []any, got %T", result)
				}
				if len(results) != 2 {
					t.Fatalf("Expected 2 results, got %d", len(results))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transformer, err := NewJMESPathTransformer(WithJMESPathExpression(tt.expression))
			if err != nil {
				t.Fatalf("Failed to create transformer: %v", err)
			}

			in := OutputInput{
				Structured: tt.input,
			}

			out, err := transformer.Transform(HookContext{}, in)
			if (err != nil) != tt.wantErr {
				t.Errorf("Transform() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.checkFn != nil {
				tt.checkFn(t, out.Structured)
			}
		})
	}
}

func TestCompileJMESPath(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		wantErr    bool
	}{
		{
			name:       "valid expression",
			expression: "[].url",
			wantErr:    false,
		},
		{
			name:       "invalid expression",
			expression: "[?invalid",
			wantErr:    true,
		},
		{
			name:       "empty expression",
			expression: "",
			wantErr:    true, // JMESPath library returns error for empty expression
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CompileJMESPath(tt.expression)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompileJMESPath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestApplyJMESPath(t *testing.T) {
	data := []any{
		map[string]any{"url": "http://example.com", "title": "Example"},
		map[string]any{"url": "http://test.com", "title": "Test"},
	}

	tests := []struct {
		name       string
		expression string
		data       any
		wantErr    bool
		checkFn    func(t *testing.T, result any)
	}{
		{
			name:       "project urls",
			expression: "[].url",
			data:       data,
			wantErr:    false,
			checkFn: func(t *testing.T, result any) {
				results, ok := result.([]any)
				if !ok {
					t.Fatalf("Expected []any, got %T", result)
				}
				if len(results) != 2 {
					t.Fatalf("Expected 2 results, got %d", len(results))
				}
				if results[0] != "http://example.com" {
					t.Errorf("Expected 'http://example.com', got %v", results[0])
				}
			},
		},
		{
			name:       "invalid expression",
			expression: "[?invalid",
			data:       data,
			wantErr:    true,
		},
		{
			name:       "count items",
			expression: "length(@)",
			data:       data,
			wantErr:    false,
			checkFn: func(t *testing.T, result any) {
				// JMESPath returns float64 for numbers
				count, ok := result.(float64)
				if !ok {
					t.Fatalf("Expected float64, got %T", result)
				}
				if count != 2 {
					t.Errorf("Expected count 2, got %v", count)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ApplyJMESPath(tt.expression, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ApplyJMESPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.checkFn != nil {
				tt.checkFn(t, result)
			}
		})
	}
}
