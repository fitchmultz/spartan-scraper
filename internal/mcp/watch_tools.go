// Package mcp provides shared watch-tool decoding and validation helpers.
//
// Purpose:
// - Keep watch-management argument handling out of the main MCP handler switch.
//
// Responsibilities:
// - Decode create/update arguments for watch tools.
// - Apply MCP defaults that mirror the API watch contract.
// - Validate and normalize optional watch job-trigger requests before persistence.
//
// Scope:
// - MCP watch tool input shaping only; persistence and execution live in internal/watch.
//
// Usage:
// - Used by watch_create, watch_update, and watch_check handler branches.
//
// Invariants/Assumptions:
// - Watch create/update defaults should match the API watch surface.
// - Omitted update fields preserve existing watch values.
// - jobTrigger.request is normalized before it is persisted.
package mcp

import (
	"encoding/json"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/submission"
	"github.com/fitchmultz/spartan-scraper/internal/watch"
)

type watchCreateArgs struct {
	URL                 string          `json:"url"`
	Selector            *string         `json:"selector,omitempty"`
	IntervalSeconds     *int            `json:"intervalSeconds,omitempty"`
	Enabled             *bool           `json:"enabled,omitempty"`
	DiffFormat          *string         `json:"diffFormat,omitempty"`
	WebhookConfig       json.RawMessage `json:"webhookConfig,omitempty"`
	NotifyOnChange      *bool           `json:"notifyOnChange,omitempty"`
	MinChangeSize       *int            `json:"minChangeSize,omitempty"`
	IgnorePatterns      *[]string       `json:"ignorePatterns,omitempty"`
	Headless            *bool           `json:"headless,omitempty"`
	UsePlaywright       *bool           `json:"usePlaywright,omitempty"`
	ExtractMode         *string         `json:"extractMode,omitempty"`
	ScreenshotEnabled   *bool           `json:"screenshotEnabled,omitempty"`
	ScreenshotConfig    json.RawMessage `json:"screenshotConfig,omitempty"`
	VisualDiffThreshold *float64        `json:"visualDiffThreshold,omitempty"`
	JobTrigger          json.RawMessage `json:"jobTrigger,omitempty"`
}

type watchUpdateArgs struct {
	ID                  string          `json:"id"`
	URL                 *string         `json:"url,omitempty"`
	Selector            *string         `json:"selector,omitempty"`
	IntervalSeconds     *int            `json:"intervalSeconds,omitempty"`
	Enabled             *bool           `json:"enabled,omitempty"`
	DiffFormat          *string         `json:"diffFormat,omitempty"`
	WebhookConfig       json.RawMessage `json:"webhookConfig,omitempty"`
	NotifyOnChange      *bool           `json:"notifyOnChange,omitempty"`
	MinChangeSize       *int            `json:"minChangeSize,omitempty"`
	IgnorePatterns      *[]string       `json:"ignorePatterns,omitempty"`
	Headless            *bool           `json:"headless,omitempty"`
	UsePlaywright       *bool           `json:"usePlaywright,omitempty"`
	ExtractMode         *string         `json:"extractMode,omitempty"`
	ScreenshotEnabled   *bool           `json:"screenshotEnabled,omitempty"`
	ScreenshotConfig    json.RawMessage `json:"screenshotConfig,omitempty"`
	VisualDiffThreshold *float64        `json:"visualDiffThreshold,omitempty"`
	JobTrigger          json.RawMessage `json:"jobTrigger,omitempty"`
}

func decodeOptionalRaw[T any](raw json.RawMessage, field string) (T, error) {
	var zero T
	if len(raw) == 0 {
		return zero, nil
	}
	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		return zero, apperrors.Validation(field + " is invalid: " + err.Error())
	}
	return value, nil
}

func boolValueOr(value *bool, fallback bool) bool {
	if value != nil {
		return *value
	}
	return fallback
}

func intValueOr(value *int, fallback int) int {
	if value != nil {
		return *value
	}
	return fallback
}

func floatValueOr(value *float64, fallback float64) float64 {
	if value != nil {
		return *value
	}
	return fallback
}

func stringValueOr(value *string, fallback string) string {
	if value != nil {
		return *value
	}
	return fallback
}

func stringSliceValue(value *[]string) []string {
	if value == nil {
		return nil
	}
	return append([]string(nil), (*value)...)
}

func (s *Server) watchSubmissionDefaults() submission.Defaults {
	defaults := submission.Defaults{
		DefaultTimeoutSeconds: s.cfg.RequestTimeoutSecs,
		DefaultUsePlaywright:  s.cfg.UsePlaywright,
		ResolveAuth:           false,
	}
	if s.manager != nil {
		defaults.DefaultTimeoutSeconds = s.manager.DefaultTimeoutSeconds()
		defaults.DefaultUsePlaywright = s.manager.DefaultUsePlaywright()
	}
	return defaults
}

func (s *Server) validateWatchJobTrigger(trigger *watch.JobTrigger) error {
	if trigger == nil {
		return nil
	}
	if len(trigger.Request) == 0 {
		return apperrors.Validation("jobTrigger.request is required when jobTrigger is set")
	}
	normalizedRequest, err := submission.NormalizeRawRequest(trigger.Kind, trigger.Request)
	if err != nil {
		return err
	}
	if _, _, err := submission.JobSpecFromRawRequest(s.cfg, s.watchSubmissionDefaults(), trigger.Kind, normalizedRequest); err != nil {
		return err
	}
	trigger.Request = normalizedRequest
	return nil
}

func (s *Server) buildWatchCreate(args watchCreateArgs) (*watch.Watch, error) {
	webhookConfig, err := decodeOptionalRaw[*model.WebhookSpec](args.WebhookConfig, "webhookConfig")
	if err != nil {
		return nil, err
	}
	screenshotConfig, err := decodeOptionalRaw[*fetch.ScreenshotConfig](args.ScreenshotConfig, "screenshotConfig")
	if err != nil {
		return nil, err
	}
	jobTrigger, err := decodeOptionalRaw[*watch.JobTrigger](args.JobTrigger, "jobTrigger")
	if err != nil {
		return nil, err
	}

	intervalSeconds := 3600
	if args.IntervalSeconds != nil && *args.IntervalSeconds > 0 {
		intervalSeconds = *args.IntervalSeconds
	}
	diffFormat := "unified"
	if args.DiffFormat != nil && strings.TrimSpace(*args.DiffFormat) != "" {
		diffFormat = strings.TrimSpace(*args.DiffFormat)
	}

	item := &watch.Watch{
		URL:                 strings.TrimSpace(args.URL),
		Selector:            strings.TrimSpace(stringValueOr(args.Selector, "")),
		IntervalSeconds:     intervalSeconds,
		Enabled:             boolValueOr(args.Enabled, true),
		DiffFormat:          diffFormat,
		WebhookConfig:       webhookConfig,
		NotifyOnChange:      boolValueOr(args.NotifyOnChange, false),
		MinChangeSize:       intValueOr(args.MinChangeSize, 0),
		IgnorePatterns:      stringSliceValue(args.IgnorePatterns),
		Headless:            boolValueOr(args.Headless, false),
		UsePlaywright:       boolValueOr(args.UsePlaywright, false),
		ExtractMode:         strings.TrimSpace(stringValueOr(args.ExtractMode, "")),
		ScreenshotEnabled:   boolValueOr(args.ScreenshotEnabled, false),
		ScreenshotConfig:    screenshotConfig,
		VisualDiffThreshold: floatValueOr(args.VisualDiffThreshold, 0.1),
		JobTrigger:          jobTrigger,
	}
	if item.URL == "" {
		return nil, apperrors.Validation("url is required")
	}
	if err := s.validateWatchJobTrigger(item.JobTrigger); err != nil {
		return nil, err
	}
	if err := item.Validate(); err != nil {
		return nil, apperrors.Validation(err.Error())
	}
	return item, nil
}

func (s *Server) applyWatchUpdate(existing *watch.Watch, args watchUpdateArgs) error {
	if args.URL != nil {
		existing.URL = strings.TrimSpace(*args.URL)
	}
	if args.Selector != nil {
		existing.Selector = strings.TrimSpace(*args.Selector)
	}
	if args.IntervalSeconds != nil && *args.IntervalSeconds > 0 {
		existing.IntervalSeconds = *args.IntervalSeconds
	}
	if args.Enabled != nil {
		existing.Enabled = *args.Enabled
	}
	if args.DiffFormat != nil && strings.TrimSpace(*args.DiffFormat) != "" {
		existing.DiffFormat = strings.TrimSpace(*args.DiffFormat)
	}
	if args.NotifyOnChange != nil {
		existing.NotifyOnChange = *args.NotifyOnChange
	}
	if args.MinChangeSize != nil {
		existing.MinChangeSize = *args.MinChangeSize
	}
	if args.IgnorePatterns != nil {
		existing.IgnorePatterns = stringSliceValue(args.IgnorePatterns)
	}
	if args.Headless != nil {
		existing.Headless = *args.Headless
	}
	if args.UsePlaywright != nil {
		existing.UsePlaywright = *args.UsePlaywright
	}
	if args.ExtractMode != nil {
		existing.ExtractMode = strings.TrimSpace(*args.ExtractMode)
	}
	if args.ScreenshotEnabled != nil {
		existing.ScreenshotEnabled = *args.ScreenshotEnabled
	}
	if args.VisualDiffThreshold != nil {
		existing.VisualDiffThreshold = *args.VisualDiffThreshold
	}
	if args.WebhookConfig != nil {
		webhookConfig, err := decodeOptionalRaw[*model.WebhookSpec](args.WebhookConfig, "webhookConfig")
		if err != nil {
			return err
		}
		existing.WebhookConfig = webhookConfig
	}
	if args.ScreenshotConfig != nil {
		screenshotConfig, err := decodeOptionalRaw[*fetch.ScreenshotConfig](args.ScreenshotConfig, "screenshotConfig")
		if err != nil {
			return err
		}
		existing.ScreenshotConfig = screenshotConfig
	}
	if args.JobTrigger != nil {
		jobTrigger, err := decodeOptionalRaw[*watch.JobTrigger](args.JobTrigger, "jobTrigger")
		if err != nil {
			return err
		}
		existing.JobTrigger = jobTrigger
	}
	if err := s.validateWatchJobTrigger(existing.JobTrigger); err != nil {
		return err
	}
	if err := existing.Validate(); err != nil {
		return apperrors.Validation(err.Error())
	}
	return nil
}
