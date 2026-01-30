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
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/fsutil"
	"github.com/google/uuid"
)

func LoadAll(dataDir string) ([]Schedule, error) {
	path := schedulesPath(dataDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Schedule{}, nil
		}
		return nil, err
	}
	var s scheduleStore
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return s.Schedules, nil
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

	if err := validateScheduleParams(schedule); err != nil {
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
