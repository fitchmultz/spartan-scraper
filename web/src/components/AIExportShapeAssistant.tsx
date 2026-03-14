import { useEffect, useMemo, useState } from "react";
import {
  aiExportShape,
  type AiExportShapeResponse,
  type ExportShapeConfig,
} from "../api";
import { getApiBaseUrl } from "../lib/api-config";
import { formatExportShapeSummary } from "../lib/export-schedule-utils";

interface AIExportShapeAssistantProps {
  isOpen: boolean;
  onClose: () => void;
  format: "md" | "csv" | "xlsx";
  currentShape?: ExportShapeConfig;
  initialJobId?: string;
  onApplyShape: (shape: ExportShapeConfig) => void;
}

interface AssistantState {
  jobId: string;
  instructions: string;
  isLoading: boolean;
  response: AiExportShapeResponse | null;
  error: string | null;
}

function buildInitialState(initialJobId?: string): AssistantState {
  return {
    jobId: initialJobId ?? "",
    instructions: "",
    isLoading: false,
    response: null,
    error: null,
  };
}

function hasCurrentShape(shape: ExportShapeConfig | undefined): boolean {
  return formatExportShapeSummary(shape) !== "Default";
}

function renderFieldList(title: string, values: string[] | undefined) {
  if (!values?.length) {
    return null;
  }

  return (
    <div style={{ marginTop: 12 }}>
      <div style={{ fontWeight: 600 }}>{title}</div>
      <ul>
        {values.map((value) => (
          <li key={value}>
            <code>{value}</code>
          </li>
        ))}
      </ul>
    </div>
  );
}

export function AIExportShapeAssistant({
  isOpen,
  onClose,
  format,
  currentShape,
  initialJobId,
  onApplyShape,
}: AIExportShapeAssistantProps) {
  const [state, setState] = useState<AssistantState>(
    buildInitialState(initialJobId),
  );
  const currentShapeSummary = useMemo(
    () => formatExportShapeSummary(currentShape),
    [currentShape],
  );

  useEffect(() => {
    if (isOpen) {
      setState(buildInitialState(initialJobId));
    }
  }, [initialJobId, isOpen]);

  if (!isOpen) {
    return null;
  }

  const handleClose = () => {
    setState(buildInitialState(initialJobId));
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
      const { data, error: apiError } = await aiExportShape({
        baseUrl: getApiBaseUrl(),
        body: {
          job_id: state.jobId.trim(),
          format,
          currentShape: hasCurrentShape(currentShape)
            ? currentShape
            : undefined,
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
        response: (data as AiExportShapeResponse) || null,
      }));
    } catch (error) {
      setState((prev) => ({
        ...prev,
        isLoading: false,
        error:
          error instanceof Error
            ? error.message
            : "Failed to generate export shape",
      }));
    }
  };

  const shape = state.response?.shape;

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
            <span className="mr-2 text-purple-400">🧭</span>
            Shape Export with AI
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
                <div style={{ fontWeight: 600 }}>Target format</div>
                <div>{format.toUpperCase()}</div>
              </div>
              <div className="job-item">
                <div style={{ fontWeight: 600 }}>Current shape</div>
                <div>{currentShapeSummary}</div>
              </div>
            </div>

            <div className="form-group" style={{ marginTop: 12 }}>
              <label htmlFor="ai-export-shape-job-id" className="form-label">
                Representative job ID
              </label>
              <input
                id="ai-export-shape-job-id"
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
                export you want to shape.
              </p>
            </div>

            <div className="form-group">
              <label
                htmlFor="ai-export-shape-instructions"
                className="form-label"
              >
                Instructions (optional)
              </label>
              <textarea
                id="ai-export-shape-instructions"
                value={state.instructions}
                onChange={(event) =>
                  setState((prev) => ({
                    ...prev,
                    instructions: event.target.value,
                  }))
                }
                rows={4}
                className="form-textarea"
                placeholder="Prioritize pricing, titles, and concise summary fields for operator handoff."
                disabled={state.isLoading}
              />
              <p className="form-help">
                Spartan only analyzes the representative result and current
                shape. The AI does not fetch new data or expand scope.
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
              {state.isLoading ? "Generating…" : "Generate Shape"}
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
                {shape ? (
                  <div className="badge running">
                    {formatExportShapeSummary(shape)}
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
                      Field options {state.response.inputStats.fieldOptionCount}
                    </div>
                    <div className="job-item">
                      Top-level {state.response.inputStats.topLevelFieldCount} ·
                      Normalized{" "}
                      {state.response.inputStats.normalizedFieldCount}
                    </div>
                    <div className="job-item">
                      Evidence {state.response.inputStats.evidenceFieldCount} ·
                      Sample rows {state.response.inputStats.sampleRecordCount}
                    </div>
                  </div>
                </div>
              ) : null}

              {shape ? (
                <div style={{ marginTop: 12 }}>
                  <h3>Generated Shape</h3>
                  {renderFieldList("Top-level fields", shape.topLevelFields)}
                  {renderFieldList("Normalized fields", shape.normalizedFields)}
                  {renderFieldList("Evidence fields", shape.evidenceFields)}
                  {renderFieldList("Summary fields", shape.summaryFields)}

                  {shape.fieldLabels &&
                  Object.keys(shape.fieldLabels).length > 0 ? (
                    <div style={{ marginTop: 12 }}>
                      <div style={{ fontWeight: 600 }}>Field labels</div>
                      <ul>
                        {Object.entries(shape.fieldLabels).map(
                          ([key, value]) => (
                            <li key={key}>
                              <code>{key}</code> → {value}
                            </li>
                          ),
                        )}
                      </ul>
                    </div>
                  ) : null}

                  {shape.formatting ? (
                    <div style={{ marginTop: 12 }}>
                      <div style={{ fontWeight: 600 }}>Formatting hints</div>
                      <ul>
                        {shape.formatting.emptyValue ? (
                          <li>Empty value: {shape.formatting.emptyValue}</li>
                        ) : null}
                        {shape.formatting.multiValueJoin ? (
                          <li>
                            Multi-value join: {shape.formatting.multiValueJoin}
                          </li>
                        ) : null}
                        {shape.formatting.markdownTitle ? (
                          <li>
                            Markdown title: {shape.formatting.markdownTitle}
                          </li>
                        ) : null}
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
                  disabled={!shape}
                  onClick={() => {
                    if (!shape) {
                      return;
                    }
                    onApplyShape(shape);
                    handleClose();
                  }}
                >
                  Apply Shape
                </button>
              </div>
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );
}
