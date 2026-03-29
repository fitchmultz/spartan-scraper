/**
 * Purpose: Render the saved-results navigator rail for the dominant reader.
 * Responsibilities: Show pagination controls when needed, present crawl/research result cards, and drive selected-item changes without leaving the reader route.
 * Scope: Reader navigator presentation only.
 * Usage: Render from `ResultsViewer.tsx` with the currently visible result page.
 * Invariants/Assumptions: Selected indexes are relative to the provided visible result set, and pagination only applies to paged JSONL output.
 */

import { getSimpleHttpStatusClass } from "../../lib/http-status";
import type { ResultItem } from "../../types";
import { isCrawlResultItem } from "../../lib/form-utils";
import {
  getResearchResultMeta,
  getResearchResultTitle,
  getResearchSummaryPreview,
} from "./resultsReaderShared";

interface ResultsNavigatorProps {
  jobKind?: "scrape" | "crawl" | "research";
  resultItems: ResultItem[];
  selectedResultIndex: number;
  currentPage: number;
  maxPage: number;
  totalResults: number;
  hasPagination: boolean;
  visibleCountLabel: string;
  jumpInputValue: string;
  onJumpInputChange: (value: string) => void;
  onLoadPage: (page: number) => void;
  setSelectedResultIndex: (index: number) => void;
}

export function ResultsNavigator({
  jobKind,
  resultItems,
  selectedResultIndex,
  currentPage,
  maxPage,
  totalResults,
  hasPagination,
  visibleCountLabel,
  jumpInputValue,
  onJumpInputChange,
  onLoadPage,
  setSelectedResultIndex,
}: ResultsNavigatorProps) {
  const renderPaginationControls = () => {
    if (!hasPagination) {
      return null;
    }

    return (
      <div className="pagination-controls">
        <button
          type="button"
          disabled={currentPage <= 1}
          onClick={() => onLoadPage(currentPage - 1)}
        >
          Previous page
        </button>

        <span className="pagination-info">
          Page {currentPage} of {maxPage} ({totalResults} total results)
        </span>

        <button
          type="button"
          disabled={currentPage >= maxPage}
          onClick={() => onLoadPage(currentPage + 1)}
        >
          Next page
        </button>

        <div className="pagination-jump">
          <input
            type="number"
            min="1"
            max={maxPage}
            value={jumpInputValue}
            onChange={(event) => onJumpInputChange(event.target.value)}
          />
          <button
            type="button"
            onClick={() => {
              const page = Number.parseInt(jumpInputValue, 10);
              if (Number.isInteger(page) && page >= 1 && page <= maxPage) {
                onLoadPage(page);
              }
            }}
          >
            Go
          </button>
        </div>
      </div>
    );
  };

  return (
    <aside className="results-viewer__navigator">
      <div className="results-viewer__navigator-header">
        <div>
          <div className="results-viewer__section-label">Navigator</div>
          <h5>Saved items</h5>
          <p className="form-help">
            Pick a result to update the detail panel without leaving this route.
          </p>
        </div>
        <div className="results-viewer__badge-row">
          <span>{visibleCountLabel}</span>
        </div>
      </div>

      {renderPaginationControls()}

      {resultItems.length > 0 ? (
        <div className="result-items-list">
          {resultItems.map((item, index) => {
            const isCrawl = isCrawlResultItem(item);
            const itemKey = isCrawl ? item.url : `result-${index}`;
            const researchSummaryPreview = !isCrawl
              ? getResearchSummaryPreview(item.summary, 140)
              : null;

            return (
              <button
                key={itemKey}
                type="button"
                className={`result-item ${
                  index === selectedResultIndex ? "selected" : ""
                }`}
                onClick={() => setSelectedResultIndex(index)}
              >
                {isCrawl ? (
                  <>
                    <div className="result-item-header">
                      <span className="result-item-url">{item.url}</span>
                      <span
                        className={`badge ${getSimpleHttpStatusClass(
                          item.status,
                          { emptyWhenZero: true },
                        )}`}
                      >
                        {item.status}
                      </span>
                    </div>
                    <div className="result-item-title">
                      {item.title || "Untitled page"}
                    </div>
                    <div className="result-item-meta">
                      {item.links?.length ?? 0} links
                    </div>
                  </>
                ) : (
                  <>
                    <div className="result-item-title">
                      {getResearchResultTitle(item, index)}
                    </div>
                    {researchSummaryPreview ? (
                      <div className="result-item-summary">
                        {researchSummaryPreview}
                      </div>
                    ) : null}
                    <div className="result-item-meta">
                      {getResearchResultMeta(item, jobKind || "saved result")}
                    </div>
                  </>
                )}
              </button>
            );
          })}
        </div>
      ) : (
        <div className="results-viewer__empty-detail">
          <h5>No results match the current reader filters</h5>
          <p className="form-help">
            Clear the search or status filter to inspect more saved output.
          </p>
        </div>
      )}
    </aside>
  );
}
