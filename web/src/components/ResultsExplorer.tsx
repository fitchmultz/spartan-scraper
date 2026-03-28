/**
 * Purpose: Provide the result-focused workspace for `/jobs/:id` with one dominant default reader.
 * Responsibilities: Coordinate reader filters, selected-item continuity, export flows, comparison tools, tree navigation, diff loading, and assistant state around the primary reader.
 * Scope: Results exploration UI only; route framing and authoritative result fetching stay outside this component.
 * Usage: Render from `ResultsContainer` with the active job's saved results and the surrounding jobs list for compare workflows.
 * Invariants/Assumptions: A selected job ID exists before rendering, the default reader always remains visible on first paint, and comparison/export actions operate on saved job results.
 */

import { useEffect, useMemo, useState } from "react";

import type {
  ComponentStatus,
  ExportInspection,
  ExportShapeConfig,
} from "../api";
import {
  type CrawlDiffResult,
  diffResults,
  type ResearchDiffResult,
} from "../lib/diff-utils";
import { getApiErrorMessage } from "../lib/api-errors";
import { exportResults, loadResults } from "../lib/results";
import { buildPromotionOptions } from "../lib/promotion";
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
import type { PromotionDestination } from "../types/promotion";
import { useAIAssistant, type ResultsAssistantMode } from "./ai-assistant";
import { ResultsViewer } from "./ResultsViewer";
import { JobPromotionPanel } from "./promotion/JobPromotionPanel";
import {
  ExportOutcomeSummary,
  GuidedExportDrawer,
  ReaderToolbar,
  ResultsAssistantRail,
  ResultsToolPanel,
  SecondaryToolsDrawer,
} from "./results-explorer/ResultsExplorerPanels";
import {
  buildDefaultExpandedTreeIds,
  collectTreeNodeIds,
  type ExportFormat,
  filterResultItems,
  findComparableJobs,
  getAvailableSecondaryTools,
  getExportGuidanceOptions,
  getJobByID,
  hasResearchVisualization,
  isCrawlResult,
  type SecondaryToolId,
  type StatusFilter,
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
  resultAgentic: AgenticResearchItem | null;
  rawResult: string | null;
  resultFormat: string;
  currentPage: number;
  totalResults: number;
  resultsPerPage: number;
  onLoadPage: (page: number) => void;
  availableJobs: Job[];
  currentJob: Job | null;
  jobType?: "scrape" | "crawl" | "research";
  aiStatus?: ComponentStatus | null;
  onPromote: (
    destination: PromotionDestination,
    options?: {
      preferredExportFormat?: "json" | "jsonl" | "md" | "csv" | "xlsx";
    },
  ) => void;
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
  for (let index = 0; index < binaryString.length; index++) {
    bytes[index] = binaryString.charCodeAt(index);
  }
  return bytes.buffer;
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
  currentJob,
  jobType = "crawl",
  aiStatus = null,
  onPromote,
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
  const [latestExportOutcome, setLatestExportOutcome] =
    useState<ExportInspection | null>(null);
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
  const promotionOptions = useMemo(
    () =>
      currentJob
        ? buildPromotionOptions(
            currentJob,
            latestExportOutcome?.request.format as
              | "json"
              | "jsonl"
              | "md"
              | "csv"
              | "xlsx"
              | undefined,
          )
        : [],
    [currentJob, latestExportOutcome?.request.format],
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
      } catch (error) {
        setDiffError(
          getApiErrorMessage(
            error,
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
    setExportError(null);
    setLatestExportOutcome(null);
    setIsExporting(true);
    try {
      const result = await exportResults(jobId, { format });
      setLatestExportOutcome(result.outcome);
      if (result.outcome.status === "succeeded" && result.content) {
        downloadFile(
          result.content,
          result.filename,
          result.contentType,
          result.isBinary,
        );
      }
    } catch (error) {
      setExportError(
        getApiErrorMessage(error, "Failed to export the saved result."),
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
    setExportError(null);
    setShapeConfigError(null);
    setLatestExportOutcome(null);
    setIsExporting(true);
    try {
      const result = await exportResults(jobId, {
        format,
        transform: {
          expression,
          language,
        },
      });
      setLatestExportOutcome(result.outcome);
      if (result.outcome.status === "succeeded" && result.content) {
        downloadFile(
          result.content,
          result.filename,
          result.contentType,
          result.isBinary,
        );
      }
    } catch (error) {
      setExportError(
        getApiErrorMessage(error, "Failed to export the transformed result."),
      );
    } finally {
      setIsExporting(false);
    }
  };

  const handleShapeExport = async () => {
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
    setLatestExportOutcome(null);
    setIsExporting(true);
    try {
      const result = await exportResults(jobId, {
        format: shapeExportFormat,
        shape,
      });
      setLatestExportOutcome(result.outcome);
      if (result.outcome.status === "succeeded" && result.content) {
        downloadFile(
          result.content,
          result.filename,
          result.contentType,
          result.isBinary,
        );
      }
    } catch (error) {
      setShapeConfigError(
        getApiErrorMessage(error, "Failed to export the shaped result."),
      );
    } finally {
      setIsExporting(false);
    }
  };

  const openResultsAssistant = (mode: ResultsAssistantMode) => {
    setAssistantMode(mode);
    aiAssistant.open({
      surface: "results",
      jobId,
      resultFormat,
      selectedResultIndex,
      resultSummary,
    });
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
                onClick={() => openResultsAssistant("shape")}
              >
                Open AI assistant
              </button>
            </div>
          </div>

          {currentJob?.status === "succeeded" ? (
            <JobPromotionPanel
              options={promotionOptions}
              onPromote={(destination) => onPromote(destination)}
            />
          ) : null}

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
              onSelectTool={setActiveTool}
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

          {latestExportOutcome ? (
            <ExportOutcomeSummary
              outcome={latestExportOutcome}
              onPromoteExportSchedule={(preferredExportFormat) =>
                onPromote("export-schedule", { preferredExportFormat })
              }
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
            onOpenResearchAssistant={() => openResultsAssistant("research")}
          />

          <ResultsToolPanel
            activeTool={activeTool}
            activeToolConfig={activeToolConfig}
            treeNodes={treeNodes}
            treeSelectedId={treeSelectedId}
            treeExpandedIds={treeExpandedIds}
            searchQuery={searchQuery}
            statusFilter={statusFilter}
            compareJobId={compareJobId}
            comparableJobs={comparableJobs}
            currentJob={currentJob}
            compareJob={compareJob}
            diffResult={diffResult}
            diffLoading={diffLoading}
            diffError={diffError}
            isResearchJob={isResearchJob}
            resultEvidence={resultEvidence}
            resultClusters={resultClusters}
            selectedEvidenceUrl={selectedEvidenceUrl}
            selectedClusterId={selectedClusterId}
            exportError={exportError}
            jobId={jobId}
            aiStatus={aiStatus}
            shapeExportFormat={shapeExportFormat}
            shapeConfigText={shapeConfigText}
            shapeConfigError={shapeConfigError}
            onCloseTool={() => setActiveTool(null)}
            onExpandAllTreeNodes={expandAllTreeNodes}
            onCollapseAllTreeNodes={collapseAllTreeNodes}
            onTreeSelect={handleTreeSelect}
            onTreeToggle={handleTreeToggle}
            onChangeCompareJobID={setCompareJobId}
            onSelectEvidenceUrl={setSelectedEvidenceUrl}
            onSelectClusterId={setSelectedClusterId}
            onTransformApply={(format, expression, language) => {
              void handleExportWithTransform(format, expression, language);
            }}
            onShapeFormatChange={setShapeExportFormat}
            onOpenShapeAssistant={() => openResultsAssistant("shape")}
            onClearShape={() => {
              setExportError(null);
              setShapeConfigText("");
              setShapeConfigError(null);
            }}
            onShapeConfigTextChange={(value) => {
              setExportError(null);
              setShapeConfigText(value);
              setShapeConfigError(null);
            }}
            onShapeExport={() => {
              void handleShapeExport();
            }}
          />
        </div>

        <ResultsAssistantRail
          jobId={jobId}
          jobType={jobType}
          resultFormat={resultFormat}
          aiStatus={aiStatus}
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
