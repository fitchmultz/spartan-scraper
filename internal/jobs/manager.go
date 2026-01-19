package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"

	"spartan-scraper/internal/crawl"
	"spartan-scraper/internal/extract"
	"spartan-scraper/internal/fetch"
	"spartan-scraper/internal/model"
	"spartan-scraper/internal/pipeline"
	"spartan-scraper/internal/research"
	"spartan-scraper/internal/scrape"
	"spartan-scraper/internal/store"
)

type Manager struct {
	store            *store.Store
	dataDir          string
	userAgent        string
	requestTimeout   time.Duration
	maxConcurrency   int
	limiter          *fetch.HostLimiter
	maxRetries       int
	retryBase        time.Duration
	usePlaywright    bool
	queue            chan model.Job
	pipelineRegistry *pipeline.Registry
	jsRegistry       *pipeline.JSRegistry
	wg               sync.WaitGroup
}

func NewManager(store *store.Store, dataDir, userAgent string, requestTimeout time.Duration, maxConcurrency int, rateLimitQPS int, rateLimitBurst int, maxRetries int, retryBase time.Duration, usePlaywright bool) *Manager {
	jsRegistry, err := pipeline.LoadJSRegistry(dataDir)
	if err != nil {
		slog.Warn("failed to load JS registry", "error", err)
	}
	return &Manager{
		store:            store,
		dataDir:          dataDir,
		userAgent:        userAgent,
		requestTimeout:   requestTimeout,
		maxConcurrency:   maxConcurrency,
		limiter:          fetch.NewHostLimiter(rateLimitQPS, rateLimitBurst),
		maxRetries:       maxRetries,
		retryBase:        retryBase,
		usePlaywright:    usePlaywright,
		queue:            make(chan model.Job, 128),
		pipelineRegistry: pipeline.NewRegistry(),
		jsRegistry:       jsRegistry,
	}
}

func (m *Manager) Start(ctx context.Context) {
	slog.Info("starting job manager", "concurrency", m.maxConcurrency)
	for i := 0; i < m.maxConcurrency; i++ {
		m.wg.Add(1)
		go func(workerID int) {
			defer m.wg.Done()
			slog.Debug("starting worker", "workerID", workerID)
			for {
				select {
				case <-ctx.Done():
					slog.Debug("stopping worker, draining queue", "workerID", workerID)
					// Drain the queue before returning
					for {
						select {
						case job, ok := <-m.queue:
							if !ok {
								return
							}
							slog.Info("worker picked up job (draining)", "workerID", workerID, "jobID", job.ID, "kind", job.Kind)
							if err := m.run(job); err != nil {
								slog.Error("job failed during drain", "jobID", job.ID, "error", err)
							}
						default:
							return
						}
					}
				case job, ok := <-m.queue:
					if !ok {
						slog.Debug("job queue closed, stopping worker", "workerID", workerID)
						return
					}
					slog.Info("worker picked up job", "workerID", workerID, "jobID", job.ID, "kind", job.Kind)
					if err := m.run(job); err != nil {
						slog.Error("job failed", "jobID", job.ID, "error", err)
					}
				}
			}
		}(i)
	}
}

func (m *Manager) Wait() {
	m.wg.Wait()
}

func (m *Manager) DefaultTimeoutSeconds() int {
	return int(m.requestTimeout.Seconds())
}

func (m *Manager) DefaultUsePlaywright() bool {
	return m.usePlaywright
}

func (m *Manager) Enqueue(job model.Job) error {
	slog.Debug("enqueuing job", "jobID", job.ID, "kind", job.Kind)
	select {
	case m.queue <- job:
		return nil
	default:
		slog.Warn("job queue full", "jobID", job.ID)
		return errors.New("job queue full")
	}
}

func (m *Manager) CreateScrapeJob(url string, headless bool, usePlaywright bool, auth fetch.AuthOptions, timeoutSeconds int, extractOpts extract.ExtractOptions, pipelineOpts pipeline.Options, incremental bool) (model.Job, error) {
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
			"pipeline":    pipelineOpts,
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

func (m *Manager) CreateCrawlJob(url string, maxDepth, maxPages int, headless bool, usePlaywright bool, auth fetch.AuthOptions, timeoutSeconds int, extractOpts extract.ExtractOptions, pipelineOpts pipeline.Options, incremental bool) (model.Job, error) {
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
			"pipeline":    pipelineOpts,
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

func (m *Manager) CreateResearchJob(query string, urls []string, maxDepth, maxPages int, headless bool, usePlaywright bool, auth fetch.AuthOptions, timeoutSeconds int, extractOpts extract.ExtractOptions, pipelineOpts pipeline.Options, incremental bool) (model.Job, error) {
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
			"pipeline":    pipelineOpts,
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
	slog.Info("running job", "jobID", job.ID, "kind", job.Kind)
	if err := m.store.UpdateStatus(job.ID, model.StatusRunning, ""); err != nil {
		slog.Error("failed to update job status to running", "jobID", job.ID, "error", err)
	}

	resultDir := filepath.Dir(job.ResultPath)
	if err := os.MkdirAll(resultDir, 0o755); err != nil {
		slog.Error("failed to create result directory", "jobID", job.ID, "error", err)
		if err := m.store.UpdateStatus(job.ID, model.StatusFailed, err.Error()); err != nil {
			slog.Error("failed to update job status to failed", "jobID", job.ID, "error", err)
		}
		return err
	}

	file, err := os.Create(job.ResultPath)
	if err != nil {
		slog.Error("failed to create result file", "jobID", job.ID, "error", err)
		if err := m.store.UpdateStatus(job.ID, model.StatusFailed, err.Error()); err != nil {
			slog.Error("failed to update job status to failed", "jobID", job.ID, "error", err)
		}
		return err
	}
	defer file.Close()

	switch job.Kind {
	case model.KindScrape:
		url, _ := job.Params["url"].(string)
		slog.Info("processing scrape job", "jobID", job.ID, "url", url)
		headless, _ := job.Params["headless"].(bool)
		usePlaywright := toBool(job.Params["playwright"], m.usePlaywright)
		timeoutSecs := toInt(job.Params["timeout"], int(m.requestTimeout.Seconds()))
		auth := decodeAuth(job.Params["auth"])
		extractOpts := decodeExtract(job.Params["extract"])
		pipelineOpts := decodePipeline(job.Params["pipeline"])
		incremental := toBool(job.Params["incremental"], false)
		result, err := scrape.Run(scrape.Request{
			URL:           url,
			RequestID:     job.ID,
			Headless:      headless,
			UsePlaywright: usePlaywright,
			Auth:          auth,
			Extract:       extractOpts,
			Pipeline:      pipelineOpts,
			Timeout:       time.Duration(timeoutSecs) * time.Second,
			UserAgent:     m.userAgent,
			Limiter:       m.limiter,
			MaxRetries:    m.maxRetries,
			RetryBase:     m.retryBase,
			DataDir:       m.dataDir,
			Incremental:   incremental,
			Store:         m.store,
			Registry:      m.pipelineRegistry,
			JSRegistry:    m.jsRegistry,
		})
		if err != nil {
			slog.Error("scrape job failed", "jobID", job.ID, "url", url, "error", err)
			if err := m.store.UpdateStatus(job.ID, model.StatusFailed, err.Error()); err != nil {
				slog.Error("failed to update job status to failed", "jobID", job.ID, "error", err)
			}
			return err
		}
		payload, err := json.Marshal(result)
		if err != nil {
			slog.Error("failed to marshal scrape result", "jobID", job.ID, "error", err)
			return err
		}
		if _, err := file.Write(append(payload, '\n')); err != nil {
			slog.Error("failed to write scrape result", "jobID", job.ID, "error", err)
			return err
		}
	case model.KindCrawl:
		url, _ := job.Params["url"].(string)
		slog.Info("processing crawl job", "jobID", job.ID, "url", url)
		maxDepth := toInt(job.Params["maxDepth"], 2)
		maxPages := toInt(job.Params["maxPages"], 200)
		headless, _ := job.Params["headless"].(bool)
		usePlaywright := toBool(job.Params["playwright"], m.usePlaywright)
		timeoutSecs := toInt(job.Params["timeout"], int(m.requestTimeout.Seconds()))
		auth := decodeAuth(job.Params["auth"])
		extractOpts := decodeExtract(job.Params["extract"])
		pipelineOpts := decodePipeline(job.Params["pipeline"])
		incremental := toBool(job.Params["incremental"], false)
		results, err := crawl.Run(crawl.Request{
			URL:           url,
			RequestID:     job.ID,
			MaxDepth:      maxDepth,
			MaxPages:      maxPages,
			Concurrency:   m.maxConcurrency,
			Headless:      headless,
			UsePlaywright: usePlaywright,
			Auth:          auth,
			Extract:       extractOpts,
			Pipeline:      pipelineOpts,
			Timeout:       time.Duration(timeoutSecs) * time.Second,
			UserAgent:     m.userAgent,
			Limiter:       m.limiter,
			MaxRetries:    m.maxRetries,
			RetryBase:     m.retryBase,
			DataDir:       m.dataDir,
			Incremental:   incremental,
			Store:         m.store,
			Registry:      m.pipelineRegistry,
			JSRegistry:    m.jsRegistry,
		})
		if err != nil {
			slog.Error("crawl job failed", "jobID", job.ID, "url", url, "error", err)
			if err := m.store.UpdateStatus(job.ID, model.StatusFailed, err.Error()); err != nil {
				slog.Error("failed to update job status to failed", "jobID", job.ID, "error", err)
			}
			return err
		}
		for _, item := range results {
			payload, err := json.Marshal(item)
			if err != nil {
				slog.Error("failed to marshal crawl result item", "jobID", job.ID, "error", err)
				continue
			}
			if _, err := file.Write(append(payload, '\n')); err != nil {
				slog.Error("failed to write crawl result item", "jobID", job.ID, "error", err)
				return err
			}
		}
	case model.KindResearch:
		query, _ := job.Params["query"].(string)
		slog.Info("processing research job", "jobID", job.ID, "query", query)
		urls := toStringSlice(job.Params["urls"])
		maxDepth := toInt(job.Params["maxDepth"], 2)
		maxPages := toInt(job.Params["maxPages"], 200)
		headless, _ := job.Params["headless"].(bool)
		usePlaywright := toBool(job.Params["playwright"], m.usePlaywright)
		timeoutSecs := toInt(job.Params["timeout"], int(m.requestTimeout.Seconds()))
		auth := decodeAuth(job.Params["auth"])
		extractOpts := decodeExtract(job.Params["extract"])
		pipelineOpts := decodePipeline(job.Params["pipeline"])
		incremental := toBool(job.Params["incremental"], false)
		result, err := research.Run(research.Request{
			Query:         query,
			RequestID:     job.ID,
			URLs:          urls,
			MaxDepth:      maxDepth,
			MaxPages:      maxPages,
			Concurrency:   m.maxConcurrency,
			Headless:      headless,
			UsePlaywright: usePlaywright,
			Auth:          auth,
			Extract:       extractOpts,
			Pipeline:      pipelineOpts,
			Timeout:       time.Duration(timeoutSecs) * time.Second,
			UserAgent:     m.userAgent,
			Limiter:       m.limiter,
			MaxRetries:    m.maxRetries,
			RetryBase:     m.retryBase,
			DataDir:       m.dataDir,
			Incremental:   incremental,
			Store:         m.store,
			Registry:      m.pipelineRegistry,
			JSRegistry:    m.jsRegistry,
		})
		if err != nil {
			slog.Error("research job failed", "jobID", job.ID, "query", query, "error", err)
			if err := m.store.UpdateStatus(job.ID, model.StatusFailed, err.Error()); err != nil {
				slog.Error("failed to update job status to failed", "jobID", job.ID, "error", err)
			}
			return err
		}
		payload, err := json.Marshal(result)
		if err != nil {
			slog.Error("failed to marshal research result", "jobID", job.ID, "error", err)
			return err
		}
		if _, err := file.Write(append(payload, '\n')); err != nil {
			slog.Error("failed to write research result", "jobID", job.ID, "error", err)
			return err
		}
	default:
		slog.Error("unknown job kind", "jobID", job.ID, "kind", job.Kind)
		if err := m.store.UpdateStatus(job.ID, model.StatusFailed, "unknown job kind"); err != nil {
			slog.Error("failed to update job status to failed", "jobID", job.ID, "error", err)
		}
		return errors.New("unknown job kind")
	}

	slog.Info("job succeeded", "jobID", job.ID)
	if err := m.store.UpdateStatus(job.ID, model.StatusSucceeded, ""); err != nil {
		slog.Error("failed to update job status to succeeded", "jobID", job.ID, "error", err)
	}
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

func decodePipeline(value interface{}) pipeline.Options {
	if value == nil {
		return pipeline.Options{}
	}
	if opts, ok := value.(pipeline.Options); ok {
		return opts
	}
	data, err := json.Marshal(value)
	if err != nil {
		return pipeline.Options{}
	}
	var opts pipeline.Options
	if err := json.Unmarshal(data, &opts); err != nil {
		return pipeline.Options{}
	}
	return opts
}
