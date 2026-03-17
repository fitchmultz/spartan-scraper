/**
 * Purpose: Bridge route-owned results state into the lazy-loaded results explorer workspace.
 * Responsibilities: Select the active job metadata, pass authoritative result state into the explorer, and keep results rendering isolated from the application shell.
 * Scope: Results-route container only.
 * Usage: Render from `App.tsx` on `/jobs/:id` with the shared results hook state and the current jobs list.
 * Invariants/Assumptions: The route owner loads results before rendering this surface, the selected job ID comes from `useResultsState`, and the explorer owns reader-only UI state locally.
 */

import { lazy, Suspense, useMemo } from "react";
import type { Job } from "../../api";
import {
  RESULTS_PER_PAGE,
  type ResultsActions,
  type ResultsState,
} from "../../hooks/useResultsState";

const ResultsExplorer = lazy(() =>
  import("../../components/ResultsExplorer").then((mod) => ({
    default: mod.ResultsExplorer,
  })),
);

interface ResultsContainerProps {
  resultsState: ResultsState & ResultsActions;
  jobs: Job[];
}

export function ResultsContainer({
  resultsState,
  jobs,
}: ResultsContainerProps) {
  const {
    selectedJobId,
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
    loadResults,
  } = resultsState;

  const selectedJob = useMemo(
    () => jobs.find((job) => job.id === selectedJobId) ?? null,
    [jobs, selectedJobId],
  );

  return (
    <section id="results">
      <Suspense
        fallback={
          <div className="loading-placeholder">Loading results explorer...</div>
        }
      >
        <ResultsExplorer
          jobId={selectedJobId}
          resultItems={resultItems}
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
          resultsPerPage={RESULTS_PER_PAGE}
          onLoadPage={(page) => {
            if (!selectedJobId) {
              return;
            }
            void loadResults(selectedJobId, resultFormat, page);
          }}
          availableJobs={jobs}
          jobType={selectedJob?.kind as "scrape" | "crawl" | "research"}
        />
      </Suspense>
    </section>
  );
}
