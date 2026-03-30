/**
 * Purpose: Verify the saved-results route hook derives selection metadata during render instead of mirroring it with effects.
 * Responsibilities: Assert result loading resets route state, selected-item metadata follows the active row, and out-of-range selections clamp to the loaded page.
 * Scope: `useResultsState` only.
 * Usage: Run with Vitest.
 * Invariants/Assumptions: The results loader is mocked, selection updates are synchronous, and research rows are the only ones that expose summary metadata.
 */

import { renderHook, act } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { loadResults as loadResultsUtil } from "../lib/results";
import type { ResultItem } from "../types";
import { useResultsState } from "./useResultsState";

vi.mock("../lib/results", () => ({
  loadResults: vi.fn(),
}));

describe("useResultsState", () => {
  it("derives active selection metadata and clamps out-of-range indexes", async () => {
    const results: ResultItem[] = [
      {
        query: "cats",
        summary: "Research summary",
        confidence: 0.91,
        evidence: [{ url: "https://example.com/source", title: "Source" }],
        clusters: [{ id: "cluster-1", name: "Cluster 1" }],
        citations: [{ url: "https://example.com/cite", title: "Citation" }],
      } as never,
      {
        url: "https://example.com/article",
        status: 200,
        title: "Article",
        text: "Body copy",
        links: [],
      } as never,
    ];

    vi.mocked(loadResultsUtil).mockResolvedValue({
      data: results,
      raw: JSON.stringify(results),
      totalCount: results.length,
    } as never);

    const { result } = renderHook(() => useResultsState());

    await act(async () => {
      await result.current.loadResults("job-1");
    });

    expect(result.current.selectedJobId).toBe("job-1");
    expect(result.current.selectedResultIndex).toBe(0);
    expect(result.current.resultSummary).toBe("Research summary");
    expect(result.current.resultConfidence).toBe(0.91);
    expect(result.current.resultEvidence).toHaveLength(1);
    expect(result.current.resultClusters).toHaveLength(1);
    expect(result.current.resultCitations).toHaveLength(1);

    act(() => {
      result.current.setSelectedResultIndex(99);
    });

    expect(result.current.selectedResultIndex).toBe(1);
    expect(result.current.resultSummary).toBeNull();
    expect(result.current.resultConfidence).toBeNull();
    expect(result.current.resultEvidence).toEqual([]);
    expect(result.current.resultClusters).toEqual([]);
    expect(result.current.resultCitations).toEqual([]);
  });
});
