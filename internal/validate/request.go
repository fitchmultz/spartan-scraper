// Package validate provides request validators for scrape, crawl, and research operations.
// It handles validation of URLs, timeouts, depths, pages, and profile names.
// It does NOT define validation rules (validate.go does).
package validate

import (
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

// JobValidationOpts is a unified set of validation inputs for all job kinds.
//
// Invariants/assumptions:
// - For scrape/crawl: URL is required.
// - For research: Query and URLs are required.
// - MaxDepth/MaxPages/Timeout use "0 means default" semantics (0 is valid).
// - AuthProfile is optional.
type JobValidationOpts struct {
	URL         string   // Required for scrape/crawl
	Query       string   // Required for research
	URLs        []string // Required for research
	MaxDepth    int      // Optional (0 = default)
	MaxPages    int      // Optional (0 = default)
	Timeout     int      // Optional (0 = default)
	AuthProfile string   // Optional
}

// ValidateJob validates job parameters based on job kind.
//
// It preserves exact validation behavior and error messages that previously
// lived in per-kind validator structs.
func ValidateJob(opts JobValidationOpts, kind model.Kind) error {
	switch kind {
	case model.KindScrape:
		if err := ValidateURL(opts.URL); err != nil {
			return err
		}
		if err := ValidateTimeout(opts.Timeout); err != nil {
			return err
		}
		if err := ValidateAuthProfileName(opts.AuthProfile); err != nil {
			return err
		}
		return nil

	case model.KindCrawl:
		if err := ValidateURL(opts.URL); err != nil {
			return err
		}
		if err := ValidateMaxDepth(opts.MaxDepth); err != nil {
			return err
		}
		if err := ValidateMaxPages(opts.MaxPages); err != nil {
			return err
		}
		if err := ValidateTimeout(opts.Timeout); err != nil {
			return err
		}
		if err := ValidateAuthProfileName(opts.AuthProfile); err != nil {
			return err
		}
		return nil

	case model.KindResearch:
		// Preserve old ordering: query required check happens before URL list validation.
		if opts.Query == "" {
			return apperrors.Validation("query is required")
		}
		if err := ValidateURLs(opts.URLs); err != nil {
			return err
		}
		if err := ValidateMaxDepth(opts.MaxDepth); err != nil {
			return err
		}
		if err := ValidateMaxPages(opts.MaxPages); err != nil {
			return err
		}
		if err := ValidateTimeout(opts.Timeout); err != nil {
			return err
		}
		if err := ValidateAuthProfileName(opts.AuthProfile); err != nil {
			return err
		}
		return nil

	default:
		return apperrors.Validation("unknown job kind")
	}
}
