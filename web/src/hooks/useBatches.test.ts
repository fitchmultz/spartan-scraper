/**
 * Purpose: Verify batch hook mapping and normalization logic that feeds BatchList rendering.
 * Responsibilities: Assert stable stats normalization for API/localStorage inputs and guard against undefined stats regressions.
 * Scope: Pure helper functions exported by useBatches (no network/UI rendering).
 * Usage: Run via `pnpm run test` or `make test-ci`.
 * Invariants/Assumptions: Batch entries consumed by UI must always include non-negative numeric stats.
 */

import { describe, expect, it } from "vitest";
import type { BatchResponse, BatchStatusResponse, Job } from "../api";
import {
  createEmptyBatchStats,
  deriveBatchStatsFromJobs,
  mapBatchCreateResponse,
  mapBatchStatusResponse,
  normalizeBatchStats,
  normalizeStoredBatchEntries,
} from "./useBatches";

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

  it("mapBatchCreateResponse always returns safe stats for BatchList", () => {
    const response: BatchResponse = {
      id: "batch-create-1",
      kind: "scrape",
      status: "pending",
      jobCount: 2,
      jobs: [makeJob("1", "queued"), makeJob("2", "queued")],
      createdAt: "2026-03-05T10:00:00.000Z",
    };

    expect(mapBatchCreateResponse(response)).toEqual({
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

  it("mapBatchStatusResponse normalizes API stats before UI consumption", () => {
    const response: BatchStatusResponse = {
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
    };

    expect(mapBatchStatusResponse(response)).toEqual({
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

  it("mapBatchCreateResponse falls back to jobCount when jobs are omitted", () => {
    expect(
      mapBatchCreateResponse({
        id: "batch-create-2",
        kind: "research",
        status: "pending",
        jobCount: 3,
        jobs: [],
        createdAt: "2026-03-05T12:00:00.000Z",
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
