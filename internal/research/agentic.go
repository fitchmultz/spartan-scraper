package research

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"log/slog"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/crawl"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/scrape"
)

const (
	agenticStatusCompleted = "completed"
	agenticStatusFailed    = "failed"
	agenticStatusSkipped   = "skipped"
)

type researchDocument struct {
	Evidence Evidence
	Links    []string
}

type agenticPlan struct {
	Objective    string
	FocusAreas   []string
	FollowUpURLs []string
	Reasoning    string
	RouteID      string
	Provider     string
	Model        string
	Cached       bool
	Confidence   float64
}

type agenticSynthesis struct {
	Summary              string
	Objective            string
	FocusAreas           []string
	KeyFindings          []string
	OpenQuestions        []string
	RecommendedNextSteps []string
	RouteID              string
	Provider             string
	Model                string
	Cached               bool
	Confidence           float64
}

func gatherResearchDocuments(ctx context.Context, req Request, targets []string, maxDepth int, maxPages int) ([]researchDocument, int, int, error) {
	items := make([]researchDocument, 0)
	queryTokens := tokenize(req.Query)
	var successCount, failCount int

	for _, target := range targets {
		if ctx.Err() != nil {
			return nil, successCount, failCount, apperrors.Wrap(apperrors.KindInternal, "research cancelled", ctx.Err())
		}
		if strings.TrimSpace(target) == "" {
			continue
		}

		if maxDepth > 0 {
			slog.Debug("research crawling target", "url", apperrors.SanitizeURL(target), "maxDepth", maxDepth)
			pages, err := crawl.Run(ctx, crawl.Request{
				URL:              target,
				RequestID:        req.RequestID,
				MaxDepth:         maxDepth,
				MaxPages:         maxPages,
				Concurrency:      req.Concurrency,
				Headless:         req.Headless,
				UsePlaywright:    req.UsePlaywright,
				Auth:             req.Auth,
				Extract:          req.Extract,
				Pipeline:         req.Pipeline,
				Timeout:          req.Timeout,
				UserAgent:        req.UserAgent,
				Limiter:          req.Limiter,
				MaxRetries:       req.MaxRetries,
				RetryBase:        req.RetryBase,
				MaxResponseBytes: req.MaxResponseBytes,
				DataDir:          req.DataDir,
				Store:            req.Store,
				Registry:         req.Registry,
				JSRegistry:       req.JSRegistry,
				TemplateRegistry: req.TemplateRegistry,
				Screenshot:       req.Screenshot,
				Device:           req.Device,
				NetworkIntercept: req.NetworkIntercept,
				ProxyPool:        req.ProxyPool,
				AIExtractor:      req.AIExtractor,
			})
			if err != nil {
				if ctx.Err() != nil {
					return nil, successCount, failCount, apperrors.Wrap(apperrors.KindInternal, "research cancelled", ctx.Err())
				}
				slog.Error("research crawl failed", "url", apperrors.SanitizeURL(target), "error", err)
				failCount++
				continue
			}
			successCount++
			for _, page := range pages {
				if page.Status == 304 {
					continue
				}
				fields := cloneEvidenceFields(page.Normalized.Fields)
				searchText := evidenceSearchText(page.Normalized.Title, page.Normalized.Text, fields)
				items = append(items, researchDocument{
					Evidence: Evidence{
						URL:     page.URL,
						Title:   page.Normalized.Title,
						Snippet: makeEvidenceSnippet(page.Normalized.Text, fields),
						Score:   scoreText(queryTokens, searchText),
						Fields:  fields,
					},
					Links: normalizeDocumentLinks(page.URL, page.Normalized.Links),
				})
			}
			continue
		}

		slog.Debug("research scraping target", "url", apperrors.SanitizeURL(target))
		res, err := scrape.Run(ctx, scrape.Request{
			URL:              target,
			RequestID:        req.RequestID,
			Headless:         req.Headless,
			UsePlaywright:    req.UsePlaywright,
			Auth:             req.Auth,
			Extract:          req.Extract,
			Pipeline:         req.Pipeline,
			Timeout:          req.Timeout,
			UserAgent:        req.UserAgent,
			Limiter:          req.Limiter,
			MaxRetries:       req.MaxRetries,
			RetryBase:        req.RetryBase,
			MaxResponseBytes: req.MaxResponseBytes,
			DataDir:          req.DataDir,
			Store:            req.Store,
			Registry:         req.Registry,
			JSRegistry:       req.JSRegistry,
			TemplateRegistry: req.TemplateRegistry,
			Screenshot:       req.Screenshot,
			Device:           req.Device,
			NetworkIntercept: req.NetworkIntercept,
			ProxyPool:        req.ProxyPool,
			AIExtractor:      req.AIExtractor,
		})
		if err != nil {
			if ctx.Err() != nil {
				return nil, successCount, failCount, apperrors.Wrap(apperrors.KindInternal, "research cancelled", ctx.Err())
			}
			slog.Error("research scrape failed", "url", apperrors.SanitizeURL(target), "error", err)
			failCount++
			continue
		}
		successCount++
		if res.Status == 304 {
			continue
		}
		fields := cloneEvidenceFields(res.Normalized.Fields)
		searchText := evidenceSearchText(res.Normalized.Title, res.Normalized.Text, fields)
		items = append(items, researchDocument{
			Evidence: Evidence{
				URL:     res.URL,
				Title:   res.Normalized.Title,
				Snippet: makeEvidenceSnippet(res.Normalized.Text, fields),
				Score:   scoreText(queryTokens, searchText),
				Fields:  fields,
			},
			Links: normalizeDocumentLinks(res.URL, res.Normalized.Links),
		})
	}

	return items, successCount, failCount, nil
}

func buildResearchResult(query string, docs []researchDocument) Result {
	items := make([]Evidence, 0, len(docs))
	for _, doc := range docs {
		items = append(items, doc.Evidence)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Score > items[j].Score
	})

	items = enrichEvidence(items)
	items = dedupEvidence(items, 3)
	clusters, items := clusterEvidence(items, 8, 1)
	citations := buildCitations(items)
	confidence := overallConfidence(items, clusters)
	summary := summarize(tokenize(query), items)

	return Result{
		Query:      query,
		Summary:    summary,
		Evidence:   items,
		Clusters:   clusters,
		Citations:  citations,
		Confidence: confidence,
	}
}

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

func collectCandidateURLs(docs []researchDocument, visited map[string]struct{}) []string {
	candidates := make([]string, 0)
	seen := map[string]struct{}{}
	for _, doc := range docs {
		for _, link := range doc.Links {
			if _, ok := visited[link]; ok {
				continue
			}
			if _, ok := seen[link]; ok {
				continue
			}
			seen[link] = struct{}{}
			candidates = append(candidates, link)
			if len(candidates) >= 50 {
				return candidates
			}
		}
	}
	return candidates
}

func filterSelectedFollowUpURLs(selected []string, candidates []string, maxURLs int) []string {
	candidateSet := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		candidateSet[candidate] = struct{}{}
	}
	filtered := make([]string, 0, minInt(len(selected), maxURLs))
	for _, raw := range selected {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		if _, ok := candidateSet[trimmed]; !ok {
			continue
		}
		filtered = appendUniqueStrings(filtered, trimmed)
		if len(filtered) >= maxURLs {
			break
		}
	}
	return filtered
}

func normalizeDocumentLinks(baseURL string, links []string) []string {
	if len(links) == 0 {
		return nil
	}
	out := make([]string, 0, len(links))
	seen := map[string]struct{}{}
	for _, raw := range links {
		normalized := normalizeFollowUpURL(baseURL, raw)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

func normalizeFollowUpURL(baseURL string, raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return ""
	}
	resolved := base.ResolveReference(parsed)
	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return ""
	}
	resolved.Fragment = ""
	return resolved.String()
}

func renderAgenticPlanningHTML(req Request, cfg *model.ResearchAgenticConfig, base Result, docs []researchDocument, candidates []string) string {
	var b strings.Builder
	b.WriteString("<html><body>")
	b.WriteString("<h1>Research planning bundle</h1>")
	b.WriteString(tag("p", "Query: "+req.Query))
	if cfg.Instructions != "" {
		b.WriteString(tag("p", "Operator instructions: "+cfg.Instructions))
	}
	b.WriteString(tag("p", fmt.Sprintf("Deterministic summary: %s", base.Summary)))
	b.WriteString("<h2>Evidence</h2><ol>")
	for i, doc := range docs {
		b.WriteString("<li>")
		b.WriteString(tag("strong", fmt.Sprintf("%d. %s", i+1, fallbackString(doc.Evidence.Title, doc.Evidence.URL))))
		b.WriteString(tag("p", doc.Evidence.URL))
		b.WriteString(tag("p", doc.Evidence.Snippet))
		if fieldSummary := summarizeEvidenceFields(doc.Evidence.Fields); fieldSummary != "" {
			b.WriteString(tag("p", fieldSummary))
		}
		b.WriteString("</li>")
		if i >= 11 {
			break
		}
	}
	b.WriteString("</ol>")
	b.WriteString("<h2>Candidate follow-up URLs</h2><ul>")
	for _, candidate := range candidates {
		b.WriteString(tag("li", candidate))
	}
	b.WriteString("</ul></body></html>")
	return b.String()
}

func renderAgenticSynthesisHTML(req Request, cfg *model.ResearchAgenticConfig, base Result, docs []researchDocument, rounds []AgenticResearchRound) string {
	var b strings.Builder
	b.WriteString("<html><body>")
	b.WriteString("<h1>Research synthesis bundle</h1>")
	b.WriteString(tag("p", "Query: "+req.Query))
	if cfg.Instructions != "" {
		b.WriteString(tag("p", "Operator instructions: "+cfg.Instructions))
	}
	b.WriteString(tag("p", fmt.Sprintf("Deterministic summary: %s", base.Summary)))
	if len(rounds) > 0 {
		b.WriteString("<h2>Follow-up rounds</h2><ol>")
		for _, round := range rounds {
			b.WriteString("<li>")
			b.WriteString(tag("p", fmt.Sprintf("Round %d goal: %s", round.Round, round.Goal)))
			if len(round.FocusAreas) > 0 {
				b.WriteString(tag("p", "Focus areas: "+strings.Join(round.FocusAreas, ", ")))
			}
			if len(round.SelectedURLs) > 0 {
				b.WriteString(tag("p", "Selected URLs: "+strings.Join(round.SelectedURLs, ", ")))
			}
			if round.Reasoning != "" {
				b.WriteString(tag("p", "Reasoning: "+round.Reasoning))
			}
			b.WriteString("</li>")
		}
		b.WriteString("</ol>")
	}
	b.WriteString("<h2>Evidence</h2><ol>")
	for i, doc := range docs {
		b.WriteString("<li>")
		b.WriteString(tag("strong", fmt.Sprintf("%d. %s", i+1, fallbackString(doc.Evidence.Title, doc.Evidence.URL))))
		b.WriteString(tag("p", doc.Evidence.URL))
		b.WriteString(tag("p", doc.Evidence.Snippet))
		if fieldSummary := summarizeEvidenceFields(doc.Evidence.Fields); fieldSummary != "" {
			b.WriteString(tag("p", fieldSummary))
		}
		b.WriteString("</li>")
		if i >= 15 {
			break
		}
	}
	b.WriteString("</ol></body></html>")
	return b.String()
}

func tag(name string, content string) string {
	return "<" + name + ">" + html.EscapeString(strings.TrimSpace(content)) + "</" + name + ">"
}

func stringField(fields map[string]extract.FieldValue, key string) string {
	field, ok := fields[key]
	if !ok {
		return ""
	}
	for _, value := range field.Values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	trimmed := strings.TrimSpace(field.RawObject)
	trimmed = strings.Trim(trimmed, `"`)
	return strings.TrimSpace(trimmed)
}

func stringSliceField(fields map[string]extract.FieldValue, key string) []string {
	field, ok := fields[key]
	if !ok {
		return nil
	}
	if len(field.Values) > 0 {
		return appendUniqueStrings(nil, field.Values...)
	}
	if strings.TrimSpace(field.RawObject) == "" {
		return nil
	}
	var decoded []string
	if err := json.Unmarshal([]byte(field.RawObject), &decoded); err == nil {
		return appendUniqueStrings(nil, decoded...)
	}
	return appendUniqueStrings(nil, field.RawObject)
}

func appendUniqueStrings(dst []string, values ...string) []string {
	seen := make(map[string]struct{}, len(dst))
	for _, existing := range dst {
		if trimmed := strings.TrimSpace(existing); trimmed != "" {
			seen[trimmed] = struct{}{}
		}
	}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		dst = append(dst, trimmed)
	}
	return dst
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func firstOrEmpty(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func fallbackString(primary string, fallback string) string {
	if trimmed := strings.TrimSpace(primary); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(fallback)
}

func minInt(a int, b int) int {
	if a <= 0 {
		return b
	}
	if b <= 0 || a < b {
		return a
	}
	return b
}
