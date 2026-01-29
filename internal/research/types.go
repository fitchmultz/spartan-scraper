// Package research provides multi-source research workflows for crawling, extracting, and clustering.
// It handles evidence aggregation, simhash deduplication, and clustering.
// It does NOT handle individual scraping or crawling (scrape/crawl packages do).
package research

import (
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
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
}

// Evidence represents a single piece of gathered evidence with computed metrics.
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

// Result contains the complete research output including summary, evidence, and clusters.
type Result struct {
	Query      string            `json:"query"`
	Summary    string            `json:"summary"`
	Evidence   []Evidence        `json:"evidence"`
	Clusters   []EvidenceCluster `json:"clusters"`
	Citations  []Citation        `json:"citations"`
	Confidence float64           `json:"confidence"`
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
