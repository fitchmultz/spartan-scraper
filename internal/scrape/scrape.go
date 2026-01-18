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
	Timeout       time.Duration
	UserAgent     string
	Limiter       *fetch.HostLimiter
	MaxRetries    int
	RetryBase     time.Duration
}

type Result struct {
	URL      string         `json:"url"`
	Status   int            `json:"status"`
	Title    string         `json:"title"`
	Text     string         `json:"text"`
	Links    []string       `json:"links"`
	Metadata extract.Result `json:"metadata"`
}

func Run(req Request) (Result, error) {
	fetcher := fetch.NewFetcher(req.Headless, req.UsePlaywright)

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
	})
	if err != nil {
		return Result{}, err
	}

	extracted, err := extract.FromHTML(res.HTML)
	if err != nil {
		return Result{}, err
	}

	return Result{
		URL:      res.URL,
		Status:   res.Status,
		Title:    extracted.Title,
		Text:     extracted.Text,
		Links:    extracted.Links,
		Metadata: extracted,
	}, nil
}
