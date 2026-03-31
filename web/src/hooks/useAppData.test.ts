/**
 * Purpose: Verify the app-shell data hook keeps operator orchestration resilient across setup mode and live job updates.
 * Responsibilities: Mock the transport helpers used by `useAppData`, prove setup-required health short-circuits the heavier loaders, and confirm job WebSocket events refresh the active detail context.
 * Scope: `useAppData` orchestration only.
 * Usage: Run with Vitest.
 * Invariants/Assumptions: Hook dependencies are mocked, setup mode should suppress the normal background fetch fanout, and live job events should refresh any open job detail.
 */

import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useAppData } from "./useAppData";

const hoisted = vi.hoisted(() => ({
  loadHealth: vi.fn(),
  loadJobs: vi.fn(),
  loadJobFailures: vi.fn(),
  loadMetrics: vi.fn(),
  loadProfiles: vi.fn(),
  loadSchedules: vi.fn(),
  loadTemplates: vi.fn(),
  loadCrawlStates: vi.fn(),
  loadJobDetail: vi.fn(),
  getWebSocketUrl: vi.fn(() => "ws://localhost:8741/v1/ws"),
  useWebSocket: vi.fn(),
  wsOptions: null as Record<string, unknown> | null,
  wsState: "disconnected",
}));

vi.mock("./app-data/api", () => ({
  POLL_INTERVAL: 4000,
  getWebSocketUrl: hoisted.getWebSocketUrl,
  loadHealth: hoisted.loadHealth,
  loadJobs: hoisted.loadJobs,
  loadJobFailures: hoisted.loadJobFailures,
  loadMetrics: hoisted.loadMetrics,
  loadProfiles: hoisted.loadProfiles,
  loadSchedules: hoisted.loadSchedules,
  loadTemplates: hoisted.loadTemplates,
  loadCrawlStates: hoisted.loadCrawlStates,
  loadJobDetail: hoisted.loadJobDetail,
}));

vi.mock("./useWebSocket", () => ({
  useWebSocket: hoisted.useWebSocket,
}));

function buildJob(id: string) {
  return {
    id,
    status: "succeeded",
    kind: "scrape",
    createdAt: "2026-03-31T00:00:00.000Z",
    updatedAt: "2026-03-31T00:00:00.000Z",
    specVersion: 1,
    spec: {},
    run: { waitMs: 0, runMs: 1, totalMs: 1 },
  };
}

describe("useAppData", () => {
  beforeEach(() => {
    hoisted.wsState = "disconnected";
    hoisted.wsOptions = null;

    hoisted.loadHealth.mockReset();
    hoisted.loadJobs.mockReset();
    hoisted.loadJobFailures.mockReset();
    hoisted.loadMetrics.mockReset();
    hoisted.loadProfiles.mockReset();
    hoisted.loadSchedules.mockReset();
    hoisted.loadTemplates.mockReset();
    hoisted.loadCrawlStates.mockReset();
    hoisted.loadJobDetail.mockReset();
    hoisted.getWebSocketUrl.mockClear();
    hoisted.useWebSocket.mockReset();

    hoisted.loadHealth.mockResolvedValue({
      health: {
        setup: { required: false },
      },
      managerStatus: { queued: 2, active: 1 },
    });
    hoisted.loadJobs.mockResolvedValue({ jobs: [buildJob("job-1")], total: 1 });
    hoisted.loadJobFailures.mockResolvedValue([]);
    hoisted.loadMetrics.mockResolvedValue(null);
    hoisted.loadProfiles.mockResolvedValue([]);
    hoisted.loadSchedules.mockResolvedValue([]);
    hoisted.loadTemplates.mockResolvedValue([]);
    hoisted.loadCrawlStates.mockResolvedValue({ crawlStates: [], total: 0 });
    hoisted.loadJobDetail.mockResolvedValue(buildJob("job-1"));
    hoisted.useWebSocket.mockImplementation(
      (options: Record<string, unknown>) => {
        hoisted.wsOptions = options;
        return { state: hoisted.wsState };
      },
    );
  });

  it("skips the normal data fanout when startup requires setup mode", async () => {
    hoisted.loadHealth.mockResolvedValue({
      health: {
        setup: { required: true },
      },
      managerStatus: null,
    });

    const { result } = renderHook(() => useAppData());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(result.current.setupRequired).toBe(true);
    expect(result.current.jobs).toEqual([]);
    expect(result.current.failedJobs).toEqual([]);
    expect(result.current.metrics).toBeNull();
    expect(hoisted.loadJobs).not.toHaveBeenCalled();
    expect(hoisted.loadJobFailures).not.toHaveBeenCalled();
    expect(hoisted.loadMetrics).not.toHaveBeenCalled();
    expect(hoisted.loadProfiles).not.toHaveBeenCalled();
    expect(hoisted.loadSchedules).not.toHaveBeenCalled();
    expect(hoisted.loadTemplates).not.toHaveBeenCalled();
    expect(hoisted.loadCrawlStates).not.toHaveBeenCalled();
    expect((hoisted.wsOptions as { enabled?: boolean } | null)?.enabled).toBe(
      false,
    );
  });

  it("refreshes jobs, failures, health, and the active detail job on live job events", async () => {
    hoisted.wsState = "connected";
    const detailJob = buildJob("job-1");
    hoisted.loadJobDetail.mockResolvedValue(detailJob);

    const { result } = renderHook(() => useAppData());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    await act(async () => {
      await result.current.refreshJobDetail(detailJob.id);
    });

    expect(result.current.detailJob?.id).toBe(detailJob.id);
    expect(result.current.connectionState).toBe("connected");

    act(() => {
      const options = hoisted.wsOptions as {
        onMessage?: (message: {
          type: string;
          timestamp: number;
          payload: unknown;
        }) => void;
      };
      options.onMessage?.({
        type: "job_completed",
        timestamp: Date.now(),
        payload: null,
      });
    });

    await waitFor(() => {
      expect(hoisted.loadJobs).toHaveBeenCalledTimes(2);
      expect(hoisted.loadJobFailures).toHaveBeenCalledTimes(2);
      expect(hoisted.loadHealth).toHaveBeenCalledTimes(2);
      expect(hoisted.loadJobDetail).toHaveBeenCalledTimes(2);
    });
  });
});
