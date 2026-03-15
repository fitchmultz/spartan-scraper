// Package research provides multi-source research workflows for crawling, extracting, and clustering.
// It handles evidence aggregation, simhash deduplication, and clustering.
// It does NOT handle individual scraping or crawling (scrape/crawl packages do).
package research

import (
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
	"github.com/fitchmultz/spartan-scraper/internal/scrape"
)

// Request contains all parameters for a research operation.
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
	Store            scrape.CrawlStateStore
	Registry         *pipeline.Registry
	JSRegistry       *pipeline.JSRegistry
	TemplateRegistry *extract.TemplateRegistry
	// Screenshot config for headless fetchers (chromedp, playwright).
	Screenshot *fetch.ScreenshotConfig
	// Device emulation for responsive/mobile content.
	Device *fetch.DeviceEmulation
	// NetworkIntercept captures matching browser requests/responses during headless execution.
	NetworkIntercept *fetch.NetworkInterceptConfig
	// ProxyPool for proxy rotation. If nil, no proxy pool is used.
	ProxyPool *fetch.ProxyPool
	// AIExtractor enables optional AI-assisted extraction during evidence gathering.
	AIExtractor *extract.AIExtractor
	// Agentic enables optional bounded pi-powered follow-up and synthesis.
	Agentic *model.ResearchAgenticConfig
}

// Evidence represents a single piece of gathered evidence with computed metrics.
type Evidence struct {
	URL         string                        `json:"url"`
	Title       string                        `json:"title"`
	Snippet     string                        `json:"snippet"`
	Score       float64                       `json:"score"`
	SimHash     uint64                        `json:"simhash"`
	ClusterID   string                        `json:"clusterId"`
	Confidence  float64                       `json:"confidence"`
	CitationURL string                        `json:"citationUrl"`
	Fields      map[string]extract.FieldValue `json:"fields,omitempty"`
}

// Result contains the complete research output including summary, evidence, and clusters.
type Result struct {
	Query      string                 `json:"query"`
	Summary    string                 `json:"summary"`
	Evidence   []Evidence             `json:"evidence"`
	Clusters   []EvidenceCluster      `json:"clusters"`
	Citations  []Citation             `json:"citations"`
	Confidence float64                `json:"confidence"`
	Agentic    *AgenticResearchResult `json:"agentic,omitempty"`
}

// EvidenceCluster represents a group of similar evidence items.
type EvidenceCluster struct {
	ID         string     `json:"id"`
	Label      string     `json:"label"`
	Evidence   []Evidence `json:"evidence"`
	Confidence float64    `json:"confidence"`
}

// Citation represents a normalized reference to a source.
type Citation struct {
	URL       string `json:"url"`
	Anchor    string `json:"anchor,omitempty"`
	Canonical string `json:"canonical"`
}

// AgenticResearchRound records a bounded AI-guided follow-up round.
type AgenticResearchRound struct {
	Round              int      `json:"round"`
	Goal               string   `json:"goal,omitempty"`
	FocusAreas         []string `json:"focusAreas,omitempty"`
	SelectedURLs       []string `json:"selectedUrls,omitempty"`
	AddedEvidenceCount int      `json:"addedEvidenceCount,omitempty"`
	Reasoning          string   `json:"reasoning,omitempty"`
}

// AgenticResearchResult captures additive pi-guided research planning and synthesis.
type AgenticResearchResult struct {
	Status               string                 `json:"status"`
	Instructions         string                 `json:"instructions,omitempty"`
	Summary              string                 `json:"summary,omitempty"`
	Objective            string                 `json:"objective,omitempty"`
	FocusAreas           []string               `json:"focusAreas,omitempty"`
	KeyFindings          []string               `json:"keyFindings,omitempty"`
	OpenQuestions        []string               `json:"openQuestions,omitempty"`
	RecommendedNextSteps []string               `json:"recommendedNextSteps,omitempty"`
	FollowUpURLs         []string               `json:"followUpUrls,omitempty"`
	Rounds               []AgenticResearchRound `json:"rounds,omitempty"`
	Confidence           float64                `json:"confidence,omitempty"`
	RouteID              string                 `json:"route_id,omitempty"`
	Provider             string                 `json:"provider,omitempty"`
	Model                string                 `json:"model,omitempty"`
	Cached               bool                   `json:"cached,omitempty"`
	Error                string                 `json:"error,omitempty"`
}
