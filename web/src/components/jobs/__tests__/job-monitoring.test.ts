/**
 * Purpose: Verify jobs dashboard view-model derivation and persisted jobs-route state helpers.
 * Responsibilities: Assert lane grouping, failure prioritization, and localStorage save/restore behavior.
 * Scope: Unit coverage for `web/src/lib/job-monitoring.ts`.
 * Usage: Run with Vitest as part of the web test suite.
 * Invariants/Assumptions: Test jobs mimic the generated API contract closely enough for dashboard derivation.
 */

import { beforeEach, describe, expect, it } from "vitest";
import {
  buildJobMonitoringDashboardModel,
  clearJobsViewState,
  loadJobsViewState,
  saveJobsViewState,
} from "../../../lib/job-monitoring";
import type { JobEntry } from "../../../types";

function createJob(
  overrides: Partial<JobEntry> & Pick<JobEntry, "id" | "status">,
): JobEntry {
  return {
    kind: "scrape",
    createdAt: "2026-03-16T12:00:00.000Z",
    updatedAt: "2026-03-16T12:05:00.000Z",
    specVersion: 1,
    spec: {},
    ...overrides,
    run: {
      waitMs: 400,
      runMs: 1200,
      totalMs: 1600,
      ...overrides.run,
    },
  };
}

describe("job-monitoring view model", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("groups failed and blocked work into the attention lane", () => {
    const failed = createJob({
      id: "failed-job",
      status: "failed",
      run: {
        waitMs: 200,
        runMs: 500,
        totalMs: 700,
        failure: {
          category: "Timeout",
          summary: "The remote host timed out.",
          retryable: true,
          terminal: false,
        },
      },
    });

    const blocked = createJob({
      id: "blocked-job",
      status: "queued",
      dependencyStatus: "pending",
    });

    const running = createJob({
      id: "running-job",
      status: "running",
    });

    const completed = createJob({
      id: "done-job",
      status: "succeeded",
    });

    const model = buildJobMonitoringDashboardModel({
      jobs: [blocked, running, completed],
      failedJobs: [failed],
      totalJobs: 4,
      connectionState: "connected",
      now: Date.parse("2026-03-16T12:10:00.000Z"),
    });

    expect(model.lanes.attention.map((job) => job.id)).toEqual([
      "failed-job",
      "blocked-job",
    ]);
    expect(model.lanes.progress.map((job) => job.id)).toEqual(["running-job"]);
    expect(model.lanes.completed.map((job) => job.id)).toEqual(["done-job"]);
  });

  it("prefers manager health counts for queued and running summary cards", () => {
    const queued = createJob({
      id: "queued-job",
      status: "queued",
    });

    const model = buildJobMonitoringDashboardModel({
      jobs: [queued],
      failedJobs: [],
      totalJobs: 1,
      connectionState: "polling",
      managerStatus: {
        queued: 7,
        active: 3,
      },
    });

    expect(model.summary.queued).toBe(7);
    expect(model.summary.running).toBe(3);
  });

  it("persists and restores jobs view state", () => {
    saveJobsViewState({
      statusFilter: "failed",
      currentPage: 3,
      scrollY: 280,
    });

    expect(loadJobsViewState()).toEqual({
      statusFilter: "failed",
      currentPage: 3,
      scrollY: 280,
    });

    clearJobsViewState();
    expect(loadJobsViewState()).toBeNull();
  });
});
