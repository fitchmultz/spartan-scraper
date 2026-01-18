package scrape

import (
	"time"

	"spartan-scraper/internal/extract"
	"spartan-scraper/internal/fetch"
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

func Run(req Request) (Result, error) {
	fetcher := fetch.NewFetcher()

	res, err := fetcher.Fetch(fetch.Request{
		URL:            req.URL,
		Timeout:        req.Timeout,
		UserAgent:      req.UserAgent,
		Headless:       req.Headless,
		UsePlaywright:  req.UsePlaywright,
		Auth:           req.Auth,
		Limiter:        req.Limiter,
		MaxRetries:     req.MaxRetries,
		RetryBaseDelay: req.RetryBase,
		DataDir:        req.DataDir,
	})
	if err != nil {
		return Result{}, err
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
