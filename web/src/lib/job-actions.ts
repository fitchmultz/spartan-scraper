/**
 * Purpose: Centralize job submission and lifecycle mutations behind testable, UI-agnostic helpers.
 * Responsibilities: Execute generated API calls, keep legacy loading/error callbacks synchronized, and return explicit action outcomes for toast-driven callers.
 * Scope: Web-client job submit/cancel/delete orchestration only.
 * Usage: Import these helpers from route containers or tests instead of repeating API mutation boilerplate.
 * Invariants/Assumptions: Callers own any confirmation UX, `setLoading`/`setError` remain safe no-op hooks, and helpers never throw after API/network failures are normalized.
 */

import type { CrawlRequest, ResearchRequest, ScrapeRequest } from "../api";
import { getApiErrorMessage } from "./api-errors";

export interface JobActionContext {
  request: ScrapeRequest | CrawlRequest | ResearchRequest;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
  refreshJobs: () => Promise<void>;
  getApiBaseUrl: () => string;
}

export type JobActionResult =
  | { status: "success" }
  | { status: "error"; message: string }
  | { status: "canceled" };

async function runJobMutation(
  execute: () => Promise<{ error?: unknown }>,
  setLoading: (loading: boolean) => void,
  setError: (error: string | null) => void,
  refreshJobs: () => Promise<void>,
  fallbackMessage: string,
): Promise<JobActionResult> {
  setLoading(true);
  try {
    const { error: apiError } = await execute();
    if (apiError) {
      const message = getApiErrorMessage(apiError, fallbackMessage);
      setError(message);
      return { status: "error", message };
    }
    setError(null);
    await refreshJobs();
    return { status: "success" };
  } catch (err) {
    const message = getApiErrorMessage(err, fallbackMessage);
    setError(message);
    return { status: "error", message };
  } finally {
    setLoading(false);
  }
}

export async function submitScrapeJob(
  postV1Scrape: (params: {
    baseUrl: string;
    body: ScrapeRequest;
  }) => Promise<{ error?: unknown }>,
  context: JobActionContext,
): Promise<JobActionResult> {
  const { request, setLoading, setError, refreshJobs, getApiBaseUrl } = context;

  return runJobMutation(
    () =>
      postV1Scrape({
        baseUrl: getApiBaseUrl(),
        body: request as ScrapeRequest,
      }),
    setLoading,
    setError,
    refreshJobs,
    "Failed to submit scrape job.",
  );
}

export async function submitCrawlJob(
  postV1Crawl: (params: {
    baseUrl: string;
    body: CrawlRequest;
  }) => Promise<{ error?: unknown }>,
  context: JobActionContext,
): Promise<JobActionResult> {
  const { request, setLoading, setError, refreshJobs, getApiBaseUrl } = context;

  return runJobMutation(
    () =>
      postV1Crawl({
        baseUrl: getApiBaseUrl(),
        body: request as CrawlRequest,
      }),
    setLoading,
    setError,
    refreshJobs,
    "Failed to submit crawl job.",
  );
}

export async function submitResearchJob(
  postV1Research: (params: {
    baseUrl: string;
    body: ResearchRequest;
  }) => Promise<{ error?: unknown }>,
  context: JobActionContext,
): Promise<JobActionResult> {
  const { request, setLoading, setError, refreshJobs, getApiBaseUrl } = context;

  return runJobMutation(
    () =>
      postV1Research({
        baseUrl: getApiBaseUrl(),
        body: request as ResearchRequest,
      }),
    setLoading,
    setError,
    refreshJobs,
    "Failed to submit research job.",
  );
}

export async function cancelJob(
  deleteV1JobsById: (params: {
    baseUrl: string;
    path: { id: string };
  }) => Promise<{ error?: unknown }>,
  jobId: string,
  setLoading: (loading: boolean) => void,
  setError: (error: string | null) => void,
  refreshJobs: () => Promise<void>,
  getApiBaseUrl: () => string,
): Promise<JobActionResult> {
  return runJobMutation(
    () =>
      deleteV1JobsById({
        baseUrl: getApiBaseUrl(),
        path: { id: jobId },
      }),
    setLoading,
    setError,
    refreshJobs,
    "Failed to cancel job.",
  );
}

export async function deleteJob(
  deleteV1JobsById: (params: {
    baseUrl: string;
    path: { id: string };
    query?: { force?: boolean };
  }) => Promise<{ error?: unknown }>,
  jobId: string,
  setLoading: (loading: boolean) => void,
  setError: (error: string | null) => void,
  refreshJobs: () => Promise<void>,
  getApiBaseUrl: () => string,
  confirmDelete: () => boolean | Promise<boolean>,
  selectedJobId: string | null,
  onJobDeleted: (jobId: string) => void,
): Promise<JobActionResult> {
  const confirmed = await confirmDelete();
  if (!confirmed) {
    return { status: "canceled" };
  }

  const result = await runJobMutation(
    () =>
      deleteV1JobsById({
        baseUrl: getApiBaseUrl(),
        path: { id: jobId },
        query: { force: true },
      }),
    setLoading,
    setError,
    refreshJobs,
    "Failed to delete job.",
  );

  if (result.status === "success" && selectedJobId === jobId) {
    onJobDeleted(jobId);
  }

  return result;
}
