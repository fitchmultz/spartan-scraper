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
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/fsutil"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
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
		url, _ := job.Params["url"].(string)
		slog.Info("processing scrape job", "jobID", job.ID, "url", apperrors.SanitizeURL(url), "request_id", getJobRequestID(job))
		method, _ := job.Params["method"].(string)
		if method == "" {
			method = "GET"
		}
		body := decodeBytes(job.Params["body"])
		contentType, _ := job.Params["contentType"].(string)
		headless, _ := job.Params["headless"].(bool)
		usePlaywright := toBool(job.Params["playwright"], m.usePlaywright)
		timeoutSecs := toInt(job.Params["timeout"], int(m.requestTimeout.Seconds()))
		var auth fetch.AuthOptions
		auth = decodeAuth(job.Params["auth"])
		var extractOpts extract.ExtractOptions
		extractOpts = decodeExtract(job.Params["extract"])
		var pipelineOpts pipeline.Options
		pipelineOpts = decodePipeline(job.Params["pipeline"])
		incremental := toBool(job.Params["incremental"], false)
		screenshot := decodeScreenshot(job.Params["screenshot"])
		result, err := scrape.Run(jobCtx, scrape.Request{
			URL:              url,
			Method:           method,
			Body:             body,
			ContentType:      contentType,
			RequestID:        getJobRequestID(job),
			Headless:         headless,
			UsePlaywright:    usePlaywright,
			Auth:             auth,
			Extract:          extractOpts,
			Pipeline:         pipelineOpts,
			Timeout:          time.Duration(timeoutSecs) * time.Second,
			UserAgent:        m.userAgent,
			Limiter:          m.limiter,
			MaxRetries:       m.maxRetries,
			RetryBase:        m.retryBase,
			MaxResponseBytes: m.maxResponseBytes,
			DataDir:          m.dataDir,
			Incremental:      incremental,
			Store:            m.store,
			Registry:         m.pipelineRegistry,
			JSRegistry:       m.jsRegistry,
			TemplateRegistry: m.templateRegistry,
			MetricsCallback:  m.metricsCallback,
			Screenshot:       screenshot,
			ProxyPool:        m.proxyPool,
		})
		if err != nil {
			if jobCtx.Err() != nil {
				slog.Info("job canceled during scrape", "jobID", job.ID)
				m.updateStatusWithEvent(job, model.StatusRunning, model.StatusCanceled, "canceled by user")
				m.propagateFailure(context.Background(), job)
				return nil
			}
			slog.Error("scrape job failed", "jobID", job.ID, "url", apperrors.SanitizeURL(url), "error", err)
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
		url, _ := job.Params["url"].(string)
		slog.Info("processing crawl job", "jobID", job.ID, "url", apperrors.SanitizeURL(url), "request_id", getJobRequestID(job))
		maxDepth := toInt(job.Params["maxDepth"], 2)
		maxPages := toInt(job.Params["maxPages"], 200)
		headless, _ := job.Params["headless"].(bool)
		usePlaywright := toBool(job.Params["playwright"], m.usePlaywright)
		timeoutSecs := toInt(job.Params["timeout"], int(m.requestTimeout.Seconds()))
		var auth fetch.AuthOptions
		auth = decodeAuth(job.Params["auth"])
		var extractOpts extract.ExtractOptions
		extractOpts = decodeExtract(job.Params["extract"])
		var pipelineOpts pipeline.Options
		pipelineOpts = decodePipeline(job.Params["pipeline"])
		incremental := toBool(job.Params["incremental"], false)
		sitemapURL, _ := job.Params["sitemapURL"].(string)
		sitemapOnly := toBool(job.Params["sitemapOnly"], false)
		includePatterns := toStringSlice(job.Params["includePatterns"])
		excludePatterns := toStringSlice(job.Params["excludePatterns"])
		screenshot := decodeScreenshot(job.Params["screenshot"])
		respectRobotsTxt := toBool(job.Params["respectRobotsTxt"], false)
		skipDuplicates := toBool(job.Params["skipDuplicates"], false)
		simHashThreshold := toInt(job.Params["simHashThreshold"], 3)

		// Create robots cache if enabled
		var robotsCache *crawl.Cache
		if respectRobotsTxt {
			robotsCache = crawl.NewCache(nil, time.Hour)
		}

		results, err := crawl.Run(jobCtx, crawl.Request{
			URL:               url,
			RequestID:         getJobRequestID(job),
			MaxDepth:          maxDepth,
			MaxPages:          maxPages,
			Concurrency:       m.maxConcurrency,
			Headless:          headless,
			UsePlaywright:     usePlaywright,
			Auth:              auth,
			Extract:           extractOpts,
			Pipeline:          pipelineOpts,
			Timeout:           time.Duration(timeoutSecs) * time.Second,
			UserAgent:         m.userAgent,
			Limiter:           m.limiter,
			MaxRetries:        reqRetries(m.maxRetries),
			RetryBase:         m.retryBase,
			MaxResponseBytes:  m.maxResponseBytes,
			DataDir:           m.dataDir,
			Incremental:       incremental,
			Store:             m.store,
			Registry:          m.pipelineRegistry,
			JSRegistry:        m.jsRegistry,
			TemplateRegistry:  m.templateRegistry,
			MetricsCallback:   m.metricsCallback,
			SitemapURL:        sitemapURL,
			SitemapOnly:       sitemapOnly,
			IncludePatterns:   includePatterns,
			ExcludePatterns:   excludePatterns,
			Screenshot:        screenshot,
			RobotsCache:       robotsCache,
			SkipDuplicates:    skipDuplicates,
			SimHashThreshold:  simHashThreshold,
			ProxyPool:         m.proxyPool,
			WebhookDispatcher: m.webhookDispatcher,
			WebhookConfig:     job.ExtractWebhookConfig(),
		})
		if err != nil {
			if jobCtx.Err() != nil {
				slog.Info("job canceled during crawl", "jobID", job.ID)
				m.updateStatusWithEvent(job, model.StatusRunning, model.StatusCanceled, "canceled by user")
				m.propagateFailure(context.Background(), job)
				return nil
			}
			slog.Error("crawl job failed", "jobID", job.ID, "url", apperrors.SanitizeURL(url), "error", err)
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
		query, _ := job.Params["query"].(string)
		slog.Info("processing research job", "jobID", job.ID, "query", query, "request_id", getJobRequestID(job))
		urls := toStringSlice(job.Params["urls"])
		maxDepth := toInt(job.Params["maxDepth"], 2)
		maxPages := toInt(job.Params["maxPages"], 200)
		headless, _ := job.Params["headless"].(bool)
		usePlaywright := toBool(job.Params["playwright"], m.usePlaywright)
		timeoutSecs := toInt(job.Params["timeout"], int(m.requestTimeout.Seconds()))
		var auth fetch.AuthOptions
		auth = decodeAuth(job.Params["auth"])
		var extractOpts extract.ExtractOptions
		extractOpts = decodeExtract(job.Params["extract"])
		var pipelineOpts pipeline.Options
		pipelineOpts = decodePipeline(job.Params["pipeline"])
		researchScreenshot := decodeScreenshot(job.Params["screenshot"])
		result, err := research.Run(jobCtx, research.Request{
			Query:            query,
			RequestID:        getJobRequestID(job),
			URLs:             urls,
			MaxDepth:         maxDepth,
			MaxPages:         maxPages,
			Concurrency:      m.maxConcurrency,
			Headless:         headless,
			UsePlaywright:    usePlaywright,
			Auth:             auth,
			Extract:          extractOpts,
			Pipeline:         pipelineOpts,
			Timeout:          time.Duration(timeoutSecs) * time.Second,
			UserAgent:        m.userAgent,
			Limiter:          m.limiter,
			MaxRetries:       m.maxRetries,
			RetryBase:        m.retryBase,
			MaxResponseBytes: m.maxResponseBytes,
			DataDir:          m.dataDir,
			Store:            m.store,
			Registry:         m.pipelineRegistry,
			JSRegistry:       m.jsRegistry,
			TemplateRegistry: m.templateRegistry,
			Screenshot:       researchScreenshot,
			ProxyPool:        m.proxyPool,
		})
		if err != nil {
			if jobCtx.Err() != nil {
				slog.Info("job canceled during research", "jobID", job.ID)
				m.updateStatusWithEvent(job, model.StatusRunning, model.StatusCanceled, "canceled by user")
				m.propagateFailure(context.Background(), job)
				return nil
			}
			slog.Error("research job failed", "jobID", job.ID, "query", query, "error", err)
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
