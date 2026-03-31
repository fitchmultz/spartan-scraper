/**
 * Purpose: Manage batch job lifecycle state for the web UI.
 * Responsibilities: Submit batches, fetch authoritative batch pages, refresh batch details/jobs, and expose polling-driven updates.
 * Scope: Client-side batch state and API orchestration for scrape/crawl/research batches only.
 * Usage: Call `useBatches()` from UI containers that render batch forms/lists and wire returned actions to controls.
 * Invariants/Assumptions: Batch entries always include normalized non-negative stats; persisted browser state is limited to the last submitted batch notice.
 */
import { useState, useEffect, useCallback, useRef } from "react";
import {
  getV1JobsBatch,
  getV1JobsBatchById,
  deleteV1JobsBatchById,
  postV1JobsBatchScrape,
  postV1JobsBatchCrawl,
  postV1JobsBatchResearch,
  type BatchListResponse,
  type BatchResponse,
  type BatchScrapeRequest,
  type BatchCrawlRequest,
  type BatchResearchRequest,
  type Job,
} from "../api";
import { getApiBaseUrl } from "../lib/api-config";
import { reportRuntimeError } from "../lib/runtime-errors";

const POLL_INTERVAL_MS = 5000;
const DEFAULT_BATCH_PAGE_SIZE = 25;
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
  progress: {
    completed: number;
    remaining: number;
    percent: number;
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
    error: typeof job.error === "string" ? job.error : "",
    run:
      job.run && typeof job.run === "object"
        ? job.run
        : { waitMs: 0, runMs: 0, totalMs: 0 },
  } as Job;
}

function readBatchJobs(
  jobs: BatchResponse["jobs"] | undefined,
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

export function createEmptyBatchProgress(): BatchEntry["progress"] {
  return {
    completed: 0,
    remaining: 0,
    percent: 0,
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

export function normalizeBatchProgress(
  progress: Partial<Record<keyof BatchEntry["progress"], unknown>> | undefined,
): BatchEntry["progress"] {
  return {
    completed: toNonNegativeNumber(progress?.completed),
    remaining: toNonNegativeNumber(progress?.remaining),
    percent: toNonNegativeNumber(progress?.percent),
  };
}

export function deriveBatchProgressFromStats(
  stats: BatchEntry["stats"],
  jobCount: number,
): BatchEntry["progress"] {
  const completed = stats.succeeded + stats.failed + stats.canceled;
  const remaining = Math.max(0, jobCount - completed);
  const percent = jobCount > 0 ? Math.round((completed / jobCount) * 100) : 0;
  return {
    completed,
    remaining,
    percent,
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

function mapBatchSummary(
  summary: BatchListResponse["batches"][number],
): BatchEntry {
  const createdAt = readIsoDate(summary.createdAt, new Date(0).toISOString());
  const jobCount = toNonNegativeNumber(summary.jobCount);
  const stats = normalizeBatchStats(summary.stats);
  const reportedProgress = normalizeBatchProgress(summary.progress);
  const hasReportedProgress =
    reportedProgress.completed +
      reportedProgress.remaining +
      reportedProgress.percent >
    0;
  return {
    id: summary.id,
    kind: isBatchKind(summary.kind) ? summary.kind : "scrape",
    status: isBatchStatus(summary.status) ? summary.status : "pending",
    jobCount,
    stats,
    progress: hasReportedProgress
      ? reportedProgress
      : deriveBatchProgressFromStats(stats, jobCount),
    createdAt,
    updatedAt: readIsoDate(summary.updatedAt, createdAt),
  };
}

export function mapBatchResponse(response: BatchResponse): BatchEntry {
  const batch = response.batch;
  const createdAt = readIsoDate(batch.createdAt, new Date(0).toISOString());
  const jobs = readBatchJobs(response.jobs);
  const reportedJobCount = toNonNegativeNumber(batch.jobCount);
  const jobCount =
    reportedJobCount > 0
      ? reportedJobCount
      : jobs && jobs.length > 0
        ? jobs.length
        : 0;
  const normalizedStats = normalizeBatchStats(batch.stats);
  const hasReportedStats =
    normalizedStats.queued +
      normalizedStats.running +
      normalizedStats.succeeded +
      normalizedStats.failed +
      normalizedStats.canceled >
    0;
  const stats = hasReportedStats
    ? normalizedStats
    : deriveBatchStatsFromJobs(jobs, jobCount);
  const reportedProgress = normalizeBatchProgress(batch.progress);
  const hasReportedProgress =
    reportedProgress.completed +
      reportedProgress.remaining +
      reportedProgress.percent >
    0;

  return {
    id: batch.id,
    kind: isBatchKind(batch.kind) ? batch.kind : "scrape",
    status: isBatchStatus(batch.status) ? batch.status : "pending",
    jobCount,
    stats,
    progress: hasReportedProgress
      ? reportedProgress
      : deriveBatchProgressFromStats(stats, jobCount),
    createdAt,
    updatedAt: readIsoDate(batch.updatedAt, createdAt),
  };
}

function readStoredLastSubmittedBatch(): BatchSubmissionRecord | null {
  try {
    const stored = localStorage.getItem(LAST_SUBMITTED_BATCH_STORAGE_KEY);
    if (!stored) {
      return null;
    }

    const parsed = JSON.parse(stored) as unknown;
    return normalizeStoredBatchSubmission(parsed);
  } catch {
    return null;
  }
}

type BatchSubmitter<TRequest> = (params: {
  baseUrl: string;
  body: TRequest;
}) => Promise<{ data?: BatchResponse; error?: unknown }>;

export function useBatches() {
  const [batches, setBatches] = useState<BatchEntry[]>([]);
  const [batchJobs, setBatchJobs] = useState<Map<string, Job[]>>(new Map());
  const [lastSubmittedBatch, setLastSubmittedBatch] =
    useState<BatchSubmissionRecord | null>(() =>
      readStoredLastSubmittedBatch(),
    );
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [total, setTotal] = useState(0);
  const [limit, setLimit] = useState(DEFAULT_BATCH_PAGE_SIZE);
  const [offset, setOffset] = useState(0);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const initializedRef = useRef(false);

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

  const getBatchList = useCallback(async (pageOffset = 0) => {
    const { data, error: apiError } = await getV1JobsBatch({
      baseUrl: getApiBaseUrl(),
      query: {
        limit: DEFAULT_BATCH_PAGE_SIZE,
        offset: pageOffset,
      },
    });

    if (apiError) {
      throw new Error(formatApiError(apiError));
    }
    if (!data) {
      throw new Error("No data returned");
    }

    return data;
  }, []);

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

  const refreshBatch = useCallback(
    async (batchId: string) => {
      try {
        const response = await getBatchStatus(batchId, true);
        const updated = mapBatchResponse(response);

        setBatches((current) => {
          const existing = current.find((batch) => batch.id === batchId);
          if (existing) {
            return current.map((batch) =>
              batch.id === batchId ? updated : batch,
            );
          }
          if (offset === 0) {
            return [updated, ...current].slice(0, limit);
          }
          return current;
        });

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
        reportRuntimeError("Failed to refresh batch", err);
        throw err;
      }
    },
    [getBatchStatus, limit, offset],
  );

  const refreshBatches = useCallback(
    async (nextOffset = offset) => {
      setLoading(true);
      setError(null);

      try {
        const response = await getBatchList(nextOffset);
        const nextBatches = response.batches.map((batch) =>
          mapBatchSummary(batch),
        );
        const resolvedLimit =
          toNonNegativeNumber(response.limit) || DEFAULT_BATCH_PAGE_SIZE;
        const resolvedOffset = toNonNegativeNumber(response.offset);
        const resolvedTotal = toNonNegativeNumber(response.total);
        const visibleBatchIDs = new Set(nextBatches.map((batch) => batch.id));

        setBatches(nextBatches);
        setLimit(resolvedLimit);
        setOffset(resolvedOffset);
        setTotal(resolvedTotal);
        setBatchJobs((current) => {
          const next = new Map<string, Job[]>();
          for (const [batchId, jobs] of current.entries()) {
            if (visibleBatchIDs.has(batchId)) {
              next.set(batchId, jobs);
            }
          }
          return next;
        });
      } catch (err) {
        setError(
          err instanceof Error ? err.message : "Failed to refresh batches",
        );
      } finally {
        setLoading(false);
      }
    },
    [getBatchList, offset],
  );

  const cancelBatch = useCallback(
    async (batchId: string) => {
      const { data, error: apiError } = await deleteV1JobsBatchById({
        baseUrl: getApiBaseUrl(),
        path: { id: batchId },
      });

      if (apiError) {
        throw new Error(formatApiError(apiError));
      }
      if (!data) {
        throw new Error("No data returned");
      }

      const updated = mapBatchResponse(data);
      setBatches((current) =>
        current.map((batch) => (batch.id === batchId ? updated : batch)),
      );

      if (batchJobs.has(batchId)) {
        void refreshBatch(batchId).catch(() => {
          // Best-effort detail refresh so any loaded job rows reflect cancellation.
        });
      }
    },
    [batchJobs, refreshBatch],
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

      const entry = mapBatchResponse(data);
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

      setBatches((current) =>
        [entry, ...current.filter((batch) => batch.id !== entry.id)].slice(
          0,
          limit,
        ),
      );
      setOffset(0);

      const jobs = readBatchJobs(data.jobs);
      if (jobs) {
        setBatchJobs((current) => {
          const next = new Map(current);
          next.set(entry.id, jobs);
          return next;
        });
      }

      setLastSubmittedBatch(submissionRecord);
      saveLastSubmittedBatch(submissionRecord);
      await refreshBatches(0);
      return entry;
    },
    [limit, refreshBatches, saveLastSubmittedBatch],
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

  const hasProcessing = batches.some(
    (batch) => batch.status === "pending" || batch.status === "processing",
  );

  useEffect(() => {
    if (initializedRef.current) {
      return;
    }
    initializedRef.current = true;
    void refreshBatches(0);
  }, [refreshBatches]);

  useEffect(() => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
      intervalRef.current = null;
    }

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
    total,
    limit,
    offset,
    refreshBatch,
    refreshBatches,
    cancelBatch,
    submitBatchScrape,
    submitBatchCrawl,
    submitBatchResearch,
    clearLastSubmittedBatch,
    hasProcessing,
  };
}
