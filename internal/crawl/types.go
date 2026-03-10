// Package crawl provides functionality for crawling multiple pages of a website.
// It implements a concurrent crawler that respects depth and page limits,
// avoids cycles by tracking visited URLs, supports incremental crawling
// using ETags and content hashes, and detects near-duplicate content using simhash.
package crawl

import (
	"context"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/dedup"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
	"github.com/fitchmultz/spartan-scraper/internal/webhook"
)

// Request represents a website crawl request.
type Request struct {
	URL              string
	RequestID        string
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
	Store            CrawlStateStore
	Registry         *pipeline.Registry
	JSRegistry       *pipeline.JSRegistry
	TemplateRegistry *extract.TemplateRegistry
	// MetricsCallback is called for each fetch operation to record metrics.
	MetricsCallback fetch.MetricsCallback
	// SitemapURL is an optional URL to a sitemap.xml file.
	// If provided, URLs from the sitemap will be used as crawl seeds.
	SitemapURL string
	// SitemapOnly indicates whether to only crawl URLs from the sitemap.
	// If false (default), the root URL is also crawled plus sitemap URLs.
	SitemapOnly bool
	// IncludePatterns are glob patterns for URL paths to include.
	// If specified, only URLs matching at least one pattern are crawled.
	// Supports * (matches any chars except /) and ** (matches any chars including /).
	IncludePatterns []string
	// ExcludePatterns are glob patterns for URL paths to exclude.
	// Excluded URLs take precedence over included ones.
	// Supports * (matches any chars except /) and ** (matches any chars including /).
	ExcludePatterns []string
	// Screenshot config for headless fetchers (chromedp, playwright).
	Screenshot *fetch.ScreenshotConfig
	// RobotsCache is an optional cache for robots.txt compliance checking.
	// If nil, robots.txt is not checked.
	RobotsCache *Cache
	// SkipDuplicates enables near-duplicate content detection during crawling.
	// When enabled, pages with content similar to already-crawled pages are marked as duplicates.
	SkipDuplicates bool
	// SimHashThreshold is the maximum Hamming distance for content to be considered a duplicate.
	// Default is 3. Lower values require more similarity (0 = exact match).
	SimHashThreshold int
	// CrossJobDedup enables cross-job duplicate detection using ContentIndex.
	// When enabled, the crawler queries the ContentIndex before fetching to detect
	// if similar content has been indexed by previous jobs.
	CrossJobDedup bool
	// CrossJobDedupThreshold is the Hamming distance threshold for cross-job duplicate detection.
	// Default is 3 (near-duplicates).
	CrossJobDedupThreshold int
	// ProxyPool for proxy rotation. If nil, no proxy pool is used.
	ProxyPool *fetch.ProxyPool
	// WebhookDispatcher is an optional dispatcher for page crawled events.
	// If nil, no webhook events are dispatched.
	WebhookDispatcher interface {
		Dispatch(ctx context.Context, url string, payload webhook.Payload, secret string)
	}
	// WebhookConfig holds webhook configuration for the crawl.
	WebhookConfig *model.WebhookConfig
	// AIExtractor for AI-powered extraction. If nil, AI extraction is disabled.
	AIExtractor *extract.AIExtractor
	// ContentIndex for cross-job deduplication. If nil, cross-job dedup is disabled.
	ContentIndex ContentIndex
}

// ContentIndex provides cross-job content deduplication capabilities.
// This is a subset of the dedup.ContentIndex interface for crawl's needs.
type ContentIndex interface {
	// Index stores a content fingerprint for a URL.
	Index(ctx context.Context, jobID, url string, simhash uint64) error
	// FindDuplicates returns URLs with similar content across all jobs.
	FindDuplicates(ctx context.Context, simhash uint64, threshold int) ([]dedup.DuplicateMatch, error)
}

// CrawlStateStore defines the interface for persisting and retrieving crawl states.
type CrawlStateStore interface {
	GetCrawlState(ctx context.Context, url string) (model.CrawlState, error)
	UpsertCrawlState(ctx context.Context, state model.CrawlState) error
}

// PageResult represents the scraping result for a single page during a crawl.
type PageResult struct {
	URL                string                     `json:"url"`
	Status             int                        `json:"status"`
	Title              string                     `json:"title"`
	Text               string                     `json:"text"`
	Links              []string                   `json:"links"`
	Metadata           extract.Result             `json:"metadata"` // Compatibility summary for existing result consumers.
	Extracted          extract.Extracted          `json:"extracted"`
	Normalized         extract.NormalizedDocument `json:"normalized"`
	SimHash            uint64                     `json:"simhash"`                      // Content fingerprint for duplicate detection
	DuplicateOf        string                     `json:"duplicateOf,omitempty"`        // URL of original page if this is a duplicate (same crawl)
	CrossJobDuplicates []CrossJobDuplicate        `json:"crossJobDuplicates,omitempty"` // Duplicates found in other jobs
}

// CrossJobDuplicate represents a duplicate match from a different job.
type CrossJobDuplicate struct {
	JobID     string `json:"jobId"`
	URL       string `json:"url"`
	Distance  int    `json:"distance"`
	IndexedAt string `json:"indexedAt"`
}

// task represents a single crawl task.
type task struct {
	URL   string
	Depth int
}
