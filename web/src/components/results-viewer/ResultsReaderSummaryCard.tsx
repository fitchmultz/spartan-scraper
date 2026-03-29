/**
 * Purpose: Render the saved-results reader summary card above the navigator and detail panel.
 * Responsibilities: Present the dominant reader framing, summarize the selected research result when present, and surface the current reader-level badges.
 * Scope: Reader summary presentation only.
 * Usage: Render from `ResultsViewer.tsx` after selected-item state has been derived.
 * Invariants/Assumptions: Research-only source chips stay hidden for crawl results, and pagination badges only appear when the reader is paging JSONL output.
 */

import type { ResearchResultItem } from "../../types";

interface ResultsReaderSummaryCardProps {
  jobKind?: "scrape" | "crawl" | "research";
  selectedResearchResult: ResearchResultItem | null;
  selectedResearchPreview: string | null;
  selectedResearchSourceLabels: string[];
  resultSummary: string | null;
  visibleCountLabel: string;
  resultConfidence: number | null;
  totalResults: number;
  hasPagination: boolean;
}

export function ResultsReaderSummaryCard({
  jobKind,
  selectedResearchResult,
  selectedResearchPreview,
  selectedResearchSourceLabels,
  resultSummary,
  visibleCountLabel,
  resultConfidence,
  totalResults,
  hasPagination,
}: ResultsReaderSummaryCardProps) {
  return (
    <div className="results-viewer__summary-card">
      <div>
        <div className="results-viewer__section-label">Default reader</div>
        <h4>Understand the saved output before using secondary tools</h4>
        {selectedResearchResult?.query ? (
          <div className="results-viewer__summary-query">
            Query: {selectedResearchResult.query}
          </div>
        ) : null}
        {jobKind === "research" ? (
          selectedResearchPreview ? (
            <p className="results-viewer__lead">{selectedResearchPreview}</p>
          ) : (
            <p className="results-viewer__lead">
              Start with the query, coverage, and source list, then open the
              full detail panel when you need the complete synthesis.
            </p>
          )
        ) : resultSummary ? (
          <p className="results-viewer__lead">{resultSummary}</p>
        ) : (
          <p className="results-viewer__lead">
            Start with the navigator and detail panel, then open comparison,
            transform, structure, or visualization tools only when you need
            them.
          </p>
        )}
        {jobKind === "research" && selectedResearchSourceLabels.length > 0 ? (
          <div className="results-viewer__source-list">
            {selectedResearchSourceLabels.map((label) => (
              <span key={label} className="results-viewer__source-chip">
                {label}
              </span>
            ))}
          </div>
        ) : null}
      </div>

      <div className="results-viewer__badge-row">
        <span>{visibleCountLabel}</span>
        {typeof resultConfidence === "number" ? (
          <span className="badge running">
            Confidence {resultConfidence.toFixed(2)}
          </span>
        ) : null}
        {hasPagination ? <span>{totalResults} total results</span> : null}
        {jobKind ? <span>{jobKind} job</span> : null}
      </div>
    </div>
  );
}
