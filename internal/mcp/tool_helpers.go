// Package mcp provides shared helpers for focused MCP tool handlers.
//
// Purpose:
// - Centralize the runtime helpers reused across multiple MCP tool domains.
//
// Responsibilities:
// - Build shared job and batch defaults.
// - Load persisted results and webhook-delivery storage.
// - Resolve auth and transport overrides consistently with other operator surfaces.
//
// Scope:
// - MCP execution helpers only; tool-specific branching lives in domain handler files.
//
// Usage:
// - Called by AI, job, batch, export, watch, and observability tool handlers.
//
// Invariants/Assumptions:
// - Helpers preserve existing MCP behavior and response envelopes.
// - Auth resolution uses the canonical auth package.
// - Batch and job helpers use manager defaults when a manager is available.
package mcp

import (
	"context"
	"os"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/api"
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/auth"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/paramdecode"
	"github.com/fitchmultz/spartan-scraper/internal/store"
	"github.com/fitchmultz/spartan-scraper/internal/submission"
	"github.com/fitchmultz/spartan-scraper/internal/webhook"
)

func (s *Server) jobSubmissionDefaults() api.JobSubmissionDefaults {
	return api.JobSubmissionDefaults{
		DefaultTimeoutSeconds: s.manager.DefaultTimeoutSeconds(),
		DefaultUsePlaywright:  s.manager.DefaultUsePlaywright(),
		ResolveAuth:           true,
	}
}

func (s *Server) batchSubmissionDefaults() submission.BatchDefaults {
	return submission.BatchDefaults{
		Defaults: submission.Defaults{
			DefaultTimeoutSeconds: s.manager.DefaultTimeoutSeconds(),
			DefaultUsePlaywright:  s.manager.DefaultUsePlaywright(),
			ResolveAuth:           true,
		},
		MaxBatchSize: s.cfg.MaxBatchSize,
	}
}

func (s *Server) runJobAndLoadResult(ctx context.Context, spec jobs.JobSpec) (string, error) {
	job, err := s.manager.CreateJob(ctx, spec)
	if err != nil {
		return "", err
	}
	if err := s.manager.Enqueue(job); err != nil {
		return "", err
	}
	if err := waitForJob(ctx, s.store, job.ID, spec.TimeoutSeconds); err != nil {
		return "", err
	}
	return loadResult(ctx, s.store, job.ID)
}

func (s *Server) createAndEnqueueBatch(ctx context.Context, kind model.Kind, specs []jobs.JobSpec) (api.BatchResponse, error) {
	batchID := jobs.GenerateBatchID()
	createdJobs, err := s.manager.CreateBatchJobs(ctx, kind, specs, batchID)
	if err != nil {
		return api.BatchResponse{}, err
	}
	if err := s.manager.EnqueueBatch(createdJobs); err != nil {
		return api.BatchResponse{}, err
	}
	return s.buildBatchResponse(ctx, batchID, true, len(createdJobs), 0)
}

func (s *Server) buildBatchResponse(ctx context.Context, batchID string, includeJobs bool, limit, offset int) (api.BatchResponse, error) {
	batch, stats, err := s.manager.GetBatchStatus(ctx, batchID)
	if err != nil {
		return api.BatchResponse{}, err
	}
	if !includeJobs {
		return api.BuildStoreBackedBatchResponse(ctx, s.store, batch, stats, nil, batch.JobCount, 0, 0)
	}
	jobs, err := s.store.ListJobsByBatch(ctx, batchID, store.ListOptions{Limit: limit, Offset: offset})
	if err != nil {
		return api.BatchResponse{}, err
	}
	return api.BuildStoreBackedBatchResponse(ctx, s.store, batch, stats, jobs, batch.JobCount, limit, offset)
}

func loadWebhookDeliveryStore(dataDir string) (*webhook.Store, error) {
	deliveryStore := webhook.NewStore(dataDir)
	if err := deliveryStore.Load(); err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to load webhook deliveries", err)
	}
	return deliveryStore, nil
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

func resolveAuthForTool(cfg config.Config, url string, profile string, override fetch.AuthOptions) (fetch.AuthOptions, error) {
	input := auth.ResolveInput{
		ProfileName: profile,
		URL:         url,
		Env:         &cfg.AuthOverrides,
	}
	resolved, err := auth.Resolve(cfg.DataDir, input)
	if err != nil {
		return fetch.AuthOptions{}, err
	}
	authOptions := auth.ToFetchOptions(resolved)
	if override.Proxy != nil {
		authOptions.Proxy = override.Proxy
	}
	if override.ProxyHints != nil {
		authOptions.ProxyHints = fetch.NormalizeProxySelectionHints(override.ProxyHints)
	}
	authOptions.NormalizeTransport()
	if err := authOptions.ValidateTransport(); err != nil {
		return fetch.AuthOptions{}, err
	}
	return authOptions, nil
}

func decodeTransportOverrides(args map[string]interface{}) fetch.AuthOptions {
	proxyURL := strings.TrimSpace(paramdecode.String(args, "proxy"))
	proxyUsername := strings.TrimSpace(paramdecode.String(args, "proxyUsername"))
	proxyPassword := strings.TrimSpace(paramdecode.String(args, "proxyPassword"))
	var proxy *fetch.ProxyConfig
	if proxyURL != "" || proxyUsername != "" || proxyPassword != "" {
		proxy = &fetch.ProxyConfig{
			URL:      proxyURL,
			Username: proxyUsername,
			Password: proxyPassword,
		}
	}
	return fetch.AuthOptions{
		Proxy: proxy,
		ProxyHints: fetch.NormalizeProxySelectionHints(&fetch.ProxySelectionHints{
			PreferredRegion: strings.TrimSpace(paramdecode.String(args, "proxyRegion")),
			RequiredTags:    paramdecode.StringSlice(args, "proxyTags"),
			ExcludeProxyIDs: paramdecode.StringSlice(args, "excludeProxyIds"),
		}),
	}
}
