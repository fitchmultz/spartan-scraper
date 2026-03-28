/**
 * Purpose: Render the dominant saved-results reader for a selected job.
 * Responsibilities: Own reader-level selection state wiring, derive the selected crawl/research context, and compose the summary, navigator, and detail sections.
 * Scope: Results detail presentation only; search, export, and secondary tools stay outside this component.
 * Usage: Render from `ResultsExplorer` with the currently visible result page and selection state.
 * Invariants/Assumptions: A job ID exists before rendering, `selectedResultIndex` is relative to the provided `resultItems`, and research-only affordances stay hidden for crawl-style results.
 */

import { useEffect, useMemo, useState } from "react";

import { isCrawlResultItem, isResearchResultItem } from "../lib/form-utils";
import type {
  AgenticResearchItem,
  CitationItem,
  ClusterItem,
  EvidenceItem,
  ResearchResultItem,
  ResultItem,
} from "../types";
import {
  getResearchSourceLabels,
  getResearchSummaryPreview,
  ResultsDetailPanel,
  ResultsNavigator,
  ResultsReaderSummaryCard,
  truncateExcerpt,
} from "./results-viewer/ResultsReaderSections";

interface ResultsViewerProps {
  jobId: string | null;
  jobKind?: "scrape" | "crawl" | "research";
  resultItems: ResultItem[];
  selectedResultIndex: number;
  setSelectedResultIndex: (index: number) => void;
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
  resultsPerPage: number;
  onLoadPage: (page: number) => void;
  onOpenResearchAssistant?: () => void;
}

export function ResultsViewer({
  jobId,
  jobKind,
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
  resultsPerPage,
  onLoadPage,
  onOpenResearchAssistant,
}: ResultsViewerProps) {
  const [jumpInputValue, setJumpInputValue] = useState(currentPage.toString());

  const selectedItem = resultItems[selectedResultIndex] ?? null;
  const selectedResearchResult =
    selectedItem && isResearchResultItem(selectedItem)
      ? (selectedItem as ResearchResultItem)
      : null;
  const selectedCrawlExcerpt = useMemo(
    () =>
      selectedItem && isCrawlResultItem(selectedItem)
        ? truncateExcerpt(selectedItem.text)
        : null,
    [selectedItem],
  );
  const selectedResearchSummary =
    selectedResearchResult?.summary ?? resultSummary ?? null;
  const selectedResearchEvidence =
    selectedResearchResult?.evidence ?? resultEvidence;
  const selectedResearchClusters =
    selectedResearchResult?.clusters ?? resultClusters;
  const selectedResearchCitations =
    selectedResearchResult?.citations ?? resultCitations;
  const selectedResearchAgentic =
    selectedResearchResult?.agentic ?? resultAgentic;
  const selectedResearchPreview = useMemo(
    () => getResearchSummaryPreview(selectedResearchSummary, 220),
    [selectedResearchSummary],
  );
  const selectedResearchSourceLabels = useMemo(
    () => getResearchSourceLabels(selectedResearchEvidence),
    [selectedResearchEvidence],
  );

  useEffect(() => {
    setJumpInputValue(currentPage.toString());
  }, [currentPage]);

  if (!jobId) {
    return null;
  }

  const maxPage = Math.max(1, Math.ceil(totalResults / resultsPerPage));
  const hasPagination = resultFormat === "jsonl" && maxPage > 1;
  const visibleCountLabel =
    resultItems.length === 1
      ? "1 visible item"
      : `${resultItems.length} visible items`;

  return (
    <div className="panel results-viewer">
      {resultItems.length > 1 ? (
        <div className="results-viewer__header">
          <div>
            <div className="results-viewer__section-label">Reader controls</div>
            <strong>
              Inspecting item {selectedResultIndex + 1} of {resultItems.length}
            </strong>
          </div>
          <div className="results-explorer__actions">
            <button
              type="button"
              className="secondary"
              onClick={() =>
                setSelectedResultIndex(Math.max(0, selectedResultIndex - 1))
              }
              disabled={selectedResultIndex === 0}
            >
              ← Previous item
            </button>
            <button
              type="button"
              className="secondary"
              onClick={() =>
                setSelectedResultIndex(
                  Math.min(resultItems.length - 1, selectedResultIndex + 1),
                )
              }
              disabled={selectedResultIndex === resultItems.length - 1}
            >
              Next item →
            </button>
          </div>
        </div>
      ) : null}

      <ResultsReaderSummaryCard
        jobKind={jobKind}
        selectedResearchResult={selectedResearchResult}
        selectedResearchPreview={selectedResearchPreview}
        selectedResearchSourceLabels={selectedResearchSourceLabels}
        resultSummary={resultSummary}
        visibleCountLabel={visibleCountLabel}
        resultConfidence={resultConfidence}
        totalResults={totalResults}
        hasPagination={hasPagination}
      />

      <div className="results-viewer__workspace">
        <ResultsNavigator
          jobKind={jobKind}
          resultItems={resultItems}
          selectedResultIndex={selectedResultIndex}
          currentPage={currentPage}
          maxPage={maxPage}
          totalResults={totalResults}
          hasPagination={hasPagination}
          visibleCountLabel={visibleCountLabel}
          jumpInputValue={jumpInputValue}
          onJumpInputChange={setJumpInputValue}
          onLoadPage={onLoadPage}
          setSelectedResultIndex={setSelectedResultIndex}
        />

        <ResultsDetailPanel
          jobKind={jobKind}
          selectedResultIndex={selectedResultIndex}
          selectedItem={selectedItem}
          selectedCrawlExcerpt={selectedCrawlExcerpt}
          selectedResearchResult={selectedResearchResult}
          selectedResearchSummary={selectedResearchSummary}
          selectedResearchEvidence={selectedResearchEvidence}
          selectedResearchClusters={selectedResearchClusters}
          selectedResearchCitations={selectedResearchCitations}
          selectedResearchAgentic={selectedResearchAgentic}
          selectedResearchSourceLabels={selectedResearchSourceLabels}
          rawResult={rawResult}
          onOpenResearchAssistant={onOpenResearchAssistant}
        />
      </div>
    </div>
  );
}
