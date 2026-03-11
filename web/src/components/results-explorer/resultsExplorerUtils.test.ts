/**
 * resultsExplorerUtils.test
 *
 * Purpose:
 * - Verify the reduced 1.0 results explorer helpers stay aligned with the
 *   supported view modes and export formats.
 *
 * Responsibilities:
 * - Cover filtering, job lookup, tree expansion, and export metadata.
 *
 * Scope:
 * - Pure helper tests only.
 *
 * Usage:
 * - Run via Vitest.
 *
 * Invariants/Assumptions:
 * - The 1.0 explorer supports only json, jsonl, csv, md, and xlsx exports.
 */

import { describe, expect, it } from "vitest";

import type { TreeNode } from "../../lib/tree-utils";
import type { Job, ResultItem } from "../../types";

import {
  buildDefaultExpandedTreeIds,
  buildExportFilename,
  collectTreeNodeIds,
  filterResultItems,
  findComparableJobs,
  getExportMimeType,
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
  },
  {
    id: "job-2",
    status: "failed",
    kind: "crawl",
    createdAt: "2026-03-10T00:02:00Z",
    updatedAt: "2026-03-10T00:03:00Z",
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
    expect(findComparableJobs(jobs, "job-1")).toEqual([jobs[1]]);
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

  it("normalizes export metadata for supported formats", () => {
    expect(getExportMimeType("jsonl")).toBe("application/x-ndjson");
    expect(getExportMimeType("xlsx")).toBe(
      "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
    );
    expect(
      buildExportFilename("job-1", "json", "2026-03-10T10-00-00Z", "filtered"),
    ).toBe("results-job-1-filtered-2026-03-10T10-00-00Z.json");
  });
});
