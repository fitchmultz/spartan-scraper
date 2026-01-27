package fetch

import (
	"testing"
)

func TestDefaultRenderProfile(t *testing.T) {
	prof := defaultRenderProfile()

	if prof.Wait.Mode != RenderWaitModeDOMReady {
		t.Errorf("expected WaitModeDOMReady, got %v", prof.Wait.Mode)
	}

	if prof.Timeouts.MaxRenderMs != 30000 {
		t.Errorf("expected MaxRenderMs 30000, got %d", prof.Timeouts.MaxRenderMs)
	}

	if prof.Timeouts.ScriptEvalMs != 5000 {
		t.Errorf("expected ScriptEvalMs 5000, got %d", prof.Timeouts.ScriptEvalMs)
	}

	if prof.Timeouts.NavigationMs != 30000 {
		t.Errorf("expected NavigationMs 30000, got %d", prof.Timeouts.NavigationMs)
	}
}

func TestMergeRenderProfile(t *testing.T) {
	base := defaultRenderProfile()

	tests := []struct {
		name     string
		base     RenderProfile
		override *RenderProfile
		validate func(t *testing.T, result RenderProfile)
	}{
		{
			name:     "override name is applied",
			base:     base,
			override: &RenderProfile{Name: "override"},
			validate: func(t *testing.T, result RenderProfile) {
				if result.Name != "override" {
					t.Errorf("expected name 'override', got %s", result.Name)
				}
			},
		},
		{
			name:     "override host patterns is applied",
			base:     base,
			override: &RenderProfile{HostPatterns: []string{"*.example.com"}},
			validate: func(t *testing.T, result RenderProfile) {
				if len(result.HostPatterns) != 1 {
					t.Errorf("expected 1 host pattern, got %d", len(result.HostPatterns))
				}
				if result.HostPatterns[0] != "*.example.com" {
					t.Errorf("expected '*.example.com', got %s", result.HostPatterns[0])
				}
			},
		},
		{
			name:     "force engine is applied",
			base:     base,
			override: &RenderProfile{ForceEngine: RenderEnginePlaywright},
			validate: func(t *testing.T, result RenderProfile) {
				if result.ForceEngine != RenderEnginePlaywright {
					t.Errorf("expected ForceEngine playwright, got %s", result.ForceEngine)
				}
			},
		},
		{
			name:     "prefer headless is applied",
			base:     base,
			override: &RenderProfile{PreferHeadless: true},
			validate: func(t *testing.T, result RenderProfile) {
				if !result.PreferHeadless {
					t.Errorf("expected PreferHeadless true, got false")
				}
			},
		},
		{
			name:     "assume JS heavy is applied",
			base:     base,
			override: &RenderProfile{AssumeJSHeavy: true},
			validate: func(t *testing.T, result RenderProfile) {
				if !result.AssumeJSHeavy {
					t.Errorf("expected AssumeJSHeavy true, got false")
				}
			},
		},
		{
			name:     "never headless is applied",
			base:     base,
			override: &RenderProfile{NeverHeadless: true},
			validate: func(t *testing.T, result RenderProfile) {
				if !result.NeverHeadless {
					t.Errorf("expected NeverHeadless true, got false")
				}
			},
		},
		{
			name:     "JS heavy threshold is applied",
			base:     base,
			override: &RenderProfile{JSHeavyThreshold: 0.7},
			validate: func(t *testing.T, result RenderProfile) {
				if result.JSHeavyThreshold != 0.7 {
					t.Errorf("expected JSHeavyThreshold 0.7, got %f", result.JSHeavyThreshold)
				}
			},
		},
		{
			name:     "block policy is applied",
			base:     base,
			override: &RenderProfile{Block: RenderBlockPolicy{ResourceTypes: []BlockedResourceType{BlockedResourceImage}}},
			validate: func(t *testing.T, result RenderProfile) {
				if len(result.Block.ResourceTypes) != 1 {
					t.Errorf("expected 1 blocked resource type, got %d", len(result.Block.ResourceTypes))
				}
			},
		},
		{
			name:     "wait policy is applied",
			base:     base,
			override: &RenderProfile{Wait: RenderWaitPolicy{Mode: RenderWaitModeNetworkIdle}},
			validate: func(t *testing.T, result RenderProfile) {
				if result.Wait.Mode != RenderWaitModeNetworkIdle {
					t.Errorf("expected WaitModeNetworkIdle, got %v", result.Wait.Mode)
				}
			},
		},
		{
			name:     "timeout policy is applied",
			base:     base,
			override: &RenderProfile{Timeouts: RenderTimeoutPolicy{MaxRenderMs: 60000}},
			validate: func(t *testing.T, result RenderProfile) {
				if result.Timeouts.MaxRenderMs != 60000 {
					t.Errorf("expected MaxRenderMs 60000, got %d", result.Timeouts.MaxRenderMs)
				}
			},
		},
		{
			name:     "base values preserved when override empty",
			base:     base,
			override: &RenderProfile{Name: "test"},
			validate: func(t *testing.T, result RenderProfile) {
				if result.Wait.Mode != RenderWaitModeDOMReady {
					t.Errorf("expected base WaitMode to be preserved, got %v", result.Wait.Mode)
				}
				if result.Timeouts.MaxRenderMs != 30000 {
					t.Errorf("expected base MaxRenderMs to be preserved, got %d", result.Timeouts.MaxRenderMs)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeRenderProfile(tt.base, tt.override)
			tt.validate(t, result)
		})
	}
}

func TestAdaptiveFetcher_New(t *testing.T) {
	f := NewAdaptiveFetcher()
	if f == nil {
		t.Fatal("NewAdaptiveFetcher() returned nil")
	}
	if f.http == nil {
		t.Error("expected http fetcher to be initialized")
	}
	if f.cdp == nil {
		t.Error("expected chromedp fetcher to be initialized")
	}
	if f.pw == nil {
		t.Error("expected playwright fetcher to be initialized")
	}
}

func TestAdaptiveFetcher_Close(t *testing.T) {
	f := NewAdaptiveFetcher()
	err := f.Close()
	if err != nil {
		t.Errorf("Close() unexpected error: %v", err)
	}
}
