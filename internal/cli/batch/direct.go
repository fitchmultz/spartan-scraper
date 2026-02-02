// Package batch provides CLI commands for batch job operations.
//
// This file contains direct execution functions for batch operations
// when the API server is not running.
//
// Responsibilities:
// - Submit batch jobs directly using the jobs package
// - Get batch status directly from the store
// - Cancel batches directly using the job manager
//
// Does NOT handle:
// - HTTP API calls
// - CLI command parsing
// - File parsing
package batch

import (
	"context"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/cli/common"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

func submitBatchScrapeDirect(ctx context.Context, cfg config.Config, req BatchScrapeRequest) (*BatchResponse, error) {
	st, err := store.Open(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	defer st.Close()

	manager := common.InitJobManager(ctx, cfg, st)

	// Build job specs
	specs := make([]jobs.JobSpec, len(req.Jobs))
	for i, job := range req.Jobs {
		specs[i] = jobs.JobSpec{
			Kind:           model.KindScrape,
			URL:            job.URL,
			Method:         job.Method,
			Body:           []byte(job.Body),
			ContentType:    job.ContentType,
			Headless:       req.Headless,
			UsePlaywright:  req.Playwright != nil && *req.Playwright,
			TimeoutSeconds: req.TimeoutSeconds,
			Auth:           *req.Auth,
			Extract:        *req.Extract,
			Pipeline:       *req.Pipeline,
			Incremental:    req.Incremental != nil && *req.Incremental,
		}
	}

	// Create batch
	batchID := jobs.GenerateBatchID()
	createdJobs, err := manager.CreateBatchJobs(ctx, model.KindScrape, specs, batchID)
	if err != nil {
		return nil, err
	}

	// Enqueue all jobs
	if err := manager.EnqueueBatch(createdJobs); err != nil {
		return nil, err
	}

	return &BatchResponse{
		ID:        batchID,
		Kind:      string(model.KindScrape),
		Status:    string(model.BatchStatusPending),
		JobCount:  len(createdJobs),
		CreatedAt: time.Now(),
	}, nil
}

func submitBatchCrawlDirect(ctx context.Context, cfg config.Config, req BatchCrawlRequest) (*BatchResponse, error) {
	st, err := store.Open(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	defer st.Close()

	manager := common.InitJobManager(ctx, cfg, st)

	// Build job specs
	specs := make([]jobs.JobSpec, len(req.Jobs))
	for i, job := range req.Jobs {
		specs[i] = jobs.JobSpec{
			Kind:           model.KindCrawl,
			URL:            job.URL,
			MaxDepth:       req.MaxDepth,
			MaxPages:       req.MaxPages,
			Headless:       req.Headless,
			UsePlaywright:  req.Playwright != nil && *req.Playwright,
			TimeoutSeconds: req.TimeoutSeconds,
			SitemapURL:     req.SitemapURL,
			SitemapOnly:    req.SitemapOnly != nil && *req.SitemapOnly,
			Auth:           *req.Auth,
			Extract:        *req.Extract,
			Pipeline:       *req.Pipeline,
			Incremental:    req.Incremental != nil && *req.Incremental,
		}
	}

	// Create batch
	batchID := jobs.GenerateBatchID()
	createdJobs, err := manager.CreateBatchJobs(ctx, model.KindCrawl, specs, batchID)
	if err != nil {
		return nil, err
	}

	// Enqueue all jobs
	if err := manager.EnqueueBatch(createdJobs); err != nil {
		return nil, err
	}

	return &BatchResponse{
		ID:        batchID,
		Kind:      string(model.KindCrawl),
		Status:    string(model.BatchStatusPending),
		JobCount:  len(createdJobs),
		CreatedAt: time.Now(),
	}, nil
}

func submitBatchResearchDirect(ctx context.Context, cfg config.Config, req BatchResearchRequest) (*BatchResponse, error) {
	st, err := store.Open(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	defer st.Close()

	manager := common.InitJobManager(ctx, cfg, st)

	// Collect URLs from jobs
	urls := make([]string, len(req.Jobs))
	for i, job := range req.Jobs {
		urls[i] = job.URL
	}

	// Research jobs only need one job with all URLs
	spec := jobs.JobSpec{
		Kind:           model.KindResearch,
		Query:          req.Query,
		URLs:           urls,
		MaxDepth:       req.MaxDepth,
		MaxPages:       req.MaxPages,
		Headless:       req.Headless,
		UsePlaywright:  req.Playwright != nil && *req.Playwright,
		TimeoutSeconds: req.TimeoutSeconds,
		Auth:           *req.Auth,
		Extract:        *req.Extract,
		Pipeline:       *req.Pipeline,
	}

	// Create batch
	batchID := jobs.GenerateBatchID()
	createdJobs, err := manager.CreateBatchJobs(ctx, model.KindResearch, []jobs.JobSpec{spec}, batchID)
	if err != nil {
		return nil, err
	}

	// Enqueue all jobs
	if err := manager.EnqueueBatch(createdJobs); err != nil {
		return nil, err
	}

	return &BatchResponse{
		ID:        batchID,
		Kind:      string(model.KindResearch),
		Status:    string(model.BatchStatusPending),
		JobCount:  len(createdJobs),
		CreatedAt: time.Now(),
	}, nil
}

func getBatchStatusDirect(ctx context.Context, cfg config.Config, batchID string) (*BatchStatusResponse, error) {
	st, err := store.Open(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	defer st.Close()

	batch, err := st.GetBatch(ctx, batchID)
	if err != nil {
		return nil, err
	}

	stats, err := st.CountJobsByBatchAndStatus(ctx, batchID)
	if err != nil {
		return nil, err
	}

	// Calculate current batch status
	batch.Status = model.CalculateBatchStatus(stats, batch.JobCount)

	return &BatchStatusResponse{
		ID:        batch.ID,
		Kind:      string(batch.Kind),
		Status:    string(batch.Status),
		JobCount:  batch.JobCount,
		Stats:     stats,
		CreatedAt: batch.CreatedAt,
		UpdatedAt: batch.UpdatedAt,
	}, nil
}

func cancelBatchDirect(ctx context.Context, cfg config.Config, batchID string) error {
	st, err := store.Open(cfg.DataDir)
	if err != nil {
		return err
	}
	defer st.Close()

	manager := common.InitJobManager(ctx, cfg, st)
	_, err = manager.CancelBatch(ctx, batchID)
	return err
}
