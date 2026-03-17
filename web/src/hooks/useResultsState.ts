/**
 * Purpose: Own the canonical saved-results loading state for the `/jobs/:id` route.
 * Responsibilities: Load saved job results, track pagination and selection, and derive selected-item summary metadata for the active result.
 * Scope: Results-route data state only; reader filters, secondary tools, and export UI stay local to the results explorer surface.
 * Usage: Call `useResultsState()` once from the application shell and pass the returned state/actions into `ResultsContainer`.
 * Invariants/Assumptions: `loadResults()` is the only authoritative fetch path, selection must stay within the currently loaded result page, and the route defaults to paginated `jsonl` inspection.
 */

import { useCallback, useEffect, useState } from "react";
import { loadResults as loadResultsUtil } from "../lib/results";
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

function resetSelectedResultState(
  setResultSummary: (value: string | null) => void,
  setResultConfidence: (value: number | null) => void,
  setResultEvidence: (value: EvidenceItem[]) => void,
  setResultClusters: (value: ClusterItem[]) => void,
  setResultCitations: (value: CitationItem[]) => void,
  setResultAgentic: (value: AgenticResearchItem | null) => void,
) {
  setResultSummary(null);
  setResultConfidence(null);
  setResultEvidence([]);
  setResultClusters([]);
  setResultCitations([]);
  setResultAgentic(null);
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
  const [resultAgentic, setResultAgentic] =
    useState<AgenticResearchItem | null>(null);
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
      resetSelectedResultState(
        setResultSummary,
        setResultConfidence,
        setResultEvidence,
        setResultClusters,
        setResultCitations,
        setResultAgentic,
      );

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
        setRawResult(result.raw ?? JSON.stringify(result.data, null, 2));
        return;
      }

      setRawResult(result.raw ?? null);
    },
    [],
  );

  useEffect(() => {
    if (resultItems.length === 0) {
      resetSelectedResultState(
        setResultSummary,
        setResultConfidence,
        setResultEvidence,
        setResultClusters,
        setResultCitations,
        setResultAgentic,
      );
      return;
    }

    const clampedIndex = Math.min(
      Math.max(selectedResultIndex, 0),
      resultItems.length - 1,
    );
    if (clampedIndex !== selectedResultIndex) {
      setSelectedResultIndex(clampedIndex);
      return;
    }

    const item = resultItems[clampedIndex];
    if (item && "summary" in item) {
      setResultSummary((item as { summary?: string }).summary ?? null);
      setResultConfidence((item as { confidence?: number }).confidence ?? null);
      setResultEvidence((item as { evidence?: EvidenceItem[] }).evidence ?? []);
      setResultClusters((item as { clusters?: ClusterItem[] }).clusters ?? []);
      setResultCitations(
        (item as { citations?: CitationItem[] }).citations ?? [],
      );
      setResultAgentic(
        (item as { agentic?: AgenticResearchItem }).agentic ?? null,
      );
      return;
    }

    resetSelectedResultState(
      setResultSummary,
      setResultConfidence,
      setResultEvidence,
      setResultClusters,
      setResultCitations,
      setResultAgentic,
    );
  }, [resultItems, selectedResultIndex]);

  return {
    selectedJobId,
    resultItems,
    selectedResultIndex,
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
    loadResults,
    setSelectedResultIndex,
  };
}
