/**
 * Results State Hook
 *
 * Custom React hook for managing results loading, display, and pagination.
 * Handles result format switching, pagination, and extraction of detailed
 * result information (summary, confidence, evidence, clusters, citations).
 * Supports tree view state, search/filter, and diff comparison state.
 *
 * @module useResultsState
 */

import { useCallback, useEffect, useRef, useState } from "react";
import type {
  ResultItem,
  EvidenceItem,
  ClusterItem,
  CitationItem,
} from "../types";
import { loadResults as loadResultsUtil } from "../lib/results";

const RESULTS_PER_PAGE = 100;

export type ViewMode = "explorer" | "tree" | "diff" | "visualize";
export type StatusFilter = "all" | "success" | "error";

export interface ResultsState {
  selectedJobId: string | null;
  resultItems: ResultItem[];
  selectedResultIndex: number;
  resultSummary: string | null;
  resultConfidence: number | null;
  resultEvidence: EvidenceItem[];
  resultClusters: ClusterItem[];
  resultCitations: CitationItem[];
  rawResult: string | null;
  resultFormat: string;
  currentPage: number;
  totalResults: number;
  // Tree view state
  viewMode: ViewMode;
  treeExpandedIds: Set<string>;
  treeSelectedId: string | null;
  // Search/filter state
  searchQuery: string;
  statusFilter: StatusFilter;
}

export interface ResultsActions {
  loadResults: (jobId: string, format?: string, page?: number) => Promise<void>;
  updateResultFormat: (format: string) => void;
  setSelectedResultIndex: (index: number) => void;
  setCurrentPage: (page: number) => void;
  // Tree view actions
  setViewMode: (mode: ViewMode) => void;
  toggleTreeNode: (nodeId: string) => void;
  expandAllTreeNodes: () => void;
  collapseAllTreeNodes: () => void;
  setTreeSelectedId: (id: string | null) => void;
  // Search/filter actions
  setSearchQuery: (query: string) => void;
  setStatusFilter: (filter: StatusFilter) => void;
  clearFilters: () => void;
}

export function useResultsState(): ResultsState & ResultsActions {
  const [selectedJobId, setSelectedJobId] = useState<string | null>(null);
  const [resultItems, setResultItems] = useState<ResultItem[]>([]);
  const [selectedResultIndex, setSelectedResultIndex] = useState(0);
  const [resultSummary, setResultSummary] = useState<string | null>(null);
  const [resultConfidence, setResultConfidence] = useState<number | null>(null);
  const [resultEvidence, setResultEvidence] = useState<EvidenceItem[]>([]);
  const [resultClusters, setResultClusters] = useState<ClusterItem[]>([]);
  const [resultCitations, setResultCitations] = useState<CitationItem[]>([]);
  const [rawResult, setRawResult] = useState<string | null>(null);
  const [resultFormat, setResultFormat] = useState<string>("jsonl");
  const [currentPage, setCurrentPage] = useState(1);
  const [totalResults, setTotalResults] = useState(0);

  // Tree view state
  const [viewMode, setViewMode] = useState<ViewMode>("explorer");
  const [treeExpandedIds, setTreeExpandedIds] = useState<Set<string>>(
    new Set(),
  );
  const [treeSelectedId, setTreeSelectedId] = useState<string | null>(null);

  // Search/filter state
  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("all");

  const selectedJobIdRef = useRef<string | null>(null);

  const loadResults = useCallback(
    async (jobId: string, format = "jsonl", page = 1) => {
      setSelectedJobId(jobId);
      setResultFormat(format);

      if (page === 1) {
        setCurrentPage(1);
        setTotalResults(0);
        setResultItems([]);
        setSelectedResultIndex(0);
        setResultSummary(null);
        setResultConfidence(null);
        setResultEvidence([]);
        setResultClusters([]);
        setResultCitations([]);
        setRawResult(null);
      }

      const result = await loadResultsUtil(
        jobId,
        format,
        page,
        RESULTS_PER_PAGE,
      );

      if (result.error) {
        console.error(result.error);
        return;
      }

      if (format === "jsonl" && result.data) {
        if (typeof result.totalCount === "number") {
          setTotalResults(result.totalCount);
        }

        setResultItems(result.data as ResultItem[]);
        setRawResult(JSON.stringify(result.data, null, 2));
      } else if (result.raw) {
        setRawResult(result.raw);
        setResultItems([]);
      }
    },
    [],
  );

  const updateResultFormat = useCallback((format: string) => {
    setResultFormat(format);
  }, []);

  // Tree view actions
  const toggleTreeNode = useCallback((nodeId: string) => {
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

  const expandAllTreeNodes = useCallback(() => {
    // This will be populated when tree nodes are available
    setTreeExpandedIds(new Set());
  }, []);

  const collapseAllTreeNodes = useCallback(() => {
    setTreeExpandedIds(new Set());
  }, []);

  // Search/filter actions
  const clearFilters = useCallback(() => {
    setSearchQuery("");
    setStatusFilter("all");
  }, []);

  useEffect(() => {
    selectedJobIdRef.current = selectedJobId;
  }, [selectedJobId]);

  useEffect(() => {
    if (selectedJobId && resultFormat) {
      void loadResults(selectedJobId, resultFormat, 1);
    }
  }, [selectedJobId, resultFormat, loadResults]);

  useEffect(() => {
    if (resultItems.length === 0) {
      setResultSummary(null);
      setResultConfidence(null);
      setResultEvidence([]);
      setResultClusters([]);
      setResultCitations([]);
      return;
    }
    const item = resultItems[selectedResultIndex];
    if (item && "summary" in item) {
      setResultSummary((item as { summary?: string }).summary ?? null);
      setResultConfidence((item as { confidence?: number }).confidence ?? null);
      setResultEvidence((item as { evidence?: EvidenceItem[] }).evidence ?? []);
      setResultClusters((item as { clusters?: ClusterItem[] }).clusters ?? []);
      setResultCitations(
        (item as { citations?: CitationItem[] }).citations ?? [],
      );
    } else {
      setResultSummary(null);
      setResultConfidence(null);
      setResultEvidence([]);
      setResultClusters([]);
      setResultCitations([]);
    }
  }, [selectedResultIndex, resultItems]);

  return {
    selectedJobId,
    resultItems,
    selectedResultIndex,
    resultSummary,
    resultConfidence,
    resultEvidence,
    resultClusters,
    resultCitations,
    rawResult,
    resultFormat,
    currentPage,
    totalResults,
    // Tree view state
    viewMode,
    treeExpandedIds,
    treeSelectedId,
    // Search/filter state
    searchQuery,
    statusFilter,
    // Actions
    loadResults,
    updateResultFormat,
    setSelectedResultIndex,
    setCurrentPage,
    setViewMode,
    toggleTreeNode,
    expandAllTreeNodes,
    collapseAllTreeNodes,
    setTreeSelectedId,
    setSearchQuery,
    setStatusFilter,
    clearFilters,
  };
}
