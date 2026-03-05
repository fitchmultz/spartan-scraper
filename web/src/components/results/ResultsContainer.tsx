/**
 * ResultsContainer - Container component for results viewing functionality
 *
 * This component encapsulates all results-related rendering:
 * - Displaying results explorer with lazy loading
 * - Managing results state integration
 *
 * It does NOT handle:
 * - Job submission
 * - Watch or chain management
 * - Batch operations
 *
 * @module ResultsContainer
 */

import { Suspense, lazy } from "react";
import type { Job } from "../../api";
import type { ResultsState, ResultsActions } from "../../hooks/useResultsState";

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
    rawResult,
    resultFormat,
    currentPage,
    totalResults,
    setCurrentPage,
  } = resultsState;

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
          rawResult={rawResult}
          resultFormat={resultFormat}
          currentPage={currentPage}
          totalResults={totalResults}
          resultsPerPage={100}
          onLoadPage={setCurrentPage}
          availableJobs={jobs}
        />
      </Suspense>
    </section>
  );
}
