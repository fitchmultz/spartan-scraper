/**
 * Purpose: Present the modal AI helper that debugs saved pipeline JavaScript scripts against representative pages.
 * Responsibilities: Collect bounded AI inputs, call the pipeline-JS debug endpoint, retain full session attempt history, hand selected attempts off to Settings, and save the operator-selected suggested script.
 * Scope: Modal debugging flow for Settings pipeline scripts only.
 * Usage: Mount from `PipelineJSEditor` when operators opt into AI-assisted tuning.
 * Invariants/Assumptions: Suggested scripts are never auto-saved, image attachments stay request-scoped, closing preserves the current tab-scoped session until operators explicitly reset or discard it, and retry/save always target the selected attempt when one exists.
 */

import { useEffect, useRef } from "react";
import {
  aiPipelineJsDebug,
  putV1PipelineJsByName,
  type AiPipelineJsDebugResponse,
  type ComponentStatus,
  type JsTargetScript,
} from "../api";
import type {
  AIAttempt,
  AIAttemptHistoryController,
} from "../hooks/useAIAttemptHistory";
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
import { AIImageAttachments } from "./AIImageAttachments";
import { describeAICapability } from "./ai-assistant";
import { BrowserExecutionControls } from "./BrowserExecutionControls";
import { AIAuthoringAttemptComparison } from "./ai-authoring/AIAuthoringAttemptComparison";
import { AIAuthoringModalShell } from "./ai-authoring/AIAuthoringModalShell";
import { AIAuthoringSessionFooter } from "./ai-authoring/AIAuthoringSessionFooter";
import { useAIAuthoringSession } from "./ai-authoring/useAIAuthoringSession";

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
  const {
    state,
    setState,
    history,
    activeAttempt,
    hasSessionDraft,
    clearSession,
    resetSession,
    discardSession,
  } = useAIAuthoringSession<JsTargetScript, DebugState>({
    storageKey,
    initialState: createInitialState,
    providedHistory,
    hasSessionDraft: (currentState, attemptCount) =>
      Boolean(
        currentState.url.trim() ||
          currentState.instructions.trim() ||
          currentState.images.length > 0 ||
          hasAIAuthoringBrowserRuntimeDraft(currentState) ||
          attemptCount > 0,
      ),
    clearError: (currentState) => ({
      ...currentState,
      error: null,
    }),
    resetDescription:
      "This clears the tuning attempt history and selected candidate, but keeps the current URL, instructions, browser options, and uploaded images so you can run a fresh pass without re-entering them.",
    discardDescription:
      "This removes the in-progress tuning draft, selected candidate, attempt history, and uploaded images from this browser tab.",
    onClose,
    onSessionCleared,
  });
  const aiCapability = describeAICapability(
    aiStatus,
    "Tune pipeline scripts manually in Settings.",
  );
  const aiUnavailable = aiCapability.unavailable;
  const aiUnavailableMessage = aiCapability.message;
  const effectiveScript = activeAttempt?.artifact ?? script;
  const resetSignalRef = useRef(resetSignal);

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
    <AIAuthoringModalShell
      title="Tune Pipeline JS with AI"
      titleIcon="🛠️"
      onClose={onClose}
      aiUnavailableMessage={aiUnavailableMessage}
      sessionNotice={
        hasSessionDraft
          ? "Close keeps this AI session available in the current browser tab. Reset session keeps the request inputs but clears tuning attempts. Discard session removes everything."
          : null
      }
      footer={
        <AIAuthoringSessionFooter
          onClose={onClose}
          hasSessionDraft={hasSessionDraft}
          hasAttempts={history.attempts.length > 0}
          onDiscardSession={() => void discardSession()}
          onResetSession={() => void resetSession()}
          onRetry={handleRetry}
          onSave={() => void handleSave()}
          onRun={() => void handleDebug()}
          discardDisabled={state.isLoading || state.isSaving}
          resetDisabled={state.isLoading || state.isSaving}
          retryDisabled={state.isLoading || state.isSaving || aiUnavailable}
          saveDisabled={
            state.isLoading ||
            state.isSaving ||
            aiUnavailable ||
            !activeAttempt?.artifact
          }
          runDisabled={state.isLoading || state.isSaving || aiUnavailable}
          actionTitle={aiUnavailableMessage ?? undefined}
          runLabel="Tune script"
          runningLabel="Tuning..."
          retryLabel="Retry with changes"
          retryingLabel="Retrying..."
          saveLabel="Save selected tuned script"
          savingLabel="Saving..."
          isRunning={state.isLoading}
          isSaving={state.isSaving}
        />
      }
    >
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
                    ...updateAIAuthoringVisualState(prev, event.target.checked),
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

        <AIAuthoringAttemptComparison
          history={history}
          artifactKind="pipeline-js"
          emptyBaselineMessage="No tuned pipeline JS script was returned for this attempt."
          emptySelectedMessage="No tuned pipeline JS script was returned for this attempt."
          onRestoreGuidance={(attempt) =>
            setState((prev) => ({
              ...prev,
              instructions: attempt.guidanceText || prev.instructions,
              error: null,
            }))
          }
          onEditInSettings={onEditInSettings}
        />
      </fieldset>
    </AIAuthoringModalShell>
  );
}
