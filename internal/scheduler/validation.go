package scheduler

import (
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/validate"
)

func validateScheduleParams(schedule Schedule) error {
	switch schedule.Kind {
	case model.KindScrape:
		opts := validate.JobValidationOpts{
			URL:         stringParam(schedule.Params, "url"),
			Timeout:     intParam(schedule.Params, "timeout", 0),
			AuthProfile: stringParam(schedule.Params, "authProfile"),
		}
		if err := validate.ValidateJob(opts, model.KindScrape); err != nil {
			return apperrors.Wrap(apperrors.KindValidation, "invalid scrape schedule", err)
		}
	case model.KindCrawl:
		opts := validate.JobValidationOpts{
			URL:         stringParam(schedule.Params, "url"),
			MaxDepth:    intParam(schedule.Params, "maxDepth", 0),
			MaxPages:    intParam(schedule.Params, "maxPages", 0),
			Timeout:     intParam(schedule.Params, "timeout", 0),
			AuthProfile: stringParam(schedule.Params, "authProfile"),
		}
		if err := validate.ValidateJob(opts, model.KindCrawl); err != nil {
			return apperrors.Wrap(apperrors.KindValidation, "invalid crawl schedule", err)
		}
	case model.KindResearch:
		opts := validate.JobValidationOpts{
			Query:       stringParam(schedule.Params, "query"),
			URLs:        stringSliceParam(schedule.Params, "urls"),
			MaxDepth:    intParam(schedule.Params, "maxDepth", 0),
			MaxPages:    intParam(schedule.Params, "maxPages", 0),
			Timeout:     intParam(schedule.Params, "timeout", 0),
			AuthProfile: stringParam(schedule.Params, "authProfile"),
		}
		if err := validate.ValidateJob(opts, model.KindResearch); err != nil {
			return apperrors.Wrap(apperrors.KindValidation, "invalid research schedule", err)
		}
	default:
		return apperrors.Validation("unknown schedule kind")
	}
	return nil
}
