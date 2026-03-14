package aiauthoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/andybalholm/cascadia"
	piai "github.com/fitchmultz/spartan-scraper/internal/ai"
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/hostmatch"
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

type RenderProfileResult struct {
	Profile           fetch.RenderProfile `json:"profile"`
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

type automationDiagnostics struct {
	Issues        []string
	RecheckStatus int
	RecheckEngine string
	RecheckError  string
}

func (s *Service) GenerateRenderProfile(ctx context.Context, req RenderProfileRequest) (RenderProfileResult, error) {
	if err := s.requireAutomationClient(); err != nil {
		return RenderProfileResult{}, err
	}
	if err := validateHTTPURL(req.URL); err != nil {
		return RenderProfileResult{}, err
	}
	if strings.TrimSpace(req.Instructions) == "" {
		return RenderProfileResult{}, apperrors.Validation("instructions are required")
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

	suggestion, err := s.generateRenderProfileSuggestion(ctx, page, renderProfilePromptInput{
		Name:           name,
		HostPatterns:   hostPatterns,
		Instructions:   strings.TrimSpace(req.Instructions),
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
	result := RenderProfileDebugResult{
		Issues:            diagnostics.Issues,
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
		Instructions:      buildRenderProfileDebugInstructions(req.Profile, req.Instructions),
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
	if strings.TrimSpace(req.Instructions) == "" {
		return PipelineJSResult{}, apperrors.Validation("instructions are required")
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

	suggestion, err := s.generatePipelineJSSuggestion(ctx, page, pipelineJSPromptInput{
		Name:           name,
		HostPatterns:   hostPatterns,
		Instructions:   strings.TrimSpace(req.Instructions),
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
	result := PipelineJSDebugResult{
		Issues:            diagnostics.Issues,
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
		Instructions:      buildPipelineJSDebugInstructions(req.Script, req.Instructions),
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

func resolveResourceIdentity(pageURL string, requestedName string, requestedHostPatterns []string) (string, []string, error) {
	parsed, err := url.Parse(strings.TrimSpace(pageURL))
	if err != nil {
		return "", nil, apperrors.Validation("invalid URL format")
	}
	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		return "", nil, apperrors.Validation("url host is required")
	}
	name := strings.TrimSpace(requestedName)
	if name == "" {
		name = host
	}
	hostPatterns := normalizeHostPatterns(requestedHostPatterns)
	if len(hostPatterns) == 0 {
		hostPatterns = []string{host}
	}
	return name, hostPatterns, nil
}

func normalizeHostPatterns(hostPatterns []string) []string {
	if len(hostPatterns) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(hostPatterns))
	for _, pattern := range hostPatterns {
		trimmed := strings.TrimSpace(pattern)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	sort.Strings(out)
	return out
}

func buildRenderProfileContextSummary(name string, hostPatterns []string, page pageContext) string {
	parts := []string{
		fmt.Sprintf("Profile name: %s", name),
		fmt.Sprintf("Host patterns: %s", strings.Join(hostPatterns, ", ")),
	}
	if page.FetchStatus > 0 {
		parts = append(parts, fmt.Sprintf("Fetch status: %d", page.FetchStatus))
	}
	if strings.TrimSpace(page.FetchEngine) != "" {
		parts = append(parts, fmt.Sprintf("Fetch engine used: %s", page.FetchEngine))
	}
	parts = append(parts,
		fmt.Sprintf("Detected JS heaviness score: %.2f", page.JSHeaviness.Score),
		fmt.Sprintf("Detected JS heaviness reasons: %s", strings.Join(page.JSHeaviness.Reasons, "; ")),
	)
	return strings.Join(parts, "\n")
}

func buildRenderProfileDebugInstructions(profile fetch.RenderProfile, instructions string) string {
	base := fmt.Sprintf("Tune the render profile named %q for the supplied page while preserving its purpose and keeping changes minimal, deterministic, and operationally useful.", profile.Name)
	if strings.TrimSpace(instructions) == "" {
		return base
	}
	return base + " Operator guidance: " + strings.TrimSpace(instructions)
}

func buildRenderProfileDebugContextSummary(profile fetch.RenderProfile, page pageContext, diagnostics automationDiagnostics) string {
	parts := []string{buildRenderProfileContextSummary(profile.Name, profile.HostPatterns, page)}
	if diagnostics.RecheckStatus > 0 {
		parts = append(parts, fmt.Sprintf("Current profile recheck status: %d", diagnostics.RecheckStatus))
	}
	if strings.TrimSpace(diagnostics.RecheckEngine) != "" {
		parts = append(parts, fmt.Sprintf("Current profile recheck engine: %s", diagnostics.RecheckEngine))
	}
	if strings.TrimSpace(diagnostics.RecheckError) != "" {
		parts = append(parts, "Current profile recheck error: "+diagnostics.RecheckError)
	}
	if len(diagnostics.Issues) > 0 {
		parts = append(parts, "Current profile issues:\n- "+strings.Join(diagnostics.Issues, "\n- "))
	}
	return strings.Join(parts, "\n")
}

func buildPipelineJSContextSummary(name string, hostPatterns []string, page pageContext) string {
	parts := []string{
		fmt.Sprintf("Script name: %s", name),
		fmt.Sprintf("Host patterns: %s", strings.Join(hostPatterns, ", ")),
	}
	if page.FetchStatus > 0 {
		parts = append(parts, fmt.Sprintf("Fetch status: %d", page.FetchStatus))
	}
	if strings.TrimSpace(page.FetchEngine) != "" {
		parts = append(parts, fmt.Sprintf("Fetch engine used: %s", page.FetchEngine))
	}
	parts = append(parts,
		fmt.Sprintf("Detected JS heaviness score: %.2f", page.JSHeaviness.Score),
		fmt.Sprintf("Detected JS heaviness reasons: %s", strings.Join(page.JSHeaviness.Reasons, "; ")),
	)
	return strings.Join(parts, "\n")
}

func buildPipelineJSDebugInstructions(script pipeline.JSTargetScript, instructions string) string {
	base := fmt.Sprintf("Tune the pipeline JS script named %q for the supplied page while preserving its intent and keeping the automation minimal and deterministic.", script.Name)
	if strings.TrimSpace(instructions) == "" {
		return base
	}
	return base + " Operator guidance: " + strings.TrimSpace(instructions)
}

func buildPipelineJSDebugContextSummary(script pipeline.JSTargetScript, page pageContext, diagnostics automationDiagnostics) string {
	parts := []string{buildPipelineJSContextSummary(script.Name, script.HostPatterns, page)}
	if diagnostics.RecheckStatus > 0 {
		parts = append(parts, fmt.Sprintf("Current script recheck status: %d", diagnostics.RecheckStatus))
	}
	if strings.TrimSpace(diagnostics.RecheckEngine) != "" {
		parts = append(parts, fmt.Sprintf("Current script recheck engine: %s", diagnostics.RecheckEngine))
	}
	if strings.TrimSpace(diagnostics.RecheckError) != "" {
		parts = append(parts, "Current script recheck error: "+diagnostics.RecheckError)
	}
	if len(diagnostics.Issues) > 0 {
		parts = append(parts, "Current script issues:\n- "+strings.Join(diagnostics.Issues, "\n- "))
	}
	return strings.Join(parts, "\n")
}

func buildRenderProfileFeedback(profile fetch.RenderProfile, issues []string) string {
	parts := []string{}
	if data, err := json.MarshalIndent(profile, "", "  "); err == nil {
		parts = append(parts, "Current profile candidate:\n"+string(data))
	}
	if len(issues) > 0 {
		parts = append(parts, "Validation issues:\n- "+strings.Join(issues, "\n- "))
	}
	return strings.Join(parts, "\n\n")
}

func buildRenderProfileDebugFeedback(profile fetch.RenderProfile, diagnostics automationDiagnostics) string {
	parts := []string{buildRenderProfileFeedback(profile, diagnostics.Issues)}
	if diagnostics.RecheckStatus > 0 {
		parts = append(parts, fmt.Sprintf("Current profile live recheck status: %d", diagnostics.RecheckStatus))
	}
	if strings.TrimSpace(diagnostics.RecheckEngine) != "" {
		parts = append(parts, fmt.Sprintf("Current profile live recheck engine: %s", diagnostics.RecheckEngine))
	}
	if strings.TrimSpace(diagnostics.RecheckError) != "" {
		parts = append(parts, "Current profile live recheck error: "+diagnostics.RecheckError)
	}
	return strings.Join(parts, "\n\n")
}

func buildPipelineJSFeedback(script pipeline.JSTargetScript, issues []string) string {
	parts := []string{}
	if data, err := json.MarshalIndent(script, "", "  "); err == nil {
		parts = append(parts, "Current script candidate:\n"+string(data))
	}
	if len(issues) > 0 {
		parts = append(parts, "Validation issues:\n- "+strings.Join(issues, "\n- "))
	}
	return strings.Join(parts, "\n\n")
}

func buildPipelineJSDebugFeedback(script pipeline.JSTargetScript, diagnostics automationDiagnostics) string {
	parts := []string{buildPipelineJSFeedback(script, diagnostics.Issues)}
	if diagnostics.RecheckStatus > 0 {
		parts = append(parts, fmt.Sprintf("Current script live recheck status: %d", diagnostics.RecheckStatus))
	}
	if strings.TrimSpace(diagnostics.RecheckEngine) != "" {
		parts = append(parts, fmt.Sprintf("Current script live recheck engine: %s", diagnostics.RecheckEngine))
	}
	if strings.TrimSpace(diagnostics.RecheckError) != "" {
		parts = append(parts, "Current script live recheck error: "+diagnostics.RecheckError)
	}
	return strings.Join(parts, "\n\n")
}

func validateGeneratedRenderProfile(html string, profile fetch.RenderProfile) []string {
	issues := validateRenderProfileStructure(profile)
	if profile.Wait.Mode == fetch.RenderWaitModeSelector {
		issues = append(issues, validateSelectorAgainstHTML(html, profile.Wait.Selector, "wait.selector")...)
	}
	return dedupeIssues(issues)
}

func validateRenderProfileStructure(profile fetch.RenderProfile) []string {
	issues := []string{}
	if err := fetch.ValidateRenderProfile(profile); err != nil {
		issues = append(issues, err.Error())
	}
	issues = append(issues, validateRenderProfileEnums(profile)...)
	if profile.Wait.Mode == fetch.RenderWaitModeSelector && strings.TrimSpace(profile.Wait.Selector) == "" {
		issues = append(issues, "wait.selector is required")
	}
	return dedupeIssues(issues)
}

func validateRenderProfileLive(pageURL string, page pageContext, profile fetch.RenderProfile) []string {
	issues := []string{}
	if page.FetchStatus >= 400 {
		issues = append(issues, fmt.Sprintf("live recheck returned HTTP %d", page.FetchStatus))
	}
	if strings.TrimSpace(page.HTML) == "" {
		issues = append(issues, "live recheck returned empty HTML")
	}
	if profile.Wait.Mode == fetch.RenderWaitModeSelector {
		issues = append(issues, validateSelectorAgainstHTML(page.HTML, profile.Wait.Selector, "wait.selector")...)
	}
	threshold := profile.JSHeavyThreshold
	if threshold <= 0 {
		threshold = 0.5
	}
	if page.FetchEngine == string(fetch.RenderEngineHTTP) && fetch.IsJSHeavy(page.JSHeaviness, threshold) && !profile.NeverHeadless {
		issues = append(issues, "live recheck still used HTTP while the page appears JS-heavy")
	}
	if profile.ForceEngine != "" && page.FetchEngine != string(profile.ForceEngine) {
		issues = append(issues, fmt.Sprintf("live recheck used engine %q instead of forced engine %q", page.FetchEngine, profile.ForceEngine))
	}
	host := hostmatch.HostFromURL(pageURL)
	if !hostmatch.HostMatchesAnyPattern(host, profile.HostPatterns) {
		issues = append(issues, fmt.Sprintf("hostPatterns do not match target host %q", host))
	}
	return dedupeIssues(issues)
}

func validateGeneratedPipelineJS(html string, script pipeline.JSTargetScript) []string {
	issues := validatePipelineJSStructure(script)
	for idx, selector := range script.Selectors {
		issues = append(issues, validateSelectorAgainstHTML(html, selector, fmt.Sprintf("selectors[%d]", idx))...)
	}
	return dedupeIssues(issues)
}

func validatePipelineJSStructure(script pipeline.JSTargetScript) []string {
	issues := []string{}
	if err := pipeline.ValidateJSTargetScript(script); err != nil {
		issues = append(issues, err.Error())
	}
	if strings.TrimSpace(script.Engine) == "" && strings.TrimSpace(script.PreNav) == "" && strings.TrimSpace(script.PostNav) == "" && len(script.Selectors) == 0 {
		issues = append(issues, "script must set at least one of engine, preNav, postNav, or selectors")
	}
	return dedupeIssues(issues)
}

func scriptUsesOnlySelectors(script pipeline.JSTargetScript) bool {
	return strings.TrimSpace(script.Engine) == "" && strings.TrimSpace(script.PreNav) == "" && strings.TrimSpace(script.PostNav) == ""
}

func validatePipelineJSLive(page pageContext, script pipeline.JSTargetScript) []string {
	issues := []string{}
	if page.FetchStatus >= 400 {
		issues = append(issues, fmt.Sprintf("live recheck returned HTTP %d", page.FetchStatus))
	}
	if strings.TrimSpace(page.FetchEngine) == string(fetch.RenderEngineHTTP) {
		issues = append(issues, "live recheck fell back to HTTP so the pipeline script did not execute")
	}
	if strings.TrimSpace(page.HTML) == "" {
		issues = append(issues, "live recheck returned empty HTML")
	}
	for idx, selector := range script.Selectors {
		issues = append(issues, validateSelectorAgainstHTML(page.HTML, selector, fmt.Sprintf("selectors[%d]", idx))...)
	}
	if strings.TrimSpace(script.Engine) != "" && page.FetchEngine != strings.ToLower(strings.TrimSpace(script.Engine)) {
		issues = append(issues, fmt.Sprintf("live recheck used engine %q instead of requested engine %q", page.FetchEngine, strings.ToLower(strings.TrimSpace(script.Engine))))
	}
	return dedupeIssues(issues)
}

func validateRenderProfileEnums(profile fetch.RenderProfile) []string {
	issues := []string{}
	switch profile.Wait.Mode {
	case "", fetch.RenderWaitModeDOMReady, fetch.RenderWaitModeNetworkIdle, fetch.RenderWaitModeStability, fetch.RenderWaitModeSelector:
	default:
		issues = append(issues, fmt.Sprintf("wait.mode %q is invalid", profile.Wait.Mode))
	}
	switch profile.Screenshot.Format {
	case "", fetch.ScreenshotFormatPNG, fetch.ScreenshotFormatJPEG:
	default:
		issues = append(issues, fmt.Sprintf("screenshot.format %q is invalid", profile.Screenshot.Format))
	}
	for _, resourceType := range profile.Block.ResourceTypes {
		switch resourceType {
		case fetch.BlockedResourceImage, fetch.BlockedResourceMedia, fetch.BlockedResourceFont, fetch.BlockedResourceStylesheet, fetch.BlockedResourceOther:
		default:
			issues = append(issues, fmt.Sprintf("block.resourceTypes contains invalid value %q", resourceType))
		}
	}
	if profile.Screenshot.Quality < 0 || profile.Screenshot.Quality > 100 {
		issues = append(issues, "screenshot.quality must be between 0 and 100")
	}
	return issues
}

func validateSelectorAgainstHTML(html string, selector string, label string) []string {
	trimmed := strings.TrimSpace(selector)
	if trimmed == "" {
		return []string{fmt.Sprintf("%s is required", label)}
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return []string{"fetched HTML could not be parsed for selector validation"}
	}
	if _, err := cascadia.ParseGroup(trimmed); err != nil {
		return []string{fmt.Sprintf("%s is invalid: %s", label, err.Error())}
	}
	if doc.Find(trimmed).Length() == 0 {
		return []string{fmt.Sprintf("%s matched no elements", label)}
	}
	return nil
}

func dedupeIssues(issues []string) []string {
	if len(issues) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(issues))
	for _, issue := range issues {
		trimmed := strings.TrimSpace(issue)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func joinFeedback(existing string, next string) string {
	existing = strings.TrimSpace(existing)
	next = strings.TrimSpace(next)
	switch {
	case existing == "":
		return next
	case next == "":
		return existing
	default:
		return existing + "\n\n" + next
	}
}
