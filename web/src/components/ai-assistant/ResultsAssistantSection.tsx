/**
 * Purpose: Embed results-focused AI tooling into the saved-results workspace.
 * Responsibilities: Keep assistant context current, generate export shapes, refine research summaries, and expose explicit apply or copy actions back into the results workflow.
 * Scope: `/jobs/:id` assistant behavior only.
 * Usage: Mount from `ResultsExplorer` beside the dominant results reader.
 * Invariants/Assumptions: Export shapes are only applied through explicit confirmation and research refinement never mutates the saved result automatically.
 */

import { useEffect, useMemo, useState } from "react";
import {
  aiExportShape,
  aiResearchRefine,
  type AiExportShapeResponse,
  type AiResearchRefineResponse,
  type ComponentStatus,
  type ExportShapeConfig,
} from "../../api";
import { getApiBaseUrl } from "../../lib/api-config";
import { getApiErrorMessage } from "../../lib/api-errors";
import { formatExportShapeSummary } from "../../lib/export-schedule-utils";
import { isResearchResultItem } from "../../lib/form-utils";
import type { ResultItem } from "../../types";
import type { AssistantContext } from "./AIAssistantProvider";
import { AIAssistantPanel } from "./AIAssistantPanel";
import { describeAICapability } from "./aiCapability";
import { useAIAssistant } from "./useAIAssistant";

export type ResultsAssistantMode = "shape" | "research";

interface ResultsAssistantSectionProps {
  jobId: string;
  jobType: "scrape" | "crawl" | "research";
  resultFormat: string;
  aiStatus?: ComponentStatus | null;
  selectedResultIndex: number;
  resultSummary: string | null;
  selectedResult: ResultItem | null;
  mode: ResultsAssistantMode;
  onModeChange: (mode: ResultsAssistantMode) => void;
  shapeFormat: "md" | "csv" | "xlsx";
  onShapeFormatChange: (format: "md" | "csv" | "xlsx") => void;
  currentShape?: ExportShapeConfig;
  onApplyShape: (shape: ExportShapeConfig) => void;
}

interface ShapeState {
  instructions: string;
  isLoading: boolean;
  response: AiExportShapeResponse | null;
  error: string | null;
}

interface RefineState {
  instructions: string;
  isLoading: boolean;
  response: AiResearchRefineResponse | null;
  error: string | null;
}

const INITIAL_SHAPE_STATE: ShapeState = {
  instructions: "",
  isLoading: false,
  response: null,
  error: null,
};

const INITIAL_REFINE_STATE: RefineState = {
  instructions: "",
  isLoading: false,
  response: null,
  error: null,
};

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

export function ResultsAssistantSection({
  jobId,
  jobType,
  resultFormat,
  aiStatus = null,
  selectedResultIndex,
  resultSummary,
  selectedResult,
  mode,
  onModeChange,
  shapeFormat,
  onShapeFormatChange,
  currentShape,
  onApplyShape,
}: ResultsAssistantSectionProps) {
  const { setContext } = useAIAssistant();
  const [shapeState, setShapeState] = useState<ShapeState>(INITIAL_SHAPE_STATE);
  const [refineState, setRefineState] =
    useState<RefineState>(INITIAL_REFINE_STATE);

  const researchResult =
    selectedResult && isResearchResultItem(selectedResult)
      ? selectedResult
      : null;
  const aiCapability = describeAICapability(
    aiStatus,
    "Shape exports or refine research manually from the saved results workspace.",
  );
  const aiUnavailable = aiCapability.unavailable;
  const aiUnavailableMessage = aiCapability.message;

  const assistantContext = useMemo<AssistantContext>(
    () => ({
      surface: "results",
      jobId,
      resultFormat,
      selectedResultIndex,
      resultSummary,
    }),
    [jobId, resultFormat, resultSummary, selectedResultIndex],
  );

  const selectionResetKey = `${jobId}:${selectedResultIndex}`;

  useEffect(() => {
    setContext(assistantContext);
  }, [assistantContext, setContext]);

  useEffect(() => {
    if (!selectionResetKey) {
      return;
    }

    setShapeState(INITIAL_SHAPE_STATE);
    setRefineState(INITIAL_REFINE_STATE);
  }, [selectionResetKey]);

  const handleGenerateShape = async () => {
    if (aiUnavailable) {
      return;
    }

    setShapeState((previous) => ({
      ...previous,
      isLoading: true,
      response: null,
      error: null,
    }));

    try {
      const response = await aiExportShape({
        baseUrl: getApiBaseUrl(),
        body: {
          job_id: jobId,
          format: shapeFormat,
          currentShape: hasCurrentShape(currentShape)
            ? currentShape
            : undefined,
          instructions: shapeState.instructions.trim() || undefined,
        },
      });

      if (response.error) {
        throw new Error(
          getApiErrorMessage(
            response.error,
            "Failed to generate export shape.",
          ),
        );
      }

      setShapeState((previous) => ({
        ...previous,
        isLoading: false,
        response: (response.data as AiExportShapeResponse) ?? null,
      }));
    } catch (error) {
      setShapeState((previous) => ({
        ...previous,
        isLoading: false,
        error:
          error instanceof Error
            ? error.message
            : "Failed to generate export shape.",
      }));
    }
  };

  const handleRefine = async () => {
    if (aiUnavailable) {
      return;
    }
    if (!researchResult) {
      setRefineState((previous) => ({
        ...previous,
        error: "Select a research result before running refinement.",
      }));
      return;
    }

    setRefineState((previous) => ({
      ...previous,
      isLoading: true,
      response: null,
      error: null,
    }));

    try {
      const response = await aiResearchRefine({
        baseUrl: getApiBaseUrl(),
        body: {
          result: researchResult,
          instructions: refineState.instructions.trim() || undefined,
        },
      });

      if (response.error) {
        throw new Error(
          getApiErrorMessage(
            response.error,
            "Failed to refine research result.",
          ),
        );
      }

      setRefineState((previous) => ({
        ...previous,
        isLoading: false,
        response: (response.data as AiResearchRefineResponse) ?? null,
      }));
    } catch (error) {
      setRefineState((previous) => ({
        ...previous,
        isLoading: false,
        error:
          error instanceof Error
            ? error.message
            : "Failed to refine research result.",
      }));
    }
  };

  return (
    <AIAssistantPanel
      title="Results assistant"
      routeLabel="/jobs/:id"
      aiStatus={aiStatus}
      aiManualFallback="Shape exports or refine research manually from the saved results workspace."
      suggestedActions={
        <>
          <button
            type="button"
            className={mode === "shape" ? "active" : "secondary"}
            onClick={() => onModeChange("shape")}
          >
            Export shape
          </button>
          {jobType === "research" ? (
            <button
              type="button"
              className={mode === "research" ? "active" : "secondary"}
              onClick={() => onModeChange("research")}
            >
              Refine research
            </button>
          ) : null}
        </>
      }
    >
      {mode === "shape" ? (
        <fieldset
          className="form-section"
          disabled={shapeState.isLoading || aiUnavailable}
          style={{ border: 0, margin: 0, minInlineSize: 0, padding: 0 }}
        >
          <div className="form-group">
            <label
              htmlFor="results-assistant-shape-format"
              className="form-label"
            >
              Export format
            </label>
            <select
              id="results-assistant-shape-format"
              value={shapeFormat}
              onChange={(event) =>
                onShapeFormatChange(event.target.value as "md" | "csv" | "xlsx")
              }
            >
              <option value="md">Markdown</option>
              <option value="csv">CSV</option>
              <option value="xlsx">XLSX</option>
            </select>
          </div>

          <div className="form-group">
            <label
              htmlFor="results-assistant-shape-instructions"
              className="form-label"
            >
              Instructions
            </label>
            <textarea
              id="results-assistant-shape-instructions"
              rows={4}
              className="form-textarea"
              value={shapeState.instructions}
              onChange={(event) =>
                setShapeState((previous) => ({
                  ...previous,
                  instructions: event.target.value,
                }))
              }
              disabled={shapeState.isLoading}
            />
          </div>

          {shapeState.error ? (
            <div className="form-error">{shapeState.error}</div>
          ) : null}

          <div className="form-actions">
            <button
              type="button"
              onClick={() => void handleGenerateShape()}
              disabled={shapeState.isLoading || aiUnavailable}
              title={aiUnavailableMessage ?? undefined}
            >
              {shapeState.isLoading ? "Generating…" : "Generate shape"}
            </button>
          </div>

          {shapeState.response?.shape ? (
            <div className="template-assistant-panel__result">
              {renderFieldList(
                "Top-level fields",
                shapeState.response.shape.topLevelFields,
              )}
              {renderFieldList(
                "Normalized fields",
                shapeState.response.shape.normalizedFields,
              )}
              {renderFieldList(
                "Evidence fields",
                shapeState.response.shape.evidenceFields,
              )}
              {renderFieldList(
                "Summary fields",
                shapeState.response.shape.summaryFields,
              )}

              {shapeState.response.explanation ? (
                <div className="template-assistant-panel__callout">
                  {shapeState.response.explanation}
                </div>
              ) : null}

              <button
                type="button"
                onClick={() =>
                  onApplyShape(shapeState.response?.shape as ExportShapeConfig)
                }
                disabled={aiUnavailable}
                title={aiUnavailableMessage ?? undefined}
              >
                Apply shape
              </button>
            </div>
          ) : null}
        </fieldset>
      ) : (
        <fieldset
          className="form-section"
          disabled={refineState.isLoading || aiUnavailable}
          style={{ border: 0, margin: 0, minInlineSize: 0, padding: 0 }}
        >
          <div className="form-group">
            <label
              htmlFor="results-assistant-refine-instructions"
              className="form-label"
            >
              Refinement instructions
            </label>
            <textarea
              id="results-assistant-refine-instructions"
              rows={4}
              className="form-textarea"
              value={refineState.instructions}
              onChange={(event) =>
                setRefineState((previous) => ({
                  ...previous,
                  instructions: event.target.value,
                }))
              }
              disabled={refineState.isLoading}
            />
          </div>

          {refineState.error ? (
            <div className="form-error">{refineState.error}</div>
          ) : null}

          <div className="form-actions">
            <button
              type="button"
              onClick={() => void handleRefine()}
              disabled={
                !researchResult || refineState.isLoading || aiUnavailable
              }
              title={aiUnavailableMessage ?? undefined}
            >
              {refineState.isLoading ? "Refining…" : "Refine result"}
            </button>
          </div>

          {refineState.response?.refined ? (
            <div className="template-assistant-panel__result">
              <h5>Refined brief</h5>
              <p>{refineState.response.refined.summary}</p>

              {refineState.response.refined.keyFindings?.length ? (
                <ul>
                  {refineState.response.refined.keyFindings.map((item) => (
                    <li key={item}>{item}</li>
                  ))}
                </ul>
              ) : null}

              {refineState.response.explanation ? (
                <div className="template-assistant-panel__callout">
                  {refineState.response.explanation}
                </div>
              ) : null}

              {refineState.response.markdown ? (
                <button
                  type="button"
                  className="secondary"
                  onClick={() =>
                    void navigator.clipboard.writeText(
                      refineState.response?.markdown ?? "",
                    )
                  }
                  disabled={aiUnavailable}
                  title={aiUnavailableMessage ?? undefined}
                >
                  Copy markdown
                </button>
              ) : null}
            </div>
          ) : null}
        </fieldset>
      )}
    </AIAssistantPanel>
  );
}
