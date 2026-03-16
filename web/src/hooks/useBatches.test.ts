/**
 * Purpose: Verify batch hook mapping and authoritative API hydration for the Web batch surface.
 * Responsibilities: Assert stable helper normalization, last-submitted persistence, page loading, and detail hydration.
 * Scope: Hook/helper behavior only; component rendering and styling stay under separate tests.
 * Usage: Run via `pnpm run test` or `make test-ci`.
 * Invariants/Assumptions: Batch entries consumed by the UI must always include non-negative numeric stats and come from the API list endpoint.
 */

import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import type { BatchListResponse, BatchResponse, Job } from "../api";
import {
  getV1JobsBatch,
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
  useBatches,
} from "./useBatches";

vi.mock("../api", () => ({
  getV1JobsBatch: vi.fn(),
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

function makeBatchListResponse(
  overrides: Partial<BatchListResponse> = {},
): BatchListResponse {
  return {
    batches: [],
    total: 0,
    limit: 25,
    offset: 0,
    ...overrides,
  };
}

function makeBatchResponse(
  overrides: Partial<BatchResponse> = {},
): BatchResponse {
  return {
    batch: {
      id: "batch-1",
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
    ...overrides,
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
  vi.mocked(getV1JobsBatch).mockResolvedValue({
    data: makeBatchListResponse(),
    request: new Request("http://127.0.0.1:8741/v1/jobs/batch"),
    response: new Response(null, { status: 200 }),
  } as never);
  vi.mocked(getV1JobsBatchById).mockResolvedValue({
    data: makeBatchResponse(),
    request: new Request("http://127.0.0.1:8741/v1/jobs/batch/batch-1"),
    response: new Response(null, { status: 200 }),
  } as never);
  vi.mocked(postV1JobsBatchScrape).mockResolvedValue({} as never);
  vi.mocked(postV1JobsBatchCrawl).mockResolvedValue({} as never);
  vi.mocked(postV1JobsBatchResearch).mockResolvedValue({} as never);
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

  it("mapBatchResponse always returns safe stats for BatchList", () => {
    expect(mapBatchResponse(makeBatchResponse())).toEqual({
      id: "batch-1",
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
});

describe("useBatches authoritative loading", () => {
  it("restores the last submitted batch from localStorage", async () => {
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

  it("hydrates the batch page from the API list endpoint", async () => {
    vi.mocked(getV1JobsBatch).mockResolvedValue({
      data: makeBatchListResponse({
        batches: [
          {
            id: "batch-list-1",
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
        ],
        total: 1,
        limit: 25,
        offset: 0,
      }),
      request: new Request(
        "http://127.0.0.1:8741/v1/jobs/batch?limit=25&offset=0",
      ),
      response: new Response(null, { status: 200 }),
    } as never);

    const { result } = renderHook(() => useBatches());

    await waitFor(() => {
      expect(result.current.batches).toEqual([
        {
          id: "batch-list-1",
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
      ]);
      expect(result.current.total).toBe(1);
      expect(result.current.limit).toBe(25);
      expect(result.current.offset).toBe(0);
    });
  });

  it("persists the submitted batch summary and refreshes page zero", async () => {
    vi.mocked(postV1JobsBatchScrape).mockResolvedValue({
      data: makeBatchResponse({
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
      }),
      request: new Request("http://127.0.0.1:8741/v1/jobs/batch/scrape"),
      response: new Response(null, { status: 201 }),
    } as never);
    vi.mocked(getV1JobsBatch).mockResolvedValue({
      data: makeBatchListResponse({
        batches: [
          {
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
        ],
        total: 1,
        limit: 25,
        offset: 0,
      }),
      request: new Request(
        "http://127.0.0.1:8741/v1/jobs/batch?limit=25&offset=0",
      ),
      response: new Response(null, { status: 200 }),
    } as never);

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
    expect(result.current.batches[0]).toMatchObject({
      id: "batch-submit-1",
      kind: "scrape",
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
  });

  it("loads batch job details on demand", async () => {
    vi.mocked(getV1JobsBatch).mockResolvedValue({
      data: makeBatchListResponse({
        batches: [
          {
            id: "batch-detail-1",
            kind: "scrape",
            status: "processing",
            jobCount: 2,
            stats: {
              queued: 1,
              running: 1,
              succeeded: 0,
              failed: 0,
              canceled: 0,
            },
            createdAt: "2026-03-05T10:00:00.000Z",
            updatedAt: "2026-03-05T10:01:00.000Z",
          },
        ],
        total: 1,
        limit: 25,
        offset: 0,
      }),
      request: new Request(
        "http://127.0.0.1:8741/v1/jobs/batch?limit=25&offset=0",
      ),
      response: new Response(null, { status: 200 }),
    } as never);
    vi.mocked(getV1JobsBatchById).mockResolvedValue({
      data: makeBatchResponse({
        batch: {
          id: "batch-detail-1",
          kind: "scrape",
          status: "processing",
          jobCount: 2,
          stats: {
            queued: 1,
            running: 1,
            succeeded: 0,
            failed: 0,
            canceled: 0,
          },
          createdAt: "2026-03-05T10:00:00.000Z",
          updatedAt: "2026-03-05T10:01:00.000Z",
        },
        jobs: [makeJob("job-1", "queued"), makeJob("job-2", "running")],
        total: 2,
        limit: 2,
        offset: 0,
      }),
      request: new Request(
        "http://127.0.0.1:8741/v1/jobs/batch/batch-detail-1",
      ),
      response: new Response(null, { status: 200 }),
    } as never);

    const { result } = renderHook(() => useBatches());

    await waitFor(() => {
      expect(result.current.batches).toHaveLength(1);
    });

    await act(async () => {
      await result.current.refreshBatch("batch-detail-1");
    });

    expect(result.current.batchJobs.get("batch-detail-1")).toEqual([
      makeJob("job-1", "queued"),
      makeJob("job-2", "running"),
    ]);
  });
});
