package pipeline

import (
	"testing"
)

func TestBaseTransformer_Priority(t *testing.T) {
	bt := BaseTransformer{}
	if priority := bt.Priority(); priority != 0 {
		t.Errorf("expected priority 0, got %d", priority)
	}
}

func TestBaseTransformer_Enabled(t *testing.T) {
	bt := BaseTransformer{}
	tests := []struct {
		name   string
		target Target
		opts   Options
		want   bool
	}{
		{
			name:   "empty target and options",
			target: Target{},
			opts:   Options{},
			want:   true,
		},
		{
			name:   "populated target and options",
			target: Target{URL: "https://example.com", Kind: "scrape"},
			opts:   Options{PreProcessors: []string{"plugin1"}},
			want:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if enabled := bt.Enabled(tt.target, tt.opts); enabled != tt.want {
				t.Errorf("expected %v, got %v", tt.want, enabled)
			}
		})
	}
}

func TestBaseTransformer_Transform(t *testing.T) {
	bt := BaseTransformer{}
	tests := []struct {
		name string
		in   OutputInput
	}{
		{
			name: "with non-nil Raw and Structured",
			in: OutputInput{
				Raw:        []byte("test data"),
				Structured: map[string]any{"key": "value"},
			},
		},
		{
			name: "with nil Structured",
			in: OutputInput{
				Raw:        []byte("test data"),
				Structured: nil,
			},
		},
		{
			name: "with empty Raw and nil Structured",
			in: OutputInput{
				Raw:        []byte{},
				Structured: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := HookContext{}
			out, err := bt.Transform(ctx, tt.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(out.Raw) != string(tt.in.Raw) {
				t.Errorf("expected Raw %v, got %v", tt.in.Raw, out.Raw)
			}
			if tt.in.Structured == nil {
				if out.Structured != nil {
					t.Errorf("expected nil Structured, got %v", out.Structured)
				}
			} else if m, ok := tt.in.Structured.(map[string]any); ok {
				if outMap, ok := out.Structured.(map[string]any); ok {
					if len(outMap) != len(m) {
						t.Errorf("expected Structured with %d keys, got %d", len(m), len(outMap))
					}
					for k, v := range m {
						if outMap[k] != v {
							t.Errorf("expected Structured[%s] = %v, got %v", k, v, outMap[k])
						}
					}
				} else {
					t.Errorf("expected map[string]any Structured, got %T", out.Structured)
				}
			}
		})
	}
}
