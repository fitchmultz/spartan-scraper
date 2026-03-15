// Package scheduler provides typed spec helpers for recurring scheduled jobs.
//
// This file is responsible for:
// - Extracting the shared execution config from a typed schedule spec
// - Resolving auth for scheduled jobs from typed execution specs
// - Deriving schedule target URLs for auth resolution and diagnostics
//
// This file does NOT handle:
// - Schedule validation (validation.go does this)
// - Schedule persistence or execution
// - Direct auth vault access (uses auth.Resolve)
//
// Invariants:
// - Schedules use the same typed V1 specs as persisted jobs
// - Auth profiles are resolved at execution time for recurring schedules
package scheduler

import (
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/auth"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func executionSpecForSchedule(schedule Schedule) (model.ExecutionSpec, error) {
	switch typed := schedule.Spec.(type) {
	case model.ScrapeSpecV1:
		return typed.Execution, nil
	case *model.ScrapeSpecV1:
		if typed == nil {
			return model.ExecutionSpec{}, unsupportedScheduleSpecError(schedule.Kind)
		}
		return typed.Execution, nil
	case model.CrawlSpecV1:
		return typed.Execution, nil
	case *model.CrawlSpecV1:
		if typed == nil {
			return model.ExecutionSpec{}, unsupportedScheduleSpecError(schedule.Kind)
		}
		return typed.Execution, nil
	case model.ResearchSpecV1:
		return typed.Execution, nil
	case *model.ResearchSpecV1:
		if typed == nil {
			return model.ExecutionSpec{}, unsupportedScheduleSpecError(schedule.Kind)
		}
		return typed.Execution, nil
	default:
		return model.ExecutionSpec{}, unsupportedScheduleSpecError(schedule.Kind)
	}
}

func targetURLForSchedule(schedule Schedule) string {
	switch typed := schedule.Spec.(type) {
	case model.ScrapeSpecV1:
		return typed.URL
	case *model.ScrapeSpecV1:
		if typed == nil {
			return ""
		}
		return typed.URL
	case model.CrawlSpecV1:
		return typed.URL
	case *model.CrawlSpecV1:
		if typed == nil {
			return ""
		}
		return typed.URL
	case model.ResearchSpecV1:
		if len(typed.URLs) > 0 {
			return typed.URLs[0]
		}
	case *model.ResearchSpecV1:
		if typed == nil {
			return ""
		}
		if len(typed.URLs) > 0 {
			return typed.URLs[0]
		}
	}
	return ""
}

func resolveScheduleAuth(schedule Schedule, dataDir string, env auth.EnvOverrides) (fetch.AuthOptions, error) {
	exec, err := executionSpecForSchedule(schedule)
	if err != nil {
		return fetch.AuthOptions{}, err
	}
	if exec.AuthProfile == "" {
		return exec.Auth, nil
	}

	input := model.AuthOverridesFromExecution(exec)
	input.URL = targetURLForSchedule(schedule)
	input.Env = &env

	resolved, err := auth.Resolve(dataDir, input)
	if err != nil {
		return fetch.AuthOptions{}, err
	}
	options := auth.ToFetchOptions(resolved)
	options.Proxy = exec.Auth.Proxy
	options.ProxyHints = fetch.NormalizeProxySelectionHints(exec.Auth.ProxyHints)
	options.OAuth2 = exec.Auth.OAuth2
	options.NormalizeTransport()
	if err := options.ValidateTransport(); err != nil {
		return fetch.AuthOptions{}, err
	}
	return options, nil
}

func unsupportedScheduleSpecError(kind model.Kind) error {
	return apperrors.Validation("unsupported typed schedule spec for kind " + string(kind))
}
