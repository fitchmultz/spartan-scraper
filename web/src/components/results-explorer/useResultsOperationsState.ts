/**
 * Purpose: Own export, compare/diff, secondary-tools, and assistant operation state for the results workspace.
 * Responsibilities: Manage export flows (direct, transform, shape), diff comparison loading, drawer toggles, active tool selection, and AI assistant integration.
 * Scope: Side-effect operations only; selection and filtering state lives in `useResultsSelectionState`.
 * Usage: Called from `useResultsExplorer` with the selection state it depends on.
 * Invariants/Assumptions: A selected job ID exists before export/compare actions run; assistant actions only open through explicit operator commands.
 */

import { useEffect, useMemo, useState } from "react";

import type { ExportInspection, ExportShapeConfig } from "../../api";
import {
  type CrawlDiffResult,
  diffResults,
  type ResearchDiffResult,
} from "../../lib/diff-utils";
import { getApiErrorMessage } from "../../lib/api-errors";
import { downloadFile, exportResults, loadResults } from "../../lib/results";
import { buildPromotionOptions } from "../../lib/promotion";
import type { EvidenceItem, Job, ResultItem } from "../../types";
import { useAIAssistant, type ResultsAssistantMode } from "../ai-assistant";
import {
  findComparableJobs,
  getAvailableSecondaryTools,
  getExportGuidanceOptions,
  getJobByID,
  type ExportFormat,
  type SecondaryToolId,
  type StatusFilter,
} from "./resultsExplorerUtils";

export interface UseResultsOperationsStateOptions {
  jobId: string | null;
  resultItems: ResultItem[];
  selectedResultIndex: number;
  resultSummary: string | null;
  resultEvidence: EvidenceItem[];
  currentJob: Job | null;
  availableJobs: Job[];
  jobType: "scrape" | "crawl" | "research";
  resultFormat: string;
  totalResults: number;
  filteredResultItems: ResultItem[];
  searchQuery: string;
  statusFilter: StatusFilter;
  isResearchJob: boolean;
}

export function useResultsOperationsState({
  jobId,
  resultItems,
  selectedResultIndex,
  resultSummary,
  resultEvidence,
  currentJob,
  availableJobs,
  jobType,
  resultFormat,
  totalResults,
  filteredResultItems,
  searchQuery,
  statusFilter,
  isResearchJob,
}: UseResultsOperationsStateOptions) {
  const aiAssistant = useAIAssistant();
  const [assistantMode, setAssistantMode] = useState<ResultsAssistantMode>(
    jobType === "research" ? "research" : "shape",
  );
  const [isToolsOpen, setIsToolsOpen] = useState(false);
  const [isExportOpen, setIsExportOpen] = useState(false);
  const [activeTool, setActiveTool] = useState<SecondaryToolId | null>(null);
  const [compareJobId, setCompareJobId] = useState<string | null>(null);
  const [diffResult, setDiffResult] = useState<
    CrawlDiffResult | ResearchDiffResult | null
  >(null);
  const [diffLoading, setDiffLoading] = useState(false);
  const [diffError, setDiffError] = useState<string | null>(null);
  const [isExporting, setIsExporting] = useState(false);
  const [exportError, setExportError] = useState<string | null>(null);
  const [latestExportOutcome, setLatestExportOutcome] =
    useState<ExportInspection | null>(null);
  const [shapeExportFormat, setShapeExportFormat] = useState<
    "md" | "csv" | "xlsx"
  >("md");
  const [shapeConfigText, setShapeConfigText] = useState("");
  const [shapeConfigError, setShapeConfigError] = useState<string | null>(null);

  useEffect(() => {
    if (jobType !== "research" && assistantMode === "research") {
      setAssistantMode("shape");
    }
  }, [assistantMode, jobType]);

  const compareJob = useMemo(
    () => getJobByID(availableJobs, compareJobId),
    [availableJobs, compareJobId],
  );
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
    clearShapeConfig,
    comparableJobs,
    compareJob,
    compareJobId,
    currentShapeConfig,
    diffError,
    diffLoading,
    diffResult,
    exportError,
    exportOptions,
    handleDirectExport,
    handleExportWithTransform,
    handleShapeExport,
    isExportOpen,
    isExporting,
    isToolsOpen,
    latestExportOutcome,
    openResultsAssistant,
    openTransformTool,
    promotionOptions,
    secondaryTools,
    setActiveTool,
    setAssistantMode,
    setCompareJobId,
    setIsExportOpen,
    setIsToolsOpen,
    setShapeExportFormat,
    shapeConfigError,
    shapeConfigText,
    shapeExportFormat,
    toggleExportDrawer,
    toggleToolsDrawer,
    updateShapeConfigText,
  };
}
