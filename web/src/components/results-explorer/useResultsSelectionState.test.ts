/**
 * Purpose: Verify the results explorer derives visible selection and tree expansion from the current render instead of syncing it through effects.
 * Responsibilities: Assert filtered reader state keeps the source selection intact, visible indexes follow the current filter, and tree expansion resolves against the active tree.
 * Scope: `useResultsSelectionState` only.
 * Usage: Run with Vitest.
 * Invariants/Assumptions: Result-item arrays are immutable snapshots from the route layer, and the hook is the sole owner of reader-local filter state.
 */

import { renderHook, act } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { ResultItem } from "../../types";
import { useResultsSelectionState } from "./useResultsSelectionState";

function buildCrawlResult(url: string, title: string): ResultItem {
  return {
    url,
    status: 200,
    title,
    text: title,
    links: [],
  };
}

describe("useResultsSelectionState", () => {
  it("keeps the raw selection intact while deriving the first visible filtered result", () => {
    const setSelectedResultIndex = vi.fn();
    const resultItems = [
      buildCrawlResult("https://alpha.example.com/a", "Alpha article"),
      buildCrawlResult("https://beta.example.com/b", "Beta article"),
    ];

    const { result } = renderHook(() =>
      useResultsSelectionState({
        resultItems,
        selectedResultIndex: 1,
        setSelectedResultIndex,
        resultEvidence: [],
        jobType: "crawl",
        totalResults: resultItems.length,
      }),
    );

    expect(result.current.activeResultIndex).toBe(1);
    expect(result.current.visibleSelectedIndex).toBe(1);

    act(() => {
      result.current.setSearchQuery("alpha");
    });

    expect(result.current.filteredResultItems).toHaveLength(1);
    expect(result.current.activeResultIndex).toBe(0);
    expect(result.current.visibleSelectedIndex).toBe(0);
    expect(setSelectedResultIndex).not.toHaveBeenCalled();

    act(() => {
      result.current.setSearchQuery("");
    });

    expect(result.current.activeResultIndex).toBe(1);
    expect(result.current.visibleSelectedIndex).toBe(1);
  });

  it("resolves tree expansion against the current tree on rerender", () => {
    const setSelectedResultIndex = vi.fn();
    const initialItems = [
      buildCrawlResult("https://alpha.example.com/a", "Alpha article"),
    ];
    const nextItems = [
      buildCrawlResult("https://beta.example.org/b", "Beta article"),
    ];

    const { result, rerender } = renderHook(
      ({ items }) =>
        useResultsSelectionState({
          resultItems: items,
          selectedResultIndex: 0,
          setSelectedResultIndex,
          resultEvidence: [],
          jobType: "crawl",
          totalResults: items.length,
        }),
      {
        initialProps: {
          items: initialItems,
        },
      },
    );

    act(() => {
      result.current.expandAllTreeNodes();
    });

    const initialExpandedIds = Array.from(result.current.treeExpandedIds);
    expect(initialExpandedIds.length).toBeGreaterThan(0);

    rerender({ items: nextItems });

    const nextExpandedIds = Array.from(result.current.treeExpandedIds);
    expect(nextExpandedIds.length).toBeGreaterThan(0);
    expect(nextExpandedIds).not.toEqual(initialExpandedIds);
  });
});
