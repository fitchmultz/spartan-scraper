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
import type { ExportShapeConfig } from "../api";
import {
  type CrawlDiffResult,
  diffResults,
  type ResearchDiffResult,
} from "../lib/diff-utils";
import { exportResults, loadResults } from "../lib/results";
import { buildUrlTree, type TreeNode } from "../lib/tree-utils";
import type {
  AgenticResearchItem,
  CitationItem,
  ClusterItem,
  CrawlResultItem,
  EvidenceItem,
  Job,
  ResultItem,
} from "../types";
import { AIExportShapeAssistant } from "./AIExportShapeAssistant";
import { ClusterGraph } from "./ClusterGraph";
import { DiffViewer } from "./DiffViewer";
import { EvidenceChart } from "./EvidenceChart";
import { ResultsViewer } from "./ResultsViewer";
import {
  buildDefaultExpandedTreeIds,
  collectTreeNodeIds,
  type ExportFormat,
  exportFormats,
  filterResultItems,
  findComparableJobs,
  getJobByID,
  hasResearchVisualization,
  resultsExplorerViewModes,
  type StatusFilter,
  type ViewMode,
} from "./results-explorer/resultsExplorerUtils";
import { TransformPreview } from "./TransformPreview";
import { TreeView } from "./TreeView";

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
  resultAgentic: AgenticResearchItem | null;
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
  resultAgentic,
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
  const [shapeExportFormat, setShapeExportFormat] = useState<
    "md" | "csv" | "xlsx"
  >("md");
  const [shapeConfigText, setShapeConfigText] = useState("");
  const [shapeConfigError, setShapeConfigError] = useState<string | null>(null);
  const [isShapeAssistantOpen, setIsShapeAssistantOpen] = useState(false);

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
      const result = await exportResults(jobId, { format });
      downloadFile(
        result.content,
        result.filename,
        result.contentType,
        result.isBinary,
      );
    } catch (err) {
      console.error("Export failed:", err);
    } finally {
      setIsExporting(false);
    }
  };

  const handleExportWithTransform = async (
    format: ExportFormat,
    expression: string,
    language: "jmespath" | "jsonata",
  ) => {
    if (!jobId) return;

    setIsExporting(true);
    try {
      const result = await exportResults(jobId, {
        format,
        transform: {
          expression,
          language,
        },
      });
      downloadFile(
        result.content,
        result.filename,
        result.contentType,
        result.isBinary,
      );
    } catch (err) {
      console.error("Transform export failed:", err);
    } finally {
      setIsExporting(false);
    }
  };

  const currentShapeConfig = useMemo<ExportShapeConfig | undefined>(() => {
    const trimmed = shapeConfigText.trim();
    if (!trimmed) {
      return undefined;
    }
    try {
      return JSON.parse(trimmed) as ExportShapeConfig;
    } catch {
      return undefined;
    }
  }, [shapeConfigText]);

  const handleShapeExport = async () => {
    if (!jobId) return;

    const trimmed = shapeConfigText.trim();
    if (!trimmed) {
      setShapeConfigError("Enter a shape JSON object or generate one with AI.");
      return;
    }

    let shape: ExportShapeConfig;
    try {
      shape = JSON.parse(trimmed) as ExportShapeConfig;
    } catch {
      setShapeConfigError("Shape JSON must be a valid object.");
      return;
    }

    setShapeConfigError(null);
    setIsExporting(true);
    try {
      const result = await exportResults(jobId, {
        format: shapeExportFormat,
        shape,
      });
      downloadFile(
        result.content,
        result.filename,
        result.contentType,
        result.isBinary,
      );
    } catch (err) {
      setShapeConfigError(String(err));
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
            jobKind={
              currentJob?.kind as "scrape" | "crawl" | "research" | undefined
            }
            resultItems={filteredResultItems}
            selectedResultIndex={selectedResultIndex}
            setSelectedResultIndex={setSelectedResultIndex}
            resultSummary={resultSummary}
            resultConfidence={resultConfidence}
            resultEvidence={resultEvidence}
            resultClusters={resultClusters}
            resultCitations={resultCitations}
            resultAgentic={resultAgentic}
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
          <>
            <TransformPreview
              jobId={jobId}
              onApply={(format, expression, language) => {
                void handleExportWithTransform(format, expression, language);
              }}
            />
            <div className="panel" style={{ marginTop: 16 }}>
              <h4>Direct Shape Export</h4>
              <p className="form-help">
                Apply a bounded export shape directly to this saved result for
                markdown and tabular exports.
              </p>
              <div className="row" style={{ gap: 12, alignItems: "flex-end" }}>
                <label>
                  Format
                  <select
                    value={shapeExportFormat}
                    onChange={(event) =>
                      setShapeExportFormat(
                        event.target.value as "md" | "csv" | "xlsx",
                      )
                    }
                  >
                    <option value="md">Markdown</option>
                    <option value="csv">CSV</option>
                    <option value="xlsx">XLSX</option>
                  </select>
                </label>
                <button
                  type="button"
                  className="secondary"
                  onClick={() => setIsShapeAssistantOpen(true)}
                >
                  Generate Shape with AI
                </button>
                <button
                  type="button"
                  className="secondary"
                  onClick={() => {
                    setShapeConfigText("");
                    setShapeConfigError(null);
                  }}
                >
                  Clear Shape
                </button>
                <button type="button" onClick={() => void handleShapeExport()}>
                  Export Shaped Result
                </button>
              </div>
              <textarea
                className="form-textarea"
                rows={10}
                style={{ marginTop: 12 }}
                value={shapeConfigText}
                onChange={(event) => {
                  setShapeConfigText(event.target.value);
                  setShapeConfigError(null);
                }}
                placeholder='{"summaryFields":["title","url"],"normalizedFields":["field.price"]}'
              />
              {shapeConfigError ? (
                <div className="transform-error">Error: {shapeConfigError}</div>
              ) : null}
            </div>
            <AIExportShapeAssistant
              isOpen={isShapeAssistantOpen}
              onClose={() => setIsShapeAssistantOpen(false)}
              format={shapeExportFormat}
              currentShape={currentShapeConfig}
              initialJobId={jobId}
              onApplyShape={(shape) => {
                setShapeConfigText(JSON.stringify(shape, null, 2));
                setShapeConfigError(null);
                setIsShapeAssistantOpen(false);
              }}
            />
          </>
        )}
      </div>
    </div>
  );
}

export default ResultsExplorer;
