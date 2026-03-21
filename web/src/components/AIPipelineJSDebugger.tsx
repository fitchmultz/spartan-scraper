/**
 * Purpose: Present the modal AI helper that debugs saved pipeline JavaScript scripts against representative pages.
 * Responsibilities: Collect bounded AI inputs, call the pipeline-JS debug endpoint, preserve retry context, and require explicit save confirmation.
 * Scope: Modal debugging flow for Settings pipeline scripts only.
 * Usage: Mount from `PipelineJSEditor` when operators opt into AI-assisted tuning.
 * Invariants/Assumptions: Suggested scripts are never auto-saved, image attachments stay request-scoped, and retrying must preserve operator inputs.
 */

import { useState } from "react";

import {
  aiPipelineJsDebug,
  putV1PipelineJsByName,
  type AiPipelineJsDebugResponse,
  type ComponentStatus,
  type JsTargetScript,
} from "../api";
import { AIAuthoringAttemptPanel } from "./AIAuthoringAttemptPanel";
import { AIImageAttachments } from "./AIImageAttachments";
import { AIUnavailableNotice, describeAICapability } from "./ai-assistant";
import { getApiBaseUrl } from "../lib/api-config";
import { getApiErrorMessage } from "../lib/api-errors";
import { toAIImagePayloads, type AttachedAIImage } from "../lib/ai-image-utils";

interface AIPipelineJSDebuggerProps {
  isOpen: boolean;
  aiStatus?: ComponentStatus | null;
  script: JsTargetScript | null;
  onClose: () => void;
  onSaved: () => void;
}

interface DebugState {
  url: string;
  instructions: string;
  images: AttachedAIImage[];
  headless: boolean;
  playwright: boolean;
  visual: boolean;
  isLoading: boolean;
  isSaving: boolean;
  error: string | null;
  result: AiPipelineJsDebugResponse | null;
  previousResult: AiPipelineJsDebugResponse | null;
}

function createInitialState(): DebugState {
  return {
    url: "",
    instructions: "",
    images: [],
    headless: false,
    playwright: false,
    visual: false,
    isLoading: false,
    isSaving: false,
    error: null,
    result: null,
    previousResult: null,
  };
}

export function AIPipelineJSDebugger({
  isOpen,
  aiStatus = null,
  script,
  onClose,
  onSaved,
}: AIPipelineJSDebuggerProps) {
  const [state, setState] = useState<DebugState>(createInitialState);
  const aiCapability = describeAICapability(
    aiStatus,
    "Tune pipeline scripts manually in Settings.",
  );
  const aiUnavailable = aiCapability.unavailable;
  const aiUnavailableMessage = aiCapability.message;

  if (!isOpen || !script) {
    return null;
  }

  const handleClose = () => {
    setState(createInitialState());
    onClose();
  };

  const handleDebug = async (options?: {
    preserveCurrentAsPrevious?: boolean;
  }) => {
    if (aiUnavailable) {
      return;
    }
    if (!state.url.trim()) {
      setState((prev) => ({ ...prev, error: "URL is required" }));
      return;
    }
    try {
      new URL(state.url);
    } catch {
      setState((prev) => ({ ...prev, error: "Please enter a valid URL" }));
      return;
    }

    const requestState = state;
    const nextPreviousResult = options?.preserveCurrentAsPrevious
      ? requestState.result
      : requestState.previousResult;

    setState((prev) => ({
      ...prev,
      isLoading: true,
      error: null,
    }));

    try {
      const { data, error } = await aiPipelineJsDebug({
        baseUrl: getApiBaseUrl(),
        body: {
          url: requestState.url.trim(),
          script,
          ...(requestState.instructions.trim()
            ? { instructions: requestState.instructions.trim() }
            : {}),
          ...(requestState.images.length > 0
            ? { images: toAIImagePayloads(requestState.images) }
            : {}),
          headless: requestState.headless,
          ...(requestState.headless
            ? { playwright: requestState.playwright }
            : {}),
          visual: requestState.visual,
        },
      });
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Failed to debug pipeline JS script"),
        );
      }
      const nextResult = (data as AiPipelineJsDebugResponse) ?? null;
      setState((prev) => ({
        ...prev,
        isLoading: false,
        instructions: nextResult?.resolved_goal?.text ?? prev.instructions,
        result: nextResult,
        previousResult: nextPreviousResult,
      }));
    } catch (error) {
      setState((prev) => ({
        ...prev,
        isLoading: false,
        error:
          error instanceof Error
            ? error.message
            : "Failed to debug pipeline JS script",
      }));
    }
  };

  const handleRetry = () => {
    if (!state.result || aiUnavailable) {
      return;
    }
    void handleDebug({ preserveCurrentAsPrevious: true });
  };

  const handleSave = async () => {
    if (!state.result?.suggested_script) {
      return;
    }

    setState((prev) => ({ ...prev, isSaving: true, error: null }));
    try {
      const { error } = await putV1PipelineJsByName({
        baseUrl: getApiBaseUrl(),
        path: { name: script.name },
        body: state.result.suggested_script,
      });
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Failed to save pipeline JS script"),
        );
      }
      onSaved();
      handleClose();
    } catch (error) {
      setState((prev) => ({
        ...prev,
        isSaving: false,
        error:
          error instanceof Error
            ? error.message
            : "Failed to save pipeline JS script",
      }));
    }
  };

  return (
    // biome-ignore lint/a11y/noStaticElementInteractions: modal overlay pattern
    // biome-ignore lint/a11y/useKeyWithClickEvents: handled via close controls
    <div className="modal-overlay" onClick={handleClose}>
      {/* biome-ignore lint/a11y/noStaticElementInteractions: modal content container */}
      {/* biome-ignore lint/a11y/useKeyWithClickEvents: handled via close controls */}
      <div
        className="modal-content modal-content--large"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="modal-header">
          <h2 className="modal-title">
            <span className="mr-2 text-purple-400">🛠️</span>
            Tune Pipeline JS with AI
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
          {aiUnavailableMessage ? (
            <AIUnavailableNotice message={aiUnavailableMessage} />
          ) : null}

          <fieldset
            disabled={state.isLoading || state.isSaving || aiUnavailable}
            style={{ border: 0, margin: 0, minInlineSize: 0, padding: 0 }}
          >
            <div className="rounded-md border border-slate-700 bg-slate-900/60 p-4">
              <h3 className="mb-2 text-sm font-medium text-slate-200">
                Current pipeline JS script
              </h3>
              <p className="text-sm text-slate-400">
                Tuning <code>{script.name}</code> for hosts{" "}
                {script.hostPatterns.join(", ")}
              </p>
            </div>

            <div className="form-group">
              <label htmlFor="ai-pipeline-js-debug-url" className="form-label">
                Target URL <span className="required">*</span>
              </label>
              <input
                id="ai-pipeline-js-debug-url"
                type="url"
                className="form-input"
                value={state.url}
                onChange={(event) =>
                  setState((prev) => ({ ...prev, url: event.target.value }))
                }
                placeholder="https://example.com/app"
                disabled={state.isLoading || state.isSaving}
              />
            </div>

            <div className="form-group">
              <label
                htmlFor="ai-pipeline-js-debug-instructions"
                className="form-label"
              >
                Tuning instructions (optional)
              </label>
              <textarea
                id="ai-pipeline-js-debug-instructions"
                className="form-textarea"
                value={state.instructions}
                onChange={(event) =>
                  setState((prev) => ({
                    ...prev,
                    instructions: event.target.value,
                  }))
                }
                rows={3}
                placeholder="Prefer selector waits over custom JavaScript, keep the script minimal, and reset scroll position only if needed."
                disabled={state.isLoading || state.isSaving}
              />
            </div>

            <AIImageAttachments
              images={state.images}
              onChange={(images) => setState((prev) => ({ ...prev, images }))}
              disabled={state.isLoading || state.isSaving || aiUnavailable}
              disabledReason={aiUnavailableMessage}
            />

            <div className="rounded-md border border-slate-700 bg-slate-900/60 p-4">
              <h3 className="mb-3 text-sm font-medium text-slate-200">
                Baseline page fetch
              </h3>
              <div className="flex flex-wrap gap-4">
                <label className="form-label m-0 flex items-center gap-2 text-sm font-normal text-slate-300">
                  <input
                    type="checkbox"
                    checked={state.headless}
                    onChange={(event) =>
                      setState((prev) => ({
                        ...prev,
                        headless: event.target.checked,
                        playwright: event.target.checked
                          ? prev.playwright
                          : false,
                        visual: event.target.checked ? prev.visual : false,
                      }))
                    }
                    disabled={state.isLoading || state.isSaving}
                  />
                  Use headless browser
                </label>
                <label className="form-label m-0 flex items-center gap-2 text-sm font-normal text-slate-300">
                  <input
                    type="checkbox"
                    checked={state.playwright}
                    onChange={(event) =>
                      setState((prev) => ({
                        ...prev,
                        playwright: event.target.checked,
                        headless: event.target.checked ? true : prev.headless,
                      }))
                    }
                    disabled={state.isLoading || state.isSaving}
                  />
                  Use Playwright
                </label>
                <label className="form-label m-0 flex items-center gap-2 text-sm font-normal text-slate-300">
                  <input
                    type="checkbox"
                    checked={state.visual}
                    onChange={(event) =>
                      setState((prev) => ({
                        ...prev,
                        visual: event.target.checked,
                        headless: event.target.checked ? true : prev.headless,
                      }))
                    }
                    disabled={state.isLoading || state.isSaving}
                  />
                  Include screenshot context
                </label>
              </div>
            </div>

            {state.error ? (
              <div className="error" role="alert">
                {state.error}
              </div>
            ) : null}

            {state.previousResult ? (
              <AIAuthoringAttemptPanel
                label="Previous candidate"
                routeId={state.previousResult.route_id}
                provider={state.previousResult.provider}
                model={state.previousResult.model}
                visualContextUsed={state.previousResult.visual_context_used}
                recheckStatus={state.previousResult.recheck_status}
                recheckEngine={state.previousResult.recheck_engine}
                recheckError={state.previousResult.recheck_error}
                issues={state.previousResult.issues}
                resolvedGoal={state.previousResult.resolved_goal}
                explanation={state.previousResult.explanation}
                muted
              >
                {state.previousResult.suggested_script ? (
                  <pre className="overflow-auto rounded bg-slate-950 p-3 text-xs text-slate-100">
                    {JSON.stringify(
                      state.previousResult.suggested_script,
                      null,
                      2,
                    )}
                  </pre>
                ) : null}
              </AIAuthoringAttemptPanel>
            ) : null}

            {state.result ? (
              <AIAuthoringAttemptPanel
                label="Latest candidate"
                routeId={state.result.route_id}
                provider={state.result.provider}
                model={state.result.model}
                visualContextUsed={state.result.visual_context_used}
                recheckStatus={state.result.recheck_status}
                recheckEngine={state.result.recheck_engine}
                recheckError={state.result.recheck_error}
                issues={state.result.issues}
                resolvedGoal={state.result.resolved_goal}
                explanation={state.result.explanation}
              >
                {state.result.suggested_script ? (
                  <pre className="overflow-auto rounded bg-slate-950 p-3 text-xs text-slate-100">
                    {JSON.stringify(state.result.suggested_script, null, 2)}
                  </pre>
                ) : null}
              </AIAuthoringAttemptPanel>
            ) : null}
          </fieldset>

          <div className="modal-footer gap-3">
            <button
              type="button"
              className="button-secondary"
              onClick={handleClose}
            >
              Cancel
            </button>
            {state.result ? (
              <>
                <button
                  type="button"
                  className="button-secondary"
                  onClick={handleRetry}
                  disabled={state.isLoading || state.isSaving || aiUnavailable}
                  title={aiUnavailableMessage ?? undefined}
                >
                  {state.isLoading ? "Retrying..." : "Retry with changes"}
                </button>
                {state.result.suggested_script ? (
                  <button
                    type="button"
                    className="button-primary"
                    onClick={handleSave}
                    disabled={
                      state.isLoading || state.isSaving || aiUnavailable
                    }
                    title={aiUnavailableMessage ?? undefined}
                  >
                    {state.isSaving ? "Saving..." : "Save tuned script"}
                  </button>
                ) : null}
              </>
            ) : (
              <button
                type="button"
                className="button-primary"
                onClick={() => void handleDebug()}
                disabled={state.isLoading || state.isSaving || aiUnavailable}
                title={aiUnavailableMessage ?? undefined}
              >
                {state.isLoading ? "Tuning..." : "Tune script"}
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
