/**
 * Purpose: Render research-style result detail inside the dominant reader.
 * Responsibilities: Show selected research summary, sources, citations, evidence, and agentic follow-up detail without mixing crawl-only presentation.
 * Scope: Research-result detail presentation only.
 * Usage: Render from `ResultsViewer.tsx` when the selected reader item is a research-style result.
 * Invariants/Assumptions: Research detail is derived from the selected result first, then falls back to route-level aggregates, and agentic detail remains optional.
 */

import type {
  AgenticResearchItem,
  CitationItem,
  ClusterItem,
  EvidenceItem,
  ResearchResultItem,
} from "../../types";
import { renderEvidenceFields } from "./resultsReaderShared";

interface ResearchResultDetailProps {
  jobKind?: "scrape" | "crawl" | "research";
  selectedResultIndex: number;
  selectedResearchResult: ResearchResultItem | null;
  selectedResearchSummary: string | null;
  selectedResearchEvidence: EvidenceItem[];
  selectedResearchClusters: ClusterItem[];
  selectedResearchCitations: CitationItem[];
  selectedResearchAgentic: AgenticResearchItem | null;
  selectedResearchSourceLabels: string[];
  onOpenResearchAssistant?: () => void;
}

export function ResearchResultDetail({
  jobKind,
  selectedResultIndex,
  selectedResearchResult,
  selectedResearchSummary,
  selectedResearchEvidence,
  selectedResearchClusters,
  selectedResearchCitations,
  selectedResearchAgentic,
  selectedResearchSourceLabels,
  onOpenResearchAssistant,
}: ResearchResultDetailProps) {
  return (
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
            Confidence {selectedResearchResult.confidence.toFixed(2)}
          </span>
        ) : null}
      </div>

      <div className="results-viewer__detail-meta">
        <span>{selectedResearchEvidence.length} evidence items</span>
        <span>{selectedResearchClusters.length} clusters</span>
        <span>{selectedResearchCitations.length} citations</span>
      </div>

      {selectedResearchSummary ? (
        <div className="results-viewer__detail-section">
          <div className="results-viewer__section-label">Summary</div>
          <p className="results-viewer__lead">{selectedResearchSummary}</p>
        </div>
      ) : null}

      {selectedResearchSourceLabels.length > 0 ? (
        <div className="results-viewer__detail-section">
          <div className="results-viewer__section-label">Top sources</div>
          <div className="results-viewer__source-list">
            {selectedResearchSourceLabels.map((label) => (
              <span key={label} className="results-viewer__source-chip">
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
            <div className="results-viewer__section-label">Citations</div>
            <div className="job-list">
              {selectedResearchCitations.map((citation) => {
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

        {selectedResearchEvidence.length > 0 ? (
          <div className="job-list" style={{ marginTop: 12 }}>
            {selectedResearchEvidence.slice(0, 10).map((item) => (
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
      </details>

      {selectedResearchAgentic ? (
        <details className="results-viewer__disclosure">
          <summary>Agentic research details</summary>

          <div
            style={{ display: "flex", gap: 8, flexWrap: "wrap", marginTop: 12 }}
          >
            <div className="badge running">
              Status {selectedResearchAgentic.status}
            </div>
            {typeof selectedResearchAgentic.confidence === "number" ? (
              <div className="badge running">
                Confidence {selectedResearchAgentic.confidence.toFixed(2)}
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
              <div className="results-viewer__section-label">Focus areas</div>
              <ul>
                {selectedResearchAgentic.focusAreas.map((item) => (
                  <li key={item}>{item}</li>
                ))}
              </ul>
            </div>
          ) : null}

          {selectedResearchAgentic.keyFindings?.length ? (
            <div style={{ marginTop: 8 }}>
              <div className="results-viewer__section-label">Key findings</div>
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
                {selectedResearchAgentic.recommendedNextSteps.map((item) => (
                  <li key={item}>{item}</li>
                ))}
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
        </details>
      ) : null}
    </>
  );
}
