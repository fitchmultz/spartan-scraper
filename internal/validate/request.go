// Package validate provides request validators for scrape, crawl, and research operations.
// It handles validation of URLs, timeouts, depths, pages, and profile names.
// It does NOT define validation rules (validate.go does).
package validate

import "spartan-scraper/internal/apperrors"

type ScrapeRequestValidator struct {
	URL         string
	Timeout     int
	AuthProfile string
}

func (v ScrapeRequestValidator) Validate() error {
	if err := ValidateURL(v.URL); err != nil {
		return err
	}
	if err := ValidateTimeout(v.Timeout); err != nil {
		return err
	}
	if err := ValidateAuthProfileName(v.AuthProfile); err != nil {
		return err
	}
	return nil
}

type CrawlRequestValidator struct {
	URL         string
	MaxDepth    int
	MaxPages    int
	Timeout     int
	AuthProfile string
}

func (v CrawlRequestValidator) Validate() error {
	if err := ValidateURL(v.URL); err != nil {
		return err
	}
	if err := ValidateMaxDepth(v.MaxDepth); err != nil {
		return err
	}
	if err := ValidateMaxPages(v.MaxPages); err != nil {
		return err
	}
	if err := ValidateTimeout(v.Timeout); err != nil {
		return err
	}
	if err := ValidateAuthProfileName(v.AuthProfile); err != nil {
		return err
	}
	return nil
}

type ResearchRequestValidator struct {
	Query       string
	URLs        []string
	MaxDepth    int
	MaxPages    int
	Timeout     int
	AuthProfile string
}

func (v ResearchRequestValidator) Validate() error {
	if v.Query == "" {
		return apperrors.Validation("query is required")
	}
	if err := ValidateURLs(v.URLs); err != nil {
		return err
	}
	if err := ValidateMaxDepth(v.MaxDepth); err != nil {
		return err
	}
	if err := ValidateMaxPages(v.MaxPages); err != nil {
		return err
	}
	if err := ValidateTimeout(v.Timeout); err != nil {
		return err
	}
	if err := ValidateAuthProfileName(v.AuthProfile); err != nil {
		return err
	}
	return nil
}
