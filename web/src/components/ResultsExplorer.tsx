/**
 * Results Explorer Component
 *
 * Comprehensive results visualization component with multiple view modes:
 * - Explorer: Enhanced list view with pagination
 * - Tree: Hierarchical tree view for crawl site structure
 * - Diff: Side-by-side comparison between job runs
 * - Visualize: Research evidence charts and cluster graphs
 *
 * Integrates export functionality, search/filter, and view mode switching.
 *
 * @module ResultsExplorer
 */
import { useCallback, useEffect, useMemo, useState } from "react";
import { TreeView } from "./TreeView";
import { DiffViewer } from "./DiffViewer";
import { EvidenceChart } from "./EvidenceChart";
import { ClusterGraph } from "./ClusterGraph";
import { ResultsViewer } from "./ResultsViewer";
import { TransformPreview } from "./TransformPreview";
import { buildUrlTree, type TreeNode } from "../lib/tree-utils";
import {
  diffResults,
  type CrawlDiffResult,
  type ResearchDiffResult,
} from "../lib/diff-utils";
import { loadResults } from "../lib/results";
import type {
  ResultItem,
  EvidenceItem,
  ClusterItem,
  CitationItem,
  Job,
  CrawlResultItem,
} from "../types";

export type ViewMode = "explorer" | "tree" | "diff" | "visualize" | "transform";
export type StatusFilter = "all" | "success" | "error";

interface ResultsExplorerProps {
  jobId: string | null;
  resultItems: ResultItem[];
  selectedResultIndex: number;
  setSelectedResultIndex: (index: number) => void;
  resultSummary: string | null;
  resultConfidence: number | null;
  resultEvidence: EvidenceItem[];
  resultClusters: ClusterItem[];
  resultCitations: CitationItem[];
  rawResult: string | null;
  resultFormat: string;
  currentPage: number;
  totalResults: number;
  resultsPerPage: number;
  onLoadPage: (page: number) => void;
  availableJobs: Job[];
  jobType?: "scrape" | "crawl" | "research";
}

/**
 * Download content as a file.
 */
function downloadFile(
  content: string,
  filename: string,
  mimeType: string,
  isBinary = false,
) {
  const blob = isBinary
    ? new Blob([base64ToArrayBuffer(content)], { type: mimeType })
    : new Blob([content], { type: mimeType });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}

/**
 * Convert base64 string to ArrayBuffer for binary downloads.
 */
function base64ToArrayBuffer(base64: string): ArrayBuffer {
  const binaryString = atob(base64);
  const bytes = new Uint8Array(binaryString.length);
  for (let i = 0; i < binaryString.length; i++) {
    bytes[i] = binaryString.charCodeAt(i);
  }
  return bytes.buffer;
}

/**
 * Main ResultsExplorer component.
 */
export function ResultsExplorer({
  jobId,
  resultItems,
  selectedResultIndex,
  setSelectedResultIndex,
  resultSummary,
  resultConfidence,
  resultEvidence,
  resultClusters,
  resultCitations,
  rawResult,
  resultFormat,
  currentPage,
  totalResults,
  resultsPerPage,
  onLoadPage,
  availableJobs,
  jobType = "crawl",
}: ResultsExplorerProps) {
  // View mode state
  const [viewMode, setViewMode] = useState<ViewMode>("explorer");

  // Tree view state
  const [treeExpandedIds, setTreeExpandedIds] = useState<Set<string>>(
    new Set(),
  );
  const [treeSelectedId, setTreeSelectedId] = useState<string | null>(null);

  // Search and filter state
  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("all");

  // Diff view state
  const [compareJobId, setCompareJobId] = useState<string | null>(null);
  const [diffResult, setDiffResult] = useState<
    CrawlDiffResult | ResearchDiffResult | null
  >(null);
  const [diffLoading, setDiffLoading] = useState(false);
  const [diffError, setDiffError] = useState<string | null>(null);

  // Visualization state
  const [selectedEvidenceUrl, setSelectedEvidenceUrl] = useState<string | null>(
    null,
  );
  const [selectedClusterId, setSelectedClusterId] = useState<string | null>(
    null,
  );

  // Export state
  const [isExporting, setIsExporting] = useState(false);

  // Build tree from crawl results
  const treeNodes = useMemo(() => {
    const crawlItems = resultItems.filter(
      (item): item is CrawlResultItem => "url" in item && "status" in item,
    );
    return buildUrlTree(crawlItems);
  }, [resultItems]);

  // Initialize tree expansion on first load
  useEffect(() => {
    if (treeNodes.length > 0 && treeExpandedIds.size === 0) {
      // Expand all domain nodes by default
      const domainIds = treeNodes.map((n) => n.id);
      setTreeExpandedIds(new Set(domainIds));
    }
  }, [treeNodes, treeExpandedIds.size]);

  // Handle tree node selection
  const handleTreeSelect = useCallback(
    (node: TreeNode) => {
      setTreeSelectedId(node.id);
      if (node.result) {
        // Find index in resultItems
        const index = resultItems.findIndex(
          (item) => "url" in item && item.url === node.url,
        );
        if (index !== -1) {
          setSelectedResultIndex(index);
        }
      }
    },
    [resultItems, setSelectedResultIndex],
  );

  // Handle tree node expand/collapse
  const handleTreeToggle = useCallback((nodeId: string) => {
    setTreeExpandedIds((prev) => {
      const next = new Set(prev);
      if (next.has(nodeId)) {
        next.delete(nodeId);
      } else {
        next.add(nodeId);
      }
      return next;
    });
  }, []);

  // Expand/collapse all tree nodes
  const expandAllTreeNodes = useCallback(() => {
    const allIds = new Set<string>();
    const collectIds = (nodes: TreeNode[]) => {
      for (const node of nodes) {
        allIds.add(node.id);
        if (node.children.length > 0) {
          collectIds(node.children);
        }
      }
    };
    collectIds(treeNodes);
    setTreeExpandedIds(allIds);
  }, [treeNodes]);

  const collapseAllTreeNodes = useCallback(() => {
    // Keep only domain nodes expanded
    const domainIds = treeNodes.map((n) => n.id);
    setTreeExpandedIds(new Set(domainIds));
  }, [treeNodes]);

  // Compute diff when compare job is selected
  useEffect(() => {
    if (!jobId || !compareJobId || viewMode !== "diff") {
      setDiffResult(null);
      return;
    }

    const computeDiff = async () => {
      setDiffLoading(true);
      setDiffError(null);

      try {
        // Load results for both jobs
        const [baseResult, compareResult] = await Promise.all([
          loadResults(jobId, "jsonl", 1, 1000),
          loadResults(compareJobId, "jsonl", 1, 1000),
        ]);

        if (baseResult.error) {
          setDiffError(`Base job error: ${baseResult.error}`);
          return;
        }

        if (compareResult.error) {
          setDiffError(`Compare job error: ${compareResult.error}`);
          return;
        }

        const baseData = (baseResult.data || []) as ResultItem[];
        const compareData = (compareResult.data || []) as ResultItem[];

        const diff = diffResults(baseData, compareData);
        setDiffResult(diff);
      } catch (err) {
        setDiffError(String(err));
      } finally {
        setDiffLoading(false);
      }
    };

    void computeDiff();
  }, [jobId, compareJobId, viewMode]);

  // Handle export
  const handleExport = useCallback(
    async (format: "json" | "csv" | "md" | "xlsx" | "parquet" | "pdf") => {
      if (!jobId) return;

      setIsExporting(true);
      try {
        const result = await loadResults(jobId, format, 1, 1000);
        const content = result.raw || "";
        const timestamp = new Date().toISOString().replace(/[:.]/g, "-");
        const filename = `results-${jobId}-${timestamp}.${format}`;
        const mimeType =
          format === "json"
            ? "application/json"
            : format === "csv"
              ? "text/csv"
              : format === "xlsx"
                ? "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
                : format === "parquet"
                  ? "application/octet-stream"
                  : format === "pdf"
                    ? "application/pdf"
                    : "text/markdown";
        const isBinary = result.isBinary || false;
        downloadFile(content, filename, mimeType, isBinary);
      } catch (err) {
        console.error("Export failed:", err);
      } finally {
        setIsExporting(false);
      }
    },
    [jobId],
  );

  // Filter results for list view
  const filteredResultItems = useMemo(() => {
    let filtered = resultItems;

    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase();
      filtered = filtered.filter((item) => {
        if ("url" in item) {
          return (
            item.url.toLowerCase().includes(query) ||
            item.title?.toLowerCase().includes(query) ||
            item.text?.toLowerCase().includes(query)
          );
        }
        if ("summary" in item && item.summary) {
          return item.summary.toLowerCase().includes(query);
        }
        return false;
      });
    }

    if (statusFilter !== "all") {
      filtered = filtered.filter((item) => {
        if (!("status" in item)) return true;
        if (statusFilter === "success") {
          return item.status >= 200 && item.status < 300;
        }
        if (statusFilter === "error") {
          return item.status >= 400;
        }
        return true;
      });
    }

    return filtered;
  }, [resultItems, searchQuery, statusFilter]);

  // Get current job for diff
  const currentJob = useMemo(() => {
    return availableJobs.find((j) => j.id === jobId) || null;
  }, [availableJobs, jobId]);

  const compareJob = useMemo(() => {
    return availableJobs.find((j) => j.id === compareJobId) || null;
  }, [availableJobs, compareJobId]);

  // Check if research visualization is available
  const isResearchJob = jobType === "research" || resultEvidence.length > 0;

  // Get other jobs of same type for diff comparison
  const comparableJobs = useMemo(() => {
    return availableJobs.filter((j) => j.id !== jobId);
  }, [availableJobs, jobId]);

  if (!jobId) {
    return null;
  }

  return (
    <div className="panel results-explorer" style={{ marginTop: 16 }}>
      {/* Header with view mode tabs */}
      <div className="results-explorer-header">
        <h3>Results: {jobId}</h3>
        <div className="view-mode-tabs">
          <button
            type="button"
            className={viewMode === "explorer" ? "active" : ""}
            onClick={() => setViewMode("explorer")}
          >
            Explorer
          </button>
          <button
            type="button"
            className={viewMode === "tree" ? "active" : ""}
            onClick={() => setViewMode("tree")}
          >
            Tree
          </button>
          <button
            type="button"
            className={viewMode === "diff" ? "active" : ""}
            onClick={() => setViewMode("diff")}
          >
            Diff
          </button>
          {isResearchJob && (
            <button
              type="button"
              className={viewMode === "visualize" ? "active" : ""}
              onClick={() => setViewMode("visualize")}
            >
              Visualize
            </button>
          )}
          <button
            type="button"
            className={viewMode === "transform" ? "active" : ""}
            onClick={() => setViewMode("transform")}
          >
            Transform
          </button>
        </div>
      </div>

      {/* Search and filter bar */}
      {viewMode !== "diff" && viewMode !== "visualize" && (
        <div className="results-explorer-toolbar">
          <div className="search-box">
            <input
              type="text"
              placeholder="Search by URL, title, or content..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
            {searchQuery && (
              <button
                type="button"
                className="search-clear"
                onClick={() => setSearchQuery("")}
              >
                ×
              </button>
            )}
          </div>
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value as StatusFilter)}
            className="status-filter"
          >
            <option value="all">All Status</option>
            <option value="success">Success (2xx)</option>
            <option value="error">Error (4xx/5xx)</option>
          </select>
          <div className="export-buttons">
            <button
              type="button"
              className="secondary"
              onClick={() => void handleExport("json")}
              disabled={isExporting}
            >
              Export JSON
            </button>
            <button
              type="button"
              className="secondary"
              onClick={() => void handleExport("csv")}
              disabled={isExporting}
            >
              Export CSV
            </button>
            <button
              type="button"
              className="secondary"
              onClick={() => void handleExport("md")}
              disabled={isExporting}
            >
              Export MD
            </button>
            <button
              type="button"
              className="secondary"
              onClick={() => void handleExport("xlsx")}
              disabled={isExporting}
            >
              Export XLSX
            </button>
            <button
              type="button"
              className="secondary"
              onClick={() => void handleExport("parquet")}
              disabled={isExporting}
            >
              Export Parquet
            </button>
            <button
              type="button"
              className="secondary"
              onClick={() => void handleExport("pdf")}
              disabled={isExporting}
            >
              Export PDF
            </button>
          </div>
        </div>
      )}

      {/* Tree view controls */}
      {viewMode === "tree" && (
        <div className="tree-controls">
          <button
            type="button"
            className="secondary"
            onClick={expandAllTreeNodes}
          >
            Expand All
          </button>
          <button
            type="button"
            className="secondary"
            onClick={collapseAllTreeNodes}
          >
            Collapse All
          </button>
          <span className="tree-stats">
            {treeNodes.length} domains,{" "}
            {treeNodes.reduce((sum, n) => sum + n.resultCount, 0)} pages
          </span>
        </div>
      )}

      {/* Diff view controls */}
      {viewMode === "diff" && (
        <div className="diff-controls">
          <label>
            Compare with:
            <select
              value={compareJobId || ""}
              onChange={(e) => setCompareJobId(e.target.value || null)}
            >
              <option value="">Select a job...</option>
              {comparableJobs.map((job) => (
                <option key={job.id} value={job.id}>
                  {job.id} ({job.status})
                </option>
              ))}
            </select>
          </label>
        </div>
      )}

      {/* View content */}
      <div className="results-explorer-content">
        {viewMode === "explorer" && (
          <ResultsViewer
            jobId={jobId}
            resultItems={filteredResultItems}
            selectedResultIndex={selectedResultIndex}
            setSelectedResultIndex={setSelectedResultIndex}
            resultSummary={resultSummary}
            resultConfidence={resultConfidence}
            resultEvidence={resultEvidence}
            resultClusters={resultClusters}
            resultCitations={resultCitations}
            rawResult={rawResult}
            resultFormat={resultFormat}
            currentPage={currentPage}
            totalResults={totalResults}
            resultsPerPage={resultsPerPage}
            onLoadPage={onLoadPage}
          />
        )}

        {viewMode === "tree" && (
          <TreeView
            nodes={treeNodes}
            selectedId={treeSelectedId}
            onSelect={handleTreeSelect}
            onToggleExpand={handleTreeToggle}
            expandedIds={treeExpandedIds}
            searchQuery={searchQuery}
            statusFilter={statusFilter}
          />
        )}

        {viewMode === "diff" && (
          <DiffViewer
            baseJob={currentJob}
            compareJob={compareJob}
            diffResult={diffResult}
            isLoading={diffLoading}
            error={diffError}
            onClose={() => setViewMode("explorer")}
          />
        )}

        {viewMode === "visualize" && isResearchJob && (
          <div className="visualize-content">
            <EvidenceChart
              evidence={resultEvidence}
              clusters={resultClusters}
              selectedEvidenceUrl={selectedEvidenceUrl}
              onSelectEvidence={(item) => setSelectedEvidenceUrl(item.url)}
            />
            {resultClusters.length > 0 && (
              <ClusterGraph
                clusters={resultClusters}
                evidence={resultEvidence}
                selectedClusterId={selectedClusterId}
                onSelectCluster={(cluster) => setSelectedClusterId(cluster.id)}
              />
            )}
          </div>
        )}

        {viewMode === "transform" && (
          <TransformPreview
            jobId={jobId}
            onApply={(expression, language) => {
              console.log("Applying transform:", expression, language);
              // TODO: Apply transformation to export
            }}
          />
        )}
      </div>
    </div>
  );
}

export default ResultsExplorer;
