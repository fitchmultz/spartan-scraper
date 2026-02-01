// Package plugins provides a WASM-based plugin system for third-party extensions.
package plugins

import (
	"context"
	"fmt"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

// WASMPlugin wraps a WASM module to implement pipeline.Plugin.
type WASMPlugin struct {
	manifest  *PluginManifest
	runtime   *WASMRuntime
	pluginDir string
	instance  *WASMInstance
}

// NewWASMPlugin creates a new WASM plugin wrapper.
func NewWASMPlugin(manifest *PluginManifest, runtime *WASMRuntime, pluginDir string) *WASMPlugin {
	return &WASMPlugin{
		manifest:  manifest,
		runtime:   runtime,
		pluginDir: pluginDir,
	}
}

// Name returns the plugin name.
func (p *WASMPlugin) Name() string {
	return p.manifest.Name
}

// Stages returns the pipeline stages this plugin hooks into.
func (p *WASMPlugin) Stages() []pipeline.Stage {
	var stages []pipeline.Stage
	for _, hook := range p.manifest.Hooks {
		switch hook {
		case "pre_fetch":
			stages = append(stages, pipeline.StagePreFetch)
		case "post_fetch":
			stages = append(stages, pipeline.StagePostFetch)
		case "pre_extract":
			stages = append(stages, pipeline.StagePreExtract)
		case "post_extract":
			stages = append(stages, pipeline.StagePostExtract)
		case "pre_output":
			stages = append(stages, pipeline.StagePreOutput)
		case "post_output":
			stages = append(stages, pipeline.StagePostOutput)
		}
	}
	return stages
}

// Priority returns the plugin execution priority.
func (p *WASMPlugin) Priority() int {
	return p.manifest.Priority
}

// Enabled returns whether the plugin is enabled for the given target and options.
func (p *WASMPlugin) Enabled(target pipeline.Target, opts pipeline.Options) bool {
	return p.manifest.Enabled
}

// initInstance initializes the WASM instance if needed.
func (p *WASMPlugin) initInstance(ctx context.Context) error {
	if p.instance != nil {
		return nil
	}

	wasmPath := p.manifest.GetWASMPath(p.pluginDir)

	// Load and compile the module
	compiled, err := p.runtime.LoadModule(ctx, p.manifest.Name, wasmPath)
	if err != nil {
		return fmt.Errorf("failed to load WASM module: %w", err)
	}

	// Instantiate the module
	instance, err := p.runtime.Instantiate(ctx, p.manifest.Name, compiled, p.manifest)
	if err != nil {
		return fmt.Errorf("failed to instantiate WASM module: %w", err)
	}

	p.instance = instance
	return nil
}

// Close releases the WASM instance.
func (p *WASMPlugin) Close(ctx context.Context) {
	if p.instance != nil {
		p.instance.Close(ctx)
		p.instance = nil
	}
}

// PreFetch implements the pre_fetch hook.
func (p *WASMPlugin) PreFetch(ctx pipeline.HookContext, in pipeline.FetchInput) (pipeline.FetchInput, error) {
	if !p.manifest.SupportsHook("pre_fetch") {
		return in, nil
	}

	if err := p.initInstance(ctx.Context); err != nil {
		return in, p.handleError("pre_fetch", err)
	}

	hookInput := fetchInputToHookInput(in)
	var hookOutput fetchHookOutput

	if err := p.instance.CallHook(ctx.Context, "pre_fetch", hookInput, &hookOutput); err != nil {
		return in, p.handleError("pre_fetch", err)
	}

	return hookOutput.toFetchInput(in), nil
}

// PostFetch implements the post_fetch hook.
func (p *WASMPlugin) PostFetch(ctx pipeline.HookContext, in pipeline.FetchInput, out pipeline.FetchOutput) (pipeline.FetchOutput, error) {
	if !p.manifest.SupportsHook("post_fetch") {
		return out, nil
	}

	if err := p.initInstance(ctx.Context); err != nil {
		return out, p.handleError("post_fetch", err)
	}

	hookInput := postFetchHookInput{
		Input:  fetchInputToHookInput(in),
		Output: fetchOutputToHookOutput(out),
	}
	var hookOutput fetchHookOutput

	if err := p.instance.CallHook(ctx.Context, "post_fetch", hookInput, &hookOutput); err != nil {
		return out, p.handleError("post_fetch", err)
	}

	return hookOutput.toFetchOutput(out), nil
}

// PreExtract implements the pre_extract hook.
func (p *WASMPlugin) PreExtract(ctx pipeline.HookContext, in pipeline.ExtractInput) (pipeline.ExtractInput, error) {
	if !p.manifest.SupportsHook("pre_extract") {
		return in, nil
	}

	if err := p.initInstance(ctx.Context); err != nil {
		return in, p.handleError("pre_extract", err)
	}

	hookInput := extractInputToHookInput(in)
	var hookOutput extractHookOutput

	if err := p.instance.CallHook(ctx.Context, "pre_extract", hookInput, &hookOutput); err != nil {
		return in, p.handleError("pre_extract", err)
	}

	return hookOutput.toExtractInput(in), nil
}

// PostExtract implements the post_extract hook.
func (p *WASMPlugin) PostExtract(ctx pipeline.HookContext, in pipeline.ExtractInput, out pipeline.ExtractOutput) (pipeline.ExtractOutput, error) {
	if !p.manifest.SupportsHook("post_extract") {
		return out, nil
	}

	if err := p.initInstance(ctx.Context); err != nil {
		return out, p.handleError("post_extract", err)
	}

	hookInput := postExtractHookInput{
		Input:  extractInputToHookInput(in),
		Output: extractOutputToHookOutput(out),
	}
	var hookOutput extractHookOutput

	if err := p.instance.CallHook(ctx.Context, "post_extract", hookInput, &hookOutput); err != nil {
		return out, p.handleError("post_extract", err)
	}

	return hookOutput.toExtractOutput(out), nil
}

// PreOutput implements the pre_output hook.
func (p *WASMPlugin) PreOutput(ctx pipeline.HookContext, in pipeline.OutputInput) (pipeline.OutputInput, error) {
	if !p.manifest.SupportsHook("pre_output") {
		return in, nil
	}

	if err := p.initInstance(ctx.Context); err != nil {
		return in, p.handleError("pre_output", err)
	}

	hookInput := outputInputToHookInput(in)
	var hookOutput outputHookOutput

	if err := p.instance.CallHook(ctx.Context, "pre_output", hookInput, &hookOutput); err != nil {
		return in, p.handleError("pre_output", err)
	}

	return hookOutput.toOutputInput(in), nil
}

// PostOutput implements the post_output hook.
func (p *WASMPlugin) PostOutput(ctx pipeline.HookContext, in pipeline.OutputInput, out pipeline.OutputOutput) (pipeline.OutputOutput, error) {
	if !p.manifest.SupportsHook("post_output") {
		return out, nil
	}

	if err := p.initInstance(ctx.Context); err != nil {
		return out, p.handleError("post_output", err)
	}

	hookInput := postOutputHookInput{
		Input:  outputInputToHookInput(in),
		Output: outputOutputToHookOutput(out),
	}
	var hookOutput outputHookOutput

	if err := p.instance.CallHook(ctx.Context, "post_output", hookInput, &hookOutput); err != nil {
		return out, p.handleError("post_output", err)
	}

	return hookOutput.toOutputOutput(out), nil
}

// handleError logs and returns a wrapped error.
func (p *WASMPlugin) handleError(hook string, err error) error {
	// Log the error but don't fail the pipeline
	fmt.Printf("[plugin:%s] Hook %s failed: %v\n", p.manifest.Name, hook, err)
	return apperrors.Wrap(apperrors.KindInternal, fmt.Sprintf("plugin %s hook %s failed", p.manifest.Name, hook), err)
}

// Hook input/output types for JSON serialization

type fetchHookInput struct {
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers,omitempty"`
	UserAgent string            `json:"user_agent,omitempty"`
	Timeout   int               `json:"timeout_ms,omitempty"`
	Headless  bool              `json:"headless,omitempty"`
}

type fetchHookOutput struct {
	URL       string            `json:"url,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	UserAgent string            `json:"user_agent,omitempty"`
	Timeout   int               `json:"timeout_ms,omitempty"`
	Headless  *bool             `json:"headless,omitempty"`
	Body      string            `json:"body,omitempty"`
	Status    int               `json:"status,omitempty"`
	Skip      bool              `json:"skip,omitempty"`
}

type postFetchHookInput struct {
	Input  fetchHookInput  `json:"input"`
	Output fetchHookOutput `json:"output"`
}

type extractHookInput struct {
	URL     string `json:"url"`
	HTML    string `json:"html"`
	Timeout int    `json:"timeout_ms,omitempty"`
}

type extractHookOutput struct {
	HTML      string `json:"html,omitempty"`
	Extracted any    `json:"extracted,omitempty"`
	Skip      bool   `json:"skip,omitempty"`
}

type postExtractHookInput struct {
	Input  extractHookInput  `json:"input"`
	Output extractHookOutput `json:"output"`
}

type outputHookInput struct {
	URL        string `json:"url"`
	Kind       string `json:"kind"`
	Raw        string `json:"raw,omitempty"`
	Structured any    `json:"structured,omitempty"`
}

type outputHookOutput struct {
	Raw        string `json:"raw,omitempty"`
	Structured any    `json:"structured,omitempty"`
	Skip       bool   `json:"skip,omitempty"`
}

type postOutputHookInput struct {
	Input  outputHookInput  `json:"input"`
	Output outputHookOutput `json:"output"`
}

// Conversion functions

func fetchInputToHookInput(in pipeline.FetchInput) fetchHookInput {
	return fetchHookInput{
		URL:       in.Target.URL,
		UserAgent: in.UserAgent,
		Timeout:   int(in.Timeout.Milliseconds()),
		Headless:  in.Headless,
	}
}

func (o *fetchHookOutput) toFetchInput(original pipeline.FetchInput) pipeline.FetchInput {
	if o.Skip {
		// Mark for skipping (would need pipeline support)
		return original
	}
	if o.URL != "" {
		original.Target.URL = o.URL
	}

	if o.UserAgent != "" {
		original.UserAgent = o.UserAgent
	}
	if o.Timeout > 0 {
		// Would need to convert back to Duration
	}
	if o.Headless != nil {
		original.Headless = *o.Headless
	}
	return original
}

func fetchOutputToHookOutput(out pipeline.FetchOutput) fetchHookOutput {
	return fetchHookOutput{
		Body:   out.Result.HTML,
		Status: out.Result.Status,
	}
}

func (o *fetchHookOutput) toFetchOutput(original pipeline.FetchOutput) pipeline.FetchOutput {
	if o.Skip {
		return original
	}
	if o.Body != "" {
		original.Result.HTML = o.Body
	}
	if o.Status > 0 {
		original.Result.Status = o.Status
	}
	return original
}

func extractInputToHookInput(in pipeline.ExtractInput) extractHookInput {
	return extractHookInput{
		URL:  in.Target.URL,
		HTML: in.HTML,
	}
}

func (o *extractHookOutput) toExtractInput(original pipeline.ExtractInput) pipeline.ExtractInput {
	if o.Skip {
		return original
	}
	if o.HTML != "" {
		original.HTML = o.HTML
	}
	return original
}

func extractOutputToHookOutput(out pipeline.ExtractOutput) extractHookOutput {
	return extractHookOutput{
		Extracted: out.Extracted,
	}
}

func (o *extractHookOutput) toExtractOutput(original pipeline.ExtractOutput) pipeline.ExtractOutput {
	if o.Skip {
		return original
	}
	if o.Extracted != nil {
		// Type assert to extract.Extracted if possible
		if extracted, ok := o.Extracted.(extract.Extracted); ok {
			original.Extracted = extracted
		}
	}
	if o.HTML != "" {
		// Store modified HTML if needed
	}
	return original
}

func outputInputToHookInput(in pipeline.OutputInput) outputHookInput {
	var rawStr string
	if len(in.Raw) > 0 {
		rawStr = string(in.Raw)
	}
	return outputHookInput{
		URL:        in.Target.URL,
		Kind:       in.Kind,
		Raw:        rawStr,
		Structured: in.Structured,
	}
}

func (o *outputHookOutput) toOutputInput(original pipeline.OutputInput) pipeline.OutputInput {
	if o.Skip {
		return original
	}
	if o.Raw != "" {
		original.Raw = []byte(o.Raw)
	}
	if o.Structured != nil {
		original.Structured = o.Structured
	}
	return original
}

func outputOutputToHookOutput(out pipeline.OutputOutput) outputHookOutput {
	var rawStr string
	if len(out.Raw) > 0 {
		rawStr = string(out.Raw)
	}
	return outputHookOutput{
		Raw:        rawStr,
		Structured: out.Structured,
	}
}

func (o *outputHookOutput) toOutputOutput(original pipeline.OutputOutput) pipeline.OutputOutput {
	if o.Skip {
		return original
	}
	if o.Raw != "" {
		original.Raw = []byte(o.Raw)
	}
	if o.Structured != nil {
		original.Structured = o.Structured
	}
	return original
}

// Ensure WASMPlugin implements pipeline.Plugin
var _ pipeline.Plugin = (*WASMPlugin)(nil)
