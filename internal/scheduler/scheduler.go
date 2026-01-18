package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/uuid"

	"spartan-scraper/internal/auth"
	"spartan-scraper/internal/extract"
	"spartan-scraper/internal/fetch"
	"spartan-scraper/internal/jobs"
	"spartan-scraper/internal/model"
)

type Schedule struct {
	ID              string                 `json:"id"`
	Kind            model.Kind             `json:"kind"`
	IntervalSeconds int                    `json:"intervalSeconds"`
	NextRun         time.Time              `json:"nextRun"`
	Params          map[string]interface{} `json:"params"`
}

type store struct {
	Schedules []Schedule `json:"schedules"`
}

func Run(ctx context.Context, dataDir string, manager *jobs.Manager) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			schedules, err := LoadAll(dataDir)
			if err != nil {
				continue
			}
			changed := false
			now := time.Now()
			for i := range schedules {
				if schedules[i].NextRun.After(now) {
					continue
				}
				err := enqueue(manager, dataDir, schedules[i])
				if err == nil {
					schedules[i].NextRun = now.Add(time.Duration(schedules[i].IntervalSeconds) * time.Second)
					changed = true
				}
			}
			if changed {
				_ = SaveAll(dataDir, schedules)
			}
		}
	}
}

func enqueue(manager *jobs.Manager, dataDir string, schedule Schedule) error {
	extractOpts := loadExtract(schedule.Params)
	switch schedule.Kind {
	case model.KindScrape:
		url := stringParam(schedule.Params, "url")
		headless := boolParam(schedule.Params, "headless")
		playwright := boolParamDefault(schedule.Params, "playwright", manager.DefaultUsePlaywright())
		authOptions, _ := loadAuth(schedule.Params, dataDir, url, auth.EnvOverrides{})
		incremental := boolParam(schedule.Params, "incremental")
		job, err := manager.CreateScrapeJob(url, headless, playwright, authOptions, intParam(schedule.Params, "timeout", manager.DefaultTimeoutSeconds()), extractOpts, incremental)
		if err != nil {
			return err
		}
		return manager.Enqueue(job)
	case model.KindCrawl:
		url := stringParam(schedule.Params, "url")
		headless := boolParam(schedule.Params, "headless")
		playwright := boolParamDefault(schedule.Params, "playwright", manager.DefaultUsePlaywright())
		maxDepth := intParam(schedule.Params, "maxDepth", 2)
		maxPages := intParam(schedule.Params, "maxPages", 200)
		authOptions, _ := loadAuth(schedule.Params, dataDir, url, auth.EnvOverrides{})
		incremental := boolParam(schedule.Params, "incremental")
		job, err := manager.CreateCrawlJob(url, maxDepth, maxPages, headless, playwright, authOptions, intParam(schedule.Params, "timeout", manager.DefaultTimeoutSeconds()), extractOpts, incremental)
		if err != nil {
			return err
		}
		return manager.Enqueue(job)
	case model.KindResearch:
		query := stringParam(schedule.Params, "query")
		urls := stringSliceParam(schedule.Params, "urls")
		targetURL := ""
		if len(urls) > 0 {
			targetURL = urls[0]
		}
		headless := boolParam(schedule.Params, "headless")
		playwright := boolParamDefault(schedule.Params, "playwright", manager.DefaultUsePlaywright())
		maxDepth := intParam(schedule.Params, "maxDepth", 2)
		maxPages := intParam(schedule.Params, "maxPages", 200)
		authOptions, _ := loadAuth(schedule.Params, dataDir, targetURL, auth.EnvOverrides{})
		incremental := boolParam(schedule.Params, "incremental")
		job, err := manager.CreateResearchJob(query, urls, maxDepth, maxPages, headless, playwright, authOptions, intParam(schedule.Params, "timeout", manager.DefaultTimeoutSeconds()), extractOpts, incremental)
		if err != nil {
			return err
		}
		return manager.Enqueue(job)
	default:
		return errors.New("unknown schedule kind")
	}
}

func loadExtract(params map[string]interface{}) extract.ExtractOptions {
	if params == nil {
		return extract.ExtractOptions{}
	}
	return extract.ExtractOptions{
		Template: stringParam(params, "extractTemplate"),
		Validate: boolParam(params, "extractValidate"),
	}
}

func loadAuth(params map[string]interface{}, dataDir string, targetURL string, env auth.EnvOverrides) (fetch.AuthOptions, error) {
	if params == nil {
		return fetch.AuthOptions{}, nil
	}
	profileName, _ := params["authProfile"].(string)
	input := auth.ResolveInput{
		ProfileName: profileName,
		URL:         targetURL,
		Env:         &env,
	}
	resolved, err := auth.Resolve(dataDir, input)
	if err != nil {
		return fetch.AuthOptions{}, err
	}
	return auth.ToFetchOptions(resolved), nil
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
	var s store
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return s.Schedules, nil
}

func SaveAll(dataDir string, schedules []Schedule) error {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	path := schedulesPath(dataDir)
	payload, err := json.MarshalIndent(store{Schedules: schedules}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o600)
}

func Add(dataDir string, schedule Schedule) error {
	if schedule.ID == "" {
		schedule.ID = uuid.NewString()
	}
	if schedule.IntervalSeconds <= 0 {
		schedule.IntervalSeconds = 3600
	}
	if schedule.NextRun.IsZero() {
		schedule.NextRun = time.Now().Add(time.Duration(schedule.IntervalSeconds) * time.Second)
	}
	items, err := LoadAll(dataDir)
	if err != nil {
		return err
	}
	items = append(items, schedule)
	return SaveAll(dataDir, items)
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

func stringParam(params map[string]interface{}, key string) string {
	if params == nil {
		return ""
	}
	if value, ok := params[key].(string); ok {
		return value
	}
	return ""
}

func boolParam(params map[string]interface{}, key string) bool {
	if params == nil {
		return false
	}
	if value, ok := params[key].(bool); ok {
		return value
	}
	return false
}

func boolParamDefault(params map[string]interface{}, key string, fallback bool) bool {
	if params == nil {
		return fallback
	}
	if _, ok := params[key]; !ok {
		return fallback
	}
	if value, ok := params[key].(bool); ok {
		return value
	}
	return fallback
}

func intParam(params map[string]interface{}, key string, fallback int) int {
	if params == nil {
		return fallback
	}
	switch value := params[key].(type) {
	case int:
		if value <= 0 {
			return fallback
		}
		return value
	case float64:
		if int(value) <= 0 {
			return fallback
		}
		return int(value)
	default:
		return fallback
	}
}

func stringSliceParam(params map[string]interface{}, key string) []string {
	if params == nil {
		return nil
	}
	switch value := params[key].(type) {
	case []interface{}:
		items := make([]string, 0, len(value))
		for _, item := range value {
			if s, ok := item.(string); ok {
				items = append(items, s)
			}
		}
		return items
	case []string:
		return value
	default:
		return nil
	}
}
