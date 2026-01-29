// Package mcp implements a Model Context Protocol (MCP) server for Spartan Scraper.
//
// Responsibilities:
// - Implement a JSON-RPC 2.0 based server over stdio.
// - Expose Spartan capabilities (scrape, crawl, research) as MCP tools.
// - Manage lifecycle and state for MCP sessions.
//
// Does NOT handle:
// - Implementation of the scraping or crawling logic itself.
// - Authentication/Authorization beyond what Spartan handles internally.
//
// Invariants/Assumptions:
// - Communicates over stdin/stdout as defined by the MCP stdio transport.
// - Expects a valid Spartan configuration and initialized services.
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/auth"
	"github.com/fitchmultz/spartan-scraper/internal/buildinfo"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/exporter"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
	"github.com/fitchmultz/spartan-scraper/internal/store"
	"github.com/fitchmultz/spartan-scraper/internal/validate"
	"sort"
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
				"version": buildinfo.Version,
			}})
		case "tools/list":
			_ = encoder.Encode(response{ID: id, Result: map[string]interface{}{"tools": s.toolsList()}})
		case "tools/call":
			result, err := s.handleToolCall(ctx, base)
			if err != nil {
				_ = encoder.Encode(response{ID: id, Error: &rpcError{Code: -32000, Message: apperrors.SafeMessage(err)}})
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
			InputSchema: schema(map[string]string{"query": "string", "urls": "array"}, map[string]string{"authProfile": "string", "maxDepth": "number", "maxPages": "number", "headless": "boolean", "playwright": "boolean", "timeoutSeconds": "number", "extractTemplate": "string", "extractValidate": "boolean", "preProcessors": "array", "postProcessors": "array", "transformers": "array"}),
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
		{
			Name:        "job_list",
			Description: "List all jobs with pagination",
			InputSchema: schema(nil, map[string]string{"limit": "number", "offset": "number"}),
		},
		{
			Name:        "job_cancel",
			Description: "Cancel a running or queued job by id",
			InputSchema: schema(map[string]string{"id": "string"}, nil),
		},
		{
			Name:        "job_export",
			Description: "Export job results in specified format (jsonl, json, md, csv)",
			InputSchema: schema(map[string]string{"id": "string"}, map[string]string{"format": "string"}),
		},
	}
}

func schema(required map[string]string, optional map[string]string) map[string]interface{} {
	properties := map[string]interface{}{}

	// Collect and sort all keys for deterministic ordering
	keys := make([]string, 0, len(required)+len(optional))
	for key := range required {
		keys = append(keys, key)
	}
	for key := range optional {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Build required keys slice (only from required map) in sorted order
	requiredKeys := make([]string, 0, len(required))
	for key := range required {
		requiredKeys = append(requiredKeys, key)
	}
	sort.Strings(requiredKeys)

	// Build properties map in sorted order
	for _, key := range keys {
		kind := ""
		if k, ok := required[key]; ok {
			kind = k
		} else if k, ok := optional[key]; ok {
			kind = k
		}
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
			return nil, apperrors.Validation("url is required")
		}
		authProfile := getString(params.Arguments, "authProfile")
		timeout := getInt(params.Arguments, "timeoutSeconds", s.cfg.RequestTimeoutSecs)
		opts := validate.JobValidationOpts{
			URL:         url,
			Timeout:     timeout,
			AuthProfile: authProfile,
		}
		if err := validate.ValidateJob(opts, model.KindScrape); err != nil {
			return nil, err
		}
		resolvedAuth, err := resolveAuthForTool(s.cfg, url, authProfile)
		if err != nil {
			return nil, err
		}
		extractOpts := extract.ExtractOptions{
			Template: getString(params.Arguments, "extractTemplate"),
			Validate: getBool(params.Arguments, "extractValidate"),
		}
		pipelineOpts := getPipelineOptions(params.Arguments)
		spec := jobs.JobSpec{
			Kind:           model.KindScrape,
			URL:            url,
			Headless:       getBool(params.Arguments, "headless"),
			UsePlaywright:  getBoolDefault(params.Arguments, "playwright", s.cfg.UsePlaywright),
			Auth:           resolvedAuth,
			TimeoutSeconds: timeout,
			Extract:        extractOpts,
			Pipeline:       pipelineOpts,
			Incremental:    getBoolDefault(params.Arguments, "incremental", false),
		}
		job, err := s.manager.CreateJob(ctx, spec)
		if err != nil {
			return nil, err
		}
		if err := s.manager.Enqueue(job); err != nil {
			return nil, err
		}
		if err := waitForJob(ctx, s.store, job.ID, timeout); err != nil {
			return nil, err
		}
		return loadResult(ctx, s.store, job.ID)
	case "crawl_site":
		url := getString(params.Arguments, "url")
		if url == "" {
			return nil, apperrors.Validation("url is required")
		}
		authProfile := getString(params.Arguments, "authProfile")
		maxDepth := getInt(params.Arguments, "maxDepth", 2)
		maxPages := getInt(params.Arguments, "maxPages", 200)
		timeout := getInt(params.Arguments, "timeoutSeconds", s.cfg.RequestTimeoutSecs)
		opts := validate.JobValidationOpts{
			URL:         url,
			MaxDepth:    maxDepth,
			MaxPages:    maxPages,
			Timeout:     timeout,
			AuthProfile: authProfile,
		}
		if err := validate.ValidateJob(opts, model.KindCrawl); err != nil {
			return nil, err
		}
		resolvedAuth, err := resolveAuthForTool(s.cfg, url, authProfile)
		if err != nil {
			return nil, err
		}
		extractOpts := extract.ExtractOptions{
			Template: getString(params.Arguments, "extractTemplate"),
			Validate: getBool(params.Arguments, "extractValidate"),
		}
		pipelineOpts := getPipelineOptions(params.Arguments)
		spec := jobs.JobSpec{
			Kind:           model.KindCrawl,
			URL:            url,
			MaxDepth:       maxDepth,
			MaxPages:       maxPages,
			Headless:       getBool(params.Arguments, "headless"),
			UsePlaywright:  getBoolDefault(params.Arguments, "playwright", s.cfg.UsePlaywright),
			Auth:           resolvedAuth,
			TimeoutSeconds: timeout,
			Extract:        extractOpts,
			Pipeline:       pipelineOpts,
			Incremental:    getBoolDefault(params.Arguments, "incremental", false),
		}
		job, err := s.manager.CreateJob(ctx, spec)
		if err != nil {
			return nil, err
		}
		if err := s.manager.Enqueue(job); err != nil {
			return nil, err
		}
		if err := waitForJob(ctx, s.store, job.ID, timeout); err != nil {
			return nil, err
		}
		return loadResult(ctx, s.store, job.ID)
	case "research":
		query := getString(params.Arguments, "query")
		urls := getStringSlice(params.Arguments, "urls")
		if query == "" || len(urls) == 0 {
			return nil, apperrors.Validation("query and urls are required")
		}
		authProfile := getString(params.Arguments, "authProfile")
		maxDepth := getInt(params.Arguments, "maxDepth", 2)
		maxPages := getInt(params.Arguments, "maxPages", 200)
		timeout := getInt(params.Arguments, "timeoutSeconds", s.cfg.RequestTimeoutSecs)
		opts := validate.JobValidationOpts{
			Query:       query,
			URLs:        urls,
			MaxDepth:    maxDepth,
			MaxPages:    maxPages,
			Timeout:     timeout,
			AuthProfile: authProfile,
		}
		if err := validate.ValidateJob(opts, model.KindResearch); err != nil {
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
		extractOpts := extract.ExtractOptions{
			Template: getString(params.Arguments, "extractTemplate"),
			Validate: getBool(params.Arguments, "extractValidate"),
		}
		pipelineOpts := getPipelineOptions(params.Arguments)
		spec := jobs.JobSpec{
			Kind:           model.KindResearch,
			Query:          query,
			URLs:           urls,
			MaxDepth:       maxDepth,
			MaxPages:       maxPages,
			Headless:       getBool(params.Arguments, "headless"),
			UsePlaywright:  getBoolDefault(params.Arguments, "playwright", s.cfg.UsePlaywright),
			Auth:           resolvedAuth,
			TimeoutSeconds: timeout,
			Extract:        extractOpts,
			Pipeline:       pipelineOpts,
		}
		job, err := s.manager.CreateJob(ctx, spec)
		if err != nil {
			return nil, err
		}
		if err := s.manager.Enqueue(job); err != nil {
			return nil, err
		}
		if err := waitForJob(ctx, s.store, job.ID, timeout); err != nil {
			return nil, err
		}
		return loadResult(ctx, s.store, job.ID)
	case "job_status":
		id := getString(params.Arguments, "id")
		if id == "" {
			return nil, apperrors.Validation("id is required")
		}
		job, err := s.store.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		return model.SanitizeJob(job), nil
	case "job_results":
		id := getString(params.Arguments, "id")
		if id == "" {
			return nil, apperrors.Validation("id is required")
		}
		return loadResult(ctx, s.store, id)
	case "job_list":
		limit := getInt(params.Arguments, "limit", 100)
		offset := getInt(params.Arguments, "offset", 0)
		jobs, err := s.store.ListOpts(ctx, store.ListOptions{Limit: limit, Offset: offset})
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"jobs": model.SanitizeJobs(jobs)}, nil
	case "job_cancel":
		id := getString(params.Arguments, "id")
		if id == "" {
			return nil, apperrors.Validation("id is required")
		}
		if err := s.manager.CancelJob(ctx, id); err != nil {
			return nil, err
		}
		return map[string]interface{}{"status": "canceled", "id": id}, nil
	case "job_export":
		id := getString(params.Arguments, "id")
		if id == "" {
			return nil, apperrors.Validation("id is required")
		}
		format := getString(params.Arguments, "format")
		if format == "" {
			format = "jsonl"
		}
		validFormats := map[string]bool{"jsonl": true, "json": true, "md": true, "csv": true}
		if !validFormats[format] {
			return nil, apperrors.Validation("invalid format: must be jsonl, json, md, or csv")
		}
		job, err := s.store.Get(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("job not found: %w", err)
		}
		if job.ResultPath == "" {
			return nil, apperrors.NotFound("job has no results")
		}
		rawBytes, err := os.ReadFile(job.ResultPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read result file: %w", err)
		}
		exported, err := exporter.Export(job, rawBytes, format)
		if err != nil {
			return nil, fmt.Errorf("failed to export job: %w", err)
		}
		return exported, nil
	default:
		return nil, fmt.Errorf("unknown tool: %s", params.Name)
	}
}

func loadResult(ctx context.Context, store *store.Store, id string) (string, error) {
	job, err := store.Get(ctx, id)
	if err != nil {
		return "", err
	}
	if job.ResultPath == "" {
		return "", apperrors.NotFound("no result path")
	}
	data, err := os.ReadFile(job.ResultPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// jobStore defines the minimal interface needed by waitForJob.
// This interface enables testing without requiring a full store implementation.
type jobStore interface {
	Get(ctx context.Context, id string) (model.Job, error)
}

// waitForJob polls the job store until the job reaches a terminal state (succeeded/failed) or times out.
//
// Does NOT handle:
// - Job execution or state transitions - those are managed by the job worker pool
// - Retry logic or job restarts - the function only polls and returns
//
// Invariants:
// - Polls every 200ms until the job completes
// - Has an independent timeout timer that cannot be cancelled by the caller's context
// - Respects context cancellation/deadline if shorter than the explicit timeout
// - Returns apperrors.Internal on timeout or job failure
// - Propagates store errors directly
func waitForJob(ctx context.Context, store jobStore, id string, timeoutSeconds int) error {
	pollInterval := 200 * time.Millisecond
	timeoutDuration := time.Duration(timeoutSeconds) * time.Second
	timer := time.NewTimer(pollInterval)
	timeoutTimer := time.After(timeoutDuration)
	defer timer.Stop()

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
			return apperrors.Internal("job failed")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			timer.Reset(pollInterval)
		case <-timeoutTimer:
			return apperrors.Internal(fmt.Sprintf("job timed out after %d seconds", timeoutSeconds))
		}
	}
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
