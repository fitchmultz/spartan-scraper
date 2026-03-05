// Package validate provides request validators for scrape, crawl, and research operations.
// It handles validation of URLs, timeouts, depths, pages, and profile names.
// It does NOT define validation rules (validate.go does).
package validate

import (
	"fmt"
	"net/url"
	"regexp"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

var profileNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

var (
	ErrInvalidURLScheme    = apperrors.ErrInvalidURLScheme
	ErrInvalidURLHost      = apperrors.ErrInvalidURLHost
	ErrInvalidTimeoutRange = apperrors.ErrInvalidTimeoutRange
	ErrInvalidMaxDepth     = apperrors.ErrInvalidMaxDepth
	ErrInvalidMaxPages     = apperrors.ErrInvalidMaxPages
	ErrInvalidProfileName  = apperrors.ErrInvalidProfileName
)

func ValidateURL(rawURL string) error {
	if rawURL == "" {
		return apperrors.Validation("url is required")
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return apperrors.WithKind(apperrors.KindValidation, fmt.Errorf("invalid url: %w", err))
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return apperrors.WithKind(apperrors.KindValidation, ErrInvalidURLScheme)
	}
	if u.Host == "" {
		return apperrors.WithKind(apperrors.KindValidation, ErrInvalidURLHost)
	}
	return nil
}

func ValidateURLs(urls []string) error {
	if len(urls) == 0 {
		return apperrors.Validation("urls list is empty")
	}
	for i, u := range urls {
		if err := ValidateURL(u); err != nil {
			return apperrors.WithKind(apperrors.KindValidation, fmt.Errorf("invalid url at index %d: %w", i, err))
		}
	}
	return nil
}

func ValidateTimeout(timeoutSeconds int) error {
	if timeoutSeconds == 0 {
		return nil
	}
	if timeoutSeconds < 5 || timeoutSeconds > 300 {
		return apperrors.WithKind(apperrors.KindValidation, ErrInvalidTimeoutRange)
	}
	return nil
}

func ValidateMaxDepth(maxDepth int) error {
	if maxDepth == 0 {
		return nil
	}
	if maxDepth < 1 || maxDepth > 10 {
		return apperrors.WithKind(apperrors.KindValidation, ErrInvalidMaxDepth)
	}
	return nil
}

func ValidateMaxPages(maxPages int) error {
	if maxPages == 0 {
		return nil
	}
	if maxPages < 1 || maxPages > 10000 {
		return apperrors.WithKind(apperrors.KindValidation, ErrInvalidMaxPages)
	}
	return nil
}

func ValidateAuthProfileName(name string) error {
	if name == "" {
		return nil
	}
	if !profileNameRegex.MatchString(name) {
		return apperrors.WithKind(apperrors.KindValidation, ErrInvalidProfileName)
	}
	return nil
}
