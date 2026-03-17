/**
 * Purpose: Verify the results explorer helper logic stays aligned with the dominant-reader cutover.
 * Responsibilities: Cover filtering, tree helpers, comparable job lookup, secondary tool availability, and guided export recommendations.
 * Scope: Pure helper tests only.
 * Usage: Run with Vitest.
 * Invariants/Assumptions: Guided export recommendations remain deterministic for the same result input and visualization stays hidden for non-research jobs without evidence.
 */

import { describe, expect, it } from "vitest";

import type { TreeNode } from "../../lib/tree-utils";
import type { Job, ResultItem } from "../../types";

import {
  buildDefaultExpandedTreeIds,
  collectTreeNodeIds,
  filterResultItems,
  findComparableJobs,
  getAvailableSecondaryTools,
  getExportGuidanceOptions,
  getJobByID,
  hasResearchVisualization,
} from "./resultsExplorerUtils";

const resultItems: ResultItem[] = [
  {
    url: "https://example.com/articles/one",
    status: 200,
    title: "Article One",
    text: "Alpha beta",
    links: [],
    normalized: { price: 10 },
  },
  {
    url: "https://example.com/missing",
    status: 404,
    title: "Missing",
    text: "Gone",
    links: [],
  },
  {
    summary: "Research summary about alpha signals",
  },
];

const jobs: Job[] = [
  {
    id: "job-1",
    status: "succeeded",
    kind: "crawl",
    createdAt: "2026-03-10T00:00:00Z",
    updatedAt: "2026-03-10T00:01:00Z",
    specVersion: 1,
    spec: { version: 1 },
    run: { waitMs: 0, runMs: 1000, totalMs: 1000 },
  },
  {
    id: "job-2",
    status: "failed",
    kind: "crawl",
    createdAt: "2026-03-10T00:02:00Z",
    updatedAt: "2026-03-10T00:03:00Z",
    specVersion: 1,
    spec: { version: 1 },
    run: { waitMs: 500, runMs: 250, totalMs: 750 },
  },
  {
    id: "job-3",
    status: "succeeded",
    kind: "research",
    createdAt: "2026-03-10T00:04:00Z",
    updatedAt: "2026-03-10T00:05:00Z",
    specVersion: 1,
    spec: { version: 1 },
    run: { waitMs: 200, runMs: 400, totalMs: 600 },
  },
];

const treeNodes: TreeNode[] = [
  {
    id: "domain:example.com",
    url: "https://example.com",
    title: "example.com",
    status: 0,
    depth: 0,
    resultCount: 2,
    children: [
      {
        id: "dir:example.com:articles",
        url: "https://example.com/articles",
        title: "articles",
        status: 0,
        depth: 1,
        resultCount: 1,
        children: [
          {
            id: "page:https://example.com/articles/one",
            url: "https://example.com/articles/one",
            title: "Article One",
            status: 200,
            depth: 2,
            resultCount: 1,
            children: [],
          },
        ],
      },
    ],
  },
];

describe("resultsExplorerUtils", () => {
  it("filters crawl and research results by query and status", () => {
    expect(filterResultItems(resultItems, "alpha", "all")).toHaveLength(2);
    expect(filterResultItems(resultItems, "", "success")).toHaveLength(2);
    expect(filterResultItems(resultItems, "", "error")).toHaveLength(2);
  });

  it("builds default and full tree expansion sets", () => {
    expect([...buildDefaultExpandedTreeIds(treeNodes)]).toEqual([
      "domain:example.com",
    ]);
    expect([...collectTreeNodeIds(treeNodes)]).toEqual([
      "domain:example.com",
      "dir:example.com:articles",
      "page:https://example.com/articles/one",
    ]);
  });

  it("returns comparable jobs and exact job lookups", () => {
    expect(findComparableJobs(jobs, "job-1", "crawl")).toEqual([jobs[1]]);
    expect(getJobByID(jobs, "job-2")).toEqual(jobs[1]);
    expect(getJobByID(jobs, "missing")).toBeNull();
  });

  it("detects research visualization availability", () => {
    expect(hasResearchVisualization("research", [])).toBe(true);
    expect(
      hasResearchVisualization("crawl", [
        {
          url: "https://example.com/evidence",
          title: "Evidence",
          snippet: "snippet",
          score: 0.8,
        },
      ]),
    ).toBe(true);
    expect(hasResearchVisualization("crawl", [])).toBe(false);
  });

  it("hides visualize from secondary tools for non-research jobs", () => {
    expect(getAvailableSecondaryTools(false).map((tool) => tool.id)).toEqual([
      "tree",
      "diff",
      "transform",
    ]);

    expect(getAvailableSecondaryTools(true).map((tool) => tool.id)).toContain(
      "visualize",
    );
  });

  it("builds guided export options with scope notes", () => {
    const options = getExportGuidanceOptions({
      totalResults: 12,
      visibleResults: 2,
      searchQuery: "alpha",
      statusFilter: "success",
      resultItems,
      evidence: [],
      isResearchJob: false,
    });

    expect(options.find((option) => option.format === "jsonl")).toMatchObject({
      readiness: "recommended",
    });

    expect(options.find((option) => option.format === "csv")).toMatchObject({
      readiness: "recommended",
    });

    expect(options[0]?.scopeLabel).toContain("12 results");
    expect(options[0]?.scopeNote).toMatch(/on-screen reader/i);
  });
});
