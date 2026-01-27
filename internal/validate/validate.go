// Package validate provides request validators for scrape, crawl, and research operations.
// It handles validation of URLs, timeouts, depths, pages, and profile names.
// It does NOT define validation rules (validate.go does).
package validate

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
)

var profileNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

var (
	ErrInvalidURLScheme    = errors.New("invalid url: must be http or https and have a host")
	ErrInvalidURLHost      = errors.New("invalid url: must have a host")
	ErrInvalidTimeoutRange = errors.New("timeoutSeconds must be between 5 and 300")
	ErrInvalidMaxDepth     = errors.New("maxDepth must be between 1 and 10")
	ErrInvalidMaxPages     = errors.New("maxPages must be between 1 and 10000")
	ErrInvalidProfileName  = errors.New("invalid authProfile: only alphanumeric, hyphens, and underscores allowed")
)

func ValidateURL(rawURL string) error {
	if rawURL == "" {
		return errors.New("url is required")
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ErrInvalidURLScheme
	}
	if u.Host == "" {
		return ErrInvalidURLHost
	}
	return nil
}

func ValidateURLs(urls []string) error {
	if len(urls) == 0 {
		return errors.New("urls list is empty")
	}
	for i, u := range urls {
		if err := ValidateURL(u); err != nil {
			return fmt.Errorf("invalid url at index %d (%s): %w", i, u, err)
		}
	}
	return nil
}

func ValidateTimeout(timeoutSeconds int) error {
	if timeoutSeconds == 0 {
		return nil
	}
	if timeoutSeconds < 5 || timeoutSeconds > 300 {
		return ErrInvalidTimeoutRange
	}
	return nil
}

func ValidateMaxDepth(maxDepth int) error {
	if maxDepth == 0 {
		return nil
	}
	if maxDepth < 1 || maxDepth > 10 {
		return ErrInvalidMaxDepth
	}
	return nil
}

func ValidateMaxPages(maxPages int) error {
	if maxPages == 0 {
		return nil
	}
	if maxPages < 1 || maxPages > 10000 {
		return ErrInvalidMaxPages
	}
	return nil
}

func ValidateAuthProfileName(name string) error {
	if name == "" {
		return nil
	}
	if !profileNameRegex.MatchString(name) {
		return ErrInvalidProfileName
	}
	return nil
}
