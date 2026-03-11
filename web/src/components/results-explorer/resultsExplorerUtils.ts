/**
 * resultsExplorerUtils
 *
 * Purpose:
 * - Hold pure helper logic used by the results explorer UI.
 *
 * Responsibilities:
 * - Build derived explorer state from raw result items and jobs.
 * - Normalize export metadata and filtered result sets.
 * - Extract traffic payloads from crawl results with intercepted data.
 *
 * Scope:
 * - Pure utility logic only; no React state, DOM access, or network calls.
 *
 * Usage:
 * - Used by ResultsExplorer and focused Vitest coverage.
 *
 * Invariants/Assumptions:
 * - Crawl-style results are identified by URL and HTTP status fields.
 * - Unknown export formats default to markdown-compatible naming and text mime types.
 * - Traffic extraction ignores result items without intercepted data arrays.
 */

import type {
  CrawlResultItem,
  EvidenceItem,
  Job,
  ResultItem,
} from "../../types";
import type { TreeNode } from "../../lib/tree-utils";

export type ViewMode = "explorer" | "tree" | "diff" | "visualize" | "transform";

export type StatusFilter = "all" | "success" | "error";

export const resultsExplorerViewModes: Array<{
  id: ViewMode;
  label: string;
}> = [
  { id: "explorer", label: "Explorer" },
  { id: "tree", label: "Tree" },
  { id: "diff", label: "Diff" },
  { id: "visualize", label: "Visualize" },
  { id: "transform", label: "Transform" },
];

export const exportFormats = ["json", "jsonl", "csv", "md", "xlsx"] as const;

export type ExportFormat = (typeof exportFormats)[number];

export function isCrawlResult(item: ResultItem): item is CrawlResultItem {
  return "url" in item && "status" in item;
}

export function filterResultItems(
  resultItems: ResultItem[],
  searchQuery: string,
  statusFilter: StatusFilter,
): ResultItem[] {
  let filtered = resultItems;

  const normalizedQuery = searchQuery.trim().toLowerCase();
  if (normalizedQuery) {
    filtered = filtered.filter((item) => {
      if (isCrawlResult(item)) {
        return (
          item.url.toLowerCase().includes(normalizedQuery) ||
          item.title?.toLowerCase().includes(normalizedQuery) ||
          item.text?.toLowerCase().includes(normalizedQuery)
        );
      }

      return item.summary?.toLowerCase().includes(normalizedQuery) ?? false;
    });
  }

  if (statusFilter === "all") {
    return filtered;
  }

  return filtered.filter((item) => {
    if (!isCrawlResult(item)) {
      return true;
    }

    if (statusFilter === "success") {
      return item.status >= 200 && item.status < 300;
    }

    return item.status >= 400;
  });
}

export function collectTreeNodeIds(nodes: TreeNode[]): Set<string> {
  const ids = new Set<string>();

  const visit = (nodeList: TreeNode[]) => {
    nodeList.forEach((node) => {
      ids.add(node.id);
      if (node.children.length > 0) {
        visit(node.children);
      }
    });
  };

  visit(nodes);
  return ids;
}

export function buildDefaultExpandedTreeIds(nodes: TreeNode[]): Set<string> {
  return new Set(nodes.map((node) => node.id));
}

export function findComparableJobs(availableJobs: Job[], jobId: string | null) {
  return availableJobs.filter((job) => job.id !== jobId);
}

export function hasResearchVisualization(
  jobType: "scrape" | "crawl" | "research",
  evidence: EvidenceItem[],
): boolean {
  return jobType === "research" || evidence.length > 0;
}

export function getJobByID(availableJobs: Job[], jobId: string | null) {
  return availableJobs.find((job) => job.id === jobId) ?? null;
}

export function getExportMimeType(format: ExportFormat): string {
  switch (format) {
    case "json":
      return "application/json";
    case "csv":
      return "text/csv";
    case "xlsx":
      return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet";
    case "jsonl":
      return "application/x-ndjson";
    default:
      return "text/markdown";
  }
}

export function buildExportFilename(
  jobId: string,
  format: string,
  timestamp: string,
  suffix?: string,
): string {
  const suffixSegment = suffix ? `-${suffix}` : "";
  return `results-${jobId}${suffixSegment}-${timestamp}.${format}`;
}
