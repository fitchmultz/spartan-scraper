// Package scheduler provides typed schedule validation.
//
// This file is responsible for:
// - Validating typed schedule specs based on schedule kind (scrape, crawl, research)
// - Delegating to validate package for common validation logic
// - Returning classified errors via apperrors package
//
// This file does NOT handle:
// - Schedule persistence or storage
// - Schedule execution
// - Shared execution decoding (params.go does this)
//
// Invariants:
// - Returns apperrors.KindValidation for validation failures
// - Uses validate.ValidateJob for common job validation
// - Unknown schedule kinds return validation errors
package scheduler

import (
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/validate"
)

func validateScheduleSpec(schedule Schedule) error {
	exec, err := executionSpecForSchedule(schedule)
	if err != nil {
		return err
	}

	switch typed := schedule.Spec.(type) {
	case model.ScrapeSpecV1:
		return validateTypedSchedule(schedule.Kind, validate.JobValidationOpts{
			URL:         typed.URL,
			Timeout:     exec.TimeoutSeconds,
			AuthProfile: exec.AuthProfile,
		})
	case *model.ScrapeSpecV1:
		return validateTypedSchedule(schedule.Kind, validate.JobValidationOpts{
			URL:         typed.URL,
			Timeout:     exec.TimeoutSeconds,
			AuthProfile: exec.AuthProfile,
		})
	case model.CrawlSpecV1:
		return validateTypedSchedule(schedule.Kind, validate.JobValidationOpts{
			URL:         typed.URL,
			MaxDepth:    typed.MaxDepth,
			MaxPages:    typed.MaxPages,
			Timeout:     exec.TimeoutSeconds,
			AuthProfile: exec.AuthProfile,
		})
	case *model.CrawlSpecV1:
		return validateTypedSchedule(schedule.Kind, validate.JobValidationOpts{
			URL:         typed.URL,
			MaxDepth:    typed.MaxDepth,
			MaxPages:    typed.MaxPages,
			Timeout:     exec.TimeoutSeconds,
			AuthProfile: exec.AuthProfile,
		})
	case model.ResearchSpecV1:
		return validateTypedSchedule(schedule.Kind, validate.JobValidationOpts{
			Query:       typed.Query,
			URLs:        typed.URLs,
			MaxDepth:    typed.MaxDepth,
			MaxPages:    typed.MaxPages,
			Timeout:     exec.TimeoutSeconds,
			AuthProfile: exec.AuthProfile,
		})
	case *model.ResearchSpecV1:
		return validateTypedSchedule(schedule.Kind, validate.JobValidationOpts{
			Query:       typed.Query,
			URLs:        typed.URLs,
			MaxDepth:    typed.MaxDepth,
			MaxPages:    typed.MaxPages,
			Timeout:     exec.TimeoutSeconds,
			AuthProfile: exec.AuthProfile,
		})
	default:
		return apperrors.Validation("unknown schedule kind")
	}
}

func validateTypedSchedule(kind model.Kind, opts validate.JobValidationOpts) error {
	if err := validate.ValidateJob(opts, kind); err != nil {
		return apperrors.Wrap(apperrors.KindValidation, "invalid "+string(kind)+" schedule", err)
	}
	return nil
}
