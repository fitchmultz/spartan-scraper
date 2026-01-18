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
	queue          chan model.Job
}

func NewManager(store *store.Store, dataDir, userAgent string, requestTimeout time.Duration, maxConcurrency int) *Manager {
	return &Manager{
		store:          store,
		dataDir:        dataDir,
		userAgent:      userAgent,
		requestTimeout: requestTimeout,
		maxConcurrency: maxConcurrency,
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

func (m *Manager) Enqueue(job model.Job) error {
	select {
	case m.queue <- job:
		return nil
	default:
		return errors.New("job queue full")
	}
}

func (m *Manager) CreateScrapeJob(url string, headless bool, auth fetch.AuthOptions, timeoutSeconds int) (model.Job, error) {
	job := model.Job{
		ID:        uuid.NewString(),
		Kind:      model.KindScrape,
		Status:    model.StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Params: map[string]interface{}{
			"url":      url,
			"headless": headless,
			"auth":     auth,
			"timeout":  timeoutSeconds,
		},
	}
	job.ResultPath = filepath.Join(m.dataDir, "jobs", job.ID, "results.jsonl")
	if err := m.store.Create(job); err != nil {
		return model.Job{}, err
	}
	return job, nil
}

func (m *Manager) CreateCrawlJob(url string, maxDepth, maxPages int, headless bool, auth fetch.AuthOptions, timeoutSeconds int) (model.Job, error) {
	job := model.Job{
		ID:        uuid.NewString(),
		Kind:      model.KindCrawl,
		Status:    model.StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Params: map[string]interface{}{
			"url":      url,
			"maxDepth": maxDepth,
			"maxPages": maxPages,
			"headless": headless,
			"auth":     auth,
			"timeout":  timeoutSeconds,
		},
	}
	job.ResultPath = filepath.Join(m.dataDir, "jobs", job.ID, "results.jsonl")
	if err := m.store.Create(job); err != nil {
		return model.Job{}, err
	}
	return job, nil
}

func (m *Manager) CreateResearchJob(query string, urls []string, maxDepth, maxPages int, headless bool, auth fetch.AuthOptions, timeoutSeconds int) (model.Job, error) {
	job := model.Job{
		ID:        uuid.NewString(),
		Kind:      model.KindResearch,
		Status:    model.StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Params: map[string]interface{}{
			"query":    query,
			"urls":     urls,
			"maxDepth": maxDepth,
			"maxPages": maxPages,
			"headless": headless,
			"auth":     auth,
			"timeout":  timeoutSeconds,
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
		timeoutSecs := toInt(job.Params["timeout"], int(m.requestTimeout.Seconds()))
		auth := decodeAuth(job.Params["auth"])
		result, err := scrape.Run(scrape.Request{
			URL:       url,
			Headless:  headless,
			Auth:      auth,
			Timeout:   time.Duration(timeoutSecs) * time.Second,
			UserAgent: m.userAgent,
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
		timeoutSecs := toInt(job.Params["timeout"], int(m.requestTimeout.Seconds()))
		auth := decodeAuth(job.Params["auth"])
		results, err := crawl.Run(crawl.Request{
			URL:       url,
			MaxDepth:  maxDepth,
			MaxPages:  maxPages,
			Headless:  headless,
			Auth:      auth,
			Timeout:   time.Duration(timeoutSecs) * time.Second,
			UserAgent: m.userAgent,
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
		timeoutSecs := toInt(job.Params["timeout"], int(m.requestTimeout.Seconds()))
		auth := decodeAuth(job.Params["auth"])
		result, err := research.Run(research.Request{
			Query:     query,
			URLs:      urls,
			MaxDepth:  maxDepth,
			MaxPages:  maxPages,
			Headless:  headless,
			Auth:      auth,
			Timeout:   time.Duration(timeoutSecs) * time.Second,
			UserAgent: m.userAgent,
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
	return auth
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
