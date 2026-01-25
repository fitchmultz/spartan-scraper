// Package mcp implements a Model Context Protocol (MCP) server for Spartan Scraper.
// It provides JSON-RPC based tools for scraping, crawling, and research operations
// that can be consumed by MCP-compatible clients.
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"spartan-scraper/internal/auth"
	"spartan-scraper/internal/config"
	"spartan-scraper/internal/extract"
	"spartan-scraper/internal/fetch"
	"spartan-scraper/internal/jobs"
	"spartan-scraper/internal/pipeline"
	"spartan-scraper/internal/store"
	"spartan-scraper/internal/validate"
)

type Server struct {
	store   *store.Store
	manager *jobs.Manager
	cfg     config.Config
	ctx     context.Context
	cancel  context.CancelFunc
}

type request struct {
	ID     interface{}       `json:"id"`
	Method string            `json:"method"`
	Params map[string]string `json:"params"`
}

type response struct {
	ID     interface{} `json:"id"`
	Result interface{} `json:"result,omitempty"`
	Error  *rpcError   `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

type callParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

func NewServer(cfg config.Config) (*Server, error) {
	st, err := store.Open(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	mgr := jobs.NewManager(
		st,
		cfg.DataDir,
		cfg.UserAgent,
		time.Duration(cfg.RequestTimeoutSecs)*time.Second,
		cfg.MaxConcurrency,
		cfg.RateLimitQPS,
		cfg.RateLimitBurst,
		cfg.MaxRetries,
		time.Duration(cfg.RetryBaseMs)*time.Millisecond,
		cfg.MaxResponseBytes,
		cfg.UsePlaywright,
	)
	ctx, cancel := context.WithCancel(context.Background())
	mgr.Start(ctx)
	return &Server{
		store:   st,
		manager: mgr,
		cfg:     cfg,
		ctx:     ctx,
		cancel:  cancel,
	}, nil
}

func (s *Server) Close() error {
	// Cancel the manager's context first
	if s.cancel != nil {
		s.cancel()
	}

	// Wait for all manager goroutines to finish
	if s.manager != nil {
		s.manager.Wait()
	}

	// Finally close the store
	if s.store != nil {
		return s.store.Close()
	}
	return nil
}

func (s *Server) Serve(ctx context.Context, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	encoder := json.NewEncoder(out)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var base map[string]json.RawMessage
		if err := json.Unmarshal(line, &base); err != nil {
			_ = encoder.Encode(response{ID: nil, Error: &rpcError{Code: -32700, Message: "parse error"}})
			continue
		}

		var method string
		if raw, ok := base["method"]; ok {
			_ = json.Unmarshal(raw, &method)
		}

		var id interface{}
		if raw, ok := base["id"]; ok {
			_ = json.Unmarshal(raw, &id)
		}

		switch method {
		case "initialize":
			_ = encoder.Encode(response{ID: id, Result: map[string]interface{}{
				"name":    "spartan-scraper-mcp",
				"version": "0.1.0",
			}})
		case "tools/list":
			_ = encoder.Encode(response{ID: id, Result: map[string]interface{}{"tools": s.toolsList()}})
		case "tools/call":
			result, err := s.handleToolCall(ctx, base)
			if err != nil {
				_ = encoder.Encode(response{ID: id, Error: &rpcError{Code: -32000, Message: err.Error()}})
				continue
			}
			_ = encoder.Encode(response{ID: id, Result: result})
		default:
			_ = encoder.Encode(response{ID: id, Error: &rpcError{Code: -32601, Message: "method not found"}})
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func (s *Server) toolsList() []tool {
	return []tool{
		{
			Name:        "scrape_page",
			Description: "Scrape a single page (static or headless)",
			InputSchema: schema(map[string]string{"url": "string"}, map[string]string{"authProfile": "string", "headless": "boolean", "playwright": "boolean", "timeoutSeconds": "number", "extractTemplate": "string", "extractValidate": "boolean", "preProcessors": "array", "postProcessors": "array", "transformers": "array", "incremental": "boolean"}),
		},
		{
			Name:        "crawl_site",
			Description: "Crawl a site with depth and page limits",
			InputSchema: schema(map[string]string{"url": "string"}, map[string]string{"authProfile": "string", "maxDepth": "number", "maxPages": "number", "headless": "boolean", "playwright": "boolean", "timeoutSeconds": "number", "extractTemplate": "string", "extractValidate": "boolean", "preProcessors": "array", "postProcessors": "array", "transformers": "array", "incremental": "boolean"}),
		},
		{
			Name:        "research",
			Description: "Deep research across multiple sources",
			InputSchema: schema(map[string]string{"query": "string", "urls": "array"}, map[string]string{"authProfile": "string", "maxDepth": "number", "maxPages": "number", "headless": "boolean", "playwright": "boolean", "timeoutSeconds": "number", "extractTemplate": "string", "extractValidate": "boolean", "preProcessors": "array", "postProcessors": "array", "transformers": "array", "incremental": "boolean"}),
		},
		{
			Name:        "job_status",
			Description: "Get job status by id",
			InputSchema: schema(map[string]string{"id": "string"}, nil),
		},
		{
			Name:        "job_results",
			Description: "Get job results by id",
			InputSchema: schema(map[string]string{"id": "string"}, nil),
		},
	}
}

func schema(required map[string]string, optional map[string]string) map[string]interface{} {
	properties := map[string]interface{}{}
	requiredKeys := make([]string, 0)
	for key, kind := range required {
		properties[key] = map[string]string{"type": kind}
		requiredKeys = append(requiredKeys, key)
	}
	for key, kind := range optional {
		properties[key] = map[string]string{"type": kind}
	}
	return map[string]interface{}{
		"type":       "object",
		"properties": properties,
		"required":   requiredKeys,
	}
}

func (s *Server) handleToolCall(ctx context.Context, base map[string]json.RawMessage) (interface{}, error) {
	var params callParams
	if raw, ok := base["params"]; ok {
		if err := json.Unmarshal(raw, &params); err != nil {
			return nil, err
		}
	}

	switch params.Name {
	case "scrape_page":
		url := getString(params.Arguments, "url")
		if url == "" {
			return nil, errors.New("url is required")
		}
		authProfile := getString(params.Arguments, "authProfile")
		playwright := getBoolDefault(params.Arguments, "playwright", s.cfg.UsePlaywright)
		timeout := getInt(params.Arguments, "timeoutSeconds", s.cfg.RequestTimeoutSecs)
		validator := validate.ScrapeRequestValidator{
			URL:         url,
			Timeout:     timeout,
			AuthProfile: authProfile,
		}
		if err := validator.Validate(); err != nil {
			return nil, err
		}
		resolvedAuth, err := resolveAuthForTool(s.cfg, url, authProfile)
		if err != nil {
			return nil, err
		}
		headless := getBool(params.Arguments, "headless")
		extractOpts := extract.ExtractOptions{
			Template: getString(params.Arguments, "extractTemplate"),
			Validate: getBool(params.Arguments, "extractValidate"),
		}
		pipelineOpts := getPipelineOptions(params.Arguments)
		incremental := getBoolDefault(params.Arguments, "incremental", false)
		job, err := s.manager.CreateScrapeJob(ctx, url, headless, playwright, resolvedAuth, timeout, extractOpts, pipelineOpts, incremental)
		if err != nil {
			return nil, err
		}
		if err := s.manager.Enqueue(job); err != nil {
			return nil, err
		}
		if err := waitForJob(ctx, s.store, job.ID); err != nil {
			return nil, err
		}
		return loadResult(ctx, s.store, job.ID)
	case "crawl_site":
		url := getString(params.Arguments, "url")
		if url == "" {
			return nil, errors.New("url is required")
		}
		authProfile := getString(params.Arguments, "authProfile")
		maxDepth := getInt(params.Arguments, "maxDepth", 2)
		maxPages := getInt(params.Arguments, "maxPages", 200)
		playwright := getBoolDefault(params.Arguments, "playwright", s.cfg.UsePlaywright)
		timeout := getInt(params.Arguments, "timeoutSeconds", s.cfg.RequestTimeoutSecs)
		validator := validate.CrawlRequestValidator{
			URL:         url,
			MaxDepth:    maxDepth,
			MaxPages:    maxPages,
			Timeout:     timeout,
			AuthProfile: authProfile,
		}
		if err := validator.Validate(); err != nil {
			return nil, err
		}
		resolvedAuth, err := resolveAuthForTool(s.cfg, url, authProfile)
		if err != nil {
			return nil, err
		}
		headless := getBool(params.Arguments, "headless")
		extractOpts := extract.ExtractOptions{
			Template: getString(params.Arguments, "extractTemplate"),
			Validate: getBool(params.Arguments, "extractValidate"),
		}
		pipelineOpts := getPipelineOptions(params.Arguments)
		incremental := getBoolDefault(params.Arguments, "incremental", false)
		job, err := s.manager.CreateCrawlJob(ctx, url, maxDepth, maxPages, headless, playwright, resolvedAuth, timeout, extractOpts, pipelineOpts, incremental)
		if err != nil {
			return nil, err
		}
		if err := s.manager.Enqueue(job); err != nil {
			return nil, err
		}
		if err := waitForJob(ctx, s.store, job.ID); err != nil {
			return nil, err
		}
		return loadResult(ctx, s.store, job.ID)
	case "research":
		query := getString(params.Arguments, "query")
		urls := getStringSlice(params.Arguments, "urls")
		if query == "" || len(urls) == 0 {
			return nil, errors.New("query and urls are required")
		}
		authProfile := getString(params.Arguments, "authProfile")
		maxDepth := getInt(params.Arguments, "maxDepth", 2)
		maxPages := getInt(params.Arguments, "maxPages", 200)
		playwright := getBoolDefault(params.Arguments, "playwright", s.cfg.UsePlaywright)
		timeout := getInt(params.Arguments, "timeoutSeconds", s.cfg.RequestTimeoutSecs)
		validator := validate.ResearchRequestValidator{
			Query:       query,
			URLs:        urls,
			MaxDepth:    maxDepth,
			MaxPages:    maxPages,
			Timeout:     timeout,
			AuthProfile: authProfile,
		}
		if err := validator.Validate(); err != nil {
			return nil, err
		}
		targetURL := ""
		if len(urls) > 0 {
			targetURL = urls[0]
		}
		resolvedAuth, err := resolveAuthForTool(s.cfg, targetURL, authProfile)
		if err != nil {
			return nil, err
		}
		headless := getBool(params.Arguments, "headless")
		extractOpts := extract.ExtractOptions{
			Template: getString(params.Arguments, "extractTemplate"),
			Validate: getBool(params.Arguments, "extractValidate"),
		}
		pipelineOpts := getPipelineOptions(params.Arguments)
		incremental := getBoolDefault(params.Arguments, "incremental", false)
		job, err := s.manager.CreateResearchJob(ctx, query, urls, maxDepth, maxPages, headless, playwright, resolvedAuth, timeout, extractOpts, pipelineOpts, incremental)
		if err != nil {
			return nil, err
		}
		if err := s.manager.Enqueue(job); err != nil {
			return nil, err
		}
		if err := waitForJob(ctx, s.store, job.ID); err != nil {
			return nil, err
		}
		return loadResult(ctx, s.store, job.ID)
	case "job_status":
		id := getString(params.Arguments, "id")
		if id == "" {
			return nil, errors.New("id is required")
		}
		return s.store.Get(ctx, id)
	case "job_results":
		id := getString(params.Arguments, "id")
		if id == "" {
			return nil, errors.New("id is required")
		}
		return loadResult(ctx, s.store, id)
	default:
		return nil, fmt.Errorf("unknown tool: %s", params.Name)
	}
}

func waitForJob(ctx context.Context, store *store.Store, id string) error {
	for {
		job, err := store.Get(ctx, id)
		if err != nil {
			return err
		}
		switch job.Status {
		case "succeeded":
			return nil
		case "failed":
			if job.Error != "" {
				return fmt.Errorf("job failed: %s", job.Error)
			}
			return errors.New("job failed")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
}

func loadResult(ctx context.Context, store *store.Store, id string) (string, error) {
	job, err := store.Get(ctx, id)
	if err != nil {
		return "", err
	}
	if job.ResultPath == "" {
		return "", errors.New("no result path")
	}
	data, err := os.ReadFile(job.ResultPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func getString(args map[string]interface{}, key string) string {
	if args == nil {
		return ""
	}
	if value, ok := args[key].(string); ok {
		return value
	}
	return ""
}

func getBool(args map[string]interface{}, key string) bool {
	if args == nil {
		return false
	}
	if value, ok := args[key].(bool); ok {
		return value
	}
	return false
}

func getBoolDefault(args map[string]interface{}, key string, fallback bool) bool {
	if args == nil {
		return fallback
	}
	if _, ok := args[key]; !ok {
		return fallback
	}
	if value, ok := args[key].(bool); ok {
		return value
	}
	return fallback
}

func getInt(args map[string]interface{}, key string, fallback int) int {
	if args == nil {
		return fallback
	}
	switch value := args[key].(type) {
	case float64:
		if int(value) <= 0 {
			return fallback
		}
		return int(value)
	case int:
		if value <= 0 {
			return fallback
		}
		return value
	default:
		return fallback
	}
}

func getStringSlice(args map[string]interface{}, key string) []string {
	if args == nil {
		return nil
	}
	values, ok := args[key]
	if !ok {
		return nil
	}
	switch v := values.(type) {
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return v
	default:
		return nil
	}
}

// getPipelineOptions extracts pipeline options from MCP tool arguments.
// Returns empty Options if args is nil or no pipeline options are provided.
func getPipelineOptions(args map[string]interface{}) pipeline.Options {
	if args == nil {
		return pipeline.Options{}
	}
	return pipeline.Options{
		PreProcessors:  getStringSlice(args, "preProcessors"),
		PostProcessors: getStringSlice(args, "postProcessors"),
		Transformers:   getStringSlice(args, "transformers"),
	}
}

func resolveAuthForTool(cfg config.Config, url string, profile string) (fetch.AuthOptions, error) {
	input := auth.ResolveInput{
		ProfileName: profile,
		URL:         url,
		Env:         &cfg.AuthOverrides,
	}
	resolved, err := auth.Resolve(cfg.DataDir, input)
	if err != nil {
		return fetch.AuthOptions{}, err
	}
	return auth.ToFetchOptions(resolved), nil
}
