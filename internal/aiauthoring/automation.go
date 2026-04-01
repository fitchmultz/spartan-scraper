// Package aiauthoring implements bounded AI-assisted authoring for automation artifacts.
//
// Purpose:
// - Generate and debug render profiles and pipeline JS from resolved page context.
//
// Responsibilities:
// - Define automation authoring request/result types and orchestrate page-context resolution,
// - AI suggestion calls, and bounded debug flows for render profiles and pipeline JS.
//
// Scope:
// - Public automation authoring entrypoints only; suggestion helpers, context builders,
// - and validation logic live in adjacent package files.
//
// Usage:
// - Invoked by API, CLI, MCP, and internal authoring surfaces.
//
// Invariants/Assumptions:
// - Explicit operator instructions always win over derived defaults.
// - Instructionless generation must still use real page context before prompting AI.
// - Strict validation and retry loops remain mandatory after model output.
package aiauthoring

import (
	"context"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

type RenderProfileRequest struct {
	URL           string
	Name          string
	HostPatterns  []string
	Instructions  string
	Images        []extract.AIImageInput
	Headless      bool
	UsePlaywright bool
	Visual        bool
}

type ResolvedGoal struct {
	Text   string `json:"text"`
	Source string `json:"source"`
}

type RenderProfileResult struct {
	Profile           fetch.RenderProfile `json:"profile"`
	ResolvedGoal      *ResolvedGoal       `json:"resolved_goal,omitempty"`
	Explanation       string              `json:"explanation,omitempty"`
	RouteID           string              `json:"route_id,omitempty"`
	Provider          string              `json:"provider,omitempty"`
	Model             string              `json:"model,omitempty"`
	VisualContextUsed bool                `json:"visual_context_used"`
}

type RenderProfileDebugRequest struct {
	URL           string
	Profile       fetch.RenderProfile
	Instructions  string
	Images        []extract.AIImageInput
	Headless      bool
	UsePlaywright bool
	Visual        bool
}

type RenderProfileDebugResult struct {
	Issues            []string             `json:"issues,omitempty"`
	ResolvedGoal      *ResolvedGoal        `json:"resolved_goal,omitempty"`
	Explanation       string               `json:"explanation,omitempty"`
	SuggestedProfile  *fetch.RenderProfile `json:"suggested_profile,omitempty"`
	RouteID           string               `json:"route_id,omitempty"`
	Provider          string               `json:"provider,omitempty"`
	Model             string               `json:"model,omitempty"`
	VisualContextUsed bool                 `json:"visual_context_used"`
	RecheckStatus     int                  `json:"recheck_status,omitempty"`
	RecheckEngine     string               `json:"recheck_engine,omitempty"`
	RecheckError      string               `json:"recheck_error,omitempty"`
}

type PipelineJSRequest struct {
	URL           string
	Name          string
	HostPatterns  []string
	Instructions  string
	Images        []extract.AIImageInput
	Headless      bool
	UsePlaywright bool
	Visual        bool
}

type PipelineJSResult struct {
	Script            pipeline.JSTargetScript `json:"script"`
	ResolvedGoal      *ResolvedGoal           `json:"resolved_goal,omitempty"`
	Explanation       string                  `json:"explanation,omitempty"`
	RouteID           string                  `json:"route_id,omitempty"`
	Provider          string                  `json:"provider,omitempty"`
	Model             string                  `json:"model,omitempty"`
	VisualContextUsed bool                    `json:"visual_context_used"`
}

type PipelineJSDebugRequest struct {
	URL           string
	Script        pipeline.JSTargetScript
	Instructions  string
	Images        []extract.AIImageInput
	Headless      bool
	UsePlaywright bool
	Visual        bool
}

type PipelineJSDebugResult struct {
	Issues            []string                 `json:"issues,omitempty"`
	ResolvedGoal      *ResolvedGoal            `json:"resolved_goal,omitempty"`
	Explanation       string                   `json:"explanation,omitempty"`
	SuggestedScript   *pipeline.JSTargetScript `json:"suggested_script,omitempty"`
	RouteID           string                   `json:"route_id,omitempty"`
	Provider          string                   `json:"provider,omitempty"`
	Model             string                   `json:"model,omitempty"`
	VisualContextUsed bool                     `json:"visual_context_used"`
	RecheckStatus     int                      `json:"recheck_status,omitempty"`
	RecheckEngine     string                   `json:"recheck_engine,omitempty"`
	RecheckError      string                   `json:"recheck_error,omitempty"`
}

const (
	resolvedGoalSourceExplicit = "explicit"
	resolvedGoalSourceDerived  = "derived"
)

func buildResolvedGoal(text string, explicit string) *ResolvedGoal {
	trimmedText := strings.TrimSpace(text)
	if trimmedText == "" {
		return nil
	}
	source := resolvedGoalSourceDerived
	if strings.TrimSpace(explicit) != "" {
		source = resolvedGoalSourceExplicit
	}
	return &ResolvedGoal{Text: trimmedText, Source: source}
}

func (s *Service) GenerateRenderProfile(ctx context.Context, req RenderProfileRequest) (RenderProfileResult, error) {
	if err := s.requireAutomationClient(); err != nil {
		return RenderProfileResult{}, err
	}
	if err := validateHTTPURL(req.URL); err != nil {
		return RenderProfileResult{}, err
	}

	name, hostPatterns, err := resolveResourceIdentity(req.URL, req.Name, req.HostPatterns)
	if err != nil {
		return RenderProfileResult{}, err
	}
	images, err := normalizeDirectAIImages(req.Images)
	if err != nil {
		return RenderProfileResult{}, err
	}

	ctx, cancel := s.withRequestTimeout(ctx)
	defer cancel()

	page, err := s.resolvePageContext(ctx, req.URL, "", images, req.Headless, req.UsePlaywright, req.Visual)
	if err != nil {
		return RenderProfileResult{}, err
	}

	instructions := resolveAutomationInstructions(automationGoalRenderProfile, req.Instructions, name, hostPatterns, page)
	resolvedGoal := buildResolvedGoal(instructions, req.Instructions)
	suggestion, err := s.generateRenderProfileSuggestion(ctx, page, renderProfilePromptInput{
		Name:           name,
		HostPatterns:   hostPatterns,
		Instructions:   instructions,
		ContextSummary: buildRenderProfileContextSummary(name, hostPatterns, page),
		ValidateCandidate: func(candidate fetch.RenderProfile) []string {
			return validateGeneratedRenderProfile(page.HTML, candidate)
		},
	})
	if err != nil {
		return RenderProfileResult{}, err
	}

	return RenderProfileResult{
		Profile:           suggestion.Profile,
		ResolvedGoal:      resolvedGoal,
		Explanation:       suggestion.Explanation,
		RouteID:           suggestion.RouteID,
		Provider:          suggestion.Provider,
		Model:             suggestion.Model,
		VisualContextUsed: page.VisualContextUsed,
	}, nil
}

func (s *Service) DebugRenderProfile(ctx context.Context, req RenderProfileDebugRequest) (RenderProfileDebugResult, error) {
	if err := s.requireAutomationClient(); err != nil {
		return RenderProfileDebugResult{}, err
	}
	if err := validateHTTPURL(req.URL); err != nil {
		return RenderProfileDebugResult{}, err
	}
	if err := fetch.ValidateRenderProfile(req.Profile); err != nil {
		return RenderProfileDebugResult{}, err
	}
	images, err := normalizeDirectAIImages(req.Images)
	if err != nil {
		return RenderProfileDebugResult{}, err
	}

	ctx, cancel := s.withRequestTimeout(ctx)
	defer cancel()

	baselinePage, err := s.resolvePageContext(ctx, req.URL, "", images, req.Headless, req.UsePlaywright, req.Visual)
	if err != nil {
		return RenderProfileDebugResult{}, err
	}

	diagnostics := s.collectRenderProfileDiagnostics(ctx, req.URL, req.Profile)
	instructions := buildRenderProfileDebugInstructions(req.Profile, req.Instructions)
	result := RenderProfileDebugResult{
		Issues:            diagnostics.Issues,
		ResolvedGoal:      buildResolvedGoal(instructions, req.Instructions),
		VisualContextUsed: baselinePage.VisualContextUsed,
		RecheckStatus:     diagnostics.RecheckStatus,
		RecheckEngine:     diagnostics.RecheckEngine,
		RecheckError:      diagnostics.RecheckError,
	}
	if len(diagnostics.Issues) == 0 && strings.TrimSpace(req.Instructions) == "" {
		result.Explanation = "No local render profile issues detected."
		return result, nil
	}

	suggestion, err := s.generateRenderProfileSuggestion(ctx, baselinePage, renderProfilePromptInput{
		Name:              req.Profile.Name,
		HostPatterns:      append([]string(nil), req.Profile.HostPatterns...),
		Instructions:      instructions,
		ContextSummary:    buildRenderProfileDebugContextSummary(req.Profile, baselinePage, diagnostics),
		Feedback:          buildRenderProfileDebugFeedback(req.Profile, diagnostics),
		ValidateCandidate: validateRenderProfileStructure,
		ValidateLive: func(candidate fetch.RenderProfile) []string {
			candidateDiagnostics := s.collectRenderProfileDiagnostics(ctx, req.URL, candidate)
			return candidateDiagnostics.Issues
		},
	})
	if err != nil {
		return RenderProfileDebugResult{}, err
	}

	result.SuggestedProfile = &suggestion.Profile
	result.Explanation = suggestion.Explanation
	result.RouteID = suggestion.RouteID
	result.Provider = suggestion.Provider
	result.Model = suggestion.Model
	return result, nil
}

func (s *Service) GeneratePipelineJS(ctx context.Context, req PipelineJSRequest) (PipelineJSResult, error) {
	if err := s.requireAutomationClient(); err != nil {
		return PipelineJSResult{}, err
	}
	if err := validateHTTPURL(req.URL); err != nil {
		return PipelineJSResult{}, err
	}

	name, hostPatterns, err := resolveResourceIdentity(req.URL, req.Name, req.HostPatterns)
	if err != nil {
		return PipelineJSResult{}, err
	}
	images, err := normalizeDirectAIImages(req.Images)
	if err != nil {
		return PipelineJSResult{}, err
	}

	ctx, cancel := s.withRequestTimeout(ctx)
	defer cancel()

	page, err := s.resolvePageContext(ctx, req.URL, "", images, req.Headless, req.UsePlaywright, req.Visual)
	if err != nil {
		return PipelineJSResult{}, err
	}

	instructions := resolveAutomationInstructions(automationGoalPipelineJS, req.Instructions, name, hostPatterns, page)
	resolvedGoal := buildResolvedGoal(instructions, req.Instructions)
	suggestion, err := s.generatePipelineJSSuggestion(ctx, page, pipelineJSPromptInput{
		Name:           name,
		HostPatterns:   hostPatterns,
		Instructions:   instructions,
		ContextSummary: buildPipelineJSContextSummary(name, hostPatterns, page),
		ValidateCandidate: func(candidate pipeline.JSTargetScript) []string {
			return validateGeneratedPipelineJS(page.HTML, candidate)
		},
	})
	if err != nil {
		return PipelineJSResult{}, err
	}

	return PipelineJSResult{
		Script:            suggestion.Script,
		ResolvedGoal:      resolvedGoal,
		Explanation:       suggestion.Explanation,
		RouteID:           suggestion.RouteID,
		Provider:          suggestion.Provider,
		Model:             suggestion.Model,
		VisualContextUsed: page.VisualContextUsed,
	}, nil
}

func (s *Service) DebugPipelineJS(ctx context.Context, req PipelineJSDebugRequest) (PipelineJSDebugResult, error) {
	if err := s.requireAutomationClient(); err != nil {
		return PipelineJSDebugResult{}, err
	}
	if err := validateHTTPURL(req.URL); err != nil {
		return PipelineJSDebugResult{}, err
	}
	if err := pipeline.ValidateJSTargetScript(req.Script); err != nil {
		return PipelineJSDebugResult{}, err
	}
	images, err := normalizeDirectAIImages(req.Images)
	if err != nil {
		return PipelineJSDebugResult{}, err
	}

	ctx, cancel := s.withRequestTimeout(ctx)
	defer cancel()

	baselinePage, err := s.resolvePageContext(ctx, req.URL, "", images, req.Headless, req.UsePlaywright, req.Visual)
	if err != nil {
		return PipelineJSDebugResult{}, err
	}

	diagnostics := s.collectPipelineJSDiagnostics(ctx, baselinePage, req.Script)
	instructions := buildPipelineJSDebugInstructions(req.Script, req.Instructions)
	result := PipelineJSDebugResult{
		Issues:            diagnostics.Issues,
		ResolvedGoal:      buildResolvedGoal(instructions, req.Instructions),
		VisualContextUsed: baselinePage.VisualContextUsed,
		RecheckStatus:     diagnostics.RecheckStatus,
		RecheckEngine:     diagnostics.RecheckEngine,
		RecheckError:      diagnostics.RecheckError,
	}
	if len(diagnostics.Issues) == 0 && strings.TrimSpace(req.Instructions) == "" {
		result.Explanation = "No local pipeline JS issues detected."
		return result, nil
	}

	suggestion, err := s.generatePipelineJSSuggestion(ctx, baselinePage, pipelineJSPromptInput{
		Name:              req.Script.Name,
		HostPatterns:      append([]string(nil), req.Script.HostPatterns...),
		Instructions:      instructions,
		ContextSummary:    buildPipelineJSDebugContextSummary(req.Script, baselinePage, diagnostics),
		Feedback:          buildPipelineJSDebugFeedback(req.Script, diagnostics),
		ValidateCandidate: validatePipelineJSStructure,
		ValidateLive: func(candidate pipeline.JSTargetScript) []string {
			candidateDiagnostics := s.collectPipelineJSDiagnostics(ctx, baselinePage, candidate)
			return candidateDiagnostics.Issues
		},
	})
	if err != nil {
		return PipelineJSDebugResult{}, err
	}

	result.SuggestedScript = &suggestion.Script
	result.Explanation = suggestion.Explanation
	result.RouteID = suggestion.RouteID
	result.Provider = suggestion.Provider
	result.Model = suggestion.Model
	return result, nil
}
