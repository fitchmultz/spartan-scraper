// Package aiauthoring implements bounded AI-assisted authoring for automation artifacts.
//
// Purpose:
// - Convert bridge automation suggestions into validated render-profile and pipeline-JS results.
//
// Responsibilities:
// - Define local prompt/suggestion structs, run bounded retry loops,
// - normalize bridge payloads, and collect local automation diagnostics.
//
// Scope:
// - Internal automation suggestion and diagnostic helpers only.
//
// Usage:
// - Used by `automation.go` public entrypoints.
//
// Invariants/Assumptions:
// - At most two AI attempts are allowed per bounded automation authoring request.
// - Suggestions must be validated before they leave the service.
package aiauthoring

import (
	"context"
	"fmt"
	"strings"

	piai "github.com/fitchmultz/spartan-scraper/internal/ai"
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/hostmatch"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

type automationDiagnostics struct {
	Issues        []string
	RecheckStatus int
	RecheckEngine string
	RecheckError  string
}

type renderProfilePromptInput struct {
	Name              string
	HostPatterns      []string
	Instructions      string
	ContextSummary    string
	Feedback          string
	ValidateCandidate func(fetch.RenderProfile) []string
	ValidateLive      func(fetch.RenderProfile) []string
}

type renderProfileSuggestion struct {
	Profile     fetch.RenderProfile
	Explanation string
	RouteID     string
	Provider    string
	Model       string
}

func (s *Service) generateRenderProfileSuggestion(ctx context.Context, page pageContext, input renderProfilePromptInput) (renderProfileSuggestion, error) {
	aiReq := piai.GenerateRenderProfileRequest{
		HTML:           page.HTML,
		URL:            page.URL,
		Instructions:   strings.TrimSpace(input.Instructions),
		ContextSummary: strings.TrimSpace(input.ContextSummary),
		Feedback:       strings.TrimSpace(input.Feedback),
		Images:         toAutomationImages(page.Images),
	}

	for attempt := 0; attempt < 2; attempt++ {
		aiResult, err := s.automationClient.GenerateRenderProfile(ctx, aiReq)
		if err != nil {
			return renderProfileSuggestion{}, apperrors.Wrap(apperrors.KindInternal, "AI render profile generation failed", err)
		}

		candidate := renderProfileFromBridge(aiResult.Profile)
		candidate.Name = input.Name
		candidate.HostPatterns = append([]string(nil), input.HostPatterns...)
		validateCandidate := input.ValidateCandidate
		if validateCandidate == nil {
			validateCandidate = func(candidate fetch.RenderProfile) []string {
				return validateGeneratedRenderProfile(page.HTML, candidate)
			}
		}
		issues := validateCandidate(candidate)
		if len(issues) == 0 && input.ValidateLive != nil {
			issues = append(issues, input.ValidateLive(candidate)...)
		}
		issues = dedupeIssues(issues)
		if len(issues) == 0 {
			return renderProfileSuggestion{
				Profile:     candidate,
				Explanation: aiResult.Explanation,
				RouteID:     aiResult.RouteID,
				Provider:    aiResult.Provider,
				Model:       aiResult.Model,
			}, nil
		}
		if attempt == 1 {
			return renderProfileSuggestion{}, apperrors.Validation(strings.Join(issues, "; "))
		}
		aiReq.Feedback = joinFeedback(aiReq.Feedback, buildRenderProfileFeedback(candidate, issues))
	}

	return renderProfileSuggestion{}, apperrors.Internal("AI render profile generation failed")
}

type pipelineJSPromptInput struct {
	Name              string
	HostPatterns      []string
	Instructions      string
	ContextSummary    string
	Feedback          string
	ValidateCandidate func(pipeline.JSTargetScript) []string
	ValidateLive      func(pipeline.JSTargetScript) []string
}

type pipelineJSSuggestion struct {
	Script      pipeline.JSTargetScript
	Explanation string
	RouteID     string
	Provider    string
	Model       string
}

func (s *Service) generatePipelineJSSuggestion(ctx context.Context, page pageContext, input pipelineJSPromptInput) (pipelineJSSuggestion, error) {
	aiReq := piai.GeneratePipelineJSRequest{
		HTML:           page.HTML,
		URL:            page.URL,
		Instructions:   strings.TrimSpace(input.Instructions),
		ContextSummary: strings.TrimSpace(input.ContextSummary),
		Feedback:       strings.TrimSpace(input.Feedback),
		Images:         toAutomationImages(page.Images),
	}

	for attempt := 0; attempt < 2; attempt++ {
		aiResult, err := s.automationClient.GeneratePipelineJS(ctx, aiReq)
		if err != nil {
			return pipelineJSSuggestion{}, apperrors.Wrap(apperrors.KindInternal, "AI pipeline JS generation failed", err)
		}

		candidate := pipelineScriptFromBridge(aiResult.Script)
		candidate.Name = input.Name
		candidate.HostPatterns = append([]string(nil), input.HostPatterns...)
		validateCandidate := input.ValidateCandidate
		if validateCandidate == nil {
			validateCandidate = func(candidate pipeline.JSTargetScript) []string {
				return validateGeneratedPipelineJS(page.HTML, candidate)
			}
		}
		issues := validateCandidate(candidate)
		if len(issues) == 0 && input.ValidateLive != nil {
			issues = append(issues, input.ValidateLive(candidate)...)
		}
		issues = dedupeIssues(issues)
		if len(issues) == 0 {
			return pipelineJSSuggestion{
				Script:      candidate,
				Explanation: aiResult.Explanation,
				RouteID:     aiResult.RouteID,
				Provider:    aiResult.Provider,
				Model:       aiResult.Model,
			}, nil
		}
		if attempt == 1 {
			return pipelineJSSuggestion{}, apperrors.Validation(strings.Join(issues, "; "))
		}
		aiReq.Feedback = joinFeedback(aiReq.Feedback, buildPipelineJSFeedback(candidate, issues))
	}

	return pipelineJSSuggestion{}, apperrors.Internal("AI pipeline JS generation failed")
}

func (s *Service) collectRenderProfileDiagnostics(ctx context.Context, pageURL string, profile fetch.RenderProfile) automationDiagnostics {
	issues := validateRenderProfileStructure(profile)
	host := hostmatch.HostFromURL(pageURL)
	if !hostmatch.HostMatchesAnyPattern(host, profile.HostPatterns) {
		issues = append(issues, fmt.Sprintf("hostPatterns do not match target host %q", host))
	}
	if len(issues) > 0 {
		return automationDiagnostics{Issues: dedupeIssues(issues)}
	}

	recheckPage, err := s.recheckAutomationPage(ctx, pageURL, &profile, nil)
	diagnostics := automationDiagnostics{}
	if err != nil {
		diagnostics.RecheckError = err.Error()
		diagnostics.Issues = dedupeIssues(append(issues, "live recheck failed: "+err.Error()))
		return diagnostics
	}

	diagnostics.RecheckStatus = recheckPage.FetchStatus
	diagnostics.RecheckEngine = recheckPage.FetchEngine
	issues = append(issues, validateRenderProfileLive(pageURL, recheckPage, profile)...)
	diagnostics.Issues = dedupeIssues(issues)
	return diagnostics
}

func (s *Service) collectPipelineJSDiagnostics(ctx context.Context, page pageContext, script pipeline.JSTargetScript) automationDiagnostics {
	issues := validatePipelineJSStructure(script)
	host := hostmatch.HostFromURL(page.URL)
	if !hostmatch.HostMatchesAnyPattern(host, script.HostPatterns) {
		issues = append(issues, fmt.Sprintf("hostPatterns do not match target host %q", host))
	}
	if len(issues) > 0 {
		return automationDiagnostics{Issues: dedupeIssues(issues)}
	}

	if scriptUsesOnlySelectors(script) {
		return automationDiagnostics{Issues: validateGeneratedPipelineJS(page.HTML, script)}
	}

	recheckPage, err := s.recheckAutomationPage(ctx, page.URL, nil, &script)
	diagnostics := automationDiagnostics{}
	if err != nil {
		diagnostics.RecheckError = err.Error()
		diagnostics.Issues = dedupeIssues(append(issues, "live recheck failed: "+err.Error()))
		return diagnostics
	}

	diagnostics.RecheckStatus = recheckPage.FetchStatus
	diagnostics.RecheckEngine = recheckPage.FetchEngine
	issues = append(issues, validatePipelineJSLive(recheckPage, script)...)
	diagnostics.Issues = dedupeIssues(issues)
	return diagnostics
}

func toAutomationImages(images []extract.AIImageInput) []piai.ImageInput {
	if len(images) == 0 {
		return nil
	}
	out := make([]piai.ImageInput, 0, len(images))
	for _, image := range images {
		out = append(out, piai.ImageInput{Data: image.Data, MimeType: image.MimeType})
	}
	return out
}

func renderProfileFromBridge(profile piai.BridgeRenderProfile) fetch.RenderProfile {
	resourceTypes := make([]fetch.BlockedResourceType, 0, len(profile.Block.ResourceTypes))
	for _, resourceType := range profile.Block.ResourceTypes {
		resourceTypes = append(resourceTypes, fetch.BlockedResourceType(resourceType))
	}
	return fetch.RenderProfile{
		ForceEngine:      fetch.RenderEngine(profile.ForceEngine),
		PreferHeadless:   profile.PreferHeadless,
		AssumeJSHeavy:    profile.AssumeJSHeavy,
		NeverHeadless:    profile.NeverHeadless,
		JSHeavyThreshold: profile.JSHeavyThreshold,
		RateLimitQPS:     profile.RateLimitQPS,
		RateLimitBurst:   profile.RateLimitBurst,
		Block: fetch.RenderBlockPolicy{
			ResourceTypes: resourceTypes,
			URLPatterns:   append([]string(nil), profile.Block.URLPatterns...),
		},
		Wait: fetch.RenderWaitPolicy{
			Mode:                fetch.RenderWaitMode(profile.Wait.Mode),
			Selector:            profile.Wait.Selector,
			NetworkIdleQuietMs:  profile.Wait.NetworkIdleQuietMs,
			MinTextLength:       profile.Wait.MinTextLength,
			StabilityPollMs:     profile.Wait.StabilityPollMs,
			StabilityIterations: profile.Wait.StabilityIterations,
			ExtraSleepMs:        profile.Wait.ExtraSleepMs,
		},
		Timeouts: fetch.RenderTimeoutPolicy{
			MaxRenderMs:  profile.Timeouts.MaxRenderMs,
			ScriptEvalMs: profile.Timeouts.ScriptEvalMs,
			NavigationMs: profile.Timeouts.NavigationMs,
		},
		Screenshot: fetch.ScreenshotConfig{
			Enabled:  profile.Screenshot.Enabled,
			FullPage: profile.Screenshot.FullPage,
			Format:   fetch.ScreenshotFormat(profile.Screenshot.Format),
			Quality:  profile.Screenshot.Quality,
			Width:    profile.Screenshot.Width,
			Height:   profile.Screenshot.Height,
		},
	}
}

func pipelineScriptFromBridge(script piai.BridgePipelineJSScript) pipeline.JSTargetScript {
	return pipeline.JSTargetScript{
		Engine:    script.Engine,
		PreNav:    script.PreNav,
		PostNav:   script.PostNav,
		Selectors: append([]string(nil), script.Selectors...),
	}
}
