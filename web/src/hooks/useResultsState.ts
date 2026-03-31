/**
 * Purpose: Own the canonical saved-results loading state for the `/jobs/:id` route.
 * Responsibilities: Load saved job results, track pagination and selection, and derive selected-item summary metadata for the active result.
 * Scope: Results-route data state only; reader filters, secondary tools, and export UI stay local to the results explorer surface.
 * Usage: Call `useResultsState()` once from the application shell and pass the returned state/actions into `ResultsContainer`.
 * Invariants/Assumptions: `loadResults()` is the only authoritative fetch path, selection must stay within the currently loaded result page, and the route defaults to paginated `jsonl` inspection.
 */

import { useCallback, useMemo, useState } from "react";
import { loadResults as loadResultsUtil } from "../lib/results";
import { reportRuntimeError } from "../lib/runtime-errors";
import type {
  AgenticResearchItem,
  CitationItem,
  ClusterItem,
  EvidenceItem,
  ResultItem,
} from "../types";

export const RESULTS_PER_PAGE = 100;

export interface ResultsState {
  selectedJobId: string | null;
  resultItems: ResultItem[];
  selectedResultIndex: number;
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
}

export interface ResultsActions {
  loadResults: (jobId: string, format?: string, page?: number) => Promise<void>;
  setSelectedResultIndex: (index: number) => void;
}

interface SelectedResultSnapshot {
  resultSummary: string | null;
  resultConfidence: number | null;
  resultEvidence: EvidenceItem[];
  resultClusters: ClusterItem[];
  resultCitations: CitationItem[];
  resultAgentic: AgenticResearchItem | null;
}

function clampSelectedResultIndex(index: number, itemCount: number): number {
  if (itemCount <= 0) {
    return 0;
  }

  return Math.min(Math.max(index, 0), itemCount - 1);
}

function deriveSelectedResultSnapshot(
  item: ResultItem | null | undefined,
): SelectedResultSnapshot {
  if (!item || !("summary" in item)) {
    return {
      resultSummary: null,
      resultConfidence: null,
      resultEvidence: [],
      resultClusters: [],
      resultCitations: [],
      resultAgentic: null,
    };
  }

  return {
    resultSummary: (item as { summary?: string }).summary ?? null,
    resultConfidence: (item as { confidence?: number }).confidence ?? null,
    resultEvidence: (item as { evidence?: EvidenceItem[] }).evidence ?? [],
    resultClusters: (item as { clusters?: ClusterItem[] }).clusters ?? [],
    resultCitations: (item as { citations?: CitationItem[] }).citations ?? [],
    resultAgentic: (item as { agentic?: AgenticResearchItem }).agentic ?? null,
  };
}

export function useResultsState(): ResultsState & ResultsActions {
  const [selectedJobId, setSelectedJobId] = useState<string | null>(null);
  const [resultItems, setResultItems] = useState<ResultItem[]>([]);
  const [selectedResultIndexState, setSelectedResultIndex] = useState(0);
  const [rawResult, setRawResult] = useState<string | null>(null);
  const [resultFormat, setResultFormat] = useState<string>("jsonl");
  const [currentPage, setCurrentPage] = useState(1);
  const [totalResults, setTotalResults] = useState(0);

  const loadResults = useCallback(
    async (jobId: string, format = "jsonl", page = 1) => {
      setSelectedJobId(jobId);
      setResultFormat(format);
      setCurrentPage(page);
      setSelectedResultIndex(0);
      setResultItems([]);
      setRawResult(null);
      setTotalResults(0);

      const result = await loadResultsUtil(
        jobId,
        format,
        page,
        RESULTS_PER_PAGE,
      );

      if (result.error) {
        reportRuntimeError("Failed to load results", result.error, {
          fallback: result.error,
        });
        return;
      }

      if (format === "jsonl" && result.data) {
        if (typeof result.totalCount === "number") {
          setTotalResults(result.totalCount);
        }

        setResultItems(result.data as ResultItem[]);
        setRawResult(result.raw ?? JSON.stringify(result.data, null, 2));
        return;
      }

      setRawResult(result.raw ?? null);
    },
    [],
  );

  const selectedResultIndex = clampSelectedResultIndex(
    selectedResultIndexState,
    resultItems.length,
  );
  const selectedResultSnapshot = useMemo(
    () => deriveSelectedResultSnapshot(resultItems[selectedResultIndex]),
    [resultItems, selectedResultIndex],
  );

  return {
    selectedJobId,
    resultItems,
    selectedResultIndex,
    ...selectedResultSnapshot,
    rawResult,
    resultFormat,
    currentPage,
    totalResults,
    loadResults,
    setSelectedResultIndex,
  };
}
