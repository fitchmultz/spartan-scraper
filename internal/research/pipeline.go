// Package research provides pipeline integration for research results.
package research

import (
	"context"
	"encoding/json"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

// applyResearchOutputPipeline applies pipeline hooks to research results.
func applyResearchOutputPipeline(ctx context.Context, registry *pipeline.Registry, baseCtx pipeline.HookContext, result Result) (Result, error) {
	raw, err := json.Marshal(result)
	if err != nil {
		return Result{}, apperrors.Wrap(apperrors.KindInternal, "failed to marshal research result", err)
	}
	input := pipeline.OutputInput{
		Target:     baseCtx.Target,
		Kind:       string(model.KindResearch),
		Raw:        raw,
		Structured: result,
	}

	preCtx := baseCtx
	preCtx.Stage = pipeline.StagePreOutput
	outInput, err := registry.RunPreOutput(preCtx, input)
	if err != nil {
		return Result{}, err
	}
	if typed, ok := outInput.Structured.(Result); ok {
		result = typed
		outInput.Structured = result
	}

	transformCtx := baseCtx
	transformCtx.Stage = pipeline.StagePreOutput
	out, err := registry.RunTransformers(transformCtx, outInput)
	if err != nil {
		return Result{}, err
	}

	postCtx := baseCtx
	postCtx.Stage = pipeline.StagePostOutput
	out, err = registry.RunPostOutput(postCtx, outInput, out)
	if err != nil {
		return Result{}, err
	}

	if out.Structured == nil {
		return result, nil
	}
	typed, ok := out.Structured.(Result)
	if !ok {
		return Result{}, apperrors.Internal("pipeline output type mismatch for research")
	}
	return typed, nil
}
