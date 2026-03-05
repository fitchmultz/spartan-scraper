// Package pipeline provides tests for the pipeline plugin interfaces.
// Tests cover BasePlugin default implementations for all hook stages (PreFetch, PostFetch, PreExtract, PostExtract, PreOutput, PostOutput),
// plugin stages, priority, and enabled behavior.
// Does NOT test actual plugin implementations or hook execution order.
package pipeline

import (
	"testing"
)

func TestBasePlugin_Stages(t *testing.T) {
	bp := BasePlugin{}
	stages := bp.Stages()
	all := AllStages()
	if len(stages) != len(all) {
		t.Errorf("expected %d stages, got %d", len(all), len(stages))
	}
	for i, stage := range all {
		if stages[i] != stage {
			t.Errorf("expected stage %d to be %s, got %s", i, stage, stages[i])
		}
	}
}

func TestBasePlugin_Priority(t *testing.T) {
	bp := BasePlugin{}
	if priority := bp.Priority(); priority != 0 {
		t.Errorf("expected priority 0, got %d", priority)
	}
}

func TestBasePlugin_Enabled(t *testing.T) {
	bp := BasePlugin{}
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
			if enabled := bp.Enabled(tt.target, tt.opts); enabled != tt.want {
				t.Errorf("expected %v, got %v", tt.want, enabled)
			}
		})
	}
}

func TestBasePlugin_PreFetch(t *testing.T) {
	bp := BasePlugin{}
	in := FetchInput{
		Target: Target{URL: "https://example.com"},
	}
	ctx := HookContext{}
	out, err := bp.PreFetch(ctx, in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Target.URL != in.Target.URL {
		t.Errorf("expected URL %s, got %s", in.Target.URL, out.Target.URL)
	}
	_ = out
}

func TestBasePlugin_PostFetch(t *testing.T) {
	bp := BasePlugin{}
	in := FetchInput{Target: Target{URL: "https://example.com"}}
	outInput := FetchOutput{}
	ctx := HookContext{}
	_, err := bp.PostFetch(ctx, in, outInput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBasePlugin_PreExtract(t *testing.T) {
	bp := BasePlugin{}
	in := ExtractInput{
		Target: Target{URL: "https://example.com"},
		HTML:   "<html><body>test</body></html>",
	}
	ctx := HookContext{}
	out, err := bp.PreExtract(ctx, in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.HTML != in.HTML {
		t.Errorf("expected HTML %s, got %s", in.HTML, out.HTML)
	}
}

func TestBasePlugin_PostExtract(t *testing.T) {
	bp := BasePlugin{}
	in := ExtractInput{Target: Target{URL: "https://example.com"}}
	outInput := ExtractOutput{}
	ctx := HookContext{}
	_, err := bp.PostExtract(ctx, in, outInput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBasePlugin_PreOutput(t *testing.T) {
	bp := BasePlugin{}
	in := OutputInput{
		Raw: []byte("test data"),
	}
	ctx := HookContext{}
	out, err := bp.PreOutput(ctx, in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out.Raw) != string(in.Raw) {
		t.Errorf("expected Raw %v, got %v", in.Raw, out.Raw)
	}
}

func TestBasePlugin_PostOutput(t *testing.T) {
	bp := BasePlugin{}
	in := OutputInput{Raw: []byte("test data")}
	outInput := OutputOutput{Raw: []byte("test data")}
	ctx := HookContext{}
	_, err := bp.PostOutput(ctx, in, outInput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
