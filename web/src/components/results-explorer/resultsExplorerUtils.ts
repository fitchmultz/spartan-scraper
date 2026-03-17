/**
 * Purpose: Hold pure helper logic used by the results explorer UI.
 * Responsibilities: Build derived explorer state from raw result items and jobs, filter result sets, describe secondary tools, and generate guided export recommendations.
 * Scope: Pure utility logic only; no React state, DOM access, or network calls.
 * Usage: Used by `ResultsExplorer` and focused Vitest coverage.
 * Invariants/Assumptions: Crawl-style results are identified by URL and HTTP status fields, supported direct-export formats stay aligned with the backend contract, and guided export recommendations must stay deterministic for the same result input.
 */

import type {
  CrawlResultItem,
  EvidenceItem,
  Job,
  ResultItem,
} from "../../types";
import type { TreeNode } from "../../lib/tree-utils";

export type SecondaryToolId = "tree" | "diff" | "transform" | "visualize";
export type StatusFilter = "all" | "success" | "error";
export type ExportFormat = "json" | "jsonl" | "csv" | "md" | "xlsx";
export type ExportReadiness = "recommended" | "available" | "limited";

export interface SecondaryToolOption {
  id: SecondaryToolId;
  label: string;
  description: string;
}

export interface ExportGuidanceOption {
  format: ExportFormat;
  title: string;
  description: string;
  readiness: ExportReadiness;
  scopeLabel: string;
  scopeNote: string;
}

interface ExportGuidanceInput {
  totalResults: number;
  visibleResults: number;
  searchQuery: string;
  statusFilter: StatusFilter;
  resultItems: ResultItem[];
  evidence: EvidenceItem[];
  isResearchJob: boolean;
}

const secondaryTools: SecondaryToolOption[] = [
  {
    id: "tree",
    label: "Inspect structure",
    description:
      "Navigate crawl output as domains, sections, and pages when the default reader needs more site context.",
  },
  {
    id: "diff",
    label: "Compare runs",
    description:
      "Load another saved job and inspect what changed without crowding the default reader.",
  },
  {
    id: "transform",
    label: "Transform output",
    description:
      "Shape or remap the saved result before exporting when the default download formats are not enough.",
  },
  {
    id: "visualize",
    label: "Visualize evidence",
    description:
      "Explore research evidence relationships, clusters, and linked sources in a graph-oriented view.",
  },
];

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

export function findComparableJobs(
  availableJobs: Job[],
  jobId: string | null,
  jobKind?: Job["kind"],
) {
  return availableJobs.filter((job) => {
    if (job.id === jobId) {
      return false;
    }

    if (jobKind && job.kind !== jobKind) {
      return false;
    }

    return true;
  });
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

export function getAvailableSecondaryTools(
  isResearchJob: boolean,
): SecondaryToolOption[] {
  return secondaryTools.filter(
    (tool) => isResearchJob || tool.id !== "visualize",
  );
}

function describeScopeLabel(totalResults: number): string {
  if (totalResults <= 0) {
    return "Exports the saved job output.";
  }

  if (totalResults === 1) {
    return "Exports the full saved job output for 1 result.";
  }

  return `Exports the full saved job output for ${totalResults} results.`;
}

function describeScopeNote({
  visibleResults,
  searchQuery,
  statusFilter,
}: Pick<
  ExportGuidanceInput,
  "visibleResults" | "searchQuery" | "statusFilter"
>): string {
  if (searchQuery.trim() || statusFilter !== "all") {
    return `Current search and status filters only narrow the on-screen reader${
      visibleResults > 0 ? ` (${visibleResults} visible right now)` : ""
    }. Use Transform when you need a shaped export.`;
  }

  return "Use this when you want the full saved job output, not just the currently selected item.";
}

function getFormatReadiness(
  format: ExportFormat,
  isResearchJob: boolean,
  hasCrawlRows: boolean,
  hasEvidence: boolean,
): ExportReadiness {
  if (format === "json") {
    return isResearchJob ? "recommended" : "available";
  }

  if (format === "jsonl") {
    return hasCrawlRows ? "recommended" : "available";
  }

  if (format === "md") {
    return isResearchJob ? "recommended" : "available";
  }

  if (hasCrawlRows) {
    return format === "csv" ? "recommended" : "available";
  }

  if (hasEvidence) {
    return "available";
  }

  return "limited";
}

function getFormatDescriptions(
  format: ExportFormat,
  isResearchJob: boolean,
): Pick<ExportGuidanceOption, "title" | "description"> {
  switch (format) {
    case "json":
      return {
        title: "JSON",
        description:
          "Machine-readable structured output for downstream scripting, debugging, or re-ingestion.",
      };
    case "jsonl":
      return {
        title: "JSONL",
        description:
          "Line-delimited result records that preserve per-item fidelity for crawls and bulk processing.",
      };
    case "csv":
      return {
        title: "CSV",
        description: isResearchJob
          ? "Spreadsheet-friendly rows when you need a lightweight tabular export of evidence or transformed output."
          : "Spreadsheet-friendly rows for quick auditing, sorting, and handoff outside Spartan.",
      };
    case "md":
      return {
        title: "Markdown",
        description: isResearchJob
          ? "Readable narrative export for sharing summaries, findings, and supporting context."
          : "Readable report export when operators want a human-first snapshot instead of raw records.",
      };
    case "xlsx":
      return {
        title: "XLSX",
        description:
          "Workbook export for richer spreadsheet review when CSV is too limited for the handoff.",
      };
  }
}

export function getExportGuidanceOptions({
  totalResults,
  visibleResults,
  searchQuery,
  statusFilter,
  resultItems,
  evidence,
  isResearchJob,
}: ExportGuidanceInput): ExportGuidanceOption[] {
  const hasCrawlRows = resultItems.some(isCrawlResult);
  const hasEvidence = evidence.length > 0;
  const scopeLabel = describeScopeLabel(totalResults);
  const scopeNote = describeScopeNote({
    visibleResults,
    searchQuery,
    statusFilter,
  });

  const formats: ExportFormat[] = isResearchJob
    ? ["md", "json", "jsonl", "csv", "xlsx"]
    : ["jsonl", "csv", "xlsx", "json", "md"];

  return formats.map((format) => ({
    format,
    ...getFormatDescriptions(format, isResearchJob),
    readiness: getFormatReadiness(
      format,
      isResearchJob,
      hasCrawlRows,
      hasEvidence,
    ),
    scopeLabel,
    scopeNote,
  }));
}
