// Package aiauthoring implements bounded AI-assisted authoring for automation artifacts.
//
// Purpose:
// - Validate generated automation artifacts against structure, live rechecks, and fetched HTML.
//
// Responsibilities:
// - Validate render profiles and pipeline JS, check selector matches, and deduplicate retry feedback issues.
//
// Scope:
// - Internal validation helpers only.
//
// Usage:
// - Used by automation suggestion and debug flows.
//
// Invariants/Assumptions:
// - Validation must remain deterministic and bounded to the current page/recheck context.
package aiauthoring

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/andybalholm/cascadia"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/hostmatch"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

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
