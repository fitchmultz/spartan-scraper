// Package research provides research functionality for Spartan Scraper.
//
// Purpose:
// - Run the agentic planning and synthesis workflow on top of gathered research evidence.
//
// Responsibilities:
// - Execute bounded follow-up rounds, call the AI extractor for planning and synthesis,
// - and normalize the resulting agentic metadata into the research result.
//
// Scope:
// - Agentic workflow orchestration only; prompt rendering and low-level helpers live in adjacent files.
//
// Usage:
// - Called internally when research requests enable agentic refinement.
//
// Invariants/Assumptions:
// - Agentic rounds must stay bounded by config.
// - Only candidate URLs derived from gathered evidence may be selected for follow-up.
package research

import (
	"context"
	"strconv"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func runAgenticResearch(ctx context.Context, req Request, docs []researchDocument, base Result) (*AgenticResearchResult, []researchDocument) {
	cfg := model.NormalizeResearchAgenticConfig(req.Agentic)
	if cfg == nil || !cfg.Enabled {
		return nil, docs
	}

	result := &AgenticResearchResult{
		Status:       agenticStatusSkipped,
		Instructions: cfg.Instructions,
	}
	if req.AIExtractor == nil {
		result.Status = agenticStatusFailed
		result.Error = "AI extractor not initialized"
		return result, docs
	}
	if len(docs) == 0 {
		result.Error = "no evidence gathered for agentic research"
		return result, docs
	}

	workingDocs := append([]researchDocument(nil), docs...)
	currentBase := base
	visited := map[string]struct{}{}
	for _, doc := range workingDocs {
		visited[doc.Evidence.URL] = struct{}{}
	}

	rounds := make([]AgenticResearchRound, 0, cfg.MaxRounds)
	selectedFollowUps := make([]string, 0, cfg.MaxRounds*cfg.MaxFollowUpURLs)

	for round := 1; round <= cfg.MaxRounds; round++ {
		candidates := collectCandidateURLs(workingDocs, visited)
		if len(candidates) == 0 {
			break
		}

		currentBase = buildResearchResult(req.Query, workingDocs)
		plan, err := planAgenticRound(ctx, req, cfg, currentBase, workingDocs, candidates)
		if err != nil {
			result.Status = agenticStatusFailed
			result.Error = err.Error()
			result.Rounds = rounds
			return result, workingDocs
		}

		selected := filterSelectedFollowUpURLs(plan.FollowUpURLs, candidates, cfg.MaxFollowUpURLs)
		roundResult := AgenticResearchRound{
			Round:        round,
			Goal:         plan.Objective,
			FocusAreas:   append([]string(nil), plan.FocusAreas...),
			SelectedURLs: append([]string(nil), selected...),
			Reasoning:    plan.Reasoning,
		}
		rounds = append(rounds, roundResult)
		if len(selected) == 0 {
			break
		}

		followUpDocs, successCount, _, err := gatherResearchDocuments(ctx, req, selected, 0, minInt(req.MaxPages, cfg.MaxFollowUpURLs))
		if err != nil && successCount == 0 {
			result.Status = agenticStatusFailed
			result.Error = err.Error()
			result.Rounds = rounds
			return result, workingDocs
		}
		if len(followUpDocs) == 0 {
			break
		}

		for _, selectedURL := range selected {
			visited[selectedURL] = struct{}{}
		}
		for _, doc := range followUpDocs {
			visited[doc.Evidence.URL] = struct{}{}
		}
		selectedFollowUps = appendUniqueStrings(selectedFollowUps, selected...)
		rounds[len(rounds)-1].AddedEvidenceCount = len(followUpDocs)
		workingDocs = append(workingDocs, followUpDocs...)
	}

	synthesis, err := synthesizeAgenticResearch(ctx, req, cfg, buildResearchResult(req.Query, workingDocs), workingDocs, rounds)
	if err != nil {
		result.Status = agenticStatusFailed
		result.Error = err.Error()
		result.FollowUpURLs = selectedFollowUps
		result.Rounds = rounds
		return result, workingDocs
	}

	result.Status = agenticStatusCompleted
	result.Objective = firstNonEmpty(synthesis.Objective, req.Query)
	result.Summary = synthesis.Summary
	result.FocusAreas = synthesis.FocusAreas
	if len(result.FocusAreas) == 0 && len(rounds) > 0 {
		for _, round := range rounds {
			result.FocusAreas = appendUniqueStrings(result.FocusAreas, round.FocusAreas...)
		}
	}
	result.KeyFindings = synthesis.KeyFindings
	result.OpenQuestions = synthesis.OpenQuestions
	result.RecommendedNextSteps = synthesis.RecommendedNextSteps
	result.FollowUpURLs = selectedFollowUps
	result.Rounds = rounds
	result.Confidence = synthesis.Confidence
	result.RouteID = synthesis.RouteID
	result.Provider = synthesis.Provider
	result.Model = synthesis.Model
	result.Cached = synthesis.Cached
	return result, workingDocs
}

func planAgenticRound(ctx context.Context, req Request, cfg *model.ResearchAgenticConfig, base Result, docs []researchDocument, candidates []string) (agenticPlan, error) {
	schema := map[string]interface{}{
		"objective":      req.Query,
		"focus_areas":    []string{"pricing model", "contract terms"},
		"follow_up_urls": []string{firstOrEmpty(candidates)},
		"reasoning":      "Need to inspect the pricing and support pages before drafting a final answer.",
	}
	result, err := req.AIExtractor.Extract(ctx, extract.AIExtractRequest{
		HTML:          renderAgenticPlanningHTML(req, cfg, base, docs, candidates),
		URL:           firstOrEmpty(req.URLs),
		Mode:          extract.AIModeSchemaGuided,
		SchemaExample: schema,
		Prompt: strings.TrimSpace(
			"Plan one bounded research follow-up round for the provided query. " +
				"Only choose follow_up_urls from the candidate URL list exactly as provided. " +
				"Prefer the smallest set of URLs that materially improves answer quality.",
		),
	})
	if err != nil {
		return agenticPlan{}, apperrors.Wrap(apperrors.KindInternal, "agentic planning failed", err)
	}

	return agenticPlan{
		Objective:    firstNonEmpty(stringField(result.Fields, "objective"), req.Query),
		FocusAreas:   stringSliceField(result.Fields, "focus_areas"),
		FollowUpURLs: stringSliceField(result.Fields, "follow_up_urls"),
		Reasoning:    stringField(result.Fields, "reasoning"),
		RouteID:      result.RouteID,
		Provider:     result.Provider,
		Model:        result.Model,
		Cached:       result.Cached,
		Confidence:   result.Confidence,
	}, nil
}

func synthesizeAgenticResearch(ctx context.Context, req Request, cfg *model.ResearchAgenticConfig, base Result, docs []researchDocument, rounds []AgenticResearchRound) (agenticSynthesis, error) {
	schema := map[string]interface{}{
		"summary":                "The company uses usage-based pricing with enterprise contracts and dedicated SLA-backed support.",
		"objective":              req.Query,
		"focus_areas":            []string{"pricing model", "support commitments"},
		"key_findings":           []string{"Pricing is handled through enterprise contracts, supported by the pricing and support pages."},
		"open_questions":         []string{"No public self-serve price points were found."},
		"recommended_next_steps": []string{"Verify current commercial terms directly with the vendor sales team."},
		"confidence":             "0.82",
	}
	result, err := req.AIExtractor.Extract(ctx, extract.AIExtractRequest{
		HTML:          renderAgenticSynthesisHTML(req, cfg, base, docs, rounds),
		URL:           firstOrEmpty(req.URLs),
		Mode:          extract.AIModeSchemaGuided,
		SchemaExample: schema,
		Prompt: strings.TrimSpace(
			"Synthesize a final research answer from the gathered evidence. " +
				"Base your response only on the supplied evidence and clearly note unresolved gaps.",
		),
	})
	if err != nil {
		return agenticSynthesis{}, apperrors.Wrap(apperrors.KindInternal, "agentic synthesis failed", err)
	}

	confidence := result.Confidence
	if rawConfidence := stringField(result.Fields, "confidence"); rawConfidence != "" {
		if parsed, parseErr := strconv.ParseFloat(strings.TrimSpace(rawConfidence), 64); parseErr == nil {
			confidence = parsed
		}
	}
	if confidence < 0 {
		confidence = 0
	}
	if confidence > 1 {
		confidence = 1
	}

	return agenticSynthesis{
		Summary:              stringField(result.Fields, "summary"),
		Objective:            firstNonEmpty(stringField(result.Fields, "objective"), req.Query),
		FocusAreas:           stringSliceField(result.Fields, "focus_areas"),
		KeyFindings:          stringSliceField(result.Fields, "key_findings"),
		OpenQuestions:        stringSliceField(result.Fields, "open_questions"),
		RecommendedNextSteps: stringSliceField(result.Fields, "recommended_next_steps"),
		RouteID:              result.RouteID,
		Provider:             result.Provider,
		Model:                result.Model,
		Cached:               result.Cached,
		Confidence:           confidence,
	}, nil
}
