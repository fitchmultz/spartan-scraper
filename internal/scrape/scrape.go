package scrape

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"spartan-scraper/internal/extract"
	"spartan-scraper/internal/fetch"
	"spartan-scraper/internal/model"
)

type Request struct {
	URL           string
	Headless      bool
	UsePlaywright bool
	Auth          fetch.AuthOptions
	Extract       extract.ExtractOptions
	Timeout       time.Duration
	UserAgent     string
	Limiter       *fetch.HostLimiter
	MaxRetries    int
	RetryBase     time.Duration
	DataDir       string
	Incremental   bool
	Store         CrawlStateStore
}

type Result struct {
	URL        string                     `json:"url"`
	Status     int                        `json:"status"`
	Title      string                     `json:"title"`
	Text       string                     `json:"text"`
	Links      []string                   `json:"links"`
	Metadata   extract.Result             `json:"metadata"` // Legacy
	Extracted  extract.Extracted          `json:"extracted"`
	Normalized extract.NormalizedDocument `json:"normalized"`
}

type CrawlStateStore interface {
	GetCrawlState(url string) (model.CrawlState, error)
	UpsertCrawlState(state model.CrawlState) error
}

func Run(req Request) (Result, error) {
	var state model.CrawlState
	var ifNoneMatch, ifModifiedSince string

	if req.Incremental && req.Store != nil {
		existingState, err := req.Store.GetCrawlState(req.URL)
		if err == nil {
			state = existingState
			ifNoneMatch = state.ETag
			ifModifiedSince = state.LastModified
		}
	}

	fetcher := fetch.NewFetcher()

	fetchReq := fetch.Request{
		URL:             req.URL,
		Timeout:         req.Timeout,
		UserAgent:       req.UserAgent,
		Headless:        req.Headless,
		UsePlaywright:   req.UsePlaywright,
		Auth:            req.Auth,
		Limiter:         req.Limiter,
		MaxRetries:      req.MaxRetries,
		RetryBaseDelay:  req.RetryBase,
		DataDir:         req.DataDir,
		IfNoneMatch:     ifNoneMatch,
		IfModifiedSince: ifModifiedSince,
	}

	res, err := fetcher.Fetch(fetchReq)
	if err != nil {
		return Result{}, err
	}

	if res.Status == 304 {
		if req.Incremental && req.Store != nil {
			state.LastScraped = time.Now()
			_ = req.Store.UpsertCrawlState(state)
		}
		return Result{
			URL:    res.URL,
			Status: 304,
		}, nil
	}

	currentHash := sha256.Sum256([]byte(res.HTML))
	currentHashStr := hex.EncodeToString(currentHash[:])

	if req.Incremental && state.ContentHash == currentHashStr && state.ContentHash != "" {
		if req.Incremental && req.Store != nil {
			state.LastScraped = time.Now()
			_ = req.Store.UpsertCrawlState(state)
		}
		return Result{
			URL:    res.URL,
			Status: 200,
		}, nil
	}

	output, err := extract.Execute(extract.ExecuteInput{
		URL:     res.URL,
		HTML:    res.HTML,
		Options: req.Extract,
		DataDir: req.DataDir,
	})
	if err != nil {
		return Result{}, err
	}

	if req.Incremental && req.Store != nil {
		newState := model.CrawlState{
			URL:          req.URL,
			ETag:         res.ETag,
			LastModified: res.LastModified,
			ContentHash:  currentHashStr,
			LastScraped:  time.Now(),
		}
		_ = req.Store.UpsertCrawlState(newState)
	}

	return Result{
		URL:    res.URL,
		Status: res.Status,
		Title:  output.Normalized.Title,
		Text:   output.Normalized.Text,
		Links:  output.Normalized.Links,
		Metadata: extract.Result{
			Title:       output.Normalized.Title,
			Description: output.Normalized.Description,
			Text:        output.Normalized.Text,
			Links:       output.Normalized.Links,
		},
		Extracted:  output.Extracted,
		Normalized: output.Normalized,
	}, nil
}
