/**
 * Purpose: Verify batch hook mapping and normalization logic that feeds BatchList rendering.
 * Responsibilities: Assert stable stats normalization for API/localStorage inputs and guard against undefined stats regressions.
 * Scope: Pure helper functions exported by useBatches (no network/UI rendering).
 * Usage: Run via `pnpm run test` or `make test-ci`.
 * Invariants/Assumptions: Batch entries consumed by UI must always include non-negative numeric stats.
 */

import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import type { BatchResponse, Job } from "../api";
import {
  getV1JobsBatchById,
  postV1JobsBatchCrawl,
  postV1JobsBatchResearch,
  postV1JobsBatchScrape,
} from "../api";
import {
  createEmptyBatchStats,
  deriveBatchStatsFromJobs,
  mapBatchResponse,
  normalizeBatchStats,
  normalizeStoredBatchEntries,
  useBatches,
} from "./useBatches";

vi.mock("../api", () => ({
  getV1JobsBatchById: vi.fn(),
  deleteV1JobsBatchById: vi.fn(),
  postV1JobsBatchScrape: vi.fn(),
  postV1JobsBatchCrawl: vi.fn(),
  postV1JobsBatchResearch: vi.fn(),
}));

vi.mock("../lib/api-config", () => ({
  getApiBaseUrl: () => "http://127.0.0.1:8741",
}));

function makeJob(id: string, status: Job["status"]): Job {
  const now = "2026-03-05T00:00:00.000Z";
  return {
    id,
    kind: "scrape",
    status,
    createdAt: now,
    updatedAt: now,
    specVersion: 1,
    spec: { version: 1 },
  };
}

const storage = new Map<string, string>();

beforeAll(() => {
  const localStorageMock = {
    getItem: (key: string) => storage.get(key) ?? null,
    setItem: (key: string, value: string) => {
      storage.set(key, value);
    },
    removeItem: (key: string) => {
      storage.delete(key);
    },
    clear: () => {
      storage.clear();
    },
  };

  vi.stubGlobal("localStorage", localStorageMock);
  if (typeof window !== "undefined") {
    Object.defineProperty(window, "localStorage", {
      value: localStorageMock,
      configurable: true,
    });
  }
});

beforeEach(() => {
  storage.clear();
  vi.clearAllMocks();
});

describe("useBatches helpers", () => {
  it("createEmptyBatchStats returns zeroed counters", () => {
    expect(createEmptyBatchStats()).toEqual({
      queued: 0,
      running: 0,
      succeeded: 0,
      failed: 0,
      canceled: 0,
    });
  });

  it("normalizeBatchStats coerces invalid values to non-negative numbers", () => {
    expect(
      normalizeBatchStats({
        queued: -3,
        running: Number.NaN,
        succeeded: 4,
        failed: -1,
        canceled: 2,
      }),
    ).toEqual({
      queued: 0,
      running: 0,
      succeeded: 4,
      failed: 0,
      canceled: 2,
    });

    expect(normalizeBatchStats(undefined)).toEqual({
      queued: 0,
      running: 0,
      succeeded: 0,
      failed: 0,
      canceled: 0,
    });
  });

  it("deriveBatchStatsFromJobs counts each job status", () => {
    const jobs: Job[] = [
      makeJob("1", "queued"),
      makeJob("2", "running"),
      makeJob("3", "succeeded"),
      makeJob("4", "failed"),
      makeJob("5", "canceled"),
    ];

    expect(deriveBatchStatsFromJobs(jobs, 5)).toEqual({
      queued: 1,
      running: 1,
      succeeded: 1,
      failed: 1,
      canceled: 1,
    });
  });

  it("deriveBatchStatsFromJobs falls back to queued=jobCount when jobs are absent", () => {
    expect(deriveBatchStatsFromJobs(undefined, 3)).toEqual({
      queued: 3,
      running: 0,
      succeeded: 0,
      failed: 0,
      canceled: 0,
    });

    expect(deriveBatchStatsFromJobs([], 0)).toEqual({
      queued: 0,
      running: 0,
      succeeded: 0,
      failed: 0,
      canceled: 0,
    });
  });

  it("mapBatchResponse always returns safe stats for BatchList", () => {
    const response: BatchResponse = {
      batch: {
        id: "batch-create-1",
        kind: "scrape",
        status: "pending",
        jobCount: 2,
        stats: {
          queued: 2,
          running: 0,
          succeeded: 0,
          failed: 0,
          canceled: 0,
        },
        createdAt: "2026-03-05T10:00:00.000Z",
        updatedAt: "2026-03-05T10:00:00.000Z",
      },
      jobs: [makeJob("1", "queued"), makeJob("2", "queued")],
      total: 2,
      limit: 2,
      offset: 0,
    };

    expect(mapBatchResponse(response)).toEqual({
      id: "batch-create-1",
      kind: "scrape",
      status: "pending",
      jobCount: 2,
      stats: {
        queued: 2,
        running: 0,
        succeeded: 0,
        failed: 0,
        canceled: 0,
      },
      createdAt: "2026-03-05T10:00:00.000Z",
      updatedAt: "2026-03-05T10:00:00.000Z",
    });
  });

  it("mapBatchResponse normalizes API stats before UI consumption", () => {
    const response: BatchResponse = {
      batch: {
        id: "batch-status-1",
        kind: "crawl",
        status: "processing",
        jobCount: 4,
        stats: {
          queued: 1,
          running: 2,
          succeeded: 1,
          failed: 0,
          canceled: 0,
        },
        createdAt: "2026-03-05T10:00:00.000Z",
        updatedAt: "2026-03-05T10:01:00.000Z",
      },
      jobs: [],
      total: 4,
      limit: 0,
      offset: 0,
    };

    expect(mapBatchResponse(response)).toEqual({
      id: "batch-status-1",
      kind: "crawl",
      status: "processing",
      jobCount: 4,
      stats: {
        queued: 1,
        running: 2,
        succeeded: 1,
        failed: 0,
        canceled: 0,
      },
      createdAt: "2026-03-05T10:00:00.000Z",
      updatedAt: "2026-03-05T10:01:00.000Z",
    });
  });

  it("mapBatchResponse falls back to jobCount when jobs are omitted", () => {
    expect(
      mapBatchResponse({
        batch: {
          id: "batch-create-2",
          kind: "research",
          status: "pending",
          jobCount: 3,
          stats: {
            queued: 3,
            running: 0,
            succeeded: 0,
            failed: 0,
            canceled: 0,
          },
          createdAt: "2026-03-05T12:00:00.000Z",
          updatedAt: "2026-03-05T12:00:00.000Z",
        },
        jobs: [],
        total: 3,
        limit: 0,
        offset: 0,
      }),
    ).toEqual({
      id: "batch-create-2",
      kind: "research",
      status: "pending",
      jobCount: 3,
      stats: {
        queued: 3,
        running: 0,
        succeeded: 0,
        failed: 0,
        canceled: 0,
      },
      createdAt: "2026-03-05T12:00:00.000Z",
      updatedAt: "2026-03-05T12:00:00.000Z",
    });
  });

  it("normalizeStoredBatchEntries discards malformed entries and fixes invalid fields", () => {
    const raw: unknown = [
      {
        id: "good-batch",
        kind: "research",
        status: "partial",
        jobCount: 3,
        stats: {
          queued: 0,
          running: 1,
          succeeded: 1,
          failed: 1,
          canceled: 0,
        },
        createdAt: "2026-03-05T10:00:00.000Z",
        updatedAt: "2026-03-05T10:05:00.000Z",
      },
      {
        id: "needs-defaults",
        kind: "unsupported-kind",
        status: "unsupported-status",
        jobCount: -9,
        stats: {
          queued: -1,
          running: -1,
          succeeded: -1,
          failed: -1,
          canceled: -1,
        },
      },
      { id: "" },
      null,
      42,
    ];

    expect(normalizeStoredBatchEntries(raw)).toEqual([
      {
        id: "good-batch",
        kind: "research",
        status: "partial",
        jobCount: 3,
        stats: {
          queued: 0,
          running: 1,
          succeeded: 1,
          failed: 1,
          canceled: 0,
        },
        createdAt: "2026-03-05T10:00:00.000Z",
        updatedAt: "2026-03-05T10:05:00.000Z",
      },
      {
        id: "needs-defaults",
        kind: "scrape",
        status: "pending",
        jobCount: 0,
        stats: {
          queued: 0,
          running: 0,
          succeeded: 0,
          failed: 0,
          canceled: 0,
        },
        createdAt: "1970-01-01T00:00:00.000Z",
        updatedAt: "1970-01-01T00:00:00.000Z",
      },
    ]);
  });
});

describe("useBatches persistence", () => {
  it("restores the last submitted batch from localStorage for persisted sessions", async () => {
    localStorage.setItem(
      "spartan_batches",
      JSON.stringify([
        {
          id: "batch-persisted-1",
          kind: "scrape",
          status: "pending",
          jobCount: 2,
          stats: {
            queued: 2,
            running: 0,
            succeeded: 0,
            failed: 0,
            canceled: 0,
          },
          createdAt: "2026-03-05T10:00:00.000Z",
          updatedAt: "2026-03-05T10:00:00.000Z",
        },
      ]),
    );
    localStorage.setItem(
      "spartan_last_submitted_batch",
      JSON.stringify({
        batchId: "batch-persisted-1",
        kind: "scrape",
        submittedUrls: ["https://example.com", "https://example.org"],
        submittedAt: "2026-03-05T10:00:05.000Z",
      }),
    );

    const { result } = renderHook(() => useBatches());

    await waitFor(() => {
      expect(result.current.lastSubmittedBatch).toEqual({
        batchId: "batch-persisted-1",
        kind: "scrape",
        submittedUrls: ["https://example.com", "https://example.org"],
        submittedAt: "2026-03-05T10:00:05.000Z",
      });
    });
  });

  it("persists the submitted batch summary and clears it when requested", async () => {
    vi.mocked(getV1JobsBatchById).mockResolvedValue({
      data: {
        batch: {
          id: "batch-submit-1",
          kind: "scrape",
          status: "pending",
          jobCount: 2,
          stats: {
            queued: 2,
            running: 0,
            succeeded: 0,
            failed: 0,
            canceled: 0,
          },
          createdAt: "2026-03-05T10:00:00.000Z",
          updatedAt: "2026-03-05T10:00:00.000Z",
        },
        jobs: [makeJob("1", "queued"), makeJob("2", "queued")],
        total: 2,
        limit: 2,
        offset: 0,
      },
      request: new Request(
        "http://127.0.0.1:8741/v1/jobs/batch/batch-submit-1",
      ),
      response: new Response(null, { status: 200 }),
    } as never);
    vi.mocked(postV1JobsBatchScrape).mockResolvedValue({
      data: {
        batch: {
          id: "batch-submit-1",
          kind: "scrape",
          status: "pending",
          jobCount: 2,
          stats: {
            queued: 2,
            running: 0,
            succeeded: 0,
            failed: 0,
            canceled: 0,
          },
          createdAt: "2026-03-05T10:00:00.000Z",
          updatedAt: "2026-03-05T10:00:00.000Z",
        },
        jobs: [makeJob("1", "queued"), makeJob("2", "queued")],
        total: 2,
        limit: 2,
        offset: 0,
      },
      request: new Request("http://127.0.0.1:8741/v1/jobs/batch/scrape"),
      response: new Response(null, { status: 202 }),
    } as never);
    vi.mocked(postV1JobsBatchCrawl).mockResolvedValue({} as never);
    vi.mocked(postV1JobsBatchResearch).mockResolvedValue({} as never);

    const { result } = renderHook(() => useBatches());

    await act(async () => {
      await result.current.submitBatchScrape({
        jobs: [{ url: "https://example.com" }, { url: "https://example.org" }],
      } as never);
    });

    expect(result.current.lastSubmittedBatch).toMatchObject({
      batchId: "batch-submit-1",
      kind: "scrape",
      submittedUrls: ["https://example.com", "https://example.org"],
    });
    expect(
      JSON.parse(
        localStorage.getItem("spartan_last_submitted_batch") ?? "null",
      ),
    ).toMatchObject({
      batchId: "batch-submit-1",
      kind: "scrape",
      submittedUrls: ["https://example.com", "https://example.org"],
    });

    act(() => {
      result.current.clearLastSubmittedBatch();
    });

    expect(result.current.lastSubmittedBatch).toBeNull();
    expect(localStorage.getItem("spartan_last_submitted_batch")).toBeNull();
  });
});
