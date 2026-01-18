package pipeline

import "sort"

type pluginEntry struct {
	plugin Plugin
	order  int
}

type transformerEntry struct {
	transformer Transformer
	order       int
}

type Registry struct {
	plugins      []pluginEntry
	transformers []transformerEntry
}

func NewRegistry() *Registry {
	return &Registry{
		plugins:      []pluginEntry{},
		transformers: []transformerEntry{},
	}
}

func (r *Registry) Register(plugin Plugin) {
	r.plugins = append(r.plugins, pluginEntry{
		plugin: plugin,
		order:  len(r.plugins),
	})
}

func (r *Registry) RegisterTransformer(transformer Transformer) {
	r.transformers = append(r.transformers, transformerEntry{
		transformer: transformer,
		order:       len(r.transformers),
	})
}

func (r *Registry) PluginsFor(stage Stage, target Target, opts Options) []Plugin {
	return r.pluginsFor(stage, target, opts)
}

func (r *Registry) RunPreFetch(ctx HookContext, in FetchInput) (FetchInput, error) {
	ctx.Stage = StagePreFetch
	for _, plugin := range r.pluginsFor(StagePreFetch, ctx.Target, ctx.Options) {
		var err error
		in, err = plugin.PreFetch(ctx, in)
		if err != nil {
			return FetchInput{}, err
		}
	}
	return in, nil
}

func (r *Registry) RunPostFetch(ctx HookContext, in FetchInput, out FetchOutput) (FetchOutput, error) {
	ctx.Stage = StagePostFetch
	for _, plugin := range r.pluginsFor(StagePostFetch, ctx.Target, ctx.Options) {
		var err error
		out, err = plugin.PostFetch(ctx, in, out)
		if err != nil {
			return FetchOutput{}, err
		}
	}
	return out, nil
}

func (r *Registry) RunPreExtract(ctx HookContext, in ExtractInput) (ExtractInput, error) {
	ctx.Stage = StagePreExtract
	for _, plugin := range r.pluginsFor(StagePreExtract, ctx.Target, ctx.Options) {
		var err error
		in, err = plugin.PreExtract(ctx, in)
		if err != nil {
			return ExtractInput{}, err
		}
	}
	return in, nil
}

func (r *Registry) RunPostExtract(ctx HookContext, in ExtractInput, out ExtractOutput) (ExtractOutput, error) {
	ctx.Stage = StagePostExtract
	for _, plugin := range r.pluginsFor(StagePostExtract, ctx.Target, ctx.Options) {
		var err error
		out, err = plugin.PostExtract(ctx, in, out)
		if err != nil {
			return ExtractOutput{}, err
		}
	}
	return out, nil
}

func (r *Registry) RunPreOutput(ctx HookContext, in OutputInput) (OutputInput, error) {
	ctx.Stage = StagePreOutput
	for _, plugin := range r.pluginsFor(StagePreOutput, ctx.Target, ctx.Options) {
		var err error
		in, err = plugin.PreOutput(ctx, in)
		if err != nil {
			return OutputInput{}, err
		}
	}
	return in, nil
}

func (r *Registry) RunPostOutput(ctx HookContext, in OutputInput, out OutputOutput) (OutputOutput, error) {
	ctx.Stage = StagePostOutput
	for _, plugin := range r.pluginsFor(StagePostOutput, ctx.Target, ctx.Options) {
		var err error
		out, err = plugin.PostOutput(ctx, in, out)
		if err != nil {
			return OutputOutput{}, err
		}
	}
	return out, nil
}

func (r *Registry) RunTransformers(ctx HookContext, in OutputInput) (OutputOutput, error) {
	ctx.Stage = StagePreOutput
	out := OutputOutput{
		Raw:        in.Raw,
		Structured: in.Structured,
	}
	for _, transformer := range r.transformersFor(ctx.Target, ctx.Options) {
		current := in
		current.Raw = out.Raw
		current.Structured = out.Structured
		var err error
		out, err = transformer.Transform(ctx, current)
		if err != nil {
			return OutputOutput{}, err
		}
	}
	return out, nil
}

func (r *Registry) pluginsFor(stage Stage, target Target, opts Options) []Plugin {
	entries := make([]pluginEntry, 0, len(r.plugins))
	for _, entry := range r.plugins {
		stages := entry.plugin.Stages()
		if len(stages) == 0 {
			stages = AllStages()
		}
		if !stageInList(stage, stages) {
			continue
		}
		if !entry.plugin.Enabled(target, opts) {
			continue
		}
		if !allowPluginForStage(entry.plugin.Name(), stage, opts) {
			continue
		}
		entries = append(entries, entry)
	}
	sort.SliceStable(entries, func(i, j int) bool {
		pi := entries[i].plugin.Priority()
		pj := entries[j].plugin.Priority()
		if pi == pj {
			return entries[i].order < entries[j].order
		}
		return pi < pj
	})
	out := make([]Plugin, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry.plugin)
	}
	return out
}

func (r *Registry) transformersFor(target Target, opts Options) []Transformer {
	entries := make([]transformerEntry, 0, len(r.transformers))
	for _, entry := range r.transformers {
		if !entry.transformer.Enabled(target, opts) {
			continue
		}
		if len(opts.Transformers) > 0 && !stringInList(entry.transformer.Name(), opts.Transformers) {
			continue
		}
		entries = append(entries, entry)
	}
	sort.SliceStable(entries, func(i, j int) bool {
		pi := entries[i].transformer.Priority()
		pj := entries[j].transformer.Priority()
		if pi == pj {
			return entries[i].order < entries[j].order
		}
		return pi < pj
	})
	out := make([]Transformer, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry.transformer)
	}
	return out
}

func stageInList(stage Stage, stages []Stage) bool {
	for _, s := range stages {
		if s == stage {
			return true
		}
	}
	return false
}

func allowPluginForStage(name string, stage Stage, opts Options) bool {
	if isPreStage(stage) {
		return len(opts.PreProcessors) == 0 || stringInList(name, opts.PreProcessors)
	}
	if isPostStage(stage) {
		return len(opts.PostProcessors) == 0 || stringInList(name, opts.PostProcessors)
	}
	return true
}

func isPreStage(stage Stage) bool {
	return stage == StagePreFetch || stage == StagePreExtract || stage == StagePreOutput
}

func isPostStage(stage Stage) bool {
	return stage == StagePostFetch || stage == StagePostExtract || stage == StagePostOutput
}

func stringInList(value string, items []string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}
