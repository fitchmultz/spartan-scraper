/**
 * resultsExplorerUtils.test
 *
 * Purpose:
 * - Verify pure results-explorer helper behavior stays stable across refactors.
 *
 * Responsibilities:
 * - Cover filtering, tree expansion defaults, and export metadata.
 * - Confirm job comparison and research-visualization helpers.
 * - Lock in traffic extraction behavior for crawl results with intercepted data.
 *
 * Scope:
 * - Unit tests for results-explorer helper logic only.
 *
 * Usage:
 * - Run via Vitest as part of frontend validation.
 *
 * Invariants/Assumptions:
 * - Fixtures mirror the generated job and intercepted-entry API shapes.
 * - Helpers should remain deterministic for the same inputs.
 */

import { describe, expect, it } from "vitest";

import type { TreeNode } from "../../lib/tree-utils";
import type { CrawlResultWithTraffic, Job, ResultItem } from "../../types";

import {
  buildDefaultExpandedTreeIds,
  buildExportFilename,
  collectTreeNodeIds,
  extractTrafficEntries,
  filterResultItems,
  findComparableJobs,
  getExportMimeType,
  getJobByID,
  hasResearchVisualization,
} from "./resultsExplorerUtils";

const crawlResultWithTraffic: CrawlResultWithTraffic = {
  url: "https://example.com/articles/one",
  status: 200,
  title: "Article One",
  text: "Alpha beta",
  links: [],
  interceptedData: [
    {
      request: {
        requestId: "req-1",
        method: "GET",
        resourceType: "xhr",
        url: "https://example.com/api/article",
      },
    },
  ],
};

const resultItems: ResultItem[] = [
  crawlResultWithTraffic,
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

  it("normalizes export metadata", () => {
    expect(getExportMimeType("pdf")).toBe("application/pdf");
    expect(
      buildExportFilename("job-1", "json", "2026-03-10T10-00-00Z", "filtered"),
    ).toBe("results-job-1-filtered-2026-03-10T10-00-00Z.json");
  });

  it("extracts intercepted traffic entries from crawl results only", () => {
    expect(extractTrafficEntries(resultItems)).toEqual([
      {
        request: {
          requestId: "req-1",
          method: "GET",
          resourceType: "xhr",
          url: "https://example.com/api/article",
        },
      },
    ]);
  });
});
