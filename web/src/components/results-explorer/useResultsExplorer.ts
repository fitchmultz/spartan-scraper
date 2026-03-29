/**
 * Purpose: Compose the results-workspace selection and operations hooks into a single return value for the `/jobs/:id` route.
 * Responsibilities: Delegate to `useResultsSelectionState` and `useResultsOperationsState`, then merge their return values for the consuming component.
 * Scope: Composition only; all state and logic lives in the two focused hooks.
 * Usage: Called from `ResultsExplorer.tsx` after the route-level results props have been resolved.
 * Invariants/Assumptions: Both sub-hooks receive consistent inputs from the same route-level source.
 */

import type { EvidenceItem, Job, ResultItem } from "../../types";
import { useResultsOperationsState } from "./useResultsOperationsState";
import { useResultsSelectionState } from "./useResultsSelectionState";

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

export function useResultsExplorer(options: UseResultsExplorerOptions) {
  const selection = useResultsSelectionState({
    resultItems: options.resultItems,
    selectedResultIndex: options.selectedResultIndex,
    setSelectedResultIndex: options.setSelectedResultIndex,
    resultEvidence: options.resultEvidence,
    jobType: options.jobType,
    totalResults: options.totalResults,
  });
  const operations = useResultsOperationsState({
    ...options,
    filteredResultItems: selection.filteredResultItems,
    searchQuery: selection.searchQuery,
    statusFilter: selection.statusFilter,
    isResearchJob: selection.isResearchJob,
  });

  return { ...selection, ...operations };
}
