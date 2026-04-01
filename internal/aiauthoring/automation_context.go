// Package aiauthoring implements bounded AI-assisted authoring for automation artifacts.
//
// Purpose:
// - Build automation authoring goals, context summaries, and retry feedback.
//
// Responsibilities:
// - Resolve automation resource identity, derive default instructions,
// - and format context/feedback strings used by the bounded AI prompts.
//
// Scope:
// - Prompt-context and feedback helpers only.
//
// Usage:
// - Used internally by automation generation and debug flows.
//
// Invariants/Assumptions:
// - Derived instructions must stay grounded in observed page context.
// - Feedback should remain concise, deterministic, and validation-driven.
package aiauthoring

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/hostmatch"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

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

type automationGoalKind string

const (
	automationGoalRenderProfile automationGoalKind = "render_profile"
	automationGoalPipelineJS    automationGoalKind = "pipeline_js"
)

func resolveAutomationInstructions(kind automationGoalKind, explicit string, name string, hostPatterns []string, page pageContext) string {
	if trimmed := strings.TrimSpace(explicit); trimmed != "" {
		return trimmed
	}
	switch kind {
	case automationGoalRenderProfile:
		return deriveRenderProfileInstructions(name, hostPatterns, page)
	case automationGoalPipelineJS:
		return derivePipelineJSInstructions(name, hostPatterns, page)
	default:
		return "Infer the minimal deterministic automation needed from the page context."
	}
}

func deriveRenderProfileInstructions(name string, hostPatterns []string, page pageContext) string {
	parts := []string{
		fmt.Sprintf("Generate a render profile for %s.", automationTargetLabel(name, hostPatterns, page)),
		"Prefer the lightest deterministic fetch configuration that still captures meaningful page content.",
	}
	if automationNeedsBrowser(page) {
		parts = append(parts, "The page shows JS-heavy or browser-dependent signals, so prefer headless rendering and wait for stable, user-visible content.")
	} else {
		parts = append(parts, "The page does not strongly signal JS-heavy behavior, so avoid unnecessary browser overhead unless validation requires it.")
	}
	if page.FetchStatus >= 400 {
		parts = append(parts, fmt.Sprintf("The initial fetch returned HTTP %d, so bias toward resilient content acquisition.", page.FetchStatus))
	}
	if strings.TrimSpace(page.FetchEngine) != "" {
		parts = append(parts, fmt.Sprintf("The observed fetch engine was %s.", page.FetchEngine))
	}
	if reasons := automationReasonSummary(page.JSHeaviness.Reasons); reasons != "" {
		parts = append(parts, "Relevant page signals: "+reasons+".")
	}
	return strings.Join(parts, " ")
}

func derivePipelineJSInstructions(name string, hostPatterns []string, page pageContext) string {
	parts := []string{
		fmt.Sprintf("Generate the minimal deterministic pipeline JS needed for %s.", automationTargetLabel(name, hostPatterns, page)),
		"Prefer selectors and waits over custom JavaScript whenever possible.",
	}
	if automationNeedsBrowser(page) {
		parts = append(parts, "The page shows JS-heavy or browser-dependent signals, so it is acceptable to use focused browser-side automation when selectors alone are not enough.")
	} else {
		parts = append(parts, "The page does not strongly signal JS-heavy behavior, so keep the script as small as possible and avoid unnecessary browser-side logic.")
	}
	if page.FetchStatus >= 400 {
		parts = append(parts, fmt.Sprintf("The initial fetch returned HTTP %d, so focus on revealing stable content rather than adding broad, speculative automation.", page.FetchStatus))
	}
	if strings.TrimSpace(page.FetchEngine) != "" {
		parts = append(parts, fmt.Sprintf("The observed fetch engine was %s.", page.FetchEngine))
	}
	if reasons := automationReasonSummary(page.JSHeaviness.Reasons); reasons != "" {
		parts = append(parts, "Relevant page signals: "+reasons+".")
	}
	return strings.Join(parts, " ")
}

func automationNeedsBrowser(page pageContext) bool {
	engine := strings.TrimSpace(page.FetchEngine)
	if engine == string(fetch.RenderEngineChromedp) || engine == string(fetch.RenderEnginePlaywright) {
		return true
	}
	return fetch.IsJSHeavy(page.JSHeaviness, 0.5)
}

func automationTargetLabel(name string, hostPatterns []string, page pageContext) string {
	host := automationTargetHost(hostPatterns, page)
	if trimmed := strings.TrimSpace(name); trimmed != "" {
		return fmt.Sprintf("%q on %s", trimmed, host)
	}
	return host
}

func automationTargetHost(hostPatterns []string, page pageContext) string {
	if host := strings.TrimSpace(hostmatch.HostFromURL(page.URL)); host != "" {
		return host
	}
	for _, pattern := range hostPatterns {
		if trimmed := strings.TrimSpace(pattern); trimmed != "" {
			return trimmed
		}
	}
	return "the target host"
}

func automationReasonSummary(reasons []string) string {
	trimmed := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		if value := strings.TrimSpace(reason); value != "" {
			trimmed = append(trimmed, value)
		}
		if len(trimmed) == 3 {
			break
		}
	}
	return strings.Join(trimmed, "; ")
}

func buildRenderProfileContextSummary(name string, hostPatterns []string, page pageContext) string {
	return buildAutomationContextSummary("Profile", name, hostPatterns, page)
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
	return buildAutomationContextSummary("Script", name, hostPatterns, page)
}

func buildAutomationContextSummary(kind string, name string, hostPatterns []string, page pageContext) string {
	reasons := automationReasonSummary(page.JSHeaviness.Reasons)
	if reasons == "" {
		reasons = "none observed"
	}
	fetchEngine := strings.TrimSpace(page.FetchEngine)
	if fetchEngine == "" {
		fetchEngine = "not recorded"
	}
	hostPatternSummary := strings.Join(hostPatterns, ", ")
	if hostPatternSummary == "" {
		hostPatternSummary = automationTargetHost(hostPatterns, page)
	}
	parts := []string{
		fmt.Sprintf("%s name: %s", kind, name),
		fmt.Sprintf("Resolved URL: %s", strings.TrimSpace(page.URL)),
		fmt.Sprintf("Host patterns: %s", hostPatternSummary),
		fmt.Sprintf("Fetch status: %d", page.FetchStatus),
		fmt.Sprintf("Fetch engine used: %s", fetchEngine),
		fmt.Sprintf("Detected JS heaviness score: %.2f", page.JSHeaviness.Score),
		fmt.Sprintf("Detected JS heaviness reasons: %s", reasons),
	}
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
