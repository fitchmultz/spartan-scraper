/**
 * Purpose: Verify lane rendering, loading states, and saved-state restoration for the jobs monitoring dashboard.
 * Responsibilities: Assert lane composition, skeleton behavior, and jobs-route restore flow for filters/page/scroll.
 * Scope: Component coverage for `JobMonitoringDashboard`.
 * Usage: Run with Vitest as part of the web test suite.
 * Invariants/Assumptions: The dashboard receives already-fetched job data from the application shell and restores view state exactly once.
 */

import { render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { JOBS_VIEW_STATE_KEY } from "../../../lib/job-monitoring";
import type { JobEntry } from "../../../types";
import { JobMonitoringDashboard } from "../JobMonitoringDashboard";

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

describe("JobMonitoringDashboard", () => {
  const scrollToMock = vi.fn();

  beforeEach(() => {
    localStorage.clear();
    vi.stubGlobal("requestAnimationFrame", (callback: FrameRequestCallback) => {
      callback(0);
      return 1;
    });
    vi.stubGlobal("cancelAnimationFrame", vi.fn());
    Object.defineProperty(window, "scrollTo", {
      value: scrollToMock,
      writable: true,
    });
    scrollToMock.mockReset();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders jobs into the correct lanes", () => {
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

    render(
      <JobMonitoringDashboard
        jobs={[blocked, running, completed]}
        failedJobs={[failed]}
        error={null}
        loading={false}
        statusFilter=""
        onStatusFilterChange={vi.fn()}
        onViewResults={vi.fn()}
        onCancel={vi.fn()}
        onDelete={vi.fn()}
        onRefresh={vi.fn()}
        currentPage={1}
        totalJobs={4}
        jobsPerPage={100}
        onPageChange={vi.fn()}
        connectionState="connected"
      />,
    );

    const attentionLane = screen.getByRole("region", {
      name: /needs attention/i,
    });
    const progressLane = screen.getByRole("region", {
      name: /in progress/i,
    });
    const completedLane = screen.getByRole("region", {
      name: /recent completed/i,
    });

    expect(
      within(attentionLane).getByText(/the remote host timed out/i),
    ).toBeInTheDocument();
    expect(
      within(attentionLane).getByText(/waiting for dependencies/i),
    ).toBeInTheDocument();
    expect(
      within(progressLane).getByText(/running scrape/i),
    ).toBeInTheDocument();
    expect(
      within(completedLane).getByRole("button", { name: /view results/i }),
    ).toBeInTheDocument();
  });

  it("shows skeletons while loading", () => {
    render(
      <JobMonitoringDashboard
        jobs={[]}
        failedJobs={[]}
        error={null}
        loading={true}
        statusFilter=""
        onStatusFilterChange={vi.fn()}
        onViewResults={vi.fn()}
        onCancel={vi.fn()}
        onDelete={vi.fn()}
        onRefresh={vi.fn()}
        currentPage={1}
        totalJobs={0}
        jobsPerPage={100}
        onPageChange={vi.fn()}
        connectionState="polling"
      />,
    );

    expect(screen.getAllByTestId("job-card-skeleton").length).toBeGreaterThan(
      0,
    );
  });

  it("restores saved filter, page, and scroll position", async () => {
    localStorage.setItem(
      JOBS_VIEW_STATE_KEY,
      JSON.stringify({
        statusFilter: "failed",
        currentPage: 3,
        scrollY: 280,
      }),
    );

    const onStatusFilterChange = vi.fn();
    const onPageChange = vi.fn();

    const { rerender } = render(
      <JobMonitoringDashboard
        jobs={[]}
        failedJobs={[]}
        error={null}
        loading={false}
        statusFilter=""
        onStatusFilterChange={onStatusFilterChange}
        onViewResults={vi.fn()}
        onCancel={vi.fn()}
        onDelete={vi.fn()}
        onRefresh={vi.fn()}
        currentPage={1}
        totalJobs={300}
        jobsPerPage={100}
        onPageChange={onPageChange}
        connectionState="connected"
      />,
    );

    expect(onStatusFilterChange).toHaveBeenCalledWith("failed");
    expect(onPageChange).toHaveBeenCalledWith(3);

    rerender(
      <JobMonitoringDashboard
        jobs={[]}
        failedJobs={[]}
        error={null}
        loading={false}
        statusFilter="failed"
        onStatusFilterChange={onStatusFilterChange}
        onViewResults={vi.fn()}
        onCancel={vi.fn()}
        onDelete={vi.fn()}
        onRefresh={vi.fn()}
        currentPage={3}
        totalJobs={300}
        jobsPerPage={100}
        onPageChange={onPageChange}
        connectionState="connected"
      />,
    );

    await waitFor(() => {
      expect(scrollToMock).toHaveBeenCalledWith({
        top: 280,
        behavior: "auto",
      });
    });

    expect(localStorage.getItem(JOBS_VIEW_STATE_KEY)).toBeNull();
  });
});
