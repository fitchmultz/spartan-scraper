// Package scheduler provides schedule persistence operations.
//
// This file is responsible for:
// - Loading all schedules from JSON file storage
// - Saving all schedules to JSON file storage
// - Adding new schedules with ID generation and defaults
// - Deleting schedules by ID
// - Listing schedules sorted by next run time
//
// This file does NOT handle:
// - Concurrent access (cachedScheduler handles this)
// - Schedule validation (validation.go does this)
// - Schedule execution (scheduler.go does this)
//
// Invariants:
// - File path is derived from dataDir: <dataDir>/schedules.json
// - Empty/missing file returns empty slice, not error
// - IDs are generated via uuid.NewString() if empty
// - IntervalSeconds defaults to 3600 if <= 0
// - NextRun defaults to now + interval if zero
package scheduler

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/fsutil"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/google/uuid"
)

type persistedScheduleStore struct {
	Schedules []persistedSchedule `json:"schedules"`
}

type persistedSchedule struct {
	ID              string          `json:"id"`
	Kind            model.Kind      `json:"kind"`
	IntervalSeconds int             `json:"intervalSeconds"`
	NextRun         time.Time       `json:"nextRun"`
	SpecVersion     *int            `json:"specVersion"`
	Spec            json.RawMessage `json:"spec"`
	Params          json.RawMessage `json:"params,omitempty"`
}

func LoadAll(dataDir string) ([]Schedule, error) {
	path := schedulesPath(dataDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Schedule{}, nil
		}
		return nil, err
	}
	var s persistedScheduleStore
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	schedules := make([]Schedule, 0, len(s.Schedules))
	for _, persisted := range s.Schedules {
		schedule, decodeErr := decodePersistedSchedule(persisted)
		if decodeErr != nil {
			return nil, decodeErr
		}
		schedules = append(schedules, schedule)
	}
	return schedules, nil
}

func SaveAll(dataDir string, schedules []Schedule) error {
	if err := fsutil.EnsureDataDir(dataDir); err != nil {
		return err
	}
	path := schedulesPath(dataDir)
	payload, err := json.MarshalIndent(scheduleStore{Schedules: schedules}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o600)
}

func Add(dataDir string, schedule Schedule) (*Schedule, error) {
	if schedule.ID == "" {
		schedule.ID = uuid.NewString()
	}
	if schedule.IntervalSeconds <= 0 {
		schedule.IntervalSeconds = 3600
	}
	if schedule.NextRun.IsZero() {
		schedule.NextRun = time.Now().Add(time.Duration(schedule.IntervalSeconds) * time.Second)
	}
	if schedule.SpecVersion == 0 {
		schedule.SpecVersion = model.JobSpecVersion1
	}

	spec, err := canonicalizeScheduleSpec(schedule.Kind, schedule.SpecVersion, schedule.Spec)
	if err != nil {
		return nil, err
	}
	schedule.Spec = spec

	if err := validateScheduleSpec(schedule); err != nil {
		return nil, err
	}

	items, err := LoadAll(dataDir)
	if err != nil {
		return nil, err
	}
	items = append(items, schedule)
	if err := SaveAll(dataDir, items); err != nil {
		return nil, err
	}
	return &schedule, nil
}

func Delete(dataDir, id string) error {
	items, err := LoadAll(dataDir)
	if err != nil {
		return err
	}
	filtered := make([]Schedule, 0, len(items))
	for _, item := range items {
		if item.ID != id {
			filtered = append(filtered, item)
		}
	}
	return SaveAll(dataDir, filtered)
}

func List(dataDir string) ([]Schedule, error) {
	items, err := LoadAll(dataDir)
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].NextRun.Before(items[j].NextRun) })
	return items, nil
}

func schedulesPath(dataDir string) string {
	base := dataDir
	if base == "" {
		base = ".data"
	}
	return filepath.Join(base, "schedules.json")
}

func decodePersistedSchedule(persisted persistedSchedule) (Schedule, error) {
	if len(persisted.Params) > 0 {
		return Schedule{}, apperrors.Validation("schedules.json uses the removed params contract; reset or recreate schedules")
	}
	if persisted.SpecVersion == nil {
		return Schedule{}, apperrors.Validation("schedules.json is missing specVersion; reset or recreate schedules")
	}
	if len(persisted.Spec) == 0 {
		return Schedule{}, apperrors.Validation("schedules.json is missing spec; reset or recreate schedules")
	}
	spec, err := model.DecodeJobSpec(persisted.Kind, *persisted.SpecVersion, persisted.Spec)
	if err != nil {
		return Schedule{}, apperrors.Wrap(apperrors.KindValidation, "failed to decode persisted schedule spec", err)
	}
	return Schedule{
		ID:              persisted.ID,
		Kind:            persisted.Kind,
		IntervalSeconds: persisted.IntervalSeconds,
		NextRun:         persisted.NextRun,
		SpecVersion:     *persisted.SpecVersion,
		Spec:            spec,
	}, nil
}

func canonicalizeScheduleSpec(kind model.Kind, version int, spec any) (any, error) {
	if spec == nil {
		return nil, apperrors.Validation("schedule spec is required")
	}
	switch kind {
	case model.KindScrape, model.KindCrawl, model.KindResearch:
	default:
		return nil, apperrors.Validation("unknown schedule kind")
	}

	switch typed := spec.(type) {
	case model.ScrapeSpecV1:
		if kind != model.KindScrape {
			return nil, apperrors.Validation(fmt.Sprintf("schedule kind %s does not match scrape spec", kind))
		}
		return typed, nil
	case *model.ScrapeSpecV1:
		if typed == nil {
			return nil, apperrors.Validation("schedule spec is required")
		}
		if kind != model.KindScrape {
			return nil, apperrors.Validation(fmt.Sprintf("schedule kind %s does not match scrape spec", kind))
		}
		return *typed, nil
	case model.CrawlSpecV1:
		if kind != model.KindCrawl {
			return nil, apperrors.Validation(fmt.Sprintf("schedule kind %s does not match crawl spec", kind))
		}
		return typed, nil
	case *model.CrawlSpecV1:
		if typed == nil {
			return nil, apperrors.Validation("schedule spec is required")
		}
		if kind != model.KindCrawl {
			return nil, apperrors.Validation(fmt.Sprintf("schedule kind %s does not match crawl spec", kind))
		}
		return *typed, nil
	case model.ResearchSpecV1:
		if kind != model.KindResearch {
			return nil, apperrors.Validation(fmt.Sprintf("schedule kind %s does not match research spec", kind))
		}
		return typed, nil
	case *model.ResearchSpecV1:
		if typed == nil {
			return nil, apperrors.Validation("schedule spec is required")
		}
		if kind != model.KindResearch {
			return nil, apperrors.Validation(fmt.Sprintf("schedule kind %s does not match research spec", kind))
		}
		return *typed, nil
	default:
		raw, err := json.Marshal(spec)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindValidation, "failed to marshal schedule spec", err)
		}
		decoded, err := model.DecodeJobSpec(kind, version, raw)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindValidation, "failed to decode schedule spec", err)
		}
		return decoded, nil
	}
}
