/**
 * Purpose: Present the modal AI helper that debugs saved render profiles against representative pages.
 * Responsibilities: Collect bounded AI inputs, call the render-profile debug endpoint, retain full session attempt history, hand selected attempts off to Settings, and save the operator-selected suggested profile.
 * Scope: Modal debugging flow for Settings render profiles only.
 * Usage: Mount from `RenderProfileEditor` when operators opt into AI-assisted tuning.
 * Invariants/Assumptions: Suggested profiles are never auto-saved, image attachments stay request-scoped, closing preserves the current tab-scoped session until operators explicitly reset or discard it, and retry/save always target the selected attempt when one exists.
 */

import { useEffect, useRef } from "react";
import {
  aiRenderProfileDebug,
  putV1RenderProfilesByName,
  type AiRenderProfileDebugResponse,
  type ComponentStatus,
  type RenderProfile,
} from "../api";
import type {
  AIAttempt,
  AIAttemptHistoryController,
} from "../hooks/useAIAttemptHistory";
import { toRenderProfileDebugAttempt } from "../lib/ai-authoring-attempts";
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

interface AIRenderProfileDebuggerProps {
  isOpen: boolean;
  aiStatus?: ComponentStatus | null;
  profile: RenderProfile | null;
  onClose: () => void;
  onSaved: () => void;
  history?: AIAttemptHistoryController<RenderProfile>;
  onEditInSettings?: (attempt: AIAttempt<RenderProfile>) => void;
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

export function AIRenderProfileDebugger({
  isOpen,
  aiStatus = null,
  profile,
  onClose,
  onSaved,
  history: providedHistory,
  onEditInSettings,
  storageKey,
  resetSignal,
  onSessionCleared,
}: AIRenderProfileDebuggerProps) {
  const {
    state,
    setState,
    history,
    activeAttempt,
    hasSessionDraft,
    clearSession,
    resetSession,
    discardSession,
  } = useAIAuthoringSession<RenderProfile, DebugState>({
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
    "Tune render profiles manually in Settings.",
  );
  const aiUnavailable = aiCapability.unavailable;
  const aiUnavailableMessage = aiCapability.message;
  const effectiveProfile = activeAttempt?.artifact ?? profile;
  const resetSignalRef = useRef(resetSignal);

  useEffect(() => {
    if (resetSignal === undefined || resetSignal === resetSignalRef.current) {
      return;
    }

    resetSignalRef.current = resetSignal;
    clearSession();
  }, [clearSession, resetSignal]);

  if (!isOpen || !profile) {
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
    if (!effectiveProfile) {
      setState((prev) => ({ ...prev, error: "A render profile is required" }));
      return;
    }

    const requestState = state;

    setState((prev) => ({
      ...prev,
      isLoading: true,
      error: null,
    }));

    try {
      const { data, error } = await aiRenderProfileDebug({
        baseUrl: getApiBaseUrl(),
        body: {
          url: requestState.url.trim(),
          profile: effectiveProfile,
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
          getApiErrorMessage(error, "Failed to debug render profile"),
        );
      }
      const attempt = toRenderProfileDebugAttempt(
        data as AiRenderProfileDebugResponse,
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
            : "Failed to debug render profile",
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
      const { error } = await putV1RenderProfilesByName({
        baseUrl: getApiBaseUrl(),
        path: { name: profile.name },
        body: activeAttempt.artifact,
      });
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Failed to save render profile"),
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
            : "Failed to save render profile",
      }));
    }
  };

  return (
    <AIAuthoringModalShell
      title="Tune Render Profile with AI"
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
          runLabel="Tune profile"
          runningLabel="Tuning..."
          retryLabel="Retry with changes"
          retryingLabel="Retrying..."
          saveLabel="Save selected tuned profile"
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
            Current render profile baseline
          </h3>
          {activeAttempt?.artifact ? (
            <p className="text-sm text-slate-400">
              Retrying from Attempt {activeAttempt.ordinal}
              {activeAttempt.manualEdit.edited
                ? " (manually edited in Settings)"
                : ""}
              . The last saved profile stays untouched until you choose Save.
            </p>
          ) : (
            <p className="text-sm text-slate-400">
              Tuning <code>{profile.name}</code> for hosts{" "}
              {profile.hostPatterns.join(", ")}
            </p>
          )}
        </div>

        <div className="form-group">
          <label htmlFor="ai-render-profile-debug-url" className="form-label">
            Target URL <span className="required">*</span>
          </label>
          <input
            id="ai-render-profile-debug-url"
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
            htmlFor="ai-render-profile-debug-instructions"
            className="form-label"
          >
            Tuning instructions (optional)
          </label>
          <textarea
            id="ai-render-profile-debug-instructions"
            className="form-textarea"
            value={state.instructions}
            onChange={(event) =>
              setState((prev) => ({
                ...prev,
                instructions: event.target.value,
              }))
            }
            rows={3}
            placeholder="Prefer a stable selector wait, use headless mode only when needed, and avoid unnecessary blocking rules."
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
          artifactKind="render-profile"
          emptyBaselineMessage="No tuned render profile was returned for this attempt."
          emptySelectedMessage="No tuned render profile was returned for this attempt."
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
