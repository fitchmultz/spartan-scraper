/**
 * Results Viewer Component
 *
 * Displays paginated job results with navigation, filtering, and detail views.
 * Handles both crawl results (with status codes, links, metadata) and research results
 * (with evidence clusters, citations, confidence scores). Supports result selection,
 * pagination, and raw/normalized data inspection.
 *
 * @module ResultsViewer
 */
import { useEffect, useState } from "react";
import { isCrawlResultItem } from "../lib/form-utils";
import { getSimpleHttpStatusClass } from "../lib/http-status";
import type {
  AgenticResearchItem,
  CitationItem,
  ClusterItem,
  EvidenceItem,
  ResearchResultItem,
  ResultItem,
} from "../types";
import { AIResearchRefiner } from "./AIResearchRefiner";
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
    <div style={{ marginTop: 8 }}>
      <div style={{ fontWeight: 600, marginBottom: 4 }}>Extracted fields</div>
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
}: ResultsViewerProps) {
  const [jumpInputValue, setJumpInputValue] = useState(currentPage.toString());
  const [isResearchRefinerOpen, setIsResearchRefinerOpen] = useState(false);
  const selectedItem = resultItems[selectedResultIndex] ?? null;
  const selectedResearchResult =
    selectedItem && !isCrawlResultItem(selectedItem)
      ? (selectedItem as ResearchResultItem)
      : null;

  useEffect(() => {
    setJumpInputValue(currentPage.toString());
  }, [currentPage]);

  useEffect(() => {
    if (!selectedResearchResult) {
      setIsResearchRefinerOpen(false);
    }
  }, [selectedResearchResult]);

  if (!jobId) {
    return null;
  }

  const maxPage = Math.ceil(totalResults / resultsPerPage);

  return (
    <div className="panel" style={{ marginTop: 16 }}>
      <h3>Results: {jobId}</h3>
      {resultItems.length > 1 ? (
        <div className="result-navigation">
          <div className="result-counter">
            Showing {selectedResultIndex + 1} of {resultItems.length} results
          </div>
          <div className="result-nav-buttons">
            <button
              type="button"
              className="secondary"
              onClick={() =>
                setSelectedResultIndex(Math.max(0, selectedResultIndex - 1))
              }
              disabled={selectedResultIndex === 0}
            >
              ← Previous
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
              Next →
            </button>
          </div>
        </div>
      ) : null}
      {typeof resultConfidence === "number" ? (
        <div className="badge running" style={{ marginBottom: 8 }}>
          Confidence {resultConfidence.toFixed(2)}
        </div>
      ) : null}
      {selectedResearchResult?.query ? (
        <div style={{ marginBottom: 8, color: "var(--text-muted)" }}>
          Query: {selectedResearchResult.query}
        </div>
      ) : null}
      {resultSummary ? <p>{resultSummary}</p> : null}
      {selectedResearchResult ? (
        <div style={{ marginBottom: 12 }}>
          <button
            type="button"
            className="secondary"
            onClick={() => setIsResearchRefinerOpen(true)}
          >
            Refine with AI
          </button>
        </div>
      ) : null}
      {resultAgentic ? (
        <div className="panel" style={{ marginTop: 12 }}>
          <h4>Agentic Research</h4>
          <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
            <div className="badge running">Status {resultAgentic.status}</div>
            {typeof resultAgentic.confidence === "number" ? (
              <div className="badge running">
                Confidence {resultAgentic.confidence.toFixed(2)}
              </div>
            ) : null}
            {resultAgentic.provider && resultAgentic.model ? (
              <div className="badge running">
                {resultAgentic.provider}/{resultAgentic.model}
              </div>
            ) : null}
          </div>
          {resultAgentic.summary ? <p>{resultAgentic.summary}</p> : null}
          {resultAgentic.error ? (
            <div style={{ color: "var(--status-failed)" }}>
              {resultAgentic.error}
            </div>
          ) : null}
          {resultAgentic.focusAreas?.length ? (
            <div style={{ marginTop: 8 }}>
              <div style={{ fontWeight: 600 }}>Focus areas</div>
              <ul>
                {resultAgentic.focusAreas.map((item) => (
                  <li key={item}>{item}</li>
                ))}
              </ul>
            </div>
          ) : null}
          {resultAgentic.keyFindings?.length ? (
            <div style={{ marginTop: 8 }}>
              <div style={{ fontWeight: 600 }}>Key findings</div>
              <ul>
                {resultAgentic.keyFindings.map((item) => (
                  <li key={item}>{item}</li>
                ))}
              </ul>
            </div>
          ) : null}
          {resultAgentic.openQuestions?.length ? (
            <div style={{ marginTop: 8 }}>
              <div style={{ fontWeight: 600 }}>Open questions</div>
              <ul>
                {resultAgentic.openQuestions.map((item) => (
                  <li key={item}>{item}</li>
                ))}
              </ul>
            </div>
          ) : null}
          {resultAgentic.recommendedNextSteps?.length ? (
            <div style={{ marginTop: 8 }}>
              <div style={{ fontWeight: 600 }}>Recommended next steps</div>
              <ul>
                {resultAgentic.recommendedNextSteps.map((item) => (
                  <li key={item}>{item}</li>
                ))}
              </ul>
            </div>
          ) : null}
          {resultAgentic.followUpUrls?.length ? (
            <div style={{ marginTop: 8 }}>
              <div style={{ fontWeight: 600 }}>Follow-up URLs</div>
              <ul>
                {resultAgentic.followUpUrls.map((item) => (
                  <li key={item}>
                    <a href={item} target="_blank" rel="noreferrer">
                      {item}
                    </a>
                  </li>
                ))}
              </ul>
            </div>
          ) : null}
          {resultAgentic.rounds?.length ? (
            <div style={{ marginTop: 8 }}>
              <div style={{ fontWeight: 600 }}>Follow-up rounds</div>
              <div className="job-list">
                {resultAgentic.rounds.map((round) => (
                  <div key={round.round} className="job-item">
                    <div>Round {round.round}</div>
                    {round.goal ? <div>{round.goal}</div> : null}
                    {round.selectedUrls?.length ? (
                      <div>{round.selectedUrls.length} selected URL(s)</div>
                    ) : null}
                    {typeof round.addedEvidenceCount === "number" ? (
                      <div>
                        {round.addedEvidenceCount} evidence item(s) added
                      </div>
                    ) : null}
                  </div>
                ))}
              </div>
            </div>
          ) : null}
        </div>
      ) : null}
      {resultClusters.length > 0 ? (
        <div style={{ marginTop: 12 }}>
          <h4>Evidence Clusters</h4>
          <div className="job-list">
            {resultClusters.map((cluster) => (
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
      {resultCitations.length > 0 ? (
        <div style={{ marginTop: 12 }}>
          <h4>Citations</h4>
          <div className="job-list">
            {resultCitations.map((citation) => {
              const target =
                citation.anchor && citation.canonical
                  ? `${citation.canonical}#${citation.anchor}`
                  : citation.canonical || citation.url || "";
              return (
                <div key={target} className="job-item">
                  <a href={target} target="_blank" rel="noreferrer">
                    {target}
                  </a>
                </div>
              );
            })}
          </div>
        </div>
      ) : null}
      {resultEvidence.length > 0 ? (
        <div className="job-list" style={{ marginTop: 12 }}>
          {resultEvidence.slice(0, 10).map((item) => (
            <div
              key={`${item.url}-${item.score}-${item.clusterId ?? ""}`}
              className="job-item"
            >
              <div>{item.title || item.url}</div>
              <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
                <div className="badge running">
                  Score {item.score.toFixed(2)}
                </div>
                {typeof item.confidence === "number" ? (
                  <div className="badge running">
                    Confidence {item.confidence.toFixed(2)}
                  </div>
                ) : null}
                {item.clusterId ? (
                  <div className="badge running">{item.clusterId}</div>
                ) : null}
              </div>
              {item.citationUrl ? (
                <a href={item.citationUrl} target="_blank" rel="noreferrer">
                  {item.citationUrl}
                </a>
              ) : null}
              <div>{item.snippet}</div>
              {renderEvidenceFields(item.fields)}
            </div>
          ))}
        </div>
      ) : null}
      {resultItems.length > 0 ? (
        <div style={{ marginTop: 12 }}>
          <h4>Results List</h4>
          {resultFormat === "jsonl" && totalResults > 0 ? (
            <div className="pagination-controls">
              <button
                type="button"
                disabled={currentPage <= 1}
                onClick={() => {
                  const newPage = currentPage - 1;
                  onLoadPage(newPage);
                }}
              >
                Previous
              </button>

              <span className="pagination-info">
                Page {currentPage} of {maxPage} ({totalResults} total results)
              </span>

              <button
                type="button"
                disabled={currentPage >= maxPage}
                onClick={() => {
                  const newPage = currentPage + 1;
                  onLoadPage(newPage);
                }}
              >
                Next
              </button>

              <div className="pagination-jump">
                <input
                  type="number"
                  min="1"
                  max={maxPage}
                  value={jumpInputValue}
                  onChange={(e) => {
                    const page = parseInt(e.target.value, 10);

                    if (
                      Number.isInteger(page) &&
                      page >= 1 &&
                      page <= maxPage
                    ) {
                      setJumpInputValue(e.target.value);
                    }
                  }}
                />
                <button
                  type="button"
                  onClick={() => {
                    const pageInput = document.querySelector(
                      ".pagination-jump input",
                    ) as HTMLInputElement;
                    const page = parseInt(pageInput.value, 10);
                    if (page >= 1 && page <= maxPage) {
                      onLoadPage(page);
                    }
                  }}
                >
                  Go
                </button>
              </div>
            </div>
          ) : null}
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
                        {item.title || "Untitled"}
                      </div>
                      {item.links?.length ? (
                        <div className="result-item-meta">
                          {item.links.length} links
                        </div>
                      ) : null}
                    </>
                  ) : (
                    <div className="result-item-non-crawl">
                      Result {index + 1} ({jobKind ?? "unknown"})
                    </div>
                  )}
                </button>
              );
            })}
          </div>
          {resultItems.length > 0 ? (
            <details style={{ marginTop: 12 }}>
              <summary>Normalized Data (Selected Item)</summary>
              <NormalizedView item={selectedItem} />
            </details>
          ) : null}
          <details style={{ marginTop: 8 }}>
            <summary>Raw output</summary>
            <pre>{rawResult}</pre>
          </details>
        </div>
      ) : null}
      <AIResearchRefiner
        isOpen={isResearchRefinerOpen}
        onClose={() => setIsResearchRefinerOpen(false)}
        result={selectedResearchResult}
      />
    </div>
  );
}
