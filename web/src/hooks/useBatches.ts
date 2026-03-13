/**
 * Purpose: Manage batch job lifecycle state for the web UI.
 * Responsibilities: Persist tracked batches, submit batch jobs, refresh status/jobs, and expose polling-driven updates.
 * Scope: Client-side batch state and API orchestration for scrape/crawl/research batches only.
 * Usage: Call `useBatches()` from UI containers that render batch forms/lists and wire returned actions to controls.
 * Invariants/Assumptions: Batch entries always include normalized non-negative stats; localStorage data may be malformed and must be sanitized.
 */
import { useState, useEffect, useCallback, useRef } from "react";
import {
  getV1JobsBatchById,
  deleteV1JobsBatchById,
  postV1JobsBatchScrape,
  postV1JobsBatchCrawl,
  postV1JobsBatchResearch,
  type BatchResponse,
  type BatchScrapeRequest,
  type BatchStatusResponse,
  type BatchCrawlRequest,
  type BatchResearchRequest,
  type Job,
} from "../api";
import { getApiBaseUrl } from "../lib/api-config";

const POLL_INTERVAL_MS = 5000;
const BATCHES_STORAGE_KEY = "spartan_batches";
const LAST_SUBMITTED_BATCH_STORAGE_KEY = "spartan_last_submitted_batch";

export type BatchEntry = {
  id: string;
  kind: "scrape" | "crawl" | "research";
  status:
    | "pending"
    | "processing"
    | "completed"
    | "failed"
    | "partial"
    | "canceled";
  jobCount: number;
  stats: {
    queued: number;
    running: number;
    succeeded: number;
    failed: number;
    canceled: number;
  };
  createdAt: string;
  updatedAt: string;
};

export type BatchSubmissionRecord = {
  batchId: string;
  kind: BatchEntry["kind"];
  submittedUrls: string[];
  submittedAt: string;
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function toNonNegativeNumber(value: unknown): number {
  if (typeof value !== "number" || Number.isNaN(value)) {
    return 0;
  }
  return value < 0 ? 0 : value;
}

function isBatchKind(value: unknown): value is BatchEntry["kind"] {
  return value === "scrape" || value === "crawl" || value === "research";
}

function isBatchStatus(value: unknown): value is BatchEntry["status"] {
  return (
    value === "pending" ||
    value === "processing" ||
    value === "completed" ||
    value === "failed" ||
    value === "partial" ||
    value === "canceled"
  );
}

function isJobStatus(value: unknown): value is Job["status"] {
  return (
    value === "queued" ||
    value === "running" ||
    value === "succeeded" ||
    value === "failed" ||
    value === "canceled"
  );
}

function formatApiError(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }
  if (typeof error === "string") {
    return error;
  }
  try {
    return JSON.stringify(error);
  } catch {
    return String(error);
  }
}

function readIsoDate(value: unknown, fallback: string): string {
  if (typeof value === "string" && value.length > 0) {
    return value;
  }
  return fallback;
}

function isStringArray(value: unknown): value is string[] {
  return (
    Array.isArray(value) && value.every((entry) => typeof entry === "string")
  );
}

function normalizeStoredBatchSubmission(
  input: unknown,
): BatchSubmissionRecord | null {
  if (!isRecord(input)) {
    return null;
  }

  const batchId = typeof input.batchId === "string" ? input.batchId : "";
  if (!batchId) {
    return null;
  }

  const kind = isBatchKind(input.kind) ? input.kind : "scrape";
  const submittedUrls = isStringArray(input.submittedUrls)
    ? input.submittedUrls.filter((url) => url.trim().length > 0)
    : [];

  if (submittedUrls.length === 0) {
    return null;
  }

  return {
    batchId,
    kind,
    submittedUrls,
    submittedAt: readIsoDate(input.submittedAt, new Date(0).toISOString()),
  };
}

function extractSubmittedUrls(request: unknown): string[] {
  if (!isRecord(request) || !Array.isArray(request.jobs)) {
    return [];
  }

  return request.jobs
    .map((job) =>
      isRecord(job) && typeof job.url === "string" ? job.url.trim() : "",
    )
    .filter((url) => url.length > 0);
}

function toJobSummary(job: Partial<Job>): Job | null {
  if (!job.id) {
    return null;
  }
  const createdAt = readIsoDate(job.createdAt, new Date(0).toISOString());
  return {
    id: job.id,
    kind: isBatchKind(job.kind) ? job.kind : "scrape",
    status: isJobStatus(job.status) ? job.status : "queued",
    createdAt,
    updatedAt: readIsoDate(job.updatedAt, createdAt),
    specVersion:
      typeof job.specVersion === "number" && Number.isFinite(job.specVersion)
        ? job.specVersion
        : 1,
    spec: job.spec && typeof job.spec === "object" ? job.spec : { version: 1 },
  };
}

function readBatchJobs(
  jobs: BatchResponse["jobs"] | BatchStatusResponse["jobs"] | undefined,
): Job[] | undefined {
  return jobs
    ?.map((job) => toJobSummary(job))
    .filter((job): job is Job => job !== null);
}

export function createEmptyBatchStats(): BatchEntry["stats"] {
  return {
    queued: 0,
    running: 0,
    succeeded: 0,
    failed: 0,
    canceled: 0,
  };
}

export function normalizeBatchStats(
  stats: Partial<Record<keyof BatchEntry["stats"], unknown>> | undefined,
): BatchEntry["stats"] {
  return {
    queued: toNonNegativeNumber(stats?.queued),
    running: toNonNegativeNumber(stats?.running),
    succeeded: toNonNegativeNumber(stats?.succeeded),
    failed: toNonNegativeNumber(stats?.failed),
    canceled: toNonNegativeNumber(stats?.canceled),
  };
}

export function deriveBatchStatsFromJobs(
  jobs: readonly Pick<Job, "status">[] | undefined,
  jobCount: number,
): BatchEntry["stats"] {
  const stats = createEmptyBatchStats();

  if (!jobs || jobs.length === 0) {
    stats.queued = toNonNegativeNumber(jobCount);
    return stats;
  }

  for (const job of jobs) {
    if (job.status === "queued") {
      stats.queued += 1;
    } else if (job.status === "running") {
      stats.running += 1;
    } else if (job.status === "succeeded") {
      stats.succeeded += 1;
    } else if (job.status === "failed") {
      stats.failed += 1;
    } else if (job.status === "canceled") {
      stats.canceled += 1;
    }
  }

  const totalCounted =
    stats.queued +
    stats.running +
    stats.succeeded +
    stats.failed +
    stats.canceled;
  if (totalCounted === 0 && jobCount > 0) {
    stats.queued = toNonNegativeNumber(jobCount);
  }

  return stats;
}

export function mapBatchStatusResponse(
  response: BatchStatusResponse,
): BatchEntry {
  const createdAt = readIsoDate(response.createdAt, new Date(0).toISOString());
  return {
    id: response.id,
    kind: isBatchKind(response.kind) ? response.kind : "scrape",
    status: isBatchStatus(response.status) ? response.status : "pending",
    jobCount: toNonNegativeNumber(response.jobCount),
    stats: normalizeBatchStats(response.stats),
    createdAt,
    updatedAt: readIsoDate(response.updatedAt, createdAt),
  };
}

export function mapBatchCreateResponse(response: BatchResponse): BatchEntry {
  const createdAt = readIsoDate(response.createdAt, new Date(0).toISOString());
  const jobs = readBatchJobs(response.jobs);
  const reportedJobCount = toNonNegativeNumber(response.jobCount);
  const jobCount =
    reportedJobCount > 0
      ? reportedJobCount
      : jobs && jobs.length > 0
        ? jobs.length
        : 0;

  return {
    id: response.id,
    kind: isBatchKind(response.kind) ? response.kind : "scrape",
    status: isBatchStatus(response.status) ? response.status : "pending",
    jobCount,
    stats: deriveBatchStatsFromJobs(jobs, jobCount),
    createdAt,
    updatedAt: createdAt,
  };
}

type BatchSubmitter<TRequest> = (params: {
  baseUrl: string;
  body: TRequest;
}) => Promise<{ data?: BatchResponse; error?: unknown }>;

export function normalizeStoredBatchEntries(input: unknown): BatchEntry[] {
  if (!Array.isArray(input)) {
    return [];
  }

  const normalized: BatchEntry[] = [];
  for (const entry of input) {
    if (!isRecord(entry)) {
      continue;
    }

    const id = typeof entry.id === "string" ? entry.id : "";
    if (!id) {
      continue;
    }

    const createdAt =
      typeof entry.createdAt === "string"
        ? entry.createdAt
        : new Date(0).toISOString();
    const updatedAt =
      typeof entry.updatedAt === "string" ? entry.updatedAt : createdAt;

    const rawStats = isRecord(entry.stats)
      ? {
          queued: entry.stats.queued,
          running: entry.stats.running,
          succeeded: entry.stats.succeeded,
          failed: entry.stats.failed,
          canceled: entry.stats.canceled,
        }
      : undefined;

    normalized.push({
      id,
      kind: isBatchKind(entry.kind) ? entry.kind : "scrape",
      status: isBatchStatus(entry.status) ? entry.status : "pending",
      jobCount: toNonNegativeNumber(entry.jobCount),
      stats: normalizeBatchStats(rawStats),
      createdAt,
      updatedAt,
    });
  }

  return normalized;
}

export function useBatches() {
  const [batches, setBatches] = useState<BatchEntry[]>([]);
  const [batchJobs, setBatchJobs] = useState<Map<string, Job[]>>(new Map());
  const [lastSubmittedBatch, setLastSubmittedBatch] =
    useState<BatchSubmissionRecord | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Load batches from localStorage (persisted across sessions)
  const loadBatches = useCallback(() => {
    try {
      const stored = localStorage.getItem(BATCHES_STORAGE_KEY);
      if (stored) {
        const parsed = JSON.parse(stored) as unknown;
        const normalized = normalizeStoredBatchEntries(parsed);
        setBatches(normalized);
      }
    } catch {
      // Ignore localStorage errors
    }
  }, []);

  // Save batches to localStorage
  const saveBatches = useCallback((newBatches: BatchEntry[]) => {
    try {
      localStorage.setItem(BATCHES_STORAGE_KEY, JSON.stringify(newBatches));
    } catch {
      // Ignore localStorage errors
    }
  }, []);

  const loadLastSubmittedBatch = useCallback(() => {
    try {
      const stored = localStorage.getItem(LAST_SUBMITTED_BATCH_STORAGE_KEY);
      if (!stored) {
        return;
      }

      const parsed = JSON.parse(stored) as unknown;
      setLastSubmittedBatch(normalizeStoredBatchSubmission(parsed));
    } catch {
      // Ignore localStorage errors
    }
  }, []);

  const saveLastSubmittedBatch = useCallback(
    (entry: BatchSubmissionRecord | null) => {
      try {
        if (!entry) {
          localStorage.removeItem(LAST_SUBMITTED_BATCH_STORAGE_KEY);
          return;
        }

        localStorage.setItem(
          LAST_SUBMITTED_BATCH_STORAGE_KEY,
          JSON.stringify(entry),
        );
      } catch {
        // Ignore localStorage errors
      }
    },
    [],
  );

  const clearLastSubmittedBatch = useCallback(() => {
    setLastSubmittedBatch(null);
    saveLastSubmittedBatch(null);
  }, [saveLastSubmittedBatch]);

  // Get batch status from API
  const getBatchStatus = useCallback(
    async (batchId: string, includeJobs = false) => {
      const { data, error: apiError } = await getV1JobsBatchById({
        baseUrl: getApiBaseUrl(),
        path: { id: batchId },
        query: includeJobs ? { include_jobs: true } : undefined,
      });

      if (apiError) {
        throw new Error(formatApiError(apiError));
      }
      if (!data) {
        throw new Error("No data returned");
      }

      return data;
    },
    [],
  );

  // Refresh single batch
  const refreshBatch = useCallback(
    async (batchId: string) => {
      try {
        const response = await getBatchStatus(batchId, true);
        const updated = mapBatchStatusResponse(response);

        setBatches((current) => {
          const existing = current.find((b) => b.id === batchId);
          if (!existing) {
            const newBatches = [updated, ...current];
            saveBatches(newBatches);
            return newBatches;
          }
          const newBatches = current.map((b) =>
            b.id === batchId ? updated : b,
          );
          saveBatches(newBatches);
          return newBatches;
        });

        // Store jobs if included
        const jobs = readBatchJobs(response.jobs);
        if (jobs) {
          setBatchJobs((current) => {
            const next = new Map(current);
            next.set(batchId, jobs);
            return next;
          });
        }

        return updated;
      } catch (err) {
        console.error("Failed to refresh batch:", err);
        throw err;
      }
    },
    [getBatchStatus, saveBatches],
  );

  // Refresh all batches
  const refreshBatches = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      // Refresh all known batches
      const promises = batches.map((b) => refreshBatch(b.id).catch(() => null));
      await Promise.all(promises);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to refresh batches",
      );
    } finally {
      setLoading(false);
    }
  }, [batches, refreshBatch]);

  // Cancel a batch
  const cancelBatch = useCallback(
    async (batchId: string) => {
      const { error: apiError } = await deleteV1JobsBatchById({
        baseUrl: getApiBaseUrl(),
        path: { id: batchId },
      });

      if (apiError) {
        throw new Error(formatApiError(apiError));
      }

      // Refresh to get updated status
      await refreshBatch(batchId);
    },
    [refreshBatch],
  );

  const submitBatch = useCallback(
    async <TRequest>(
      submitter: BatchSubmitter<TRequest>,
      request: TRequest,
    ) => {
      const { data, error: apiError } = await submitter({
        baseUrl: getApiBaseUrl(),
        body: request,
      });

      if (apiError) {
        throw new Error(formatApiError(apiError));
      }
      if (!data) {
        throw new Error("No response from server");
      }

      const entry = mapBatchCreateResponse(data);
      const submittedUrls = extractSubmittedUrls(request);
      const submissionRecord =
        submittedUrls.length > 0
          ? {
              batchId: entry.id,
              kind: entry.kind,
              submittedUrls,
              submittedAt: new Date().toISOString(),
            }
          : null;

      setBatches((current) => {
        const newBatches = [entry, ...current];
        saveBatches(newBatches);
        return newBatches;
      });
      setLastSubmittedBatch(submissionRecord);
      saveLastSubmittedBatch(submissionRecord);
      void refreshBatch(entry.id).catch(() => {
        // Best-effort refresh to hydrate authoritative stats.
      });
      return entry;
    },
    [refreshBatch, saveBatches, saveLastSubmittedBatch],
  );

  const submitBatchScrape = useCallback(
    (request: BatchScrapeRequest) =>
      submitBatch(postV1JobsBatchScrape, request),
    [submitBatch],
  );

  const submitBatchCrawl = useCallback(
    (request: BatchCrawlRequest) => submitBatch(postV1JobsBatchCrawl, request),
    [submitBatch],
  );

  const submitBatchResearch = useCallback(
    (request: BatchResearchRequest) =>
      submitBatch(postV1JobsBatchResearch, request),
    [submitBatch],
  );

  // Remove a batch from tracking
  const removeBatch = useCallback(
    (batchId: string) => {
      setBatches((current) => {
        const newBatches = current.filter((b) => b.id !== batchId);
        saveBatches(newBatches);
        return newBatches;
      });
      setBatchJobs((current) => {
        const next = new Map(current);
        next.delete(batchId);
        return next;
      });
      setLastSubmittedBatch((current) => {
        if (!current || current.batchId !== batchId) {
          return current;
        }

        saveLastSubmittedBatch(null);
        return null;
      });
    },
    [saveBatches, saveLastSubmittedBatch],
  );

  // Check if any batches are processing
  const hasProcessing = batches.some(
    (b) => b.status === "pending" || b.status === "processing",
  );

  // Setup polling
  useEffect(() => {
    loadBatches();
  }, [loadBatches]);

  useEffect(() => {
    loadLastSubmittedBatch();
  }, [loadLastSubmittedBatch]);

  useEffect(() => {
    // Clear existing interval
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
      intervalRef.current = null;
    }

    // Only poll if there are processing batches
    if (hasProcessing) {
      intervalRef.current = setInterval(() => {
        void refreshBatches();
      }, POLL_INTERVAL_MS);
    }

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, [hasProcessing, refreshBatches]);

  return {
    batches,
    batchJobs,
    lastSubmittedBatch,
    loading,
    error,
    refreshBatch,
    refreshBatches,
    cancelBatch,
    submitBatchScrape,
    submitBatchCrawl,
    submitBatchResearch,
    removeBatch,
    clearLastSubmittedBatch,
    hasProcessing,
  };
}
