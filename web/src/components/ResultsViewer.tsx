/**
 * Purpose: Render the dominant result-reading experience for a saved job.
 * Responsibilities: Show the selected-item navigator, primary detail panel, pagination controls, and secondary disclosures for raw output and research-specific context.
 * Scope: Results detail presentation only; search, export, and secondary tools stay outside this component.
 * Usage: Render from `ResultsExplorer` with the currently visible result page and selection state.
 * Invariants/Assumptions: A job ID exists before rendering, `selectedResultIndex` is relative to the provided `resultItems`, and research-only affordances stay hidden for crawl-style results.
 */

import { useEffect, useMemo, useState } from "react";
import { isCrawlResultItem, isResearchResultItem } from "../lib/form-utils";
import { getSimpleHttpStatusClass } from "../lib/http-status";
import type {
  AgenticResearchItem,
  CitationItem,
  ClusterItem,
  EvidenceItem,
  ResearchResultItem,
  ResultItem,
} from "../types";
import { NormalizedView } from "./NormalizedView";

function getFieldDisplayValues(
  field: NonNullable<EvidenceItem["fields"]>[string],
) {
  if (field.values && field.values.length > 0) {
    return field.values;
  }
  return [];
}

function renderEvidenceFields(fields: EvidenceItem["fields"]) {
  if (!fields || Object.keys(fields).length === 0) {
    return null;
  }

  return (
    <div className="results-viewer__field-block">
      <div className="results-viewer__section-label">Extracted fields</div>
      <div className="job-list">
        {Object.entries(fields).map(([name, field]) => {
          const values = getFieldDisplayValues(field);
          return (
            <div key={name} className="job-item">
              <div style={{ fontWeight: 600 }}>{name}</div>
              {values.length > 0 ? (
                <ul style={{ margin: "6px 0 0", paddingLeft: 18 }}>
                  {values.map((value) => (
                    <li key={`${name}-${value}`}>{value}</li>
                  ))}
                </ul>
              ) : (
                <div style={{ color: "var(--text-muted)" }}>
                  No string values returned.
                </div>
              )}
              {field.rawObject ? (
                <pre style={{ marginTop: 8, whiteSpace: "pre-wrap" }}>
                  {field.rawObject}
                </pre>
              ) : null}
            </div>
          );
        })}
      </div>
    </div>
  );
}

function truncateExcerpt(text?: string): string | null {
  if (!text) {
    return null;
  }

  const normalized = text.replace(/\s+/g, " ").trim();
  if (!normalized) {
    return null;
  }

  if (normalized.length <= 420) {
    return normalized;
  }

  return `${normalized.slice(0, 417)}…`;
}

function truncateText(text: string, maxLength: number): string {
  if (text.length <= maxLength) {
    return text;
  }

  return `${text.slice(0, Math.max(0, maxLength - 1)).trimEnd()}…`;
}

function getResearchResultTitle(
  item: ResearchResultItem,
  index: number,
): string {
  return item.query?.trim() || `Research result ${index + 1}`;
}

function getResearchSummaryPreview(
  summary?: string | null,
  maxLength = 180,
): string | null {
  const normalized = truncateExcerpt(summary ?? undefined);
  if (!normalized) {
    return null;
  }
  return truncateText(normalized, maxLength);
}

function getResearchResultMeta(
  item: ResearchResultItem,
  fallback: string,
): string {
  const parts: string[] = [];

  if (item.evidence && item.evidence.length > 0) {
    parts.push(`${item.evidence.length} evidence`);
  }
  if (item.clusters && item.clusters.length > 0) {
    parts.push(`${item.clusters.length} clusters`);
  }
  if (item.citations && item.citations.length > 0) {
    parts.push(`${item.citations.length} citations`);
  }

  return parts.join(" · ") || fallback;
}

function getResearchSourceLabels(evidence: EvidenceItem[]): string[] {
  const seen = new Set<string>();
  const labels: string[] = [];

  for (const item of evidence) {
    const label = (item.title || item.url || "").trim();
    if (!label || seen.has(label)) {
      continue;
    }
    seen.add(label);
    labels.push(label);
    if (labels.length >= 3) {
      break;
    }
  }

  return labels;
}

function hasStructuredSelectedItem(item: ResultItem | null): boolean {
  return (
    !!item &&
    isCrawlResultItem(item) &&
    !!(
      (item.normalized && Object.keys(item.normalized).length > 0) ||
      (item.extracted && Object.keys(item.extracted).length > 0) ||
      (item.metadata && Object.keys(item.metadata).length > 0)
    )
  );
}

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
            onChange={(event) => setJumpInputValue(event.target.value)}
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

      <div className="results-viewer__workspace">
        <aside className="results-viewer__navigator">
          <div className="results-viewer__navigator-header">
            <div>
              <div className="results-viewer__section-label">Navigator</div>
              <h5>Saved items</h5>
              <p className="form-help">
                Pick a result to update the detail panel without leaving this
                route.
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
                        {getResearchSummaryPreview(item.summary, 140) ? (
                          <div className="result-item-summary">
                            {getResearchSummaryPreview(item.summary, 140)}
                          </div>
                        ) : null}
                        <div className="result-item-meta">
                          {getResearchResultMeta(
                            item,
                            jobKind || "saved result",
                          )}
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

        <section className="results-viewer__detail">
          {selectedItem ? (
            <div className="results-viewer__detail-card">
              {isCrawlResultItem(selectedItem) ? (
                <>
                  <div className="results-viewer__detail-header">
                    <div>
                      <div className="results-viewer__section-label">
                        Selected page
                      </div>
                      <h4>{selectedItem.title || "Untitled page"}</h4>
                      <a
                        href={selectedItem.url}
                        target="_blank"
                        rel="noreferrer"
                      >
                        {selectedItem.url}
                      </a>
                    </div>

                    <span
                      className={`badge ${getSimpleHttpStatusClass(
                        selectedItem.status,
                        { emptyWhenZero: true },
                      )}`}
                    >
                      {selectedItem.status}
                    </span>
                  </div>

                  <div className="results-viewer__detail-meta">
                    <span>{selectedItem.links?.length ?? 0} links</span>
                    {selectedItem.metadata ? (
                      <span>
                        {Object.keys(selectedItem.metadata).length} metadata
                        fields
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
                      <div className="results-viewer__section-label">
                        Page text preview
                      </div>
                      <p className="results-viewer__lead">
                        {selectedCrawlExcerpt}
                      </p>
                    </div>
                  ) : (
                    <div className="results-viewer__empty-detail">
                      No extracted or normalized detail is available for this
                      item.
                    </div>
                  )}

                  {selectedItem.text ? (
                    <details className="results-viewer__disclosure">
                      <summary>Full page text</summary>
                      <pre>{selectedItem.text}</pre>
                    </details>
                  ) : null}
                </>
              ) : (
                <>
                  <div className="results-viewer__detail-header">
                    <div>
                      <div className="results-viewer__section-label">
                        Selected research result
                      </div>
                      <h4>
                        {selectedResearchResult?.query ||
                          `Research result ${selectedResultIndex + 1}`}
                      </h4>
                    </div>

                    {typeof selectedResearchResult?.confidence === "number" ? (
                      <span className="badge running">
                        Confidence{" "}
                        {selectedResearchResult.confidence.toFixed(2)}
                      </span>
                    ) : null}
                  </div>

                  <div className="results-viewer__detail-meta">
                    <span>
                      {selectedResearchEvidence.length} evidence items
                    </span>
                    <span>{selectedResearchClusters.length} clusters</span>
                    <span>{selectedResearchCitations.length} citations</span>
                  </div>

                  {selectedResearchSummary ? (
                    <div className="results-viewer__detail-section">
                      <div className="results-viewer__section-label">
                        Summary
                      </div>
                      <p className="results-viewer__lead">
                        {selectedResearchSummary}
                      </p>
                    </div>
                  ) : null}

                  {selectedResearchSourceLabels.length > 0 ? (
                    <div className="results-viewer__detail-section">
                      <div className="results-viewer__section-label">
                        Top sources
                      </div>
                      <div className="results-viewer__source-list">
                        {selectedResearchSourceLabels.map((label) => (
                          <span
                            key={label}
                            className="results-viewer__source-chip"
                          >
                            {label}
                          </span>
                        ))}
                      </div>
                    </div>
                  ) : null}

                  {jobKind === "research" && onOpenResearchAssistant ? (
                    <div className="results-explorer__actions">
                      <button
                        type="button"
                        className="secondary"
                        onClick={onOpenResearchAssistant}
                      >
                        Open AI assistant
                      </button>
                    </div>
                  ) : null}

                  <details className="results-viewer__disclosure">
                    <summary>Research insights</summary>

                    {selectedResearchClusters.length > 0 ? (
                      <div style={{ marginTop: 12 }}>
                        <div className="results-viewer__section-label">
                          Evidence clusters
                        </div>
                        <div className="job-list">
                          {selectedResearchClusters.map((cluster) => (
                            <div key={cluster.id} className="job-item">
                              <div>{cluster.label || cluster.id}</div>
                              <div className="badge running">
                                Confidence {cluster.confidence.toFixed(2)}
                              </div>
                              <div>{cluster.evidence.length} sources</div>
                            </div>
                          ))}
                        </div>
                      </div>
                    ) : null}

                    {selectedResearchCitations.length > 0 ? (
                      <div style={{ marginTop: 12 }}>
                        <div className="results-viewer__section-label">
                          Citations
                        </div>
                        <div className="job-list">
                          {selectedResearchCitations.map((citation) => {
                            const target =
                              citation.anchor && citation.canonical
                                ? `${citation.canonical}#${citation.anchor}`
                                : citation.canonical || citation.url || "";

                            return (
                              <div key={target} className="job-item">
                                <a
                                  href={target}
                                  target="_blank"
                                  rel="noreferrer"
                                >
                                  {target}
                                </a>
                              </div>
                            );
                          })}
                        </div>
                      </div>
                    ) : null}

                    {selectedResearchEvidence.length > 0 ? (
                      <div className="job-list" style={{ marginTop: 12 }}>
                        {selectedResearchEvidence.slice(0, 10).map((item) => (
                          <div
                            key={`${item.url}-${item.score}-${item.clusterId ?? ""}`}
                            className="job-item"
                          >
                            <div>{item.title || item.url}</div>
                            <div
                              style={{
                                display: "flex",
                                gap: 8,
                                flexWrap: "wrap",
                              }}
                            >
                              <div className="badge running">
                                Score {item.score.toFixed(2)}
                              </div>
                              {typeof item.confidence === "number" ? (
                                <div className="badge running">
                                  Confidence {item.confidence.toFixed(2)}
                                </div>
                              ) : null}
                              {item.clusterId ? (
                                <div className="badge running">
                                  {item.clusterId}
                                </div>
                              ) : null}
                            </div>
                            {item.citationUrl ? (
                              <a
                                href={item.citationUrl}
                                target="_blank"
                                rel="noreferrer"
                              >
                                {item.citationUrl}
                              </a>
                            ) : null}
                            <div>{item.snippet}</div>
                            {renderEvidenceFields(item.fields)}
                          </div>
                        ))}
                      </div>
                    ) : null}
                  </details>
                </>
              )}

              {selectedResearchAgentic ? (
                <details className="results-viewer__disclosure">
                  <summary>Agentic research details</summary>

                  <div
                    style={{
                      display: "flex",
                      gap: 8,
                      flexWrap: "wrap",
                      marginTop: 12,
                    }}
                  >
                    <div className="badge running">
                      Status {selectedResearchAgentic.status}
                    </div>
                    {typeof selectedResearchAgentic.confidence === "number" ? (
                      <div className="badge running">
                        Confidence{" "}
                        {selectedResearchAgentic.confidence.toFixed(2)}
                      </div>
                    ) : null}
                    {selectedResearchAgentic.provider &&
                    selectedResearchAgentic.model ? (
                      <div className="badge running">
                        {selectedResearchAgentic.provider}/
                        {selectedResearchAgentic.model}
                      </div>
                    ) : null}
                  </div>

                  {selectedResearchAgentic.summary ? (
                    <p>{selectedResearchAgentic.summary}</p>
                  ) : null}
                  {selectedResearchAgentic.error ? (
                    <div style={{ color: "var(--status-failed)" }}>
                      {selectedResearchAgentic.error}
                    </div>
                  ) : null}

                  {selectedResearchAgentic.focusAreas?.length ? (
                    <div style={{ marginTop: 8 }}>
                      <div className="results-viewer__section-label">
                        Focus areas
                      </div>
                      <ul>
                        {selectedResearchAgentic.focusAreas.map((item) => (
                          <li key={item}>{item}</li>
                        ))}
                      </ul>
                    </div>
                  ) : null}

                  {selectedResearchAgentic.keyFindings?.length ? (
                    <div style={{ marginTop: 8 }}>
                      <div className="results-viewer__section-label">
                        Key findings
                      </div>
                      <ul>
                        {selectedResearchAgentic.keyFindings.map((item) => (
                          <li key={item}>{item}</li>
                        ))}
                      </ul>
                    </div>
                  ) : null}

                  {selectedResearchAgentic.openQuestions?.length ? (
                    <div style={{ marginTop: 8 }}>
                      <div className="results-viewer__section-label">
                        Open questions
                      </div>
                      <ul>
                        {selectedResearchAgentic.openQuestions.map((item) => (
                          <li key={item}>{item}</li>
                        ))}
                      </ul>
                    </div>
                  ) : null}

                  {selectedResearchAgentic.recommendedNextSteps?.length ? (
                    <div style={{ marginTop: 8 }}>
                      <div className="results-viewer__section-label">
                        Recommended next steps
                      </div>
                      <ul>
                        {selectedResearchAgentic.recommendedNextSteps.map(
                          (item) => (
                            <li key={item}>{item}</li>
                          ),
                        )}
                      </ul>
                    </div>
                  ) : null}

                  {selectedResearchAgentic.followUpUrls?.length ? (
                    <div style={{ marginTop: 8 }}>
                      <div className="results-viewer__section-label">
                        Follow-up URLs
                      </div>
                      <ul>
                        {selectedResearchAgentic.followUpUrls.map((item) => (
                          <li key={item}>
                            <a href={item} target="_blank" rel="noreferrer">
                              {item}
                            </a>
                          </li>
                        ))}
                      </ul>
                    </div>
                  ) : null}

                  {selectedResearchAgentic.rounds?.length ? (
                    <div style={{ marginTop: 8 }}>
                      <div className="results-viewer__section-label">
                        Follow-up rounds
                      </div>
                      <div className="job-list">
                        {selectedResearchAgentic.rounds.map((round) => (
                          <div key={round.round} className="job-item">
                            <div>Round {round.round}</div>
                            {round.goal ? <div>{round.goal}</div> : null}
                            {round.selectedUrls?.length ? (
                              <div>
                                {round.selectedUrls.length} selected URL(s)
                              </div>
                            ) : null}
                            {typeof round.addedEvidenceCount === "number" ? (
                              <div>
                                {round.addedEvidenceCount} evidence item(s)
                                added
                              </div>
                            ) : null}
                          </div>
                        ))}
                      </div>
                    </div>
                  ) : null}
                </details>
              ) : null}

              <details className="results-viewer__disclosure">
                <summary>Raw job output</summary>
                <pre>{rawResult}</pre>
              </details>
            </div>
          ) : (
            <div className="results-viewer__empty-detail">
              <h5>No item selected</h5>
              <p className="form-help">
                Pick a result from the navigator to inspect it.
              </p>
            </div>
          )}
        </section>
      </div>
    </div>
  );
}
