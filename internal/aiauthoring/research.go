// Package aiauthoring provides aiauthoring functionality for Spartan Scraper.
//
// Purpose:
// - Implement research support for package aiauthoring.
//
// Responsibilities:
// - Define the file-local types, functions, and helpers that belong to this package concern.
//
// Scope:
// - Package-internal behavior owned by this file; broader orchestration stays in adjacent package files.
//
// Usage:
// - Used by other files in package `aiauthoring` and any exported callers that depend on this package.
//
// Invariants/Assumptions:
// - This file should preserve the package contract and rely on surrounding package configuration as the source of truth.

package aiauthoring

import (
	"context"
	"fmt"
	"strings"

	piai "github.com/fitchmultz/spartan-scraper/internal/ai"
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/research"
)

type ResearchRefineRequest struct {
	Result       research.Result
	Instructions string
}

type ResearchRefineInputStats struct {
	EvidenceCount     int  `json:"evidenceCount"`
	EvidenceUsedCount int  `json:"evidenceUsedCount"`
	ClusterCount      int  `json:"clusterCount"`
	CitationCount     int  `json:"citationCount"`
	HasAgentic        bool `json:"hasAgentic"`
}

type ResearchRefineResult struct {
	Issues      []string                    `json:"issues,omitempty"`
	InputStats  ResearchRefineInputStats    `json:"inputStats"`
	Refined     piai.ResearchRefinedContent `json:"refined"`
	Markdown    string                      `json:"markdown"`
	Explanation string                      `json:"explanation,omitempty"`
	RouteID     string                      `json:"route_id,omitempty"`
	Provider    string                      `json:"provider,omitempty"`
	Model       string                      `json:"model,omitempty"`
}

type researchInputDiagnostics struct {
	Issues        []string
	EvidenceByURL map[string]research.Evidence
	CitationURLs  map[string]struct{}
}

func (s *Service) RefineResearch(ctx context.Context, req ResearchRefineRequest) (ResearchRefineResult, error) {
	if err := s.requireAutomationClient(); err != nil {
		return ResearchRefineResult{}, err
	}
	if err := validateResearchRefineInput(req.Result); err != nil {
		return ResearchRefineResult{}, err
	}

	ctx, cancel := s.withRequestTimeout(ctx)
	defer cancel()

	diagnostics := analyzeResearchInput(req.Result)
	aiReq := piai.ResearchRefineRequest{
		Result:       bridgeResearchResult(req.Result),
		Instructions: strings.TrimSpace(req.Instructions),
	}

	for attempt := 0; attempt < 2; attempt++ {
		aiResult, err := s.automationClient.GenerateResearchRefinement(ctx, aiReq)
		if err != nil {
			return ResearchRefineResult{}, apperrors.Wrap(apperrors.KindInternal, "AI research refinement failed", err)
		}

		candidate := normalizeResearchRefinement(aiResult.Refined, diagnostics.EvidenceByURL)
		issues := validateResearchRefinementCandidate(candidate, diagnostics)
		if len(issues) == 0 {
			return ResearchRefineResult{
				Issues: dedupeIssues(diagnostics.Issues),
				InputStats: ResearchRefineInputStats{
					EvidenceCount:     len(req.Result.Evidence),
					EvidenceUsedCount: countUsedEvidence(candidate.EvidenceHighlights),
					ClusterCount:      len(req.Result.Clusters),
					CitationCount:     len(req.Result.Citations),
					HasAgentic:        req.Result.Agentic != nil,
				},
				Refined:     candidate,
				Markdown:    renderResearchRefinementMarkdown(req.Result.Query, candidate),
				Explanation: strings.TrimSpace(aiResult.Explanation),
				RouteID:     aiResult.RouteID,
				Provider:    aiResult.Provider,
				Model:       aiResult.Model,
			}, nil
		}
		if attempt == 1 {
			return ResearchRefineResult{}, apperrors.Validation(strings.Join(issues, "; "))
		}
		aiReq.Feedback = joinFeedback(aiReq.Feedback, buildResearchRefinementFeedback(candidate, issues))
	}

	return ResearchRefineResult{}, apperrors.Internal("AI research refinement failed")
}

func validateResearchRefineInput(result research.Result) error {
	if strings.TrimSpace(result.Query) == "" && strings.TrimSpace(result.Summary) == "" && len(result.Evidence) == 0 && result.Agentic == nil {
		return apperrors.Validation("result must include at least one of query, summary, evidence, or agentic synthesis")
	}
	return nil
}

func analyzeResearchInput(result research.Result) researchInputDiagnostics {
	issues := []string{}
	evidenceByURL := make(map[string]research.Evidence, len(result.Evidence))
	citationURLs := map[string]struct{}{}
	duplicateEvidenceURLs := map[string]int{}
	missingTitles := 0
	missingSnippets := 0

	if strings.TrimSpace(result.Query) == "" {
		issues = append(issues, "research query is missing")
	}
	if strings.TrimSpace(result.Summary) == "" {
		issues = append(issues, "research summary is missing")
	}
	if len(result.Evidence) == 0 {
		issues = append(issues, "research result contains no evidence items")
	}

	for _, item := range result.Evidence {
		url := strings.TrimSpace(item.URL)
		if url == "" {
			issues = append(issues, "an evidence item is missing its url")
			continue
		}
		if _, exists := evidenceByURL[url]; exists {
			duplicateEvidenceURLs[url]++
			continue
		}
		evidenceByURL[url] = item
		if strings.TrimSpace(item.Title) == "" {
			missingTitles++
		}
		if strings.TrimSpace(item.Snippet) == "" {
			missingSnippets++
		}
		if trimmedCitation := strings.TrimSpace(item.CitationURL); trimmedCitation != "" {
			citationURLs[trimmedCitation] = struct{}{}
		}
	}

	for url := range duplicateEvidenceURLs {
		issues = append(issues, fmt.Sprintf("duplicate evidence URL detected: %s", url))
	}
	if missingTitles > 0 {
		issues = append(issues, fmt.Sprintf("%d evidence item(s) are missing titles", missingTitles))
	}
	if missingSnippets > 0 {
		issues = append(issues, fmt.Sprintf("%d evidence item(s) are missing snippets", missingSnippets))
	}

	for _, citation := range result.Citations {
		if canonical := strings.TrimSpace(citation.Canonical); canonical != "" {
			citationURLs[canonical] = struct{}{}
		}
		if url := strings.TrimSpace(citation.URL); url != "" {
			citationURLs[url] = struct{}{}
		}
	}

	return researchInputDiagnostics{
		Issues:        dedupeIssues(issues),
		EvidenceByURL: evidenceByURL,
		CitationURLs:  citationURLs,
	}
}

func normalizeResearchRefinement(candidate piai.ResearchRefinedContent, evidenceByURL map[string]research.Evidence) piai.ResearchRefinedContent {
	candidate.Summary = strings.TrimSpace(candidate.Summary)
	candidate.ConciseSummary = strings.TrimSpace(candidate.ConciseSummary)
	candidate.KeyFindings = trimStringList(candidate.KeyFindings)
	candidate.OpenQuestions = trimStringList(candidate.OpenQuestions)
	candidate.RecommendedNextSteps = trimStringList(candidate.RecommendedNextSteps)
	if len(candidate.EvidenceHighlights) > 0 {
		normalizedHighlights := make([]piai.ResearchEvidenceHighlight, 0, len(candidate.EvidenceHighlights))
		for _, highlight := range candidate.EvidenceHighlights {
			highlight.URL = strings.TrimSpace(highlight.URL)
			highlight.Title = strings.TrimSpace(highlight.Title)
			highlight.Finding = strings.TrimSpace(highlight.Finding)
			highlight.Relevance = strings.TrimSpace(highlight.Relevance)
			highlight.CitationURL = strings.TrimSpace(highlight.CitationURL)
			if item, ok := evidenceByURL[highlight.URL]; ok {
				if highlight.Title == "" {
					highlight.Title = strings.TrimSpace(item.Title)
				}
				if highlight.CitationURL == "" {
					highlight.CitationURL = strings.TrimSpace(item.CitationURL)
				}
			}
			normalizedHighlights = append(normalizedHighlights, highlight)
		}
		candidate.EvidenceHighlights = normalizedHighlights
	}
	return candidate
}

func validateResearchRefinementCandidate(candidate piai.ResearchRefinedContent, diagnostics researchInputDiagnostics) []string {
	issues := []string{}
	if strings.TrimSpace(candidate.Summary) == "" {
		issues = append(issues, "refined.summary is required")
	}
	if strings.TrimSpace(candidate.ConciseSummary) == "" {
		issues = append(issues, "refined.conciseSummary is required")
	}
	if len(candidate.KeyFindings) == 0 {
		issues = append(issues, "refined.keyFindings must include at least one item")
	}
	if candidate.Confidence < 0 || candidate.Confidence > 1 {
		issues = append(issues, "refined.confidence must be between 0 and 1")
	}
	if len(diagnostics.EvidenceByURL) > 0 && len(candidate.EvidenceHighlights) == 0 {
		issues = append(issues, "refined.evidenceHighlights must reference at least one supplied evidence item")
	}

	seenURLs := map[string]struct{}{}
	for idx, highlight := range candidate.EvidenceHighlights {
		label := fmt.Sprintf("refined.evidenceHighlights[%d]", idx)
		if strings.TrimSpace(highlight.URL) == "" {
			issues = append(issues, label+".url is required")
			continue
		}
		if _, ok := diagnostics.EvidenceByURL[highlight.URL]; !ok {
			issues = append(issues, fmt.Sprintf("%s.url must reference a supplied evidence URL", label))
		}
		if strings.TrimSpace(highlight.Finding) == "" {
			issues = append(issues, label+".finding is required")
		}
		if highlight.CitationURL != "" {
			if _, ok := diagnostics.CitationURLs[highlight.CitationURL]; !ok {
				issues = append(issues, fmt.Sprintf("%s.citationUrl must reference a supplied citation or evidence citation URL", label))
			}
		}
		if _, seen := seenURLs[highlight.URL]; seen {
			issues = append(issues, fmt.Sprintf("duplicate evidence highlight URL detected: %s", highlight.URL))
			continue
		}
		seenURLs[highlight.URL] = struct{}{}
	}

	return dedupeIssues(issues)
}

func buildResearchRefinementFeedback(candidate piai.ResearchRefinedContent, issues []string) string {
	parts := []string{
		"The previous research refinement did not validate. Fix these issues: " + strings.Join(issues, "; "),
	}
	if strings.TrimSpace(candidate.Summary) != "" {
		parts = append(parts, "Previous summary: "+candidate.Summary)
	}
	if len(candidate.EvidenceHighlights) > 0 {
		urls := make([]string, 0, len(candidate.EvidenceHighlights))
		for _, highlight := range candidate.EvidenceHighlights {
			if strings.TrimSpace(highlight.URL) != "" {
				urls = append(urls, highlight.URL)
			}
		}
		if len(urls) > 0 {
			parts = append(parts, "Previous highlighted URLs: "+strings.Join(urls, ", "))
		}
	}
	return strings.Join(parts, "\n\n")
}

func countUsedEvidence(highlights []piai.ResearchEvidenceHighlight) int {
	if len(highlights) == 0 {
		return 0
	}
	seen := map[string]struct{}{}
	for _, highlight := range highlights {
		url := strings.TrimSpace(highlight.URL)
		if url == "" {
			continue
		}
		seen[url] = struct{}{}
	}
	return len(seen)
}

func renderResearchRefinementMarkdown(query string, refined piai.ResearchRefinedContent) string {
	lines := []string{"# Refined Research Brief", ""}
	if trimmedQuery := strings.TrimSpace(query); trimmedQuery != "" {
		lines = append(lines, "## Query", "", trimmedQuery, "")
	}
	lines = append(lines, "## Summary", "", refined.Summary, "")
	lines = append(lines, "## Concise Summary", "", refined.ConciseSummary, "")
	appendMarkdownList(&lines, "## Key Findings", refined.KeyFindings)
	if len(refined.EvidenceHighlights) > 0 {
		lines = append(lines, "## Evidence Highlights", "")
		for _, highlight := range refined.EvidenceHighlights {
			label := highlight.Title
			if strings.TrimSpace(label) == "" {
				label = highlight.URL
			}
			line := fmt.Sprintf("- [%s](%s): %s", label, highlight.URL, highlight.Finding)
			if trimmed := strings.TrimSpace(highlight.Relevance); trimmed != "" {
				line += " — " + trimmed
			}
			if trimmed := strings.TrimSpace(highlight.CitationURL); trimmed != "" && trimmed != highlight.URL {
				line += fmt.Sprintf(" (citation: [%s](%s))", trimmed, trimmed)
			}
			lines = append(lines, line)
		}
		lines = append(lines, "")
	}
	appendMarkdownList(&lines, "## Open Questions", refined.OpenQuestions)
	appendMarkdownList(&lines, "## Recommended Next Steps", refined.RecommendedNextSteps)
	if refined.Confidence > 0 {
		lines = append(lines, "## Confidence", "", fmt.Sprintf("%.2f", refined.Confidence), "")
	}
	return strings.TrimSpace(strings.Join(lines, "\n")) + "\n"
}

func appendMarkdownList(lines *[]string, heading string, items []string) {
	if len(items) == 0 {
		return
	}
	*lines = append(*lines, heading, "")
	for _, item := range items {
		*lines = append(*lines, "- "+item)
	}
	*lines = append(*lines, "")
}

func bridgeResearchResult(result research.Result) piai.BridgeResearchResult {
	out := piai.BridgeResearchResult{
		Query:      strings.TrimSpace(result.Query),
		Summary:    strings.TrimSpace(result.Summary),
		Confidence: result.Confidence,
	}
	if len(result.Evidence) > 0 {
		out.Evidence = make([]piai.BridgeResearchEvidence, 0, len(result.Evidence))
		for _, item := range result.Evidence {
			out.Evidence = append(out.Evidence, bridgeResearchEvidence(item))
		}
	}
	if len(result.Clusters) > 0 {
		out.Clusters = make([]piai.BridgeResearchEvidenceCluster, 0, len(result.Clusters))
		for _, cluster := range result.Clusters {
			out.Clusters = append(out.Clusters, bridgeResearchCluster(cluster))
		}
	}
	if len(result.Citations) > 0 {
		out.Citations = make([]piai.BridgeResearchCitation, 0, len(result.Citations))
		for _, citation := range result.Citations {
			out.Citations = append(out.Citations, piai.BridgeResearchCitation{
				URL:       strings.TrimSpace(citation.URL),
				Anchor:    strings.TrimSpace(citation.Anchor),
				Canonical: strings.TrimSpace(citation.Canonical),
			})
		}
	}
	if result.Agentic != nil {
		rounds := make([]piai.BridgeResearchAgenticRound, 0, len(result.Agentic.Rounds))
		for _, round := range result.Agentic.Rounds {
			rounds = append(rounds, piai.BridgeResearchAgenticRound{
				Round:              round.Round,
				Goal:               strings.TrimSpace(round.Goal),
				FocusAreas:         trimStringList(round.FocusAreas),
				SelectedURLs:       trimStringList(round.SelectedURLs),
				AddedEvidenceCount: round.AddedEvidenceCount,
				Reasoning:          strings.TrimSpace(round.Reasoning),
			})
		}
		out.Agentic = &piai.BridgeResearchAgenticResult{
			Status:               strings.TrimSpace(result.Agentic.Status),
			Instructions:         strings.TrimSpace(result.Agentic.Instructions),
			Summary:              strings.TrimSpace(result.Agentic.Summary),
			Objective:            strings.TrimSpace(result.Agentic.Objective),
			FocusAreas:           trimStringList(result.Agentic.FocusAreas),
			KeyFindings:          trimStringList(result.Agentic.KeyFindings),
			OpenQuestions:        trimStringList(result.Agentic.OpenQuestions),
			RecommendedNextSteps: trimStringList(result.Agentic.RecommendedNextSteps),
			FollowUpURLs:         trimStringList(result.Agentic.FollowUpURLs),
			Rounds:               rounds,
			Confidence:           result.Agentic.Confidence,
			RouteID:              strings.TrimSpace(result.Agentic.RouteID),
			Provider:             strings.TrimSpace(result.Agentic.Provider),
			Model:                strings.TrimSpace(result.Agentic.Model),
			Cached:               result.Agentic.Cached,
			Error:                strings.TrimSpace(result.Agentic.Error),
		}
	}
	return out
}

func bridgeResearchCluster(cluster research.EvidenceCluster) piai.BridgeResearchEvidenceCluster {
	out := piai.BridgeResearchEvidenceCluster{
		ID:         strings.TrimSpace(cluster.ID),
		Label:      strings.TrimSpace(cluster.Label),
		Confidence: cluster.Confidence,
	}
	if len(cluster.Evidence) > 0 {
		out.Evidence = make([]piai.BridgeResearchEvidence, 0, len(cluster.Evidence))
		for _, item := range cluster.Evidence {
			out.Evidence = append(out.Evidence, bridgeResearchEvidence(item))
		}
	}
	return out
}

func bridgeResearchEvidence(item research.Evidence) piai.BridgeResearchEvidence {
	out := piai.BridgeResearchEvidence{
		URL:         strings.TrimSpace(item.URL),
		Title:       strings.TrimSpace(item.Title),
		Snippet:     strings.TrimSpace(item.Snippet),
		Score:       item.Score,
		SimHash:     item.SimHash,
		ClusterID:   strings.TrimSpace(item.ClusterID),
		Confidence:  item.Confidence,
		CitationURL: strings.TrimSpace(item.CitationURL),
	}
	if len(item.Fields) > 0 {
		out.Fields = make(map[string]piai.BridgeFieldValue, len(item.Fields))
		for name, field := range item.Fields {
			out.Fields[name] = piai.BridgeFieldValue{
				Values:    append([]string(nil), field.Values...),
				Source:    string(field.Source),
				RawObject: field.RawObject,
			}
		}
	}
	return out
}

func trimStringList(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
