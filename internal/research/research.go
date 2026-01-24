package research

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log/slog"
	"math"
	"math/bits"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"spartan-scraper/internal/crawl"
	"spartan-scraper/internal/extract"
	"spartan-scraper/internal/fetch"
	"spartan-scraper/internal/model"
	"spartan-scraper/internal/pipeline"
	"spartan-scraper/internal/scrape"
)

type Request struct {
	Query            string
	RequestID        string
	URLs             []string
	MaxDepth         int
	MaxPages         int
	Concurrency      int
	Headless         bool
	UsePlaywright    bool
	Auth             fetch.AuthOptions
	Extract          extract.ExtractOptions
	Pipeline         pipeline.Options
	Timeout          time.Duration
	UserAgent        string
	Limiter          *fetch.HostLimiter
	MaxRetries       int
	RetryBase        time.Duration
	MaxResponseBytes int64
	DataDir          string
	Incremental      bool
	Store            scrape.CrawlStateStore
	Registry         *pipeline.Registry
	JSRegistry       *pipeline.JSRegistry
}

type Evidence struct {
	URL         string  `json:"url"`
	Title       string  `json:"title"`
	Snippet     string  `json:"snippet"`
	Score       float64 `json:"score"`
	SimHash     uint64  `json:"simhash"`
	ClusterID   string  `json:"clusterId"`
	Confidence  float64 `json:"confidence"`
	CitationURL string  `json:"citationUrl"`
}

type Result struct {
	Query      string            `json:"query"`
	Summary    string            `json:"summary"`
	Evidence   []Evidence        `json:"evidence"`
	Clusters   []EvidenceCluster `json:"clusters"`
	Citations  []Citation        `json:"citations"`
	Confidence float64           `json:"confidence"`
}

type EvidenceCluster struct {
	ID         string     `json:"id"`
	Label      string     `json:"label"`
	Evidence   []Evidence `json:"evidence"`
	Confidence float64    `json:"confidence"`
}

type Citation struct {
	URL       string `json:"url"`
	Anchor    string `json:"anchor,omitempty"`
	Canonical string `json:"canonical"`
}

func Run(ctx context.Context, req Request) (Result, error) {
	slog.Info("research.Run start", "query", req.Query, "urls", req.URLs)
	items := make([]Evidence, 0)
	queryTokens := tokenize(req.Query)

	for _, target := range req.URLs {
		if strings.TrimSpace(target) == "" {
			continue
		}

		if req.MaxDepth > 0 {
			slog.Debug("research crawling target", "url", target, "maxDepth", req.MaxDepth)
			pages, err := crawl.Run(ctx, crawl.Request{
				URL:              target,
				RequestID:        req.RequestID,
				MaxDepth:         req.MaxDepth,
				MaxPages:         req.MaxPages,
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
				Incremental:      req.Incremental,
				Store:            req.Store,
				Registry:         req.Registry,
				JSRegistry:       req.JSRegistry,
			})
			if err != nil {
				slog.Error("research crawl failed", "url", target, "error", err)
				continue
			}
			for _, page := range pages {
				if page.Status == 304 {
					continue
				}
				items = append(items, Evidence{
					URL:     page.URL,
					Title:   page.Title,
					Snippet: makeSnippet(page.Text),
					Score:   scoreText(queryTokens, page.Text),
				})
			}
		} else {
			slog.Debug("research scraping target", "url", target)
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
				Incremental:      req.Incremental,
				Store:            req.Store,
				Registry:         req.Registry,
				JSRegistry:       req.JSRegistry,
			})
			if err != nil {
				slog.Error("research scrape failed", "url", target, "error", err)
				continue
			}
			if res.Status != 304 {
				items = append(items, Evidence{
					URL:     res.URL,
					Title:   res.Title,
					Snippet: makeSnippet(res.Text),
					Score:   scoreText(queryTokens, res.Text),
				})
			}
		}
	}

	slog.Info("research gathering complete", "evidenceCount", len(items))
	sort.Slice(items, func(i, j int) bool {
		return items[i].Score > items[j].Score
	})

	items = enrichEvidence(items)
	items = dedupEvidence(items, 3)
	clusters, items := clusterEvidence(items, 8, 1)
	citations := buildCitations(items)
	confidence := overallConfidence(items, clusters)

	summary := summarize(queryTokens, items)
	result := Result{
		Query:      req.Query,
		Summary:    summary,
		Evidence:   items,
		Clusters:   clusters,
		Citations:  citations,
		Confidence: confidence,
	}

	registry := req.Registry
	if registry == nil {
		registry = pipeline.NewRegistry()
	}
	target := pipeline.NewTarget("", string(model.KindResearch))
	baseCtx := pipeline.HookContext{
		Context:     ctx,
		RequestID:   req.RequestID,
		Target:      target,
		Now:         time.Now(),
		DataDir:     req.DataDir,
		Options:     req.Pipeline,
		Attributes:  map[string]string{},
		Diagnostics: map[string]any{},
	}
	slog.Info("research complete", "confidence", result.Confidence)
	return applyResearchOutputPipeline(ctx, registry, baseCtx, result)
}

func tokenize(query string) []string {
	clean := strings.ToLower(query)
	re := regexp.MustCompile(`[^a-z0-9\s]+`)
	clean = re.ReplaceAllString(clean, " ")
	parts := strings.Fields(clean)
	uniq := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		if part == "" || seen[part] {
			continue
		}
		seen[part] = true
		uniq = append(uniq, part)
	}
	return uniq
}

func scoreText(tokens []string, text string) float64 {
	lower := strings.ToLower(text)
	score := 0.0
	for _, token := range tokens {
		score += float64(strings.Count(lower, token))
	}
	return score
}

func makeSnippet(text string) string {
	trimmed := strings.TrimSpace(text)
	if len(trimmed) <= 300 {
		return trimmed
	}
	return trimmed[:300] + "..."
}

func summarize(tokens []string, items []Evidence) string {
	if len(items) == 0 {
		return "No evidence gathered."
	}

	max := 5
	if len(items) < max {
		max = len(items)
	}

	sentences := make([]string, 0, len(items))
	for _, item := range items {
		sentences = append(sentences, splitSentences(item.Snippet)...)
		if len(sentences) > 40 {
			break
		}
	}

	scored := make([]scoredSentence, 0, len(sentences))
	for _, sentence := range sentences {
		scored = append(scored, scoredSentence{
			Text:  sentence,
			Score: scoreText(tokens, sentence),
		})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	selected := make([]string, 0, max)
	for i := 0; i < len(scored) && len(selected) < max; i++ {
		if strings.TrimSpace(scored[i].Text) == "" {
			continue
		}
		selected = append(selected, scored[i].Text)
	}

	if len(selected) == 0 {
		return items[0].Snippet
	}
	return strings.Join(selected, " ")
}

type scoredSentence struct {
	Text  string
	Score float64
}

func splitSentences(text string) []string {
	parts := regexp.MustCompile(`[.!?]+`).Split(text, -1)
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trim := strings.TrimSpace(part)
		if trim != "" {
			out = append(out, trim+".")
		}
	}
	return out
}

func enrichEvidence(items []Evidence) []Evidence {
	if len(items) == 0 {
		return items
	}
	maxScore := 0.0
	for _, item := range items {
		if item.Score > maxScore {
			maxScore = item.Score
		}
	}

	out := make([]Evidence, 0, len(items))
	for _, item := range items {
		text := strings.TrimSpace(item.Title + " " + item.Snippet)
		item.SimHash = computeSimHash(text)
		citation := normalizeCitation(item.URL, item.Snippet, item.Title)
		item.CitationURL = buildCitationURL(citation.Canonical, citation.Anchor)
		item.Confidence = evidenceConfidence(item, maxScore)
		out = append(out, item)
	}
	return out
}

func computeSimHash(text string) uint64 {
	tokens := tokenize(text)
	if len(tokens) == 0 {
		return 0
	}
	var weights [64]int
	for _, token := range tokens {
		h := fnv.New64a()
		_, _ = h.Write([]byte(token))
		hash := h.Sum64()
		for i := 0; i < 64; i++ {
			if hash&(1<<i) != 0 {
				weights[i]++
			} else {
				weights[i]--
			}
		}
	}
	var out uint64
	for i := 0; i < 64; i++ {
		if weights[i] >= 0 {
			out |= 1 << i
		}
	}
	return out
}

func hammingDistance(a uint64, b uint64) int {
	return bits.OnesCount64(a ^ b)
}

func dedupEvidence(items []Evidence, maxDistance int) []Evidence {
	if len(items) == 0 {
		return items
	}
	out := make([]Evidence, 0, len(items))
	for _, item := range items {
		duplicate := false
		for _, existing := range out {
			if hammingDistance(item.SimHash, existing.SimHash) <= maxDistance {
				duplicate = true
				break
			}
		}
		if !duplicate {
			out = append(out, item)
		}
	}
	return out
}

func clusterEvidence(items []Evidence, maxDistance int, minSize int) ([]EvidenceCluster, []Evidence) {
	if len(items) == 0 {
		return []EvidenceCluster{}, items
	}
	type cluster struct {
		id       string
		evidence []Evidence
	}
	clusters := make([]cluster, 0)

	for _, item := range items {
		placed := false
		for i := range clusters {
			for _, member := range clusters[i].evidence {
				if hammingDistance(item.SimHash, member.SimHash) <= maxDistance {
					clusters[i].evidence = append(clusters[i].evidence, item)
					placed = true
					break
				}
			}
			if placed {
				break
			}
		}
		if !placed {
			clusters = append(clusters, cluster{
				id:       fmtClusterID(len(clusters) + 1),
				evidence: []Evidence{item},
			})
		}
	}

	enriched := make([]Evidence, 0, len(items))
	finalClusters := make([]EvidenceCluster, 0, len(clusters))
	for _, c := range clusters {
		for i := range c.evidence {
			c.evidence[i].ClusterID = c.id
			enriched = append(enriched, c.evidence[i])
		}
		label := clusterLabel(c.evidence)
		conf := clusterConfidence(c.evidence)
		if minSize <= 1 || len(c.evidence) >= minSize {
			finalClusters = append(finalClusters, EvidenceCluster{
				ID:         c.id,
				Label:      label,
				Evidence:   c.evidence,
				Confidence: conf,
			})
		}
	}

	sort.Slice(finalClusters, func(i, j int) bool {
		return finalClusters[i].Confidence > finalClusters[j].Confidence
	})

	return finalClusters, enriched
}

func fmtClusterID(index int) string {
	return "cluster-" + strconv.Itoa(index)
}

func clusterLabel(items []Evidence) string {
	if len(items) == 0 {
		return ""
	}
	if strings.TrimSpace(items[0].Title) != "" {
		return items[0].Title
	}
	return hostFromURL(items[0].URL)
}

func hostFromURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return raw
	}
	return parsed.Host
}

func buildCitations(items []Evidence) []Citation {
	seen := map[string]bool{}
	out := make([]Citation, 0, len(items))
	for _, item := range items {
		citation := normalizeCitation(item.URL, item.Snippet, item.Title)
		key := citation.Canonical + "#" + citation.Anchor
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, citation)
	}
	return out
}

func normalizeCitation(rawURL string, snippet string, title string) Citation {
	canonical := canonicalizeURL(rawURL)
	anchor := citationAnchor(snippet, title)
	return Citation{
		URL:       rawURL,
		Anchor:    anchor,
		Canonical: canonical,
	}
}

func buildCitationURL(canonical string, anchor string) string {
	if canonical == "" {
		return ""
	}
	if anchor == "" {
		return canonical
	}
	return canonical + "#" + anchor
}

func canonicalizeURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return raw
	}
	parsed.Fragment = ""
	return parsed.String()
}

func citationAnchor(snippet string, title string) string {
	base := strings.TrimSpace(snippet)
	if base == "" {
		base = strings.TrimSpace(title)
	}
	if base == "" {
		return ""
	}
	re := regexp.MustCompile(`[^a-z0-9\s]+`)
	clean := re.ReplaceAllString(strings.ToLower(base), " ")
	words := strings.Fields(clean)
	if len(words) == 0 {
		return ""
	}
	if len(words) > 8 {
		words = words[:8]
	}
	return strings.Join(words, "-")
}

func evidenceConfidence(item Evidence, maxScore float64) float64 {
	if maxScore <= 0 {
		return 0
	}
	scoreFactor := math.Log1p(item.Score) / math.Log1p(maxScore)
	lengthFactor := 0.0
	if len(item.Snippet) > 0 {
		lengthFactor = math.Min(float64(len(item.Snippet))/300.0, 1.0)
	}
	return clamp01(0.7*scoreFactor + 0.3*lengthFactor)
}

func clusterConfidence(items []Evidence) float64 {
	if len(items) == 0 {
		return 0
	}
	sum := 0.0
	for _, item := range items {
		sum += item.Confidence
	}
	return clamp01(sum / float64(len(items)))
}

func overallConfidence(items []Evidence, clusters []EvidenceCluster) float64 {
	if len(items) == 0 {
		return 0
	}
	sum := 0.0
	for _, item := range items {
		sum += item.Confidence
	}
	evidenceScore := sum / float64(len(items))

	clusterScore := 0.0
	if len(clusters) > 0 {
		for _, cluster := range clusters {
			clusterScore += cluster.Confidence
		}
		clusterScore /= float64(len(clusters))
	}

	return clamp01(0.6*evidenceScore + 0.4*clusterScore)
}

func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func applyResearchOutputPipeline(ctx context.Context, registry *pipeline.Registry, baseCtx pipeline.HookContext, result Result) (Result, error) {
	raw, _ := json.Marshal(result)
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
		return Result{}, fmt.Errorf("pipeline output type mismatch for research")
	}
	return typed, nil
}
