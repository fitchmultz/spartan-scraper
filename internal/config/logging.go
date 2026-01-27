// Package config provides logging configuration and conventions.
// This file documents standardized logging patterns used throughout the codebase.
package config

/*
Logging Conventions

Log Levels:
  - Error: Failures requiring action or that cannot be recovered
  - Warn:  Issues that don't stop execution but may indicate problems
  - Info:  Significant state changes (start/stop, completed, recovered)
  - Debug: Detailed flow tracing, retry attempts, intermediate steps

Structured Field Naming:
  - Use camelCase for field names: jobID, workerID, scheduleID
  - Standard fields: url, jobID, workerID, kind, error, attempt, status
  - Component-specific: loginURL (safe - never includes credentials), selector, mode

Security:
  - NEVER log: passwords, tokens, API keys, cookie values, auth credentials
  - For apperrors-wrapped errors: use apperrors.SafeMessage(err) when logging
  - Login logs: only log loginURL, never include credentials

Level Guidelines by Component:

  Job Manager (internal/jobs):
    - Info: job lifecycle (create, complete, cancel, recovery)
    - Debug: worker lifecycle (start/stop), job pickup
    - Warn: queue full, checkpoint failures
    - Error: job execution failures, store errors

  Fetchers (internal/fetch):
    - Debug: fetch lifecycle, retry flow, wait strategies
    - Warn: fetch failures with retry, status codes 4xx
    - Error: setup failures, max retries exceeded, login failures

  Crawl/Scrape (internal/crawl, internal/scrape):
    - Debug: page processing, fetch/extract stages
    - Info: start/end, content not modified, page complete
    - Warn: task channel full, 4xx status codes
    - Error: failures at any stage (fetch, extract, pipeline)

  Scheduler (internal/scheduler):
    - Warn: config file read/parse failures
    - Error: enqueue failures, save failures

  Server (internal/cli/server):
    - Info: lifecycle events (start, shutdown, worker status)
    - Warn: timeout waiting for workers
    - Error: store failures, HTTP server errors

High-Volume Logs:
  - Move high-frequency logs to Debug level (e.g., worker job pickup)
  - Keep startup/recovery logs at Info for visibility
  - Use rate limiting for very noisy operations (not currently needed)
*/
