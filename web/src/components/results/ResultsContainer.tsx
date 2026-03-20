/**
 * Purpose: Bridge route-owned results state into the lazy-loaded results explorer workspace.
 * Responsibilities: Select the active job metadata, pass authoritative result state into the explorer, and keep results rendering isolated from the application shell.
 * Scope: Results-route container only.
 * Usage: Render from `App.tsx` on `/jobs/:id` with the shared results hook state and the current jobs list.
 * Invariants/Assumptions: The route owner loads results before rendering this surface, the selected job ID comes from `useResultsState`, and the explorer owns reader-only UI state locally.
 */

import { lazy, Suspense, useMemo } from "react";
import type { ComponentStatus, Job } from "../../api";
import type { PromotionDestination } from "../../types/promotion";
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
  currentJob: Job | null;
  aiStatus?: ComponentStatus | null;
  onPromote: (
    destination: PromotionDestination,
    options?: {
      preferredExportFormat?: "json" | "jsonl" | "md" | "csv" | "xlsx";
    },
  ) => void;
}

export function ResultsContainer({
  resultsState,
  jobs,
  currentJob,
  aiStatus = null,
  onPromote,
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

  const availableJobs = useMemo(() => {
    if (!currentJob || jobs.some((job) => job.id === currentJob.id)) {
      return jobs;
    }

    return [currentJob, ...jobs];
  }, [currentJob, jobs]);

  const selectedJob = useMemo(
    () => currentJob ?? jobs.find((job) => job.id === selectedJobId) ?? null,
    [currentJob, jobs, selectedJobId],
  );

  return (
    <section id="results" data-tour="job-results">
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
          availableJobs={availableJobs}
          currentJob={selectedJob}
          jobType={selectedJob?.kind as "scrape" | "crawl" | "research"}
          aiStatus={aiStatus}
          onPromote={onPromote}
        />
      </Suspense>
    </section>
  );
}
