package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"spartan-scraper/internal/auth"
	"spartan-scraper/internal/extract"
	"spartan-scraper/internal/fetch"
	"spartan-scraper/internal/jobs"
	"spartan-scraper/internal/model"
	"spartan-scraper/internal/pipeline"
	"spartan-scraper/internal/validate"
)

type Schedule struct {
	ID              string                 `json:"id"`
	Kind            model.Kind             `json:"kind"`
	IntervalSeconds int                    `json:"intervalSeconds"`
	NextRun         time.Time              `json:"nextRun"`
	Params          map[string]interface{} `json:"params"`
}

type scheduleStore struct {
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
				err := enqueue(ctx, manager, dataDir, schedules[i])
				if err == nil {
					schedules[i].NextRun = now.Add(time.Duration(schedules[i].IntervalSeconds) * time.Second)
					changed = true
				} else {
					slog.Error("failed to enqueue scheduled job",
						"schedule_id", schedules[i].ID,
						"schedule_kind", schedules[i].Kind,
						"error", err,
					)
				}
			}
			if changed {
				if err := SaveAll(dataDir, schedules); err != nil {
					slog.Error("failed to save schedules", "error", err)
				}
			}
		}
	}
}

func enqueue(ctx context.Context, manager *jobs.Manager, dataDir string, schedule Schedule) error {
	extractOpts := loadExtract(schedule.Params)
	pipelineOpts := loadPipeline(schedule.Params)
	switch schedule.Kind {
	case model.KindScrape:
		url := stringParam(schedule.Params, "url")
		headless := boolParam(schedule.Params, "headless")
		playwright := boolParamDefault(schedule.Params, "playwright", manager.DefaultUsePlaywright())
		authOptions, err := loadAuth(schedule.Params, dataDir, url, auth.EnvOverrides{})
		if err != nil {
			return fmt.Errorf("failed to resolve auth for scrape schedule %s: %w", schedule.ID, err)
		}
		incremental := boolParam(schedule.Params, "incremental")
		job, err := manager.CreateScrapeJob(ctx, url, headless, playwright, authOptions, intParam(schedule.Params, "timeout", manager.DefaultTimeoutSeconds()), extractOpts, pipelineOpts, incremental)
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
		authOptions, err := loadAuth(schedule.Params, dataDir, url, auth.EnvOverrides{})
		if err != nil {
			return fmt.Errorf("failed to resolve auth for crawl schedule %s: %w", schedule.ID, err)
		}
		incremental := boolParam(schedule.Params, "incremental")
		job, err := manager.CreateCrawlJob(ctx, url, maxDepth, maxPages, headless, playwright, authOptions, intParam(schedule.Params, "timeout", manager.DefaultTimeoutSeconds()), extractOpts, pipelineOpts, incremental)
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
		authOptions, err := loadAuth(schedule.Params, dataDir, targetURL, auth.EnvOverrides{})
		if err != nil {
			return fmt.Errorf("failed to resolve auth for research schedule %s: %w", schedule.ID, err)
		}
		incremental := boolParam(schedule.Params, "incremental")
		job, err := manager.CreateResearchJob(ctx, query, urls, maxDepth, maxPages, headless, playwright, authOptions, intParam(schedule.Params, "timeout", manager.DefaultTimeoutSeconds()), extractOpts, pipelineOpts, incremental)
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
	opts := extract.ExtractOptions{
		Template: stringParam(params, "extractTemplate"),
		Validate: boolParam(params, "extractValidate"),
	}

	// Handle inline template from extractConfig
	if extractConfigPath := stringParam(params, "extractConfig"); extractConfigPath != "" {
		data, err := os.ReadFile(extractConfigPath)
		if err != nil {
			slog.Warn("failed to read extract config", "path", extractConfigPath, "error", err)
			return opts
		}
		var tmpl extract.Template
		if err := json.Unmarshal(data, &tmpl); err != nil {
			slog.Warn("failed to parse extract config", "path", extractConfigPath, "error", err)
			return opts
		}
		opts.Inline = &tmpl
	}

	return opts
}

func loadPipeline(params map[string]interface{}) pipeline.Options {
	if params == nil {
		return pipeline.Options{}
	}
	raw, ok := params["pipeline"]
	if !ok || raw == nil {
		return pipeline.Options{}
	}
	if opts, ok := raw.(pipeline.Options); ok {
		return opts
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return pipeline.Options{}
	}
	var opts pipeline.Options
	if err := json.Unmarshal(data, &opts); err != nil {
		return pipeline.Options{}
	}
	return opts
}

func loadAuth(params map[string]interface{}, dataDir string, targetURL string, env auth.EnvOverrides) (fetch.AuthOptions, error) {
	if params == nil {
		return fetch.AuthOptions{}, nil
	}

	// Start with auth overrides from params
	input := loadAuthOverrides(params)

	// Add profile name and URL
	input.ProfileName = stringParam(params, "authProfile")
	input.URL = targetURL
	input.Env = &env

	// Resolve with overrides and profile
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
	var s scheduleStore
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
	payload, err := json.MarshalIndent(scheduleStore{Schedules: schedules}, "", "  ")
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

	if err := validateScheduleParams(schedule); err != nil {
		return err
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

func validateScheduleParams(schedule Schedule) error {
	switch schedule.Kind {
	case model.KindScrape:
		url := stringParam(schedule.Params, "url")
		timeout := intParam(schedule.Params, "timeout", 0)
		authProfile := stringParam(schedule.Params, "authProfile")
		validator := validate.ScrapeRequestValidator{
			URL:         url,
			Timeout:     timeout,
			AuthProfile: authProfile,
		}
		if err := validator.Validate(); err != nil {
			return fmt.Errorf("invalid scrape schedule: %w", err)
		}
	case model.KindCrawl:
		url := stringParam(schedule.Params, "url")
		maxDepth := intParam(schedule.Params, "maxDepth", 0)
		maxPages := intParam(schedule.Params, "maxPages", 0)
		timeout := intParam(schedule.Params, "timeout", 0)
		authProfile := stringParam(schedule.Params, "authProfile")
		validator := validate.CrawlRequestValidator{
			URL:         url,
			MaxDepth:    maxDepth,
			MaxPages:    maxPages,
			Timeout:     timeout,
			AuthProfile: authProfile,
		}
		if err := validator.Validate(); err != nil {
			return fmt.Errorf("invalid crawl schedule: %w", err)
		}
	case model.KindResearch:
		query := stringParam(schedule.Params, "query")
		urls := stringSliceParam(schedule.Params, "urls")
		maxDepth := intParam(schedule.Params, "maxDepth", 0)
		maxPages := intParam(schedule.Params, "maxPages", 0)
		timeout := intParam(schedule.Params, "timeout", 0)
		authProfile := stringParam(schedule.Params, "authProfile")
		validator := validate.ResearchRequestValidator{
			Query:       query,
			URLs:        urls,
			MaxDepth:    maxDepth,
			MaxPages:    maxPages,
			Timeout:     timeout,
			AuthProfile: authProfile,
		}
		if err := validator.Validate(); err != nil {
			return fmt.Errorf("invalid research schedule: %w", err)
		}
	default:
		return errors.New("unknown schedule kind")
	}
	return nil
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

func loadAuthOverrides(params map[string]interface{}) auth.ResolveInput {
	if params == nil {
		return auth.ResolveInput{}
	}

	input := auth.ResolveInput{
		Headers: loadHeaderKVs(params),
		Cookies: loadCookies(params),
		Tokens:  loadTokens(params),
		Login:   loadLoginFlow(params),
	}

	return input
}

func loadHeaderKVs(params map[string]interface{}) []auth.HeaderKV {
	raw, ok := params["headers"]
	if !ok || raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case []auth.HeaderKV:
		return v
	case []interface{}:
		headers := make([]auth.HeaderKV, 0, len(v))
		for _, item := range v {
			if h, ok := item.(map[string]interface{}); ok {
				headers = append(headers, auth.HeaderKV{
					Key:   stringFromMap(h, "key"),
					Value: stringFromMap(h, "value"),
				})
			}
		}
		return headers
	}
	return nil
}

func loadCookies(params map[string]interface{}) []auth.Cookie {
	raw, ok := params["cookies"]
	if !ok || raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case []auth.Cookie:
		return v
	case []interface{}:
		cookies := make([]auth.Cookie, 0, len(v))
		for _, item := range v {
			if c, ok := item.(map[string]interface{}); ok {
				cookies = append(cookies, auth.Cookie{
					Name:   stringFromMap(c, "name"),
					Value:  stringFromMap(c, "value"),
					Domain: stringFromMap(c, "domain"),
					Path:   stringFromMap(c, "path"),
				})
			}
		}
		return cookies
	}
	return nil
}

func loadTokens(params map[string]interface{}) []auth.Token {
	if params == nil {
		return nil
	}

	tokens := make([]auth.Token, 0)

	// Basic auth token
	if basic := stringParam(params, "authBasic"); basic != "" {
		tokens = append(tokens, auth.Token{Kind: auth.TokenBasic, Value: basic})
	}

	// Other tokens
	tokenKind := parseTokenKind(stringParam(params, "tokenKind"))
	tokenValues := stringSliceParam(params, "tokens")
	for _, value := range tokenValues {
		if strings.TrimSpace(value) == "" {
			continue
		}
		tokens = append(tokens, auth.Token{
			Kind:   tokenKind,
			Value:  value,
			Header: stringParam(params, "tokenHeader"),
			Query:  stringParam(params, "tokenQuery"),
			Cookie: stringParam(params, "tokenCookie"),
		})
	}

	if len(tokens) == 0 {
		return nil
	}
	return tokens
}

func loadLoginFlow(params map[string]interface{}) *auth.LoginFlow {
	loginURL := stringParam(params, "loginURL")
	if loginURL == "" {
		return nil
	}
	return &auth.LoginFlow{
		URL:            loginURL,
		UserSelector:   stringParam(params, "loginUserSelector"),
		PassSelector:   stringParam(params, "loginPassSelector"),
		SubmitSelector: stringParam(params, "loginSubmitSelector"),
		Username:       stringParam(params, "loginUser"),
		Password:       stringParam(params, "loginPass"),
	}
}

func stringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func parseTokenKind(kind string) auth.TokenKind {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "basic":
		return auth.TokenBasic
	case "api_key", "api-key", "apikey":
		return auth.TokenApiKey
	default:
		return auth.TokenBearer
	}
}
