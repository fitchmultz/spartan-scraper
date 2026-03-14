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
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

type RenderProfileRequest struct {
	URL           string
	Name          string
	HostPatterns  []string
	Instructions  string
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

type PipelineJSRequest struct {
	URL           string
	Name          string
	HostPatterns  []string
	Instructions  string
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

	ctx, cancel := s.withRequestTimeout(ctx)
	defer cancel()

	page, err := s.resolvePageContext(ctx, req.URL, "", req.Headless, req.UsePlaywright, req.Visual)
	if err != nil {
		return RenderProfileResult{}, err
	}

	aiReq := piai.GenerateRenderProfileRequest{
		HTML:           page.HTML,
		URL:            page.URL,
		Instructions:   strings.TrimSpace(req.Instructions),
		ContextSummary: buildRenderProfileContextSummary(name, hostPatterns, page),
		Images:         toAutomationImages(page.Images),
	}

	for attempt := 0; attempt < 2; attempt++ {
		aiResult, err := s.automationClient.GenerateRenderProfile(ctx, aiReq)
		if err != nil {
			return RenderProfileResult{}, apperrors.Wrap(apperrors.KindInternal, "AI render profile generation failed", err)
		}

		candidate := renderProfileFromBridge(aiResult.Profile)
		candidate.Name = name
		candidate.HostPatterns = append([]string(nil), hostPatterns...)
		issues := validateGeneratedRenderProfile(page.HTML, candidate)
		if len(issues) == 0 {
			return RenderProfileResult{
				Profile:           candidate,
				Explanation:       aiResult.Explanation,
				RouteID:           aiResult.RouteID,
				Provider:          aiResult.Provider,
				Model:             aiResult.Model,
				VisualContextUsed: page.VisualContextUsed,
			}, nil
		}
		if attempt == 1 {
			return RenderProfileResult{}, apperrors.Validation(strings.Join(issues, "; "))
		}
		aiReq.Feedback = buildRenderProfileFeedback(candidate, issues)
	}

	return RenderProfileResult{}, apperrors.Internal("AI render profile generation failed")
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

	ctx, cancel := s.withRequestTimeout(ctx)
	defer cancel()

	page, err := s.resolvePageContext(ctx, req.URL, "", req.Headless, req.UsePlaywright, req.Visual)
	if err != nil {
		return PipelineJSResult{}, err
	}

	aiReq := piai.GeneratePipelineJSRequest{
		HTML:           page.HTML,
		URL:            page.URL,
		Instructions:   strings.TrimSpace(req.Instructions),
		ContextSummary: buildPipelineJSContextSummary(name, hostPatterns, page),
		Images:         toAutomationImages(page.Images),
	}

	for attempt := 0; attempt < 2; attempt++ {
		aiResult, err := s.automationClient.GeneratePipelineJS(ctx, aiReq)
		if err != nil {
			return PipelineJSResult{}, apperrors.Wrap(apperrors.KindInternal, "AI pipeline JS generation failed", err)
		}

		candidate := pipelineScriptFromBridge(aiResult.Script)
		candidate.Name = name
		candidate.HostPatterns = append([]string(nil), hostPatterns...)
		issues := validateGeneratedPipelineJS(page.HTML, candidate)
		if len(issues) == 0 {
			return PipelineJSResult{
				Script:            candidate,
				Explanation:       aiResult.Explanation,
				RouteID:           aiResult.RouteID,
				Provider:          aiResult.Provider,
				Model:             aiResult.Model,
				VisualContextUsed: page.VisualContextUsed,
			}, nil
		}
		if attempt == 1 {
			return PipelineJSResult{}, apperrors.Validation(strings.Join(issues, "; "))
		}
		aiReq.Feedback = buildPipelineJSFeedback(candidate, issues)
	}

	return PipelineJSResult{}, apperrors.Internal("AI pipeline JS generation failed")
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

func validateGeneratedRenderProfile(html string, profile fetch.RenderProfile) []string {
	issues := []string{}
	if err := fetch.ValidateRenderProfile(profile); err != nil {
		issues = append(issues, err.Error())
	}
	issues = append(issues, validateRenderProfileEnums(profile)...)
	if profile.Wait.Mode == fetch.RenderWaitModeSelector {
		issues = append(issues, validateSelectorAgainstHTML(html, profile.Wait.Selector, "wait.selector")...)
	}
	return dedupeIssues(issues)
}

func validateGeneratedPipelineJS(html string, script pipeline.JSTargetScript) []string {
	issues := []string{}
	if err := pipeline.ValidateJSTargetScript(script); err != nil {
		issues = append(issues, err.Error())
	}
	if strings.TrimSpace(script.Engine) == "" && strings.TrimSpace(script.PreNav) == "" && strings.TrimSpace(script.PostNav) == "" && len(script.Selectors) == 0 {
		issues = append(issues, "script must set at least one of engine, preNav, postNav, or selectors")
	}
	for idx, selector := range script.Selectors {
		issues = append(issues, validateSelectorAgainstHTML(html, selector, fmt.Sprintf("selectors[%d]", idx))...)
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
