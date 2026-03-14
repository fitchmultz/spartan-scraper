import { useState } from "react";
import { aiResearchRefine, type AiResearchRefineResponse } from "../api";
import { getApiBaseUrl } from "../lib/api-config";
import type { ResearchResultItem } from "../types";

interface AIResearchRefinerProps {
  isOpen: boolean;
  onClose: () => void;
  result: ResearchResultItem | null;
}

interface RefinerState {
  instructions: string;
  isLoading: boolean;
  response: AiResearchRefineResponse | null;
  error: string | null;
}

const initialState: RefinerState = {
  instructions: "",
  isLoading: false,
  response: null,
  error: null,
};

export function AIResearchRefiner({
  isOpen,
  onClose,
  result,
}: AIResearchRefinerProps) {
  const [state, setState] = useState<RefinerState>(initialState);

  if (!isOpen) {
    return null;
  }

  const handleClose = () => {
    setState(initialState);
    onClose();
  };

  const handleRefine = async () => {
    if (!result) {
      setState((prev) => ({
        ...prev,
        error: "Select a research result before running AI refinement.",
      }));
      return;
    }

    setState((prev) => ({
      ...prev,
      isLoading: true,
      response: null,
      error: null,
    }));

    try {
      const { data, error: apiError } = await aiResearchRefine({
        baseUrl: getApiBaseUrl(),
        body: {
          result,
          instructions: state.instructions.trim() || undefined,
        },
      });

      if (apiError) {
        const errorMessage =
          typeof apiError === "object" && apiError !== null
            ? (apiError as { error?: string; message?: string }).error ||
              (apiError as { error?: string; message?: string }).message ||
              String(apiError)
            : String(apiError);
        throw new Error(errorMessage);
      }

      setState((prev) => ({
        ...prev,
        isLoading: false,
        response: (data as AiResearchRefineResponse) || null,
      }));
    } catch (error) {
      setState((prev) => ({
        ...prev,
        isLoading: false,
        error:
          error instanceof Error
            ? error.message
            : "Failed to refine research result",
      }));
    }
  };

  const refined = state.response?.refined;

  return (
    // biome-ignore lint/a11y/noStaticElementInteractions: modal overlay pattern
    // biome-ignore lint/a11y/useKeyWithClickEvents: handled by close button and overlay click
    <div className="modal-overlay" onClick={handleClose}>
      {/* biome-ignore lint/a11y/noStaticElementInteractions: modal content container */}
      {/* biome-ignore lint/a11y/useKeyWithClickEvents: handled by parent modal semantics */}
      <div
        className="modal-content modal-content--large"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="modal-header">
          <h2 className="modal-title">
            <span className="mr-2 text-purple-400">✨</span>
            Refine Research Result with AI
          </h2>
          <button
            type="button"
            className="modal-close"
            onClick={handleClose}
            aria-label="Close"
          >
            ×
          </button>
        </div>

        <div className="modal-body space-y-4">
          <div className="form-section">
            <div className="form-group">
              <label
                htmlFor="ai-research-refine-instructions"
                className="form-label"
              >
                Instructions (optional)
              </label>
              <textarea
                id="ai-research-refine-instructions"
                value={state.instructions}
                onChange={(event) =>
                  setState((prev) => ({
                    ...prev,
                    instructions: event.target.value,
                  }))
                }
                rows={4}
                className="form-textarea"
                placeholder="Condense this into an operator-ready brief focused on the strongest evidence and remaining gaps."
                disabled={state.isLoading}
              />
              <p className="form-help">
                Spartan sends only the selected research result. The AI does not
                browse, fetch, or expand the evidence set.
              </p>
            </div>

            {result ? (
              <div className="job-list" style={{ marginTop: 12 }}>
                <div className="job-item">
                  <div style={{ fontWeight: 600 }}>
                    Selected research result
                  </div>
                  {result.query ? <div>Query: {result.query}</div> : null}
                  {typeof result.confidence === "number" ? (
                    <div>Confidence: {result.confidence.toFixed(2)}</div>
                  ) : null}
                  <div>
                    Evidence items: {result.evidence?.length ?? 0} · Clusters:{" "}
                    {result.clusters?.length ?? 0} · Citations:{" "}
                    {result.citations?.length ?? 0}
                  </div>
                </div>
              </div>
            ) : null}

            {state.error ? (
              <div className="form-error" role="alert">
                {state.error}
              </div>
            ) : null}
          </div>

          <div className="modal-footer gap-3">
            <button
              type="button"
              className="secondary"
              onClick={handleClose}
              disabled={state.isLoading}
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={() => void handleRefine()}
              disabled={state.isLoading || !result}
            >
              {state.isLoading ? "Refining…" : "Refine Result"}
            </button>
          </div>

          {state.response ? (
            <div className="form-section">
              <div className="flex gap-2" style={{ flexWrap: "wrap" }}>
                {state.response.route_id ? (
                  <div className="badge running">{state.response.route_id}</div>
                ) : null}
                {state.response.provider && state.response.model ? (
                  <div className="badge running">
                    {state.response.provider}/{state.response.model}
                  </div>
                ) : null}
                {typeof refined?.confidence === "number" ? (
                  <div className="badge running">
                    Confidence {refined.confidence.toFixed(2)}
                  </div>
                ) : null}
              </div>

              {state.response.issues?.length ? (
                <div style={{ marginTop: 12 }}>
                  <div style={{ fontWeight: 600 }}>Input diagnostics</div>
                  <ul>
                    {state.response.issues.map((issue) => (
                      <li key={issue}>{issue}</li>
                    ))}
                  </ul>
                </div>
              ) : null}

              {state.response.inputStats ? (
                <div style={{ marginTop: 12 }}>
                  <div style={{ fontWeight: 600 }}>Input stats</div>
                  <div className="job-list">
                    <div className="job-item">
                      Evidence {state.response.inputStats.evidenceCount} · Used{" "}
                      {state.response.inputStats.evidenceUsedCount}
                    </div>
                    <div className="job-item">
                      Clusters {state.response.inputStats.clusterCount} ·
                      Citations {state.response.inputStats.citationCount}
                    </div>
                    <div className="job-item">
                      Agentic synthesis{" "}
                      {state.response.inputStats.hasAgentic
                        ? "present"
                        : "not present"}
                    </div>
                  </div>
                </div>
              ) : null}

              {refined ? (
                <div style={{ marginTop: 12 }}>
                  <h3>Refined Brief</h3>
                  <p>{refined.summary}</p>
                  <div className="panel" style={{ marginTop: 12 }}>
                    <h4>Concise Summary</h4>
                    <p>{refined.conciseSummary}</p>
                  </div>
                  {refined.keyFindings?.length ? (
                    <div style={{ marginTop: 12 }}>
                      <h4>Key Findings</h4>
                      <ul>
                        {refined.keyFindings.map((item) => (
                          <li key={item}>{item}</li>
                        ))}
                      </ul>
                    </div>
                  ) : null}
                  {refined.evidenceHighlights?.length ? (
                    <div style={{ marginTop: 12 }}>
                      <h4>Evidence Highlights</h4>
                      <div className="job-list">
                        {refined.evidenceHighlights.map((item) => (
                          <div
                            key={`${item.url}-${item.finding}`}
                            className="job-item"
                          >
                            <div style={{ fontWeight: 600 }}>
                              {item.title || item.url}
                            </div>
                            <a href={item.url} target="_blank" rel="noreferrer">
                              {item.url}
                            </a>
                            <div>{item.finding}</div>
                            {item.relevance ? (
                              <div style={{ color: "var(--text-muted)" }}>
                                {item.relevance}
                              </div>
                            ) : null}
                          </div>
                        ))}
                      </div>
                    </div>
                  ) : null}
                  {refined.openQuestions?.length ? (
                    <div style={{ marginTop: 12 }}>
                      <h4>Open Questions</h4>
                      <ul>
                        {refined.openQuestions.map((item) => (
                          <li key={item}>{item}</li>
                        ))}
                      </ul>
                    </div>
                  ) : null}
                  {refined.recommendedNextSteps?.length ? (
                    <div style={{ marginTop: 12 }}>
                      <h4>Recommended Next Steps</h4>
                      <ul>
                        {refined.recommendedNextSteps.map((item) => (
                          <li key={item}>{item}</li>
                        ))}
                      </ul>
                    </div>
                  ) : null}
                </div>
              ) : null}

              {state.response.explanation ? (
                <div style={{ marginTop: 12 }}>
                  <div style={{ fontWeight: 600 }}>Model explanation</div>
                  <p>{state.response.explanation}</p>
                </div>
              ) : null}

              <details style={{ marginTop: 12 }}>
                <summary>Rendered Markdown</summary>
                <pre>{state.response.markdown}</pre>
              </details>
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );
}
