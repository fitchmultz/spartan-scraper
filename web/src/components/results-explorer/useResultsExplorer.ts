/**
 * Purpose: Own the stateful results-workspace behavior for the `/jobs/:id` route.
 * Responsibilities: Coordinate reader filters, selected-item continuity, secondary-tool state, comparison loading, export flows, and results-assistant state behind a single route-local hook.
 * Scope: Results workspace state only; rendering stays in `ResultsExplorer.tsx` and its child panels.
 * Usage: Called from `ResultsExplorer.tsx` after the route-level results props have been resolved.
 * Invariants/Assumptions: A selected job ID exists before export/compare actions run, filtered selection remains aligned with the visible reader list, and assistant actions only open through explicit operator commands.
 */

import { useEffect, useMemo, useState } from "react";

import type { ExportInspection, ExportShapeConfig } from "../../api";
import {
  type CrawlDiffResult,
  diffResults,
  type ResearchDiffResult,
} from "../../lib/diff-utils";
import { getApiErrorMessage } from "../../lib/api-errors";
import { exportResults, loadResults } from "../../lib/results";
import { buildPromotionOptions } from "../../lib/promotion";
import { buildUrlTree, type TreeNode } from "../../lib/tree-utils";
import type {
  CrawlResultItem,
  EvidenceItem,
  Job,
  ResultItem,
} from "../../types";
import { useAIAssistant, type ResultsAssistantMode } from "../ai-assistant";
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
} from "./resultsExplorerUtils";

interface UseResultsExplorerOptions {
  jobId: string | null;
  resultItems: ResultItem[];
  selectedResultIndex: number;
  setSelectedResultIndex: (index: number) => void;
  resultSummary: string | null;
  resultEvidence: EvidenceItem[];
  currentJob: Job | null;
  availableJobs: Job[];
  jobType: "scrape" | "crawl" | "research";
  resultFormat: string;
  totalResults: number;
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

export function useResultsExplorer({
  jobId,
  resultItems,
  selectedResultIndex,
  setSelectedResultIndex,
  resultSummary,
  resultEvidence,
  currentJob,
  availableJobs,
  jobType,
  resultFormat,
  totalResults,
}: UseResultsExplorerOptions) {
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

  const setSelectedVisibleResultIndex = (visibleIndex: number) => {
    const sourceIndex = filteredSourceIndexes[visibleIndex];
    if (typeof sourceIndex === "number") {
      setSelectedResultIndex(sourceIndex);
    }
  };

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
    if (!jobId) {
      return;
    }

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
    if (!jobId) {
      return;
    }

    setAssistantMode(mode);
    aiAssistant.open({
      surface: "results",
      jobId,
      resultFormat,
      selectedResultIndex,
      resultSummary,
    });
  };

  const clearReaderFilters = () => {
    setSearchQuery("");
    setStatusFilter("all");
  };

  const toggleToolsDrawer = () => {
    setIsToolsOpen((open) => !open);
    setIsExportOpen(false);
  };

  const toggleExportDrawer = () => {
    setIsExportOpen((open) => !open);
    setIsToolsOpen(false);
  };

  const openTransformTool = () => {
    setExportError(null);
    setIsExportOpen(false);
    setIsToolsOpen(true);
    setActiveTool("transform");
  };

  const clearShapeConfig = () => {
    setExportError(null);
    setShapeConfigText("");
    setShapeConfigError(null);
  };

  const updateShapeConfigText = (value: string) => {
    setExportError(null);
    setShapeConfigText(value);
    setShapeConfigError(null);
  };

  const applyAssistantShape = (shape: ExportShapeConfig) => {
    setIsExportOpen(false);
    setIsToolsOpen(true);
    setActiveTool("transform");
    setShapeConfigText(JSON.stringify(shape, null, 2));
    setShapeConfigError(null);
  };

  return {
    activeTool,
    activeToolConfig,
    applyAssistantShape,
    assistantMode,
    clearReaderFilters,
    clearShapeConfig,
    collapseAllTreeNodes,
    comparableJobs,
    compareJob,
    compareJobId,
    currentShapeConfig,
    diffError,
    diffLoading,
    diffResult,
    expandAllTreeNodes,
    exportError,
    exportOptions,
    filteredResultItems,
    filteredSourceIndexes,
    handleDirectExport,
    handleExportWithTransform,
    handleShapeExport,
    handleTreeSelect,
    handleTreeToggle,
    isExportOpen,
    isExporting,
    isResearchJob,
    isToolsOpen,
    latestExportOutcome,
    openResultsAssistant,
    openTransformTool,
    promotionOptions,
    searchQuery,
    secondaryTools,
    selectedClusterId,
    selectedEvidenceUrl,
    setActiveTool,
    setAssistantMode,
    setCompareJobId,
    setIsExportOpen,
    setIsToolsOpen,
    setSearchQuery,
    setSelectedClusterId,
    setSelectedEvidenceUrl,
    setSelectedVisibleResultIndex,
    setShapeExportFormat,
    setStatusFilter,
    shapeConfigError,
    shapeConfigText,
    shapeExportFormat,
    statusFilter,
    toggleExportDrawer,
    toggleToolsDrawer,
    totalVisibleResults: Math.max(totalResults, resultItems.length),
    treeExpandedIds,
    treeNodes,
    treeSelectedId,
    updateShapeConfigText,
    visibleSelectedIndex,
  };
}
