/**
 * Purpose: Verify the results explorer only runs diff comparisons from explicit actions and ignores stale async responses.
 * Responsibilities: Assert `runDiff()` loads the selected jobs on demand, keeps newer comparisons authoritative, and discards older in-flight results.
 * Scope: `useResultsOperationsState` only.
 * Usage: Run with Vitest.
 * Invariants/Assumptions: AI assistant access is mocked, load results are asynchronous and can resolve out of order, and crawl diff comparisons use saved result snapshots.
 */

import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { Job, ResultItem } from "../../types";
import type { ResultsResponse } from "../../lib/results";
import type { CrawlDiffResult } from "../../lib/diff-utils";
import { useResultsOperationsState } from "./useResultsOperationsState";

const hoisted = vi.hoisted(() => ({
  loadResults: vi.fn(),
  assistant: {
    open: vi.fn(),
  },
}));

vi.mock("../../lib/results", async () => {
  const actual =
    await vi.importActual<typeof import("../../lib/results")>(
      "../../lib/results",
    );
  return {
    ...actual,
    loadResults: hoisted.loadResults,
  };
});

vi.mock("../ai-assistant", () => ({
  useAIAssistant: () => hoisted.assistant,
}));

function buildCrawlResult(url: string): ResultItem {
  return {
    url,
    status: 200,
    title: url,
    text: url,
    links: [],
  };
}

function buildLoadResponse(items: ResultItem[]): ResultsResponse {
  return {
    data: items,
    raw: JSON.stringify(items),
  };
}

function createDeferred<T>() {
  let resolve!: (value: T) => void;

  const promise = new Promise<T>((promiseResolve) => {
    resolve = promiseResolve;
  });

  return { promise, resolve };
}

function renderOperationsHook() {
  const jobs: Job[] = [
    {
      id: "job-1",
      status: "succeeded",
      kind: "crawl",
      createdAt: "2026-03-18T00:00:00.000Z",
      updatedAt: "2026-03-18T00:00:00.000Z",
      specVersion: 1,
      spec: {},
      run: { waitMs: 0, runMs: 1, totalMs: 1 },
    },
    {
      id: "job-2",
      status: "succeeded",
      kind: "crawl",
      createdAt: "2026-03-18T00:01:00.000Z",
      updatedAt: "2026-03-18T00:01:00.000Z",
      specVersion: 1,
      spec: {},
      run: { waitMs: 0, runMs: 1, totalMs: 1 },
    },
    {
      id: "job-3",
      status: "succeeded",
      kind: "crawl",
      createdAt: "2026-03-18T00:02:00.000Z",
      updatedAt: "2026-03-18T00:02:00.000Z",
      specVersion: 1,
      spec: {},
      run: { waitMs: 0, runMs: 1, totalMs: 1 },
    },
  ];

  return renderHook(() =>
    useResultsOperationsState({
      jobId: "job-1",
      resultItems: [],
      selectedResultIndex: 0,
      resultSummary: null,
      resultEvidence: [],
      currentJob: jobs[0],
      availableJobs: jobs,
      jobType: "crawl",
      resultFormat: "jsonl",
      totalResults: 1,
      filteredResultItems: [],
      searchQuery: "",
      statusFilter: "all",
      isResearchJob: false,
    }),
  );
}

describe("useResultsOperationsState", () => {
  beforeEach(() => {
    hoisted.loadResults.mockReset();
    hoisted.assistant.open.mockReset();
  });

  it("ignores stale diff responses after a newer comparison starts", async () => {
    const pending: Array<{
      jobId: string;
      resolve: (value: ResultsResponse) => void;
    }> = [];

    hoisted.loadResults.mockImplementation((jobId: string) => {
      const deferred = createDeferred<ResultsResponse>();
      pending.push({ jobId, resolve: deferred.resolve });
      return deferred.promise;
    });

    const { result } = renderOperationsHook();

    act(() => {
      void result.current.runDiff("job-2");
      void result.current.runDiff("job-3");
    });

    expect(pending).toHaveLength(4);
    expect(pending.map((entry) => entry.jobId)).toEqual([
      "job-1",
      "job-2",
      "job-1",
      "job-3",
    ]);

    const baseResults = [buildCrawlResult("https://example.com/base")];
    const compareTwoResults = [buildCrawlResult("https://example.com/two")];

    await act(async () => {
      pending[2].resolve(buildLoadResponse(baseResults));
      pending[3].resolve(buildLoadResponse(compareTwoResults));
      await Promise.resolve();
    });

    await waitFor(() => {
      expect(result.current.diffLoading).toBe(false);
    });

    expect(result.current.diffResult).toBeTruthy();
    expect((result.current.diffResult as CrawlDiffResult).added[0].url).toBe(
      "https://example.com/two",
    );

    const compareOneResults = [buildCrawlResult("https://example.com/one")];

    await act(async () => {
      pending[0].resolve(buildLoadResponse(baseResults));
      pending[1].resolve(buildLoadResponse(compareOneResults));
      await Promise.resolve();
    });

    expect((result.current.diffResult as CrawlDiffResult).added[0].url).toBe(
      "https://example.com/two",
    );
    expect(result.current.diffLoading).toBe(false);
  });
});
