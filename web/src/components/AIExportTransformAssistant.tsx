import { useMemo, useState } from "react";
import {
  aiTransformGenerate,
  type AiTransformGenerateResponse,
  type ResultTransformConfig,
} from "../api";
import { getApiBaseUrl } from "../lib/api-config";
import { formatExportTransformSummary } from "../lib/export-schedule-utils";

interface AIExportTransformAssistantProps {
  isOpen: boolean;
  onClose: () => void;
  currentTransform?: ResultTransformConfig;
  onApplyTransform: (transform: ResultTransformConfig) => void;
}

interface AssistantState {
  jobId: string;
  instructions: string;
  preferredLanguage: "jmespath" | "jsonata";
  isLoading: boolean;
  response: AiTransformGenerateResponse | null;
  error: string | null;
}

const initialState: AssistantState = {
  jobId: "",
  instructions: "",
  preferredLanguage: "jmespath",
  isLoading: false,
  response: null,
  error: null,
};

function hasCurrentTransform(
  transform: ResultTransformConfig | undefined,
): boolean {
  return formatExportTransformSummary(transform) !== "Default";
}

export function AIExportTransformAssistant({
  isOpen,
  onClose,
  currentTransform,
  onApplyTransform,
}: AIExportTransformAssistantProps) {
  const [state, setState] = useState<AssistantState>(initialState);
  const currentSummary = useMemo(
    () => formatExportTransformSummary(currentTransform),
    [currentTransform],
  );

  if (!isOpen) {
    return null;
  }

  const handleClose = () => {
    setState(initialState);
    onClose();
  };

  const handleGenerate = async () => {
    if (!state.jobId.trim()) {
      setState((prev) => ({
        ...prev,
        error: "Representative job ID is required.",
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
      const { data, error: apiError } = await aiTransformGenerate({
        baseUrl: getApiBaseUrl(),
        body: {
          job_id: state.jobId.trim(),
          currentTransform: hasCurrentTransform(currentTransform)
            ? currentTransform
            : undefined,
          preferredLanguage: state.preferredLanguage,
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
        response: (data as AiTransformGenerateResponse) || null,
      }));
    } catch (error) {
      setState((prev) => ({
        ...prev,
        isLoading: false,
        error:
          error instanceof Error
            ? error.message
            : "Failed to generate transform",
      }));
    }
  };

  const transform = state.response?.transform;

  return (
    // biome-ignore lint/a11y/noStaticElementInteractions: modal overlay pattern
    // biome-ignore lint/a11y/useKeyWithClickEvents: close handled by buttons/overlay
    <div className="modal-overlay" onClick={handleClose}>
      {/* biome-ignore lint/a11y/noStaticElementInteractions: modal content container */}
      {/* biome-ignore lint/a11y/useKeyWithClickEvents: handled by modal semantics */}
      <div
        className="modal-content modal-content--large"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="modal-header">
          <h2 className="modal-title">
            <span className="mr-2 text-purple-400">🪄</span>
            Generate Export Transform with AI
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
            <div className="job-list">
              <div className="job-item">
                <div style={{ fontWeight: 600 }}>Current transform</div>
                <div>{currentSummary}</div>
              </div>
            </div>

            <div className="form-group" style={{ marginTop: 12 }}>
              <label
                htmlFor="ai-export-transform-job-id"
                className="form-label"
              >
                Representative job ID
              </label>
              <input
                id="ai-export-transform-job-id"
                type="text"
                className="form-input"
                value={state.jobId}
                onChange={(event) =>
                  setState((prev) => ({ ...prev, jobId: event.target.value }))
                }
                placeholder="job-123456789abc"
                disabled={state.isLoading}
              />
              <p className="form-help">
                Use a completed job with a representative result file for the
                export you want to transform.
              </p>
            </div>

            <div className="form-group">
              <label
                htmlFor="ai-export-transform-language"
                className="form-label"
              >
                Preferred language
              </label>
              <select
                id="ai-export-transform-language"
                className="form-select"
                value={state.preferredLanguage}
                onChange={(event) =>
                  setState((prev) => ({
                    ...prev,
                    preferredLanguage: event.target.value as
                      | "jmespath"
                      | "jsonata",
                  }))
                }
                disabled={state.isLoading}
              >
                <option value="jmespath">JMESPath</option>
                <option value="jsonata">JSONata</option>
              </select>
            </div>

            <div className="form-group">
              <label
                htmlFor="ai-export-transform-instructions"
                className="form-label"
              >
                Instructions (optional)
              </label>
              <textarea
                id="ai-export-transform-instructions"
                value={state.instructions}
                onChange={(event) =>
                  setState((prev) => ({
                    ...prev,
                    instructions: event.target.value,
                  }))
                }
                rows={4}
                className="form-textarea"
                placeholder="Project the URL, title, and normalized pricing fields for recurring exports."
                disabled={state.isLoading}
              />
              <p className="form-help">
                Spartan only analyzes the representative result and current
                transform. The AI does not fetch new data or expand scope.
              </p>
            </div>

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
            <button type="button" onClick={() => void handleGenerate()}>
              {state.isLoading ? "Generating…" : "Generate Transform"}
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
                {transform ? (
                  <div className="badge running">
                    {formatExportTransformSummary(transform)}
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
                      Sample records{" "}
                      {state.response.inputStats.sampleRecordCount}
                    </div>
                    <div className="job-item">
                      Field paths {state.response.inputStats.fieldPathCount}
                    </div>
                    <div className="job-item">
                      Current transform{" "}
                      {state.response.inputStats.currentTransformProvided
                        ? "provided"
                        : "not provided"}
                    </div>
                  </div>
                </div>
              ) : null}

              {transform ? (
                <div style={{ marginTop: 12 }}>
                  <div style={{ fontWeight: 600 }}>Generated transform</div>
                  <div className="job-list" style={{ marginTop: 8 }}>
                    <div className="job-item">
                      <div style={{ fontWeight: 600 }}>Language</div>
                      <div>{transform.language}</div>
                    </div>
                  </div>
                  <pre className="preview-output">{transform.expression}</pre>
                </div>
              ) : null}

              {state.response.preview?.length ? (
                <div style={{ marginTop: 12 }}>
                  <div style={{ fontWeight: 600 }}>Preview</div>
                  <pre className="preview-output">
                    {JSON.stringify(state.response.preview, null, 2)}
                  </pre>
                </div>
              ) : null}

              {state.response.explanation ? (
                <div style={{ marginTop: 12 }}>
                  <div style={{ fontWeight: 600 }}>Model explanation</div>
                  <p>{state.response.explanation}</p>
                </div>
              ) : null}

              <div className="modal-footer gap-3" style={{ marginTop: 16 }}>
                <button
                  type="button"
                  className="secondary"
                  onClick={handleClose}
                >
                  Keep Editing
                </button>
                <button
                  type="button"
                  disabled={!transform}
                  onClick={() => {
                    if (!transform) {
                      return;
                    }
                    onApplyTransform(transform);
                    handleClose();
                  }}
                >
                  Apply Transform
                </button>
              </div>
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );
}
