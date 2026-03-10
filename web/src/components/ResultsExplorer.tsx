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
import { useEffect, useMemo, useState } from "react";
import { TreeView } from "./TreeView";
import { DiffViewer } from "./DiffViewer";
import { EvidenceChart } from "./EvidenceChart";
import { ClusterGraph } from "./ClusterGraph";
import { ResultsViewer } from "./ResultsViewer";
import { TransformPreview } from "./TransformPreview";
import { TrafficInspector } from "./TrafficInspector";
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
import {
  buildDefaultExpandedTreeIds,
  buildExportFilename,
  collectTreeNodeIds,
  exportFormats,
  extractTrafficEntries,
  filterResultItems,
  findComparableJobs,
  getExportMimeType,
  getJobByID,
  hasResearchVisualization,
  resultsExplorerViewModes,
  type ExportFormat,
  type StatusFilter,
  type ViewMode,
} from "./results-explorer/resultsExplorerUtils";

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

interface ResultsExplorerTabsProps {
  isResearchJob: boolean;
  viewMode: ViewMode;
  onSelectMode: (mode: ViewMode) => void;
}

interface ResultsExplorerToolbarProps {
  searchQuery: string;
  statusFilter: StatusFilter;
  isExporting: boolean;
  onChangeSearchQuery: (value: string) => void;
  onChangeStatusFilter: (value: StatusFilter) => void;
  onExport: (format: ExportFormat) => void;
  onExportHAR: () => void;
}

interface TreeControlsProps {
  treeNodes: TreeNode[];
  onExpandAll: () => void;
  onCollapseAll: () => void;
}

interface DiffControlsProps {
  compareJobId: string | null;
  comparableJobs: Job[];
  onChangeCompareJobID: (jobID: string | null) => void;
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

function ResultsExplorerTabs({
  isResearchJob,
  viewMode,
  onSelectMode,
}: ResultsExplorerTabsProps) {
  return (
    <div className="view-mode-tabs">
      {resultsExplorerViewModes
        .filter((mode) => isResearchJob || mode.id !== "visualize")
        .map((mode) => (
          <button
            key={mode.id}
            type="button"
            className={viewMode === mode.id ? "active" : ""}
            onClick={() => onSelectMode(mode.id)}
          >
            {mode.label}
          </button>
        ))}
    </div>
  );
}

function ResultsExplorerToolbar({
  searchQuery,
  statusFilter,
  isExporting,
  onChangeSearchQuery,
  onChangeStatusFilter,
  onExport,
  onExportHAR,
}: ResultsExplorerToolbarProps) {
  return (
    <div className="results-explorer-toolbar">
      <div className="search-box">
        <input
          type="text"
          placeholder="Search by URL, title, or content..."
          value={searchQuery}
          onChange={(event) => onChangeSearchQuery(event.target.value)}
        />
        {searchQuery && (
          <button
            type="button"
            className="search-clear"
            onClick={() => onChangeSearchQuery("")}
          >
            ×
          </button>
        )}
      </div>
      <select
        value={statusFilter}
        onChange={(event) =>
          onChangeStatusFilter(event.target.value as StatusFilter)
        }
        className="status-filter"
      >
        <option value="all">All Status</option>
        <option value="success">Success (2xx)</option>
        <option value="error">Error (4xx/5xx)</option>
      </select>
      <div className="export-buttons">
        {exportFormats.map((format) => (
          <button
            key={format}
            type="button"
            className="secondary"
            onClick={() => onExport(format)}
            disabled={isExporting}
          >
            Export {format.toUpperCase()}
          </button>
        ))}
        <button type="button" className="secondary" onClick={onExportHAR}>
          Export HAR
        </button>
      </div>
    </div>
  );
}

function ExplorerTreeControls({
  treeNodes,
  onExpandAll,
  onCollapseAll,
}: TreeControlsProps) {
  return (
    <div className="tree-controls">
      <button type="button" className="secondary" onClick={onExpandAll}>
        Expand All
      </button>
      <button type="button" className="secondary" onClick={onCollapseAll}>
        Collapse All
      </button>
      <span className="tree-stats">
        {treeNodes.length} domains,{" "}
        {treeNodes.reduce((sum, node) => sum + node.resultCount, 0)} pages
      </span>
    </div>
  );
}

function ExplorerDiffControls({
  compareJobId,
  comparableJobs,
  onChangeCompareJobID,
}: DiffControlsProps) {
  return (
    <div className="diff-controls">
      <label>
        Compare with:
        <select
          value={compareJobId || ""}
          onChange={(event) => onChangeCompareJobID(event.target.value || null)}
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
  );
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
      setTreeExpandedIds(buildDefaultExpandedTreeIds(treeNodes));
    }
  }, [treeNodes, treeExpandedIds.size]);

  const handleTreeSelect = (node: TreeNode) => {
    setTreeSelectedId(node.id);
    if (!node.result) {
      return;
    }

    const index = resultItems.findIndex(
      (item) => "url" in item && item.url === node.url,
    );
    if (index !== -1) {
      setSelectedResultIndex(index);
    }
  };

  const handleTreeToggle = (nodeId: string) => {
    setTreeExpandedIds((prev) => {
      const next = new Set(prev);
      if (next.has(nodeId)) {
        next.delete(nodeId);
      } else {
        next.add(nodeId);
      }
      return next;
    });
  };

  const expandAllTreeNodes = () => {
    setTreeExpandedIds(collectTreeNodeIds(treeNodes));
  };

  const collapseAllTreeNodes = () => {
    setTreeExpandedIds(buildDefaultExpandedTreeIds(treeNodes));
  };

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
  const handleExport = async (format: ExportFormat) => {
    if (!jobId) return;

    setIsExporting(true);
    try {
      const result = await loadResults(jobId, format, 1, 1000);
      const content = result.raw || "";
      const timestamp = new Date().toISOString().replace(/[:.]/g, "-");
      const filename = buildExportFilename(jobId, format, timestamp);
      downloadFile(
        content,
        filename,
        getExportMimeType(format),
        result.isBinary || false,
      );
    } catch (err) {
      console.error("Export failed:", err);
    } finally {
      setIsExporting(false);
    }
  };

  const handleExportWithTransform = async (
    expression: string,
    language: "jmespath" | "jsonata",
  ) => {
    if (!jobId) return;

    setIsExporting(true);
    try {
      const format = "json";
      const result = await loadResults(
        jobId,
        format,
        1,
        1000,
        expression,
        language,
      );

      if (result.error) {
        console.error("Transform export failed:", result.error);
        return;
      }

      const content = result.raw || "";
      const timestamp = new Date().toISOString().replace(/[:.]/g, "-");
      downloadFile(
        content,
        buildExportFilename(jobId, format, timestamp, "transformed"),
        getExportMimeType(format),
        false,
      );
    } catch (err) {
      console.error("Transform export failed:", err);
    } finally {
      setIsExporting(false);
    }
  };

  const filteredResultItems = useMemo(
    () => filterResultItems(resultItems, searchQuery, statusFilter),
    [resultItems, searchQuery, statusFilter],
  );

  const currentJob = useMemo(
    () => getJobByID(availableJobs, jobId),
    [availableJobs, jobId],
  );

  const compareJob = useMemo(
    () => getJobByID(availableJobs, compareJobId),
    [availableJobs, compareJobId],
  );

  const isResearchJob = hasResearchVisualization(jobType, resultEvidence);

  const comparableJobs = useMemo(
    () => findComparableJobs(availableJobs, jobId),
    [availableJobs, jobId],
  );

  const handleExportHAR = () => {
    if (!jobId) {
      return;
    }
    window.open(`/v1/jobs/${jobId}/results?format=har`, "_blank");
  };

  if (!jobId) {
    return null;
  }

  return (
    <div className="panel results-explorer" style={{ marginTop: 16 }}>
      {/* Header with view mode tabs */}
      <div className="results-explorer-header">
        <h3>Results: {jobId}</h3>
        <ResultsExplorerTabs
          isResearchJob={isResearchJob}
          viewMode={viewMode}
          onSelectMode={setViewMode}
        />
      </div>

      {viewMode !== "diff" && viewMode !== "visualize" && (
        <ResultsExplorerToolbar
          searchQuery={searchQuery}
          statusFilter={statusFilter}
          isExporting={isExporting}
          onChangeSearchQuery={setSearchQuery}
          onChangeStatusFilter={setStatusFilter}
          onExport={(format) => void handleExport(format)}
          onExportHAR={handleExportHAR}
        />
      )}

      {viewMode === "tree" && (
        <ExplorerTreeControls
          treeNodes={treeNodes}
          onExpandAll={expandAllTreeNodes}
          onCollapseAll={collapseAllTreeNodes}
        />
      )}

      {viewMode === "diff" && (
        <ExplorerDiffControls
          compareJobId={compareJobId}
          comparableJobs={comparableJobs}
          onChangeCompareJobID={setCompareJobId}
        />
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
              void handleExportWithTransform(expression, language);
            }}
          />
        )}

        {viewMode === "traffic" && (
          <TrafficInspector
            jobId={jobId}
            entries={extractTrafficEntries(resultItems)}
          />
        )}
      </div>
    </div>
  );
}

export default ResultsExplorer;
