// Package pipeline provides tests for the plugin registry.
// Tests cover plugin registration, stage-based filtering, priority ordering, and the stageInList helper.
// Does NOT test concurrent access or plugin lifecycle management.
package pipeline

import (
	"testing"
)

type mockPlugin struct {
	name     string
	priority int
	stages   []Stage
	enabled  bool
}

func (m *mockPlugin) Name() string                                         { return m.name }
func (m *mockPlugin) Priority() int                                        { return m.priority }
func (m *mockPlugin) Stages() []Stage                                      { return m.stages }
func (m *mockPlugin) Enabled(Target, Options) bool                         { return m.enabled }
func (m *mockPlugin) PreFetch(HookContext, FetchInput) (FetchInput, error) { return FetchInput{}, nil }
func (m *mockPlugin) PostFetch(HookContext, FetchInput, FetchOutput) (FetchOutput, error) {
	return FetchOutput{}, nil
}
func (m *mockPlugin) PreExtract(HookContext, ExtractInput) (ExtractInput, error) {
	return ExtractInput{}, nil
}
func (m *mockPlugin) PostExtract(HookContext, ExtractInput, ExtractOutput) (ExtractOutput, error) {
	return ExtractOutput{}, nil
}
func (m *mockPlugin) PreOutput(HookContext, OutputInput) (OutputInput, error) {
	return OutputInput{}, nil
}
func (m *mockPlugin) PostOutput(HookContext, OutputInput, OutputOutput) (OutputOutput, error) {
	return OutputOutput{}, nil
}

func TestRegistryPluginsFor(t *testing.T) {
	r := NewRegistry()
	p1 := &mockPlugin{name: "p1", priority: 10, stages: []Stage{StagePreFetch}, enabled: true}
	p2 := &mockPlugin{name: "p2", priority: 5, stages: []Stage{StagePreFetch}, enabled: true}
	p3 := &mockPlugin{name: "p3", priority: 5, stages: []Stage{StagePostFetch}, enabled: true}

	r.Register(p1)
	r.Register(p2)
	r.Register(p3)

	target := Target{}
	opts := Options{}

	plugins := r.PluginsFor(StagePreFetch, target, opts)
	if len(plugins) != 2 {
		t.Errorf("expected 2 plugins for StagePreFetch, got %d", len(plugins))
	}

	// Order should be p2 then p1 (priority 5 < 10)
	if plugins[0].Name() != "p2" || plugins[1].Name() != "p1" {
		t.Errorf("wrong plugin order: %s, %s", plugins[0].Name(), plugins[1].Name())
	}
}

func TestStageInList(t *testing.T) {
	stages := []Stage{StagePreFetch, StagePostFetch}
	if !stageInList(StagePreFetch, stages) {
		t.Error("StagePreFetch should be in list")
	}
	if stageInList(StagePreExtract, stages) {
		t.Error("StagePreExtract should NOT be in list")
	}
}
