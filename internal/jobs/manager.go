package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"spartan-scraper/internal/crawl"
	"spartan-scraper/internal/extract"
	"spartan-scraper/internal/fetch"
	"spartan-scraper/internal/model"
	"spartan-scraper/internal/research"
	"spartan-scraper/internal/scrape"
	"spartan-scraper/internal/store"
)

type Manager struct {
	store          *store.Store
	dataDir        string
	userAgent      string
	requestTimeout time.Duration
	maxConcurrency int
	limiter        *fetch.HostLimiter
	maxRetries     int
	retryBase      time.Duration
	usePlaywright  bool
	queue          chan model.Job
}

func NewManager(store *store.Store, dataDir, userAgent string, requestTimeout time.Duration, maxConcurrency int, rateLimitQPS int, rateLimitBurst int, maxRetries int, retryBase time.Duration, usePlaywright bool) *Manager {
	return &Manager{
		store:          store,
		dataDir:        dataDir,
		userAgent:      userAgent,
		requestTimeout: requestTimeout,
		maxConcurrency: maxConcurrency,
		limiter:        fetch.NewHostLimiter(rateLimitQPS, rateLimitBurst),
		maxRetries:     maxRetries,
		retryBase:      retryBase,
		usePlaywright:  usePlaywright,
		queue:          make(chan model.Job, 128),
	}
}

func (m *Manager) Start(ctx context.Context) {
	for i := 0; i < m.maxConcurrency; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case job := <-m.queue:
					_ = m.run(job)
				}
			}
		}()
	}
}

func (m *Manager) DefaultTimeoutSeconds() int {
	return int(m.requestTimeout.Seconds())
}

func (m *Manager) DefaultUsePlaywright() bool {
	return m.usePlaywright
}

func (m *Manager) Enqueue(job model.Job) error {
	select {
	case m.queue <- job:
		return nil
	default:
		return errors.New("job queue full")
	}
}

func (m *Manager) CreateScrapeJob(url string, headless bool, usePlaywright bool, auth fetch.AuthOptions, timeoutSeconds int, extractOpts extract.ExtractOptions, incremental bool) (model.Job, error) {
	job := model.Job{
		ID:        uuid.NewString(),
		Kind:      model.KindScrape,
		Status:    model.StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Params: map[string]interface{}{
			"url":         url,
			"headless":    headless,
			"playwright":  usePlaywright,
			"auth":        auth,
			"extract":     extractOpts,
			"timeout":     timeoutSeconds,
			"incremental": incremental,
		},
	}
	job.ResultPath = filepath.Join(m.dataDir, "jobs", job.ID, "results.jsonl")
	if err := m.store.Create(job); err != nil {
		return model.Job{}, err
	}
	return job, nil
}

func (m *Manager) CreateCrawlJob(url string, maxDepth, maxPages int, headless bool, usePlaywright bool, auth fetch.AuthOptions, timeoutSeconds int, extractOpts extract.ExtractOptions, incremental bool) (model.Job, error) {
	job := model.Job{
		ID:        uuid.NewString(),
		Kind:      model.KindCrawl,
		Status:    model.StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Params: map[string]interface{}{
			"url":         url,
			"maxDepth":    maxDepth,
			"maxPages":    maxPages,
			"headless":    headless,
			"playwright":  usePlaywright,
			"auth":        auth,
			"extract":     extractOpts,
			"timeout":     timeoutSeconds,
			"incremental": incremental,
		},
	}
	job.ResultPath = filepath.Join(m.dataDir, "jobs", job.ID, "results.jsonl")
	if err := m.store.Create(job); err != nil {
		return model.Job{}, err
	}
	return job, nil
}

func (m *Manager) CreateResearchJob(query string, urls []string, maxDepth, maxPages int, headless bool, usePlaywright bool, auth fetch.AuthOptions, timeoutSeconds int, extractOpts extract.ExtractOptions, incremental bool) (model.Job, error) {
	job := model.Job{
		ID:        uuid.NewString(),
		Kind:      model.KindResearch,
		Status:    model.StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Params: map[string]interface{}{
			"query":       query,
			"urls":        urls,
			"maxDepth":    maxDepth,
			"maxPages":    maxPages,
			"headless":    headless,
			"playwright":  usePlaywright,
			"auth":        auth,
			"extract":     extractOpts,
			"timeout":     timeoutSeconds,
			"incremental": incremental,
		},
	}
	job.ResultPath = filepath.Join(m.dataDir, "jobs", job.ID, "results.jsonl")
	if err := m.store.Create(job); err != nil {
		return model.Job{}, err
	}
	return job, nil
}

func (m *Manager) run(job model.Job) error {
	_ = m.store.UpdateStatus(job.ID, model.StatusRunning, "")

	resultDir := filepath.Dir(job.ResultPath)
	if err := os.MkdirAll(resultDir, 0o755); err != nil {
		_ = m.store.UpdateStatus(job.ID, model.StatusFailed, err.Error())
		return err
	}

	file, err := os.Create(job.ResultPath)
	if err != nil {
		_ = m.store.UpdateStatus(job.ID, model.StatusFailed, err.Error())
		return err
	}
	defer file.Close()

	switch job.Kind {
	case model.KindScrape:
		url, _ := job.Params["url"].(string)
		headless, _ := job.Params["headless"].(bool)
		usePlaywright := toBool(job.Params["playwright"], m.usePlaywright)
		timeoutSecs := toInt(job.Params["timeout"], int(m.requestTimeout.Seconds()))
		auth := decodeAuth(job.Params["auth"])
		extractOpts := decodeExtract(job.Params["extract"])
		incremental := toBool(job.Params["incremental"], false)
		result, err := scrape.Run(scrape.Request{
			URL:           url,
			Headless:      headless,
			UsePlaywright: usePlaywright,
			Auth:          auth,
			Extract:       extractOpts,
			Timeout:       time.Duration(timeoutSecs) * time.Second,
			UserAgent:     m.userAgent,
			Limiter:       m.limiter,
			MaxRetries:    m.maxRetries,
			RetryBase:     m.retryBase,
			DataDir:       m.dataDir,
			Incremental:   incremental,
			Store:         m.store,
		})
		if err != nil {
			_ = m.store.UpdateStatus(job.ID, model.StatusFailed, err.Error())
			return err
		}
		payload, _ := json.Marshal(result)
		_, _ = file.Write(append(payload, '\n'))
	case model.KindCrawl:
		url, _ := job.Params["url"].(string)
		maxDepth := toInt(job.Params["maxDepth"], 2)
		maxPages := toInt(job.Params["maxPages"], 200)
		headless, _ := job.Params["headless"].(bool)
		usePlaywright := toBool(job.Params["playwright"], m.usePlaywright)
		timeoutSecs := toInt(job.Params["timeout"], int(m.requestTimeout.Seconds()))
		auth := decodeAuth(job.Params["auth"])
		extractOpts := decodeExtract(job.Params["extract"])
		incremental := toBool(job.Params["incremental"], false)
		results, err := crawl.Run(crawl.Request{
			URL:           url,
			MaxDepth:      maxDepth,
			MaxPages:      maxPages,
			Concurrency:   m.maxConcurrency,
			Headless:      headless,
			UsePlaywright: usePlaywright,
			Auth:          auth,
			Extract:       extractOpts,
			Timeout:       time.Duration(timeoutSecs) * time.Second,
			UserAgent:     m.userAgent,
			Limiter:       m.limiter,
			MaxRetries:    m.maxRetries,
			RetryBase:     m.retryBase,
			DataDir:       m.dataDir,
			Incremental:   incremental,
			Store:         m.store,
		})
		if err != nil {
			_ = m.store.UpdateStatus(job.ID, model.StatusFailed, err.Error())
			return err
		}
		for _, item := range results {
			payload, _ := json.Marshal(item)
			_, _ = file.Write(append(payload, '\n'))
		}
	case model.KindResearch:
		query, _ := job.Params["query"].(string)
		urls := toStringSlice(job.Params["urls"])
		maxDepth := toInt(job.Params["maxDepth"], 2)
		maxPages := toInt(job.Params["maxPages"], 200)
		headless, _ := job.Params["headless"].(bool)
		usePlaywright := toBool(job.Params["playwright"], m.usePlaywright)
		timeoutSecs := toInt(job.Params["timeout"], int(m.requestTimeout.Seconds()))
		auth := decodeAuth(job.Params["auth"])
		extractOpts := decodeExtract(job.Params["extract"])
		incremental := toBool(job.Params["incremental"], false)
		result, err := research.Run(research.Request{
			Query:         query,
			URLs:          urls,
			MaxDepth:      maxDepth,
			MaxPages:      maxPages,
			Concurrency:   m.maxConcurrency,
			Headless:      headless,
			UsePlaywright: usePlaywright,
			Auth:          auth,
			Extract:       extractOpts,
			Timeout:       time.Duration(timeoutSecs) * time.Second,
			UserAgent:     m.userAgent,
			Limiter:       m.limiter,
			MaxRetries:    m.maxRetries,
			RetryBase:     m.retryBase,
			DataDir:       m.dataDir,
			Incremental:   incremental,
			Store:         m.store,
		})
		if err != nil {
			_ = m.store.UpdateStatus(job.ID, model.StatusFailed, err.Error())
			return err
		}
		payload, _ := json.Marshal(result)
		_, _ = file.Write(append(payload, '\n'))
	default:
		_ = m.store.UpdateStatus(job.ID, model.StatusFailed, "unknown job kind")
		return errors.New("unknown job kind")
	}

	_ = m.store.UpdateStatus(job.ID, model.StatusSucceeded, "")
	return nil
}

func decodeAuth(value interface{}) fetch.AuthOptions {
	if value == nil {
		return fetch.AuthOptions{}
	}
	if auth, ok := value.(fetch.AuthOptions); ok {
		return auth
	}
	data, ok := value.(map[string]interface{})
	if !ok {
		return fetch.AuthOptions{}
	}
	auth := fetch.AuthOptions{}
	if v, ok := data["basic"].(string); ok {
		auth.Basic = v
	}
	if v, ok := data["loginUrl"].(string); ok {
		auth.LoginURL = v
	}
	if v, ok := data["loginUserSelector"].(string); ok {
		auth.LoginUserSelector = v
	}
	if v, ok := data["loginPassSelector"].(string); ok {
		auth.LoginPassSelector = v
	}
	if v, ok := data["loginSubmitSelector"].(string); ok {
		auth.LoginSubmitSelector = v
	}
	if v, ok := data["loginUser"].(string); ok {
		auth.LoginUser = v
	}
	if v, ok := data["loginPass"].(string); ok {
		auth.LoginPass = v
	}
	if headers, ok := data["headers"].(map[string]interface{}); ok {
		m := map[string]string{}
		for k, v := range headers {
			if sv, ok := v.(string); ok {
				m[k] = sv
			}
		}
		auth.Headers = m
	}
	if cookies, ok := data["cookies"].([]interface{}); ok {
		values := make([]string, 0, len(cookies))
		for _, v := range cookies {
			if sv, ok := v.(string); ok {
				values = append(values, sv)
			}
		}
		auth.Cookies = values
	}
	if query, ok := data["query"].(map[string]interface{}); ok {
		m := map[string]string{}
		for k, v := range query {
			if sv, ok := v.(string); ok {
				m[k] = sv
			}
		}
		auth.Query = m
	}
	return auth
}

func decodeExtract(value interface{}) extract.ExtractOptions {
	if value == nil {
		return extract.ExtractOptions{}
	}
	if opts, ok := value.(extract.ExtractOptions); ok {
		return opts
	}
	data, err := json.Marshal(value)
	if err != nil {
		return extract.ExtractOptions{}
	}
	var opts extract.ExtractOptions
	if err := json.Unmarshal(data, &opts); err != nil {
		return extract.ExtractOptions{}
	}
	return opts
}

func toInt(value interface{}, fallback int) int {
	switch v := value.(type) {
	case int:
		if v <= 0 {
			return fallback
		}
		return v
	case float64:
		if int(v) <= 0 {
			return fallback
		}
		return int(v)
	default:
		return fallback
	}
}

func toStringSlice(value interface{}) []string {
	switch v := value.(type) {
	case []string:
		return v
	case []interface{}:
		items := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				items = append(items, s)
			}
		}
		return items
	default:
		return nil
	}
}

func toBool(value interface{}, fallback bool) bool {
	switch v := value.(type) {
	case bool:
		return v
	default:
		return fallback
	}
}
