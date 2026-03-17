/**
 * Purpose: Provide the result-focused workspace for `/jobs/:id` with one dominant default reader.
 * Responsibilities: Coordinate reader filters, selected-item continuity, secondary tools, guided export flows, tree navigation, diff loading, and research visualization state.
 * Scope: Results exploration UI only; route framing and authoritative result fetching stay outside this component.
 * Usage: Render from `ResultsContainer` with the active job's saved results and the surrounding jobs list for compare workflows.
 * Invariants/Assumptions: A selected job ID exists before rendering, the default reader always remains visible on first paint, and comparison/export actions operate on saved job results.
 */

import { useEffect, useMemo, useState } from "react";
import type { ExportShapeConfig } from "../api";
import {
  type CrawlDiffResult,
  diffResults,
  type ResearchDiffResult,
} from "../lib/diff-utils";
import { getApiErrorMessage } from "../lib/api-errors";
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
import {
  ResultsAssistantSection,
  type ResultsAssistantMode,
  useAIAssistant,
} from "./ai-assistant";
import { ClusterGraph } from "./ClusterGraph";
import { DiffViewer } from "./DiffViewer";
import { EvidenceChart } from "./EvidenceChart";
import { ResultsViewer } from "./ResultsViewer";
import {
  buildDefaultExpandedTreeIds,
  collectTreeNodeIds,
  type ExportFormat,
  getAvailableSecondaryTools,
  getExportGuidanceOptions,
  getJobByID,
  hasResearchVisualization,
  type SecondaryToolId,
  filterResultItems,
  findComparableJobs,
  isCrawlResult,
  type StatusFilter,
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

interface ReaderToolbarProps {
  searchQuery: string;
  statusFilter: StatusFilter;
  visibleResults: number;
  totalResults: number;
  onChangeSearchQuery: (value: string) => void;
  onChangeStatusFilter: (value: StatusFilter) => void;
  onClearFilters: () => void;
}

interface SecondaryToolsDrawerProps {
  tools: ReturnType<typeof getAvailableSecondaryTools>;
  activeTool: SecondaryToolId | null;
  onSelectTool: (tool: SecondaryToolId) => void;
  onClose: () => void;
}

interface GuidedExportDrawerProps {
  options: ReturnType<typeof getExportGuidanceOptions>;
  isExporting: boolean;
  exportError: string | null;
  onExport: (format: ExportFormat) => void;
  onClose: () => void;
  onOpenTransform: () => void;
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

function base64ToArrayBuffer(base64: string): ArrayBuffer {
  const binaryString = atob(base64);
  const bytes = new Uint8Array(binaryString.length);
  for (let i = 0; i < binaryString.length; i++) {
    bytes[i] = binaryString.charCodeAt(i);
  }
  return bytes.buffer;
}

function ReaderToolbar({
  searchQuery,
  statusFilter,
  visibleResults,
  totalResults,
  onChangeSearchQuery,
  onChangeStatusFilter,
  onClearFilters,
}: ReaderToolbarProps) {
  return (
    <div className="results-explorer-toolbar">
      <div className="search-box">
        <input
          type="text"
          placeholder="Search by URL, title, or content..."
          value={searchQuery}
          onChange={(event) => onChangeSearchQuery(event.target.value)}
        />
        {searchQuery ? (
          <button
            type="button"
            className="search-clear"
            onClick={() => onChangeSearchQuery("")}
            aria-label="Clear search"
          >
            ×
          </button>
        ) : null}
      </div>

      <select
        value={statusFilter}
        onChange={(event) =>
          onChangeStatusFilter(event.target.value as StatusFilter)
        }
        className="status-filter"
        aria-label="Result status filter"
      >
        <option value="all">All status</option>
        <option value="success">Success (2xx)</option>
        <option value="error">Error (4xx/5xx)</option>
      </select>

      <div className="results-explorer__toolbar-hint">
        {visibleResults === totalResults || totalResults === 0
          ? `${Math.max(visibleResults, totalResults)} result${
              Math.max(visibleResults, totalResults) === 1 ? "" : "s"
            } in the reader.`
          : `${visibleResults} of ${totalResults} results are visible in the reader.`}
      </div>

      {(searchQuery.trim() || statusFilter !== "all") &&
      visibleResults !== totalResults ? (
        <button type="button" className="secondary" onClick={onClearFilters}>
          Clear reader filters
        </button>
      ) : null}
    </div>
  );
}

function SecondaryToolsDrawer({
  tools,
  activeTool,
  onSelectTool,
  onClose,
}: SecondaryToolsDrawerProps) {
  return (
    <div className="results-explorer__drawer">
      <div className="results-explorer__drawer-header">
        <div>
          <div className="results-viewer__section-label">Secondary tools</div>
          <h4>Open comparison and analysis only when you need it</h4>
          <p className="form-help">
            The reader stays primary. These tools sit underneath it so you can
            branch into structure, comparison, visualization, or transforms
            without losing context.
          </p>
        </div>
        <button type="button" className="secondary" onClick={onClose}>
          Close
        </button>
      </div>

      <div className="results-explorer__tool-grid">
        {tools.map((tool) => (
          <button
            key={tool.id}
            type="button"
            className={`results-explorer__tool-card ${
              activeTool === tool.id ? "is-active" : ""
            }`}
            onClick={() => onSelectTool(tool.id)}
          >
            <strong>{tool.label}</strong>
            <span>{tool.description}</span>
          </button>
        ))}
      </div>
    </div>
  );
}

function GuidedExportDrawer({
  options,
  isExporting,
  exportError,
  onExport,
  onClose,
  onOpenTransform,
}: GuidedExportDrawerProps) {
  const scopePreview = options[0];

  return (
    <div className="results-explorer__drawer">
      <div className="results-explorer__drawer-header">
        <div>
          <div className="results-viewer__section-label">Guided export</div>
          <h4>Choose the right handoff format before you download</h4>
          <p className="form-help">
            Export stays quiet by default. Open it when you are ready to hand
            off or archive the saved result.
          </p>
        </div>
        <div className="results-explorer__export-actions">
          <button type="button" className="secondary" onClick={onOpenTransform}>
            Need a transformed export?
          </button>
          <button type="button" className="secondary" onClick={onClose}>
            Close
          </button>
        </div>
      </div>

      {scopePreview ? (
        <div className="results-explorer__export-preview">
          <strong>{scopePreview.scopeLabel}</strong>
          <p>{scopePreview.scopeNote}</p>
        </div>
      ) : null}

      {exportError ? (
        <div className="transform-error">{exportError}</div>
      ) : null}

      <div className="results-explorer__export-grid">
        {options.map((option) => (
          <div key={option.format} className="results-explorer__export-card">
            <div className="results-explorer__export-card-head">
              <div>
                <h5>{option.title}</h5>
                <p>{option.description}</p>
              </div>
              <span
                className={`results-explorer__readiness results-explorer__readiness--${option.readiness}`}
              >
                {option.readiness}
              </span>
            </div>
            <button
              type="button"
              className="secondary"
              onClick={() => onExport(option.format)}
              disabled={isExporting}
            >
              Export {option.title}
            </button>
          </div>
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
        Expand all
      </button>
      <button type="button" className="secondary" onClick={onCollapseAll}>
        Collapse to domains
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
  const aiAssistant = useAIAssistant();
  const [assistantMode, setAssistantMode] = useState<ResultsAssistantMode>(
    jobType === "research" ? "research" : "shape",
  );
  const [isToolsOpen, setIsToolsOpen] = useState(false);
  const [isExportOpen, setIsExportOpen] = useState(false);
  const [activeTool, setActiveTool] = useState<SecondaryToolId | null>(null);

  const [treeExpandedIds, setTreeExpandedIds] = useState<Set<string>>(
    new Set(),
  );
  const [treeSelectedId, setTreeSelectedId] = useState<string | null>(null);

  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("all");

  const [compareJobId, setCompareJobId] = useState<string | null>(null);
  const [diffResult, setDiffResult] = useState<
    CrawlDiffResult | ResearchDiffResult | null
  >(null);
  const [diffLoading, setDiffLoading] = useState(false);
  const [diffError, setDiffError] = useState<string | null>(null);

  const [selectedEvidenceUrl, setSelectedEvidenceUrl] = useState<string | null>(
    null,
  );
  const [selectedClusterId, setSelectedClusterId] = useState<string | null>(
    null,
  );

  const [isExporting, setIsExporting] = useState(false);
  const [exportError, setExportError] = useState<string | null>(null);
  const [shapeExportFormat, setShapeExportFormat] = useState<
    "md" | "csv" | "xlsx"
  >("md");
  const [shapeConfigText, setShapeConfigText] = useState("");
  const [shapeConfigError, setShapeConfigError] = useState<string | null>(null);

  const treeNodes = useMemo(() => {
    const crawlItems = resultItems.filter((item): item is CrawlResultItem =>
      isCrawlResult(item),
    );
    return buildUrlTree(crawlItems);
  }, [resultItems]);

  useEffect(() => {
    if (treeNodes.length > 0 && treeExpandedIds.size === 0) {
      setTreeExpandedIds(buildDefaultExpandedTreeIds(treeNodes));
    }
  }, [treeNodes, treeExpandedIds.size]);

  useEffect(() => {
    if (jobType !== "research" && assistantMode === "research") {
      setAssistantMode("shape");
    }
  }, [assistantMode, jobType]);

  const filteredResultItems = useMemo(
    () => filterResultItems(resultItems, searchQuery, statusFilter),
    [resultItems, searchQuery, statusFilter],
  );

  const filteredSourceIndexes = useMemo(
    () =>
      resultItems.reduce<number[]>((indexes, item, index) => {
        if (filteredResultItems.includes(item)) {
          indexes.push(index);
        }
        return indexes;
      }, []),
    [filteredResultItems, resultItems],
  );

  useEffect(() => {
    if (filteredSourceIndexes.length === 0) {
      return;
    }

    if (!filteredSourceIndexes.includes(selectedResultIndex)) {
      setSelectedResultIndex(filteredSourceIndexes[0]);
    }
  }, [filteredSourceIndexes, selectedResultIndex, setSelectedResultIndex]);

  const visibleSelectedIndex = Math.max(
    0,
    filteredSourceIndexes.indexOf(selectedResultIndex),
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

  const secondaryTools = useMemo(
    () => getAvailableSecondaryTools(isResearchJob),
    [isResearchJob],
  );

  const activeToolConfig = secondaryTools.find(
    (tool) => tool.id === activeTool,
  );

  const comparableJobs = useMemo(
    () => findComparableJobs(availableJobs, jobId, currentJob?.kind),
    [availableJobs, currentJob?.kind, jobId],
  );

  const exportOptions = useMemo(
    () =>
      getExportGuidanceOptions({
        totalResults,
        visibleResults: filteredResultItems.length,
        searchQuery,
        statusFilter,
        resultItems,
        evidence: resultEvidence,
        isResearchJob,
      }),
    [
      filteredResultItems.length,
      isResearchJob,
      resultEvidence,
      resultItems,
      searchQuery,
      statusFilter,
      totalResults,
    ],
  );

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

  useEffect(() => {
    if (!jobId || !compareJobId || activeTool !== "diff") {
      setDiffResult(null);
      return;
    }

    const computeDiff = async () => {
      setDiffLoading(true);
      setDiffError(null);

      try {
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

        setDiffResult(diffResults(baseData, compareData));
      } catch (err) {
        setDiffError(
          getApiErrorMessage(
            err,
            "Failed to compare the selected job results.",
          ),
        );
      } finally {
        setDiffLoading(false);
      }
    };

    void computeDiff();
  }, [activeTool, compareJobId, jobId]);

  if (!jobId) {
    return null;
  }

  const handleTreeSelect = (node: TreeNode) => {
    setTreeSelectedId(node.id);
    if (!node.result) {
      return;
    }

    const index = resultItems.findIndex(
      (item) => isCrawlResult(item) && item.url === node.url,
    );
    if (index !== -1) {
      setSelectedResultIndex(index);
    }
  };

  const handleTreeToggle = (nodeId: string) => {
    setTreeExpandedIds((previous) => {
      const next = new Set(previous);
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

  const handleDirectExport = async (format: ExportFormat) => {
    if (!jobId) {
      return;
    }

    setExportError(null);
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
      setExportError(
        getApiErrorMessage(err, "Failed to export the saved result."),
      );
    } finally {
      setIsExporting(false);
    }
  };

  const handleExportWithTransform = async (
    format: ExportFormat,
    expression: string,
    language: "jmespath" | "jsonata",
  ) => {
    if (!jobId) {
      return;
    }

    setExportError(null);
    setShapeConfigError(null);
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
      setExportError(
        getApiErrorMessage(err, "Failed to export the transformed result."),
      );
    } finally {
      setIsExporting(false);
    }
  };

  const handleShapeExport = async () => {
    if (!jobId) {
      return;
    }

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
    setExportError(null);
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
      setShapeConfigError(
        getApiErrorMessage(err, "Failed to export the shaped result."),
      );
    } finally {
      setIsExporting(false);
    }
  };

  return (
    <div className="panel results-explorer">
      <div className="ai-assistant-surface">
        <div className="ai-assistant-surface__main">
          <div className="results-explorer-header">
            <div>
              <div className="results-viewer__section-label">Result reader</div>
              <h3>Read the saved output</h3>
              <p className="form-help">
                Start with the selected item, understand what changed, and only
                then branch into comparison, structure, transforms, or exports.
              </p>
              <div className="results-explorer__surface-summary">
                <span>Job {jobId}</span>
                {currentJob?.kind ? (
                  <span>{currentJob.kind} workflow</span>
                ) : null}
                <span>
                  {totalResults > 0
                    ? `${totalResults} saved results`
                    : "Saved result route"}
                </span>
              </div>
            </div>

            <div className="results-explorer__actions">
              <button
                type="button"
                className={isToolsOpen ? "active" : "secondary"}
                onClick={() => {
                  setIsToolsOpen((open) => !open);
                  setIsExportOpen(false);
                }}
              >
                Tools
              </button>
              <button
                type="button"
                className={isExportOpen ? "active" : "secondary"}
                onClick={() => {
                  setIsExportOpen((open) => !open);
                  setIsToolsOpen(false);
                }}
              >
                Export
              </button>
              <button
                type="button"
                className="secondary"
                onClick={() => {
                  setAssistantMode("shape");
                  aiAssistant.open({
                    surface: "results",
                    jobId,
                    resultFormat,
                    selectedResultIndex,
                    resultSummary,
                  });
                }}
              >
                Open AI assistant
              </button>
            </div>
          </div>

          <ReaderToolbar
            searchQuery={searchQuery}
            statusFilter={statusFilter}
            visibleResults={filteredResultItems.length}
            totalResults={Math.max(totalResults, resultItems.length)}
            onChangeSearchQuery={setSearchQuery}
            onChangeStatusFilter={setStatusFilter}
            onClearFilters={() => {
              setSearchQuery("");
              setStatusFilter("all");
            }}
          />

          {filteredResultItems.length === 0 ? (
            <div className="results-explorer__surface-note">
              No saved results match the current reader filters.
            </div>
          ) : null}

          {isToolsOpen ? (
            <SecondaryToolsDrawer
              tools={secondaryTools}
              activeTool={activeTool}
              onSelectTool={(tool) => {
                setActiveTool(tool);
              }}
              onClose={() => setIsToolsOpen(false)}
            />
          ) : null}

          {isExportOpen ? (
            <GuidedExportDrawer
              options={exportOptions}
              isExporting={isExporting}
              exportError={exportError}
              onExport={(format) => {
                void handleDirectExport(format);
              }}
              onClose={() => setIsExportOpen(false)}
              onOpenTransform={() => {
                setExportError(null);
                setIsExportOpen(false);
                setIsToolsOpen(true);
                setActiveTool("transform");
              }}
            />
          ) : null}

          <ResultsViewer
            jobId={jobId}
            jobKind={
              currentJob?.kind as "scrape" | "crawl" | "research" | undefined
            }
            resultItems={filteredResultItems}
            selectedResultIndex={visibleSelectedIndex}
            setSelectedResultIndex={(visibleIndex) => {
              const sourceIndex = filteredSourceIndexes[visibleIndex];
              if (typeof sourceIndex === "number") {
                setSelectedResultIndex(sourceIndex);
              }
            }}
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
            onOpenResearchAssistant={() => {
              setAssistantMode("research");
              aiAssistant.open({
                surface: "results",
                jobId,
                resultFormat,
                selectedResultIndex,
                resultSummary,
              });
            }}
          />

          {activeToolConfig ? (
            <div className="results-explorer__tool-panel">
              <div className="results-explorer__drawer-header">
                <div>
                  <div className="results-viewer__section-label">
                    Secondary tool
                  </div>
                  <h4>Secondary tool: {activeToolConfig.label}</h4>
                  <p className="form-help">{activeToolConfig.description}</p>
                </div>
                <button
                  type="button"
                  className="secondary"
                  onClick={() => setActiveTool(null)}
                >
                  Hide tool
                </button>
              </div>

              {activeTool === "tree" ? (
                <>
                  <ExplorerTreeControls
                    treeNodes={treeNodes}
                    onExpandAll={expandAllTreeNodes}
                    onCollapseAll={collapseAllTreeNodes}
                  />
                  <TreeView
                    nodes={treeNodes}
                    selectedId={treeSelectedId}
                    onSelect={handleTreeSelect}
                    onToggleExpand={handleTreeToggle}
                    expandedIds={treeExpandedIds}
                    searchQuery={searchQuery}
                    statusFilter={statusFilter}
                  />
                </>
              ) : null}

              {activeTool === "diff" ? (
                <>
                  <ExplorerDiffControls
                    compareJobId={compareJobId}
                    comparableJobs={comparableJobs}
                    onChangeCompareJobID={setCompareJobId}
                  />
                  <DiffViewer
                    baseJob={currentJob}
                    compareJob={compareJob}
                    diffResult={diffResult}
                    isLoading={diffLoading}
                    error={diffError}
                    onClose={() => setActiveTool(null)}
                  />
                </>
              ) : null}

              {activeTool === "visualize" && isResearchJob ? (
                <div className="visualize-content">
                  <EvidenceChart
                    evidence={resultEvidence}
                    clusters={resultClusters}
                    selectedEvidenceUrl={selectedEvidenceUrl}
                    onSelectEvidence={(item) =>
                      setSelectedEvidenceUrl(item.url)
                    }
                  />
                  {resultClusters.length > 0 ? (
                    <ClusterGraph
                      clusters={resultClusters}
                      evidence={resultEvidence}
                      selectedClusterId={selectedClusterId}
                      onSelectCluster={(cluster) =>
                        setSelectedClusterId(cluster.id)
                      }
                    />
                  ) : null}
                </div>
              ) : null}

              {activeTool === "transform" ? (
                <div className="results-explorer__transform-stack">
                  {exportError ? (
                    <div className="transform-error">{exportError}</div>
                  ) : null}
                  <TransformPreview
                    jobId={jobId}
                    onApply={(format, expression, language) => {
                      void handleExportWithTransform(
                        format,
                        expression,
                        language,
                      );
                    }}
                  />
                  <div className="panel results-explorer__shape-export">
                    <h4>Direct shape export</h4>
                    <p className="form-help">
                      Apply a bounded export shape directly to this saved result
                      for markdown and tabular handoffs.
                    </p>
                    <div className="row results-explorer__shape-export-controls">
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
                        onClick={() => {
                          setAssistantMode("shape");
                          aiAssistant.open({
                            surface: "results",
                            jobId,
                            resultFormat,
                            selectedResultIndex,
                            resultSummary,
                          });
                        }}
                      >
                        Open AI assistant
                      </button>
                      <button
                        type="button"
                        className="secondary"
                        onClick={() => {
                          setExportError(null);
                          setShapeConfigText("");
                          setShapeConfigError(null);
                        }}
                      >
                        Clear shape
                      </button>
                      <button
                        type="button"
                        onClick={() => void handleShapeExport()}
                      >
                        Export shaped result
                      </button>
                    </div>
                    <textarea
                      className="form-textarea results-explorer__shape-export-input"
                      rows={10}
                      value={shapeConfigText}
                      onChange={(event) => {
                        setExportError(null);
                        setShapeConfigText(event.target.value);
                        setShapeConfigError(null);
                      }}
                      placeholder='{"summaryFields":["title","url"],"normalizedFields":["field.price"]}'
                    />
                    {shapeConfigError ? (
                      <div className="transform-error">
                        Error: {shapeConfigError}
                      </div>
                    ) : null}
                  </div>
                </div>
              ) : null}
            </div>
          ) : null}
        </div>

        <ResultsAssistantSection
          jobId={jobId}
          jobType={jobType}
          resultFormat={resultFormat}
          selectedResultIndex={selectedResultIndex}
          resultSummary={resultSummary}
          selectedResult={resultItems[selectedResultIndex] ?? null}
          mode={assistantMode}
          onModeChange={setAssistantMode}
          shapeFormat={shapeExportFormat}
          onShapeFormatChange={setShapeExportFormat}
          currentShape={currentShapeConfig}
          onApplyShape={(shape) => {
            setIsExportOpen(false);
            setIsToolsOpen(true);
            setActiveTool("transform");
            setShapeConfigText(JSON.stringify(shape, null, 2));
            setShapeConfigError(null);
          }}
        />
      </div>
    </div>
  );
}

export default ResultsExplorer;
