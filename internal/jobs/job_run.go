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

func (m *Manager) updateStatusWithTimeout(jobID string, status model.Status, errorMsg string, timeout time.Duration) {
	updateCtx, cancelUpdate := context.WithTimeout(context.Background(), timeout)
	if err := m.store.UpdateStatus(updateCtx, jobID, status, errorMsg); err != nil {
		slog.Error("failed to update job status", "jobID", jobID, "status", status, "error", err)
	}
	cancelUpdate()
}

func (m *Manager) run(ctx context.Context, job model.Job) error {
	slog.Info("running job", "jobID", job.ID, "kind", job.Kind, "request_id", getJobRequestID(job))

	latest, err := m.store.Get(ctx, job.ID)
	if err == nil && latest.Status.IsTerminal() {
		slog.Info("job already in terminal state, not running", "jobID", job.ID, "status", latest.Status)
		return nil
	}

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
		m.updateStatusWithTimeout(job.ID, model.StatusFailed, err.Error(), 2*time.Second)
		return err
	}

	file, err := fsutil.CreateSecure(job.ResultPath)
	if err != nil {
		slog.Error("failed to create result file", "jobID", job.ID, "error", err)
		m.updateStatusWithTimeout(job.ID, model.StatusFailed, err.Error(), 2*time.Second)
		return err
	}
	defer file.Close()

	switch job.Kind {
	case model.KindScrape:
		url, _ := job.Params["url"].(string)
		slog.Info("processing scrape job", "jobID", job.ID, "url", apperrors.SanitizeURL(url), "request_id", getJobRequestID(job))
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
		result, err := scrape.Run(jobCtx, scrape.Request{
			URL:              url,
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
		})
		if err != nil {
			if jobCtx.Err() != nil {
				slog.Info("job canceled during scrape", "jobID", job.ID)
				m.updateStatusWithTimeout(job.ID, model.StatusCanceled, "canceled by user", 5*time.Second)
				return nil
			}
			slog.Error("scrape job failed", "jobID", job.ID, "url", apperrors.SanitizeURL(url), "error", err)
			m.updateStatusWithTimeout(job.ID, model.StatusFailed, err.Error(), 2*time.Second)
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
		results, err := crawl.Run(jobCtx, crawl.Request{
			URL:              url,
			RequestID:        getJobRequestID(job),
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
			MaxRetries:       reqRetries(m.maxRetries),
			RetryBase:        m.retryBase,
			MaxResponseBytes: m.maxResponseBytes,
			DataDir:          m.dataDir,
			Incremental:      incremental,
			Store:            m.store,
			Registry:         m.pipelineRegistry,
			JSRegistry:       m.jsRegistry,
			TemplateRegistry: m.templateRegistry,
		})
		if err != nil {
			if jobCtx.Err() != nil {
				slog.Info("job canceled during crawl", "jobID", job.ID)
				m.updateStatusWithTimeout(job.ID, model.StatusCanceled, "canceled by user", 5*time.Second)
				return nil
			}
			slog.Error("crawl job failed", "jobID", job.ID, "url", apperrors.SanitizeURL(url), "error", err)
			m.updateStatusWithTimeout(job.ID, model.StatusFailed, err.Error(), 2*time.Second)
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
		incremental := toBool(job.Params["incremental"], false)
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
			Incremental:      incremental,
			Store:            m.store,
			Registry:         m.pipelineRegistry,
			JSRegistry:       m.jsRegistry,
			TemplateRegistry: m.templateRegistry,
		})
		if err != nil {
			if jobCtx.Err() != nil {
				slog.Info("job canceled during research", "jobID", job.ID)
				m.updateStatusWithTimeout(job.ID, model.StatusCanceled, "canceled by user", 5*time.Second)
				return nil
			}
			slog.Error("research job failed", "jobID", job.ID, "query", query, "error", err)
			m.updateStatusWithTimeout(job.ID, model.StatusFailed, err.Error(), 2*time.Second)
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
		m.updateStatusWithTimeout(job.ID, model.StatusFailed, "unknown job kind", 2*time.Second)
		return apperrors.Internal("unknown job kind")
	}

	if jobCtx.Err() != nil {
		slog.Info("job completed but was canceled", "jobID", job.ID)
		return nil
	}

	slog.Info("job succeeded", "jobID", job.ID)
	updateCtx, cancelUpdate := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelUpdate()
	if err := m.store.UpdateStatus(updateCtx, job.ID, model.StatusSucceeded, ""); err != nil {
		slog.Error("failed to update job status to succeeded", "jobID", job.ID, "error", err)
	}
	return nil
}
