/**
 * Hook for batch data management
 *
 * Fetches batch list, provides refresh, handles polling for updates.
 * Manages batch state including job listings for expanded batches.
 *
 * @module useBatches
 */
import { useState, useEffect, useCallback, useRef } from "react";
import {
  getV1JobsBatchById,
  deleteV1JobsBatchById,
  postV1JobsBatchScrape,
  postV1JobsBatchCrawl,
  postV1JobsBatchResearch,
  type BatchStatusResponse,
  type BatchScrapeRequest,
  type BatchCrawlRequest,
  type BatchResearchRequest,
  type Job,
} from "../api";
import { getApiBaseUrl } from "../lib/api-config";

const POLL_INTERVAL_MS = 5000;

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

function mapBatchResponse(response: BatchStatusResponse): BatchEntry {
  return {
    id: response.id,
    kind: response.kind as BatchEntry["kind"],
    status: response.status as BatchEntry["status"],
    jobCount: response.jobCount,
    stats: response.stats,
    createdAt: response.createdAt,
    updatedAt: response.updatedAt,
  };
}

export function useBatches() {
  const [batches, setBatches] = useState<BatchEntry[]>([]);
  const [batchJobs, setBatchJobs] = useState<Map<string, Job[]>>(new Map());
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Load batches from localStorage (persisted across sessions)
  const loadBatches = useCallback(() => {
    try {
      const stored = localStorage.getItem("spartan_batches");
      if (stored) {
        const parsed = JSON.parse(stored) as BatchEntry[];
        setBatches(parsed);
      }
    } catch {
      // Ignore localStorage errors
    }
  }, []);

  // Save batches to localStorage
  const saveBatches = useCallback((newBatches: BatchEntry[]) => {
    try {
      localStorage.setItem("spartan_batches", JSON.stringify(newBatches));
    } catch {
      // Ignore localStorage errors
    }
  }, []);

  // Get batch status from API
  const getBatchStatus = useCallback(
    async (batchId: string, includeJobs = false) => {
      const { data, error: apiError } = await getV1JobsBatchById({
        baseUrl: getApiBaseUrl(),
        path: { id: batchId },
        query: includeJobs ? { include_jobs: true } : undefined,
      });

      if (apiError) {
        throw new Error(String(apiError));
      }

      if (!data) {
        throw new Error("No data returned");
      }

      return data as BatchStatusResponse;
    },
    [],
  );

  // Refresh single batch
  const refreshBatch = useCallback(
    async (batchId: string) => {
      try {
        const response = await getBatchStatus(batchId, true);
        const updated = mapBatchResponse(response);

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
        if (response.jobs) {
          setBatchJobs((current) => {
            const next = new Map(current);
            next.set(batchId, response.jobs as Job[]);
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
        throw new Error(String(apiError));
      }

      // Refresh to get updated status
      await refreshBatch(batchId);
    },
    [refreshBatch],
  );

  // Submit batch scrape
  const submitBatchScrape = useCallback(
    async (request: BatchScrapeRequest) => {
      const { data, error: apiError } = await postV1JobsBatchScrape({
        baseUrl: getApiBaseUrl(),
        body: request,
      });

      if (apiError) {
        throw new Error(String(apiError));
      }

      if (!data) {
        throw new Error("No response from server");
      }

      // Add to tracked batches
      const response = data as BatchStatusResponse;
      const entry = mapBatchResponse(response);

      setBatches((current) => {
        const newBatches = [entry, ...current];
        saveBatches(newBatches);
        return newBatches;
      });

      return entry;
    },
    [saveBatches],
  );

  // Submit batch crawl
  const submitBatchCrawl = useCallback(
    async (request: BatchCrawlRequest) => {
      const { data, error: apiError } = await postV1JobsBatchCrawl({
        baseUrl: getApiBaseUrl(),
        body: request,
      });

      if (apiError) {
        throw new Error(String(apiError));
      }

      if (!data) {
        throw new Error("No response from server");
      }

      const response = data as BatchStatusResponse;
      const entry = mapBatchResponse(response);

      setBatches((current) => {
        const newBatches = [entry, ...current];
        saveBatches(newBatches);
        return newBatches;
      });

      return entry;
    },
    [saveBatches],
  );

  // Submit batch research
  const submitBatchResearch = useCallback(
    async (request: BatchResearchRequest) => {
      const { data, error: apiError } = await postV1JobsBatchResearch({
        baseUrl: getApiBaseUrl(),
        body: request,
      });

      if (apiError) {
        throw new Error(String(apiError));
      }

      if (!data) {
        throw new Error("No response from server");
      }

      const response = data as BatchStatusResponse;
      const entry = mapBatchResponse(response);

      setBatches((current) => {
        const newBatches = [entry, ...current];
        saveBatches(newBatches);
        return newBatches;
      });

      return entry;
    },
    [saveBatches],
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
    },
    [saveBatches],
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
    loading,
    error,
    refreshBatch,
    refreshBatches,
    cancelBatch,
    submitBatchScrape,
    submitBatchCrawl,
    submitBatchResearch,
    removeBatch,
    hasProcessing,
  };
}
