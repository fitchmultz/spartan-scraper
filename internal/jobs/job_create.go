package jobs

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

// CreateScrapeJob creates and persists a new scrape job.
func (m *Manager) CreateScrapeJob(ctx context.Context, url string, headless bool, usePlaywright bool, auth fetch.AuthOptions, timeoutSeconds int, extractOpts extract.ExtractOptions, pipelineOpts pipeline.Options, incremental bool, requestID string) (model.Job, error) {
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
			"requestID":   requestID,
		},
	}
	job.ResultPath = filepath.Join(m.dataDir, "jobs", job.ID, "results.jsonl")
	if err := m.store.Create(ctx, job); err != nil {
		return model.Job{}, err
	}
	return job, nil
}

// CreateCrawlJob creates and persists a new crawl job.
func (m *Manager) CreateCrawlJob(ctx context.Context, url string, maxDepth, maxPages int, headless bool, usePlaywright bool, auth fetch.AuthOptions, timeoutSeconds int, extractOpts extract.ExtractOptions, pipelineOpts pipeline.Options, incremental bool, requestID string) (model.Job, error) {
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
			"requestID":   requestID,
		},
	}
	job.ResultPath = filepath.Join(m.dataDir, "jobs", job.ID, "results.jsonl")
	if err := m.store.Create(ctx, job); err != nil {
		return model.Job{}, err
	}
	return job, nil
}

// CreateResearchJob creates and persists a new research job.
func (m *Manager) CreateResearchJob(ctx context.Context, query string, urls []string, maxDepth, maxPages int, headless bool, usePlaywright bool, auth fetch.AuthOptions, timeoutSeconds int, extractOpts extract.ExtractOptions, pipelineOpts pipeline.Options, incremental bool, requestID string) (model.Job, error) {
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
			"requestID":   requestID,
		},
	}
	job.ResultPath = filepath.Join(m.dataDir, "jobs", job.ID, "results.jsonl")
	if err := m.store.Create(ctx, job); err != nil {
		return model.Job{}, err
	}
	return job, nil
}

// CreateJob creates and persists a new job from a unified JobSpec.
// This method consolidates three kind-specific Create*Job methods into a single entry point.
// It validates the spec and dispatches to the appropriate Create*Job method.
// Returns the created job or an error if validation fails or creation fails.
func (m *Manager) CreateJob(ctx context.Context, spec JobSpec) (model.Job, error) {
	if err := spec.Validate(); err != nil {
		return model.Job{}, fmt.Errorf("invalid job spec: %w", err)
	}

	switch spec.Kind {
	case model.KindScrape:
		return m.CreateScrapeJob(ctx, spec.URL, spec.Headless, spec.UsePlaywright, spec.Auth, spec.TimeoutSeconds, spec.Extract, spec.Pipeline, spec.Incremental, spec.RequestID)
	case model.KindCrawl:
		return m.CreateCrawlJob(ctx, spec.URL, spec.MaxDepth, spec.MaxPages, spec.Headless, spec.UsePlaywright, spec.Auth, spec.TimeoutSeconds, spec.Extract, spec.Pipeline, spec.Incremental, spec.RequestID)
	case model.KindResearch:
		return m.CreateResearchJob(ctx, spec.Query, spec.URLs, spec.MaxDepth, spec.MaxPages, spec.Headless, spec.UsePlaywright, spec.Auth, spec.TimeoutSeconds, spec.Extract, spec.Pipeline, spec.Incremental, spec.RequestID)
	default:
		return model.Job{}, fmt.Errorf("unknown job kind: %s", spec.Kind)
	}
}
