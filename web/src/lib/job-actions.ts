/**
 * Job Actions
 *
 * Pure functions for submitting and managing jobs. These functions wrap
 * API calls with consistent error handling and loading state management.
 * They are not React-specific and can be easily tested and mocked.
 *
 * @module job-actions
 */

import type { ScrapeRequest, CrawlRequest, ResearchRequest } from "../api";

export interface JobActionContext {
  request: ScrapeRequest | CrawlRequest | ResearchRequest;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
  refreshJobs: () => Promise<void>;
  getApiBaseUrl: () => string;
}

/**
 * Submit a scrape job via API.
 */
export async function submitScrapeJob(
  postV1Scrape: (params: {
    baseUrl: string;
    body: ScrapeRequest;
  }) => Promise<{ error?: unknown }>,
  context: JobActionContext,
): Promise<void> {
  const { request, setLoading, setError, refreshJobs, getApiBaseUrl } = context;

  setLoading(true);
  try {
    const { error: apiError } = await postV1Scrape({
      baseUrl: getApiBaseUrl(),
      body: request as ScrapeRequest,
    });
    if (apiError) {
      setError(String(apiError));
      return;
    }
    setError(null);
    await refreshJobs();
  } catch (err) {
    setError(String(err));
  } finally {
    setLoading(false);
  }
}

/**
 * Submit a crawl job via API.
 */
export async function submitCrawlJob(
  postV1Crawl: (params: {
    baseUrl: string;
    body: CrawlRequest;
  }) => Promise<{ error?: unknown }>,
  context: JobActionContext,
): Promise<void> {
  const { request, setLoading, setError, refreshJobs, getApiBaseUrl } = context;

  setLoading(true);
  try {
    const { error: apiError } = await postV1Crawl({
      baseUrl: getApiBaseUrl(),
      body: request as CrawlRequest,
    });
    if (apiError) {
      setError(String(apiError));
      return;
    }
    setError(null);
    await refreshJobs();
  } catch (err) {
    setError(String(err));
  } finally {
    setLoading(false);
  }
}

/**
 * Submit a research job via API.
 */
export async function submitResearchJob(
  postV1Research: (params: {
    baseUrl: string;
    body: ResearchRequest;
  }) => Promise<{ error?: unknown }>,
  context: JobActionContext,
): Promise<void> {
  const { request, setLoading, setError, refreshJobs, getApiBaseUrl } = context;

  setLoading(true);
  try {
    const { error: apiError } = await postV1Research({
      baseUrl: getApiBaseUrl(),
      body: request as ResearchRequest,
    });
    if (apiError) {
      setError(String(apiError));
      return;
    }
    setError(null);
    await refreshJobs();
  } catch (err) {
    setError(String(err));
  } finally {
    setLoading(false);
  }
}

/**
 * Cancel a job via API.
 */
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
): Promise<void> {
  setLoading(true);
  try {
    const { error: apiError } = await deleteV1JobsById({
      baseUrl: getApiBaseUrl(),
      path: { id: jobId },
    });
    if (apiError) {
      setError(String(apiError));
      return;
    }
    setError(null);
    await refreshJobs();
  } catch (err) {
    setError(String(err));
  } finally {
    setLoading(false);
  }
}

/**
 * Delete a job via API with confirmation.
 */
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
  selectedJobId: string | null,
  onJobDeleted: (jobId: string) => void,
): Promise<void> {
  if (!confirm("Are you sure you want to permanently delete this job?")) {
    return;
  }

  setLoading(true);
  try {
    const { error: apiError } = await deleteV1JobsById({
      baseUrl: getApiBaseUrl(),
      path: { id: jobId },
      query: { force: true },
    });
    if (apiError) {
      setError(String(apiError));
      return;
    }
    setError(null);
    await refreshJobs();
    if (selectedJobId === jobId) {
      onJobDeleted(jobId);
    }
  } catch (err) {
    setError(String(err));
  } finally {
    setLoading(false);
  }
}
