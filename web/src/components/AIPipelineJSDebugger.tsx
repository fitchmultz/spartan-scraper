/**
 * Purpose: Present the modal AI helper that debugs saved pipeline JavaScript scripts against representative pages.
 * Responsibilities: Collect bounded AI inputs, call the pipeline-JS debug endpoint, retain full session attempt history, hand selected attempts off to Settings, and save the operator-selected suggested script.
 * Scope: Modal debugging flow for Settings pipeline scripts only.
 * Usage: Mount from `PipelineJSEditor` when operators opt into AI-assisted tuning.
 * Invariants/Assumptions: Suggested scripts are never auto-saved, image attachments stay request-scoped, closing preserves the current tab-scoped session until operators explicitly reset or discard it, and retry/save always target the selected attempt when one exists.
 */

import { useCallback, useEffect, useMemo, useRef } from "react";
import {
  aiPipelineJsDebug,
  putV1PipelineJsByName,
  type AiPipelineJsDebugResponse,
  type ComponentStatus,
  type JsTargetScript,
} from "../api";
import {
  useAIAttemptHistory,
  type AIAttempt,
  type AIAttemptHistoryController,
} from "../hooks/useAIAttemptHistory";
import { useSessionStorageState } from "../hooks/useSessionStorageState";
import { toPipelineJsDebugAttempt } from "../lib/ai-authoring-attempts";
import {
  buildAIAuthoringRequestContext,
  createAIAuthoringBrowserRuntimeState,
  hasAIAuthoringBrowserRuntimeDraft,
  updateAIAuthoringHeadlessState,
  updateAIAuthoringPlaywrightState,
  updateAIAuthoringVisualState,
  type AIAuthoringBrowserRuntimeState,
} from "../lib/ai-authoring-browser-runtime";
import { getApiBaseUrl } from "../lib/api-config";
import { getApiErrorMessage } from "../lib/api-errors";
import { isValidHttpUrl } from "../lib/form-utils";
import type { AttachedAIImage } from "../lib/ai-image-utils";
import { AIAttemptHistoryList } from "./AIAttemptHistoryList";
import { AIAuthoringAttemptPanel } from "./AIAuthoringAttemptPanel";
import { AICandidateDiffView } from "./AICandidateDiffView";
import { AIImageAttachments } from "./AIImageAttachments";
import { AIUnavailableNotice, describeAICapability } from "./ai-assistant";
import { BrowserExecutionControls } from "./BrowserExecutionControls";
import { useToast } from "./toast";

interface AIPipelineJSDebuggerProps {
  isOpen: boolean;
  aiStatus?: ComponentStatus | null;
  script: JsTargetScript | null;
  onClose: () => void;
  onSaved: () => void;
  history?: AIAttemptHistoryController<JsTargetScript>;
  onEditInSettings?: (attempt: AIAttempt<JsTargetScript>) => void;
  storageKey?: string;
  resetSignal?: number;
  onSessionCleared?: () => void;
}

interface DebugState extends AIAuthoringBrowserRuntimeState {
  url: string;
  instructions: string;
  images: AttachedAIImage[];
  isLoading: boolean;
  isSaving: boolean;
  error: string | null;
}

function createInitialState(): DebugState {
  return {
    url: "",
    instructions: "",
    images: [],
    ...createAIAuthoringBrowserRuntimeState(),
    isLoading: false,
    isSaving: false,
    error: null,
  };
}

export function AIPipelineJSDebugger({
  isOpen,
  aiStatus = null,
  script,
  onClose,
  onSaved,
  history: providedHistory,
  onEditInSettings,
  storageKey,
  resetSignal,
  onSessionCleared,
}: AIPipelineJSDebuggerProps) {
  const toast = useToast();
  const [state, setState, clearState] = useSessionStorageState<DebugState>(
    storageKey ?? null,
    createInitialState,
  );
  const ownedHistory = useAIAttemptHistory<JsTargetScript>(
    storageKey ? { storageKey: `${storageKey}.history` } : undefined,
  );
  const history = providedHistory ?? ownedHistory;
  const aiCapability = describeAICapability(
    aiStatus,
    "Tune pipeline scripts manually in Settings.",
  );
  const aiUnavailable = aiCapability.unavailable;
  const aiUnavailableMessage = aiCapability.message;
  const activeAttempt = history.activeAttempt;
  const baselineAttempt = history.baselineAttempt;
  const latestAttempt = history.latestAttempt;
  const effectiveScript = activeAttempt?.artifact ?? script;
  const hasSessionDraft = useMemo(
    () =>
      Boolean(
        state.url.trim() ||
          state.instructions.trim() ||
          state.images.length > 0 ||
          hasAIAuthoringBrowserRuntimeDraft(state) ||
          history.attempts.length > 0,
      ),
    [state, history.attempts.length],
  );
  const resetSignalRef = useRef(resetSignal);

  const clearSession = useCallback(() => {
    clearState();
    history.reset();
    onSessionCleared?.();
  }, [clearState, history, onSessionCleared]);

  useEffect(() => {
    if (resetSignal === undefined || resetSignal === resetSignalRef.current) {
      return;
    }

    resetSignalRef.current = resetSignal;
    clearSession();
  }, [clearSession, resetSignal]);

  if (!isOpen || !script) {
    return null;
  }

  const handleClose = () => {
    onClose();
  };

  const handleDebug = async () => {
    if (aiUnavailable) {
      return;
    }
    if (!state.url.trim()) {
      setState((prev) => ({ ...prev, error: "URL is required" }));
      return;
    }
    if (!isValidHttpUrl(state.url)) {
      setState((prev) => ({ ...prev, error: "Please enter a valid URL" }));
      return;
    }
    if (!effectiveScript) {
      setState((prev) => ({ ...prev, error: "A pipeline script is required" }));
      return;
    }

    const requestState = state;

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
          script: effectiveScript,
          ...(requestState.instructions.trim()
            ? { instructions: requestState.instructions.trim() }
            : {}),
          ...buildAIAuthoringRequestContext({
            source: "runtime",
            images: requestState.images,
            state: requestState,
          }),
        },
      });
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Failed to debug pipeline JS script"),
        );
      }
      const attempt = toPipelineJsDebugAttempt(
        data as AiPipelineJsDebugResponse,
      );
      history.appendAttempt(attempt);
      setState((prev) => ({
        ...prev,
        isLoading: false,
        instructions: attempt.guidanceText || prev.instructions,
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
    if (history.attempts.length === 0 || aiUnavailable) {
      return;
    }
    void handleDebug();
  };

  const handleResetSession = async () => {
    const confirmed = await toast.confirm({
      title: "Reset this AI session?",
      description:
        "This clears the tuning attempt history and selected candidate, but keeps the current URL, instructions, browser options, and uploaded images so you can run a fresh pass without re-entering them.",
      confirmLabel: "Reset session",
      cancelLabel: "Keep session",
      tone: "warning",
    });
    if (!confirmed) {
      return;
    }

    history.reset();
    setState((prev) => ({
      ...prev,
      error: null,
    }));
  };

  const handleDiscardSession = async () => {
    if (!hasSessionDraft) {
      clearSession();
      onClose();
      return;
    }

    const confirmed = await toast.confirm({
      title: "Discard this AI session?",
      description:
        "This removes the in-progress tuning draft, selected candidate, attempt history, and uploaded images from this browser tab.",
      confirmLabel: "Discard session",
      cancelLabel: "Keep session",
      tone: "warning",
    });
    if (!confirmed) {
      return;
    }

    clearSession();
    onClose();
  };

  const handleSave = async () => {
    if (!activeAttempt?.artifact) {
      return;
    }

    setState((prev) => ({ ...prev, isSaving: true, error: null }));
    try {
      const { error } = await putV1PipelineJsByName({
        baseUrl: getApiBaseUrl(),
        path: { name: script.name },
        body: activeAttempt.artifact,
      });
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Failed to save pipeline JS script"),
        );
      }
      clearSession();
      onSaved();
      onClose();
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

          {hasSessionDraft ? (
            <div className="rounded-md border border-sky-500/30 bg-sky-500/10 px-3 py-2 text-sm text-sky-100">
              Close keeps this AI session available in the current browser tab.
              Reset session keeps the request inputs but clears tuning attempts.
              Discard session removes everything.
            </div>
          ) : null}

          <fieldset
            disabled={state.isLoading || state.isSaving || aiUnavailable}
            style={{ border: 0, margin: 0, minInlineSize: 0, padding: 0 }}
          >
            <div className="rounded-md border border-slate-700 bg-slate-900/60 p-4">
              <h3 className="mb-2 text-sm font-medium text-slate-200">
                Current pipeline JS baseline
              </h3>
              {activeAttempt?.artifact ? (
                <p className="text-sm text-slate-400">
                  Retrying from Attempt {activeAttempt.ordinal}
                  {activeAttempt.manualEdit.edited
                    ? " (manually edited in Settings)"
                    : ""}
                  . The last saved script stays untouched until you choose Save.
                </p>
              ) : (
                <p className="text-sm text-slate-400">
                  Tuning <code>{script.name}</code> for hosts{" "}
                  {script.hostPatterns.join(", ")}
                </p>
              )}
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
              <div className="space-y-3">
                <BrowserExecutionControls
                  headless={state.headless}
                  setHeadless={(value) =>
                    setState((prev) => ({
                      ...prev,
                      ...updateAIAuthoringHeadlessState(prev, value),
                    }))
                  }
                  usePlaywright={state.playwright}
                  setUsePlaywright={(value) =>
                    setState((prev) => ({
                      ...prev,
                      ...updateAIAuthoringPlaywrightState(prev, value),
                    }))
                  }
                  headlessLabel="Use headless browser"
                  playwrightLabel="Use Playwright"
                  helperText="Enable headless to unlock Playwright."
                  showTimeout={false}
                  disabled={state.isLoading || state.isSaving}
                />
                <label className="form-label m-0 flex items-center gap-2 text-sm font-normal text-slate-300">
                  <input
                    type="checkbox"
                    checked={state.visual}
                    onChange={(event) =>
                      setState((prev) => ({
                        ...prev,
                        ...updateAIAuthoringVisualState(
                          prev,
                          event.target.checked,
                        ),
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

            <AIAttemptHistoryList
              attempts={history.attempts}
              activeAttemptId={history.activeAttemptId}
              baselineAttemptId={history.baselineAttemptId}
              onSelectAttempt={history.selectAttempt}
              onSelectBaseline={history.selectBaseline}
              onRestoreGuidance={(attempt) =>
                setState((prev) => ({
                  ...prev,
                  instructions: attempt.guidanceText || prev.instructions,
                  error: null,
                }))
              }
              onEditInSettings={onEditInSettings}
            />

            {baselineAttempt ? (
              <AIAuthoringAttemptPanel
                key={baselineAttempt.id}
                label={`Comparison baseline · Attempt ${baselineAttempt.ordinal}`}
                routeId={baselineAttempt.routeId}
                provider={baselineAttempt.provider}
                model={baselineAttempt.model}
                visualContextUsed={baselineAttempt.visualContextUsed}
                recheckStatus={baselineAttempt.recheckStatus}
                recheckEngine={baselineAttempt.recheckEngine}
                recheckError={baselineAttempt.recheckError}
                issues={baselineAttempt.issues}
                resolvedGoal={baselineAttempt.resolvedGoal}
                explanation={baselineAttempt.explanation}
                rawResponse={baselineAttempt.rawResponse}
                manualEdit={baselineAttempt.manualEdit}
                muted
              >
                {baselineAttempt.artifact ? (
                  <AICandidateDiffView
                    artifactKind="pipeline-js"
                    selectedArtifact={baselineAttempt.artifact}
                    selectedLabel={`Attempt ${baselineAttempt.ordinal}`}
                  />
                ) : (
                  <div className="text-sm text-slate-400">
                    No tuned pipeline JS script was returned for this attempt.
                  </div>
                )}
              </AIAuthoringAttemptPanel>
            ) : null}

            {activeAttempt ? (
              <AIAuthoringAttemptPanel
                key={activeAttempt.id}
                label={`${activeAttempt.id === latestAttempt?.id ? "Latest" : "Selected"} candidate · Attempt ${activeAttempt.ordinal}`}
                routeId={activeAttempt.routeId}
                provider={activeAttempt.provider}
                model={activeAttempt.model}
                visualContextUsed={activeAttempt.visualContextUsed}
                recheckStatus={activeAttempt.recheckStatus}
                recheckEngine={activeAttempt.recheckEngine}
                recheckError={activeAttempt.recheckError}
                issues={activeAttempt.issues}
                resolvedGoal={activeAttempt.resolvedGoal}
                explanation={activeAttempt.explanation}
                rawResponse={activeAttempt.rawResponse}
                manualEdit={activeAttempt.manualEdit}
              >
                {activeAttempt.artifact ? (
                  <AICandidateDiffView
                    artifactKind="pipeline-js"
                    baselineArtifact={baselineAttempt?.artifact ?? null}
                    selectedArtifact={activeAttempt.artifact}
                    baselineLabel={
                      baselineAttempt
                        ? `Attempt ${baselineAttempt.ordinal}`
                        : "Comparison baseline"
                    }
                    selectedLabel={`Attempt ${activeAttempt.ordinal}`}
                  />
                ) : (
                  <div className="text-sm text-slate-400">
                    No tuned pipeline JS script was returned for this attempt.
                  </div>
                )}
              </AIAuthoringAttemptPanel>
            ) : null}
          </fieldset>

          <div className="modal-footer gap-3">
            <button
              type="button"
              className="button-secondary"
              onClick={handleClose}
            >
              Close
            </button>
            {hasSessionDraft ? (
              <button
                type="button"
                className="button-secondary"
                onClick={() => void handleDiscardSession()}
                disabled={state.isLoading || state.isSaving}
              >
                Discard session
              </button>
            ) : null}
            {history.attempts.length > 0 ? (
              <>
                <button
                  type="button"
                  className="button-secondary"
                  onClick={() => void handleResetSession()}
                  disabled={state.isLoading || state.isSaving}
                >
                  Reset session
                </button>
                <button
                  type="button"
                  className="button-secondary"
                  onClick={handleRetry}
                  disabled={state.isLoading || state.isSaving || aiUnavailable}
                  title={aiUnavailableMessage ?? undefined}
                >
                  {state.isLoading ? "Retrying..." : "Retry with changes"}
                </button>
                <button
                  type="button"
                  className="button-primary"
                  onClick={handleSave}
                  disabled={
                    state.isLoading ||
                    state.isSaving ||
                    aiUnavailable ||
                    !activeAttempt?.artifact
                  }
                  title={aiUnavailableMessage ?? undefined}
                >
                  {state.isSaving ? "Saving..." : "Save selected tuned script"}
                </button>
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
