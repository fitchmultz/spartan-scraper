// Package jobs provides job execution lifecycle management.
// This file contains the core job execution logic that dispatches to scrape, crawl,
// or research packages based on job kind.
//
// Responsibilities:
// - Managing job status transitions (queued → running → succeeded/failed/canceled)
// - Creating result directories and files
// - Executing jobs with appropriate context and cancellation handling
// - Writing results to JSONL output files
//
// This file does NOT:
// - Create jobs (see job_create.go)
// - Implement scraping/crawling/research logic (delegated to respective packages)
// - Handle job queueing or scheduling (managed by Manager's worker pool)
//
// Invariants:
// - Job status updates are retried with background context on primary context cancellation
// - Result directories are created securely before writing
// - Active jobs are tracked for cancellation support
// - All errors are sanitized via apperrors before storage

package jobs

import (
	"context"
	"encoding/json"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/crawl"
	"github.com/fitchmultz/spartan-scraper/internal/fsutil"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/research"
	"github.com/fitchmultz/spartan-scraper/internal/scrape"
)

func getJobRequestID(job model.Job) string {
	if reqID, ok := job.Params["requestID"].(string); ok && reqID != "" {
		return reqID
	}
	return job.ID
}

func (m *Manager) updateStatusWithTimeout(jobID string, status model.Status, err error, timeout time.Duration) {
	updateCtx, cancelUpdate := context.WithTimeout(context.Background(), timeout)
	errorMsg := apperrors.SafeMessage(err)
	if updateErr := m.store.UpdateStatus(updateCtx, jobID, status, errorMsg); updateErr != nil {
		slog.Error("failed to update job status", "jobID", jobID, "status", status, "error", updateErr)
	}
	cancelUpdate()
}

// updateStatusWithEvent updates job status and publishes an event.
func (m *Manager) updateStatusWithEvent(job model.Job, prevStatus model.Status, status model.Status, errMsg string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if updateErr := m.store.UpdateStatus(ctx, job.ID, status, errMsg); updateErr != nil {
		slog.Error("failed to update job status", "jobID", job.ID, "status", status, "error", updateErr)
		return
	}

	// Update job for event
	job.Status = status
	if errMsg != "" {
		job.Error = errMsg
	}

	// Determine event type
	eventType := JobEventStatus
	if status.IsTerminal() {
		eventType = JobEventCompleted
	}

	m.publishEvent(JobEvent{
		Type:       eventType,
		Job:        job,
		PrevStatus: prevStatus,
	})
}

func (m *Manager) run(ctx context.Context, job model.Job) error {
	slog.Info("running job", "jobID", job.ID, "kind", job.Kind, "request_id", getJobRequestID(job))

	latest, err := m.store.Get(ctx, job.ID)
	if err == nil && latest.Status.IsTerminal() {
		slog.Info("job already in terminal state, not running", "jobID", job.ID, "status", latest.Status)
		return nil
	}

	prevStatus := job.Status

	if err := m.store.UpdateStatus(ctx, job.ID, model.StatusRunning, ""); err != nil {
		// If primary context is canceled, retry with background context
		if ctx.Err() != nil {
			updateCtx, cancelUpdate := context.WithTimeout(context.Background(), 2*time.Second)
			err = m.store.UpdateStatus(updateCtx, job.ID, model.StatusRunning, "")
			cancelUpdate()
		}
		if err != nil {
			slog.Error("failed to update job status to running", "jobID", job.ID, "error", err)
		}
	}

	// Publish job started event
	job.Status = model.StatusRunning
	m.publishEvent(JobEvent{
		Type:       JobEventStarted,
		Job:        job,
		PrevStatus: prevStatus,
	})

	jobCtx, cancel := context.WithCancel(ctx)
	m.mu.Lock()
	m.activeJobs[job.ID] = cancel
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		delete(m.activeJobs, job.ID)
		m.mu.Unlock()
		cancel()
	}()

	resultDir := filepath.Dir(job.ResultPath)
	if err := fsutil.MkdirAllSecure(resultDir); err != nil {
		slog.Error("failed to create result directory", "jobID", job.ID, "error", err)
		m.updateStatusWithTimeout(job.ID, model.StatusFailed, err, 2*time.Second)
		return err
	}

	file, err := fsutil.CreateSecure(job.ResultPath)
	if err != nil {
		slog.Error("failed to create result file", "jobID", job.ID, "error", err)
		m.updateStatusWithTimeout(job.ID, model.StatusFailed, err, 2*time.Second)
		return err
	}
	defer file.Close()

	switch job.Kind {
	case model.KindScrape:
		input, err := decodeScrapeExecutionInput(job, m)
		if err != nil {
			slog.Error("invalid scrape job configuration", "jobID", job.ID, "error", err)
			m.updateStatusWithEvent(job, model.StatusRunning, model.StatusFailed, apperrors.SafeMessage(err))
			m.propagateFailure(context.Background(), job)
			return err
		}
		slog.Info("processing scrape job", "jobID", job.ID, "url", apperrors.SanitizeURL(input.URL), "request_id", input.Config.RequestID)
		result, err := scrape.Run(jobCtx, scrape.Request{
			URL:              input.URL,
			Method:           input.Method,
			Body:             input.Body,
			ContentType:      input.ContentType,
			RequestID:        input.Config.RequestID,
			Headless:         input.Config.Headless,
			UsePlaywright:    input.Config.UsePlaywright,
			Auth:             input.Config.Auth,
			Extract:          input.Config.Extract,
			Pipeline:         input.Config.Pipeline,
			Timeout:          time.Duration(input.Config.TimeoutSeconds) * time.Second,
			UserAgent:        m.userAgent,
			Limiter:          m.limiter,
			MaxRetries:       m.maxRetries,
			RetryBase:        m.retryBase,
			MaxResponseBytes: m.maxResponseBytes,
			DataDir:          m.DataDir,
			Incremental:      input.Incremental,
			Store:            m.store,
			Registry:         m.pipelineRegistry,
			JSRegistry:       m.jsRegistry,
			TemplateRegistry: m.templateRegistry,
			MetricsCallback:  m.metricsCallback,
			Screenshot:       input.Config.Screenshot,
			ProxyPool:        m.proxyPool,
			AIExtractor:      m.aiExtractor,
		})
		if err != nil {
			if jobCtx.Err() != nil {
				slog.Info("job canceled during scrape", "jobID", job.ID)
				m.updateStatusWithEvent(job, model.StatusRunning, model.StatusCanceled, "canceled by user")
				m.propagateFailure(context.Background(), job)
				return nil
			}
			slog.Error("scrape job failed", "jobID", job.ID, "url", apperrors.SanitizeURL(input.URL), "error", err)
			m.updateStatusWithEvent(job, model.StatusRunning, model.StatusFailed, apperrors.SafeMessage(err))
			m.propagateFailure(context.Background(), job)
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
		input, err := decodeCrawlExecutionInput(job, m)
		if err != nil {
			slog.Error("invalid crawl job configuration", "jobID", job.ID, "error", err)
			m.updateStatusWithEvent(job, model.StatusRunning, model.StatusFailed, apperrors.SafeMessage(err))
			m.propagateFailure(context.Background(), job)
			return err
		}
		slog.Info("processing crawl job", "jobID", job.ID, "url", apperrors.SanitizeURL(input.URL), "request_id", input.Config.RequestID)

		// Create robots cache if enabled
		var robotsCache *crawl.Cache
		if input.RespectRobotsTxt {
			robotsCache = crawl.NewCache(nil, time.Hour)
		}

		// Get content index for cross-job deduplication
		var contentIndex crawl.ContentIndex
		if input.CrossJobDedup {
			contentIndex = m.contentIndex
		}

		results, err := crawl.Run(jobCtx, crawl.Request{
			URL:                    input.URL,
			RequestID:              input.Config.RequestID,
			MaxDepth:               input.MaxDepth,
			MaxPages:               input.MaxPages,
			Concurrency:            m.maxConcurrency,
			Headless:               input.Config.Headless,
			UsePlaywright:          input.Config.UsePlaywright,
			Auth:                   input.Config.Auth,
			Extract:                input.Config.Extract,
			Pipeline:               input.Config.Pipeline,
			Timeout:                time.Duration(input.Config.TimeoutSeconds) * time.Second,
			UserAgent:              m.userAgent,
			Limiter:                m.limiter,
			MaxRetries:             reqRetries(m.maxRetries),
			RetryBase:              m.retryBase,
			MaxResponseBytes:       m.maxResponseBytes,
			DataDir:                m.DataDir,
			Incremental:            input.Incremental,
			Store:                  m.store,
			Registry:               m.pipelineRegistry,
			JSRegistry:             m.jsRegistry,
			TemplateRegistry:       m.templateRegistry,
			MetricsCallback:        m.metricsCallback,
			SitemapURL:             input.SitemapURL,
			SitemapOnly:            input.SitemapOnly,
			IncludePatterns:        input.IncludePatterns,
			ExcludePatterns:        input.ExcludePatterns,
			Screenshot:             input.Config.Screenshot,
			RobotsCache:            robotsCache,
			SkipDuplicates:         input.SkipDuplicates,
			SimHashThreshold:       input.SimHashThreshold,
			CrossJobDedup:          input.CrossJobDedup,
			CrossJobDedupThreshold: input.CrossJobDedupThreshold,
			ProxyPool:              m.proxyPool,
			WebhookDispatcher:      m.webhookDispatcher,
			WebhookConfig:          job.ExtractWebhookConfig(),
			AIExtractor:            m.aiExtractor,
			ContentIndex:           contentIndex,
		})
		if err != nil {
			if jobCtx.Err() != nil {
				slog.Info("job canceled during crawl", "jobID", job.ID)
				m.updateStatusWithEvent(job, model.StatusRunning, model.StatusCanceled, "canceled by user")
				m.propagateFailure(context.Background(), job)
				return nil
			}
			slog.Error("crawl job failed", "jobID", job.ID, "url", apperrors.SanitizeURL(input.URL), "error", err)
			m.updateStatusWithEvent(job, model.StatusRunning, model.StatusFailed, apperrors.SafeMessage(err))
			m.propagateFailure(context.Background(), job)
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
		input, err := decodeResearchExecutionInput(job, m)
		if err != nil {
			slog.Error("invalid research job configuration", "jobID", job.ID, "error", err)
			m.updateStatusWithEvent(job, model.StatusRunning, model.StatusFailed, apperrors.SafeMessage(err))
			m.propagateFailure(context.Background(), job)
			return err
		}
		slog.Info("processing research job", "jobID", job.ID, "query", input.Query, "request_id", input.Config.RequestID)
		result, err := research.Run(jobCtx, research.Request{
			Query:            input.Query,
			RequestID:        input.Config.RequestID,
			URLs:             input.URLs,
			MaxDepth:         input.MaxDepth,
			MaxPages:         input.MaxPages,
			Concurrency:      m.maxConcurrency,
			Headless:         input.Config.Headless,
			UsePlaywright:    input.Config.UsePlaywright,
			Auth:             input.Config.Auth,
			Extract:          input.Config.Extract,
			Pipeline:         input.Config.Pipeline,
			Timeout:          time.Duration(input.Config.TimeoutSeconds) * time.Second,
			UserAgent:        m.userAgent,
			Limiter:          m.limiter,
			MaxRetries:       m.maxRetries,
			RetryBase:        m.retryBase,
			MaxResponseBytes: m.maxResponseBytes,
			DataDir:          m.DataDir,
			Store:            m.store,
			Registry:         m.pipelineRegistry,
			JSRegistry:       m.jsRegistry,
			TemplateRegistry: m.templateRegistry,
			Screenshot:       input.Config.Screenshot,
			ProxyPool:        m.proxyPool,
		})
		if err != nil {
			if jobCtx.Err() != nil {
				slog.Info("job canceled during research", "jobID", job.ID)
				m.updateStatusWithEvent(job, model.StatusRunning, model.StatusCanceled, "canceled by user")
				m.propagateFailure(context.Background(), job)
				return nil
			}
			slog.Error("research job failed", "jobID", job.ID, "query", input.Query, "error", err)
			m.updateStatusWithEvent(job, model.StatusRunning, model.StatusFailed, apperrors.SafeMessage(err))
			m.propagateFailure(context.Background(), job)
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
		m.updateStatusWithEvent(job, model.StatusRunning, model.StatusFailed, "unknown job kind")
		return apperrors.Internal("unknown job kind")
	}

	if jobCtx.Err() != nil {
		slog.Info("job completed but was canceled", "jobID", job.ID)
		return nil
	}

	slog.Info("job succeeded", "jobID", job.ID)
	m.updateStatusWithEvent(job, model.StatusRunning, model.StatusSucceeded, "")

	// Resolve dependencies for jobs waiting on this one
	m.resolveDependencies(ctx, job)

	return nil
}
