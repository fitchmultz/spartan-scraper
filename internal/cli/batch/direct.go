// Package batch provides CLI commands for batch job operations.
//
// Purpose:
// - Execute batch submissions directly when the API server is not running.
//
// Responsibilities:
// - Convert canonical batch requests into job specs for direct execution.
// - Create, enqueue, inspect, and cancel batches through local manager/store access.
// - Reuse the shared batch request-to-spec conversion from internal/submission.
//
// Scope:
// - Direct CLI batch execution only.
//
// Usage:
// - Called by CLI batch submit/status/cancel flows when local direct mode is selected.
//
// Invariants/Assumptions:
// - Direct execution should persist the same specs as equivalent API batch submissions.
// - Auth is already resolved by CLI flag handling before direct batch conversion runs.
package batch

import (
	"context"

	spartanapi "github.com/fitchmultz/spartan-scraper/internal/api"
	"github.com/fitchmultz/spartan-scraper/internal/cli/common"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/store"
	"github.com/fitchmultz/spartan-scraper/internal/submission"
)

func submitBatchScrapeDirect(ctx context.Context, cfg config.Config, req BatchScrapeRequest) (*BatchResponse, error) {
	st, err := store.Open(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	defer st.Close()

	manager, err := common.InitJobManager(ctx, cfg, st)
	if err != nil {
		return nil, err
	}

	specs, err := submission.JobSpecsFromBatchScrapeRequest(cfg, directBatchDefaults(cfg, manager), req)
	if err != nil {
		return nil, err
	}

	batchID := jobs.GenerateBatchID()
	createdJobs, err := manager.CreateBatchJobs(ctx, model.KindScrape, specs, batchID)
	if err != nil {
		return nil, err
	}
	if err := manager.EnqueueBatch(createdJobs); err != nil {
		return nil, err
	}

	response := spartanapi.BuildCreatedBatchResponse(batchID, model.KindScrape, createdJobs)
	return &response, nil
}

func submitBatchCrawlDirect(ctx context.Context, cfg config.Config, req BatchCrawlRequest) (*BatchResponse, error) {
	st, err := store.Open(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	defer st.Close()

	manager, err := common.InitJobManager(ctx, cfg, st)
	if err != nil {
		return nil, err
	}

	specs, err := submission.JobSpecsFromBatchCrawlRequest(cfg, directBatchDefaults(cfg, manager), req)
	if err != nil {
		return nil, err
	}

	batchID := jobs.GenerateBatchID()
	createdJobs, err := manager.CreateBatchJobs(ctx, model.KindCrawl, specs, batchID)
	if err != nil {
		return nil, err
	}
	if err := manager.EnqueueBatch(createdJobs); err != nil {
		return nil, err
	}

	response := spartanapi.BuildCreatedBatchResponse(batchID, model.KindCrawl, createdJobs)
	return &response, nil
}

func submitBatchResearchDirect(ctx context.Context, cfg config.Config, req BatchResearchRequest) (*BatchResponse, error) {
	st, err := store.Open(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	defer st.Close()

	manager, err := common.InitJobManager(ctx, cfg, st)
	if err != nil {
		return nil, err
	}

	specs, err := submission.JobSpecsFromBatchResearchRequest(cfg, directBatchDefaults(cfg, manager), req)
	if err != nil {
		return nil, err
	}

	batchID := jobs.GenerateBatchID()
	createdJobs, err := manager.CreateBatchJobs(ctx, model.KindResearch, specs, batchID)
	if err != nil {
		return nil, err
	}
	if err := manager.EnqueueBatch(createdJobs); err != nil {
		return nil, err
	}

	response := spartanapi.BuildCreatedBatchResponse(batchID, model.KindResearch, createdJobs)
	return &response, nil
}

func directBatchDefaults(cfg config.Config, manager *jobs.Manager) submission.BatchDefaults {
	return submission.BatchDefaults{
		Defaults: submission.Defaults{
			DefaultTimeoutSeconds: manager.DefaultTimeoutSeconds(),
			DefaultUsePlaywright:  manager.DefaultUsePlaywright(),
			ResolveAuth:           false,
		},
		MaxBatchSize: cfg.MaxBatchSize,
	}
}

func listBatchesDirect(ctx context.Context, cfg config.Config, limit, offset int) (*BatchListResponse, error) {
	st, err := store.Open(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	defer st.Close()

	opts := store.ListOptions{Limit: limit, Offset: offset}.Defaults()
	batches, stats, err := st.ListBatchesWithStats(ctx, opts)
	if err != nil {
		return nil, err
	}
	total, err := st.CountBatches(ctx)
	if err != nil {
		return nil, err
	}
	response := spartanapi.BuildBatchListResponse(batches, stats, total, opts.Limit, opts.Offset)
	return &response, nil
}

func getBatchStatusDirect(ctx context.Context, cfg config.Config, batchID string, includeJobs bool) (*BatchStatusResponse, error) {
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

	if !includeJobs {
		response := spartanapi.BuildBatchResponse(batch, stats, nil, batch.JobCount, 0, 0)
		return &response, nil
	}

	jobsByBatch, err := st.ListJobsByBatch(ctx, batchID, store.ListOptions{Limit: batch.JobCount, Offset: 0})
	if err != nil {
		return nil, err
	}
	response := spartanapi.BuildBatchResponse(batch, stats, jobsByBatch, batch.JobCount, batch.JobCount, 0)
	return &response, nil
}

func cancelBatchDirect(ctx context.Context, cfg config.Config, batchID string) (*BatchResponse, error) {
	st, err := store.Open(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	defer st.Close()

	manager, err := common.InitJobManager(ctx, cfg, st)
	if err != nil {
		return nil, err
	}
	if _, err := manager.CancelBatch(ctx, batchID); err != nil {
		return nil, err
	}
	batch, stats, err := manager.GetBatchStatus(ctx, batchID)
	if err != nil {
		return nil, err
	}
	response := spartanapi.BuildBatchResponse(batch, stats, nil, batch.JobCount, 0, 0)
	return &response, nil
}
