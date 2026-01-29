/**
 * Results State Hook
 *
 * Custom React hook for managing results loading, display, and pagination.
 * Handles result format switching, pagination, and extraction of detailed
 * result information (summary, confidence, evidence, clusters, citations).
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
import { buildApiUrl } from "../lib/api-config";

const RESULTS_PER_PAGE = 100;

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
}

export interface ResultsActions {
  loadResults: (jobId: string, format?: string, page?: number) => Promise<void>;
  updateResultFormat: (format: string) => void;
  setSelectedResultIndex: (index: number) => void;
  setCurrentPage: (page: number) => void;
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
        try {
          const resultsUrl = buildApiUrl(
            `/v1/jobs/${jobId}/results?format=${format}&limit=${RESULTS_PER_PAGE}&offset=${(page - 1) * RESULTS_PER_PAGE}`,
          );
          const response = await fetch(resultsUrl, { method: "HEAD" });
          const totalCountStr = response.headers.get("X-Total-Count");
          if (totalCountStr) {
            setTotalResults(parseInt(totalCountStr, 10));
          }
        } catch {
          // Ignore header fetch errors; results are still valid
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
    loadResults,
    updateResultFormat,
    setSelectedResultIndex,
    setCurrentPage,
  };
}
