/**
 * Purpose: Render crawl-style result detail inside the dominant reader.
 * Responsibilities: Show selected page identity, structured fields when available, excerpt fallback, and raw page text disclosure.
 * Scope: Crawl-result detail presentation only.
 * Usage: Render from `ResultsViewer.tsx` when the selected reader item is a crawl-style result.
 * Invariants/Assumptions: Structured data wins over excerpt fallback, and full page text only appears when the saved result includes it.
 */

import { getSimpleHttpStatusClass } from "../../lib/http-status";
import { NormalizedView } from "../NormalizedView";
import type { CrawlResultItem } from "../../types";
import { hasStructuredSelectedItem } from "./resultsReaderShared";

interface CrawlResultDetailProps {
  selectedItem: CrawlResultItem;
  selectedCrawlExcerpt: string | null;
}

export function CrawlResultDetail({
  selectedItem,
  selectedCrawlExcerpt,
}: CrawlResultDetailProps) {
  return (
    <>
      <div className="results-viewer__detail-header">
        <div>
          <div className="results-viewer__section-label">Selected page</div>
          <h4>{selectedItem.title || "Untitled page"}</h4>
          <a href={selectedItem.url} target="_blank" rel="noreferrer">
            {selectedItem.url}
          </a>
        </div>

        <span
          className={`badge ${getSimpleHttpStatusClass(selectedItem.status, {
            emptyWhenZero: true,
          })}`}
        >
          {selectedItem.status}
        </span>
      </div>

      <div className="results-viewer__detail-meta">
        <span>{selectedItem.links?.length ?? 0} links</span>
        {selectedItem.metadata ? (
          <span>
            {Object.keys(selectedItem.metadata).length} metadata fields
          </span>
        ) : null}
      </div>

      {hasStructuredSelectedItem(selectedItem) ? (
        <div className="results-viewer__detail-section">
          <div className="results-viewer__section-label">
            Selected item data
          </div>
          <NormalizedView item={selectedItem} />
        </div>
      ) : selectedCrawlExcerpt ? (
        <div className="results-viewer__detail-section">
          <div className="results-viewer__section-label">Page text preview</div>
          <p className="results-viewer__lead">{selectedCrawlExcerpt}</p>
        </div>
      ) : (
        <div className="results-viewer__empty-detail">
          No extracted or normalized detail is available for this item.
        </div>
      )}

      {selectedItem.text ? (
        <details className="results-viewer__disclosure">
          <summary>Full page text</summary>
          <pre>{selectedItem.text}</pre>
        </details>
      ) : null}
    </>
  );
}
