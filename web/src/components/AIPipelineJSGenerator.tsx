/**
 * Purpose: Present the modal AI helper that generates pipeline JavaScript scripts from operator guidance and sample pages.
 * Responsibilities: Collect bounded AI inputs, call the pipeline-JS generation endpoint, retain full session attempt history, hand selected attempts off to Settings, and save the operator-selected attempt.
 * Scope: Modal generation flow for Settings pipeline scripts only.
 * Usage: Mount from `PipelineJSEditor` when operators opt into AI-assisted authoring.
 * Invariants/Assumptions: Generated scripts are never auto-saved, image attachments stay request-scoped, closing preserves the current tab-scoped session until operators explicitly reset or discard it, and save always targets the selected attempt.
 */

import {
  aiPipelineJsGenerate,
  postV1PipelineJs,
  type AiPipelineJsGenerateResponse,
  type ComponentStatus,
  type JsTargetScript,
} from "../api";
import type {
  AIAttempt,
  AIAttemptHistoryController,
} from "../hooks/useAIAttemptHistory";
import { toPipelineJsGenerateAttempt } from "../lib/ai-authoring-attempts";
import {
  buildAIAuthoringRequestContext,
  createAIAuthoringBrowserRuntimeState,
  hasAIAuthoringBrowserRuntimeDraft,
  updateAIAuthoringHeadlessState,
  updateAIAuthoringPlaywrightState,
  updateAIAuthoringVisualState,
  type AIAuthoringBrowserRuntimeState,
} from "../lib/ai-authoring-browser-runtime";
import { buildManualEditContinuationGuidance } from "../lib/ai-authoring-roundtrip";
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

interface AIPipelineJSGeneratorProps {
  isOpen: boolean;
  aiStatus?: ComponentStatus | null;
  onClose: () => void;
  onSaved: () => void;
  history?: AIAttemptHistoryController<JsTargetScript>;
  onEditInSettings?: (attempt: AIAttempt<JsTargetScript>) => void;
  storageKey?: string;
}

interface GeneratorState extends AIAuthoringBrowserRuntimeState {
  url: string;
  name: string;
  hostPatterns: string;
  instructions: string;
  images: AttachedAIImage[];
  isGenerating: boolean;
  isSaving: boolean;
  error: string | null;
}

const INITIAL_STATE: GeneratorState = {
  url: "",
  name: "",
  hostPatterns: "",
  instructions: "",
  images: [],
  ...createAIAuthoringBrowserRuntimeState(),
  isGenerating: false,
  isSaving: false,
  error: null,
};

export function AIPipelineJSGenerator({
  isOpen,
  aiStatus = null,
  onClose,
  onSaved,
  history: providedHistory,
  onEditInSettings,
  storageKey,
}: AIPipelineJSGeneratorProps) {
  const {
    state,
    setState,
    history,
    activeAttempt,
    hasSessionDraft,
    clearSession,
    resetSession,
    discardSession,
  } = useAIAuthoringSession<JsTargetScript, GeneratorState>({
    storageKey,
    initialState: INITIAL_STATE,
    providedHistory,
    hasSessionDraft: (currentState, attemptCount) =>
      Boolean(
        currentState.url.trim() ||
          currentState.name.trim() ||
          currentState.hostPatterns.trim() ||
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
      "This clears the generated attempt history and selected candidate, but keeps the current URL, instructions, browser options, and uploaded images so you can start a fresh pass without re-entering them.",
    discardDescription:
      "This removes the in-progress AI request draft, selected candidate, attempt history, and uploaded images from this browser tab.",
    onClose,
  });

  const aiCapability = describeAICapability(
    aiStatus,
    "Create and edit pipeline scripts manually in Settings.",
  );
  const aiUnavailable = aiCapability.unavailable;
  const aiUnavailableMessage = aiCapability.message;

  const validateInputs = () => {
    if (!state.url.trim()) {
      return "URL is required";
    }
    if (!isValidHttpUrl(state.url)) {
      return "Please enter a valid URL";
    }
    return null;
  };

  const handleGenerate = async () => {
    if (aiUnavailable) {
      return;
    }
    const validationError = validateInputs();
    if (validationError) {
      setState((prev) => ({ ...prev, error: validationError }));
      return;
    }

    const requestState = state;

    setState((prev) => ({
      ...prev,
      isGenerating: true,
      error: null,
    }));

    try {
      const hostPatterns = requestState.hostPatterns
        .split(",")
        .map((value) => value.trim())
        .filter(Boolean);
      const continuationInstructions = buildManualEditContinuationGuidance({
        operatorInstructions: requestState.instructions,
        artifact: activeAttempt?.manualEdit.edited
          ? activeAttempt.artifact
          : null,
        artifactLabel: "pipeline JS script",
      });

      const { data, error } = await aiPipelineJsGenerate({
        baseUrl: getApiBaseUrl(),
        body: {
          url: requestState.url,
          ...(requestState.name.trim()
            ? { name: requestState.name.trim() }
            : {}),
          ...(hostPatterns.length > 0 ? { host_patterns: hostPatterns } : {}),
          ...(continuationInstructions
            ? { instructions: continuationInstructions }
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
          getApiErrorMessage(error, "Failed to generate pipeline JS script"),
        );
      }

      const attempt = toPipelineJsGenerateAttempt(
        data as AiPipelineJsGenerateResponse,
      );
      history.appendAttempt(attempt);

      setState((prev) => ({
        ...prev,
        isGenerating: false,
        instructions: attempt.guidanceText || prev.instructions,
      }));
    } catch (error) {
      setState((prev) => ({
        ...prev,
        isGenerating: false,
        error:
          error instanceof Error
            ? error.message
            : "Failed to generate pipeline JS script",
      }));
    }
  };

  const handleRetry = () => {
    if (history.attempts.length === 0 || aiUnavailable) {
      return;
    }
    void handleGenerate();
  };

  const handleSave = async () => {
    if (!activeAttempt?.artifact) {
      return;
    }

    setState((prev) => ({ ...prev, isSaving: true, error: null }));
    try {
      const { error } = await postV1PipelineJs({
        baseUrl: getApiBaseUrl(),
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

  if (!isOpen) {
    return null;
  }

  return (
    <AIAuthoringModalShell
      title="Generate Pipeline JS with AI"
      titleIcon="✨"
      onClose={onClose}
      aiUnavailableMessage={aiUnavailableMessage}
      sessionNotice={
        hasSessionDraft
          ? "Close keeps this AI session available in the current browser tab. Reset session keeps the request inputs but clears generated attempts. Discard session removes everything."
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
          onRun={() => void handleGenerate()}
          discardDisabled={state.isGenerating || state.isSaving}
          resetDisabled={state.isGenerating || state.isSaving}
          retryDisabled={state.isGenerating || state.isSaving || aiUnavailable}
          saveDisabled={
            state.isGenerating ||
            state.isSaving ||
            aiUnavailable ||
            !activeAttempt?.artifact
          }
          runDisabled={state.isGenerating || aiUnavailable}
          actionTitle={aiUnavailableMessage ?? undefined}
          runLabel="Generate Script"
          runningLabel="Generating..."
          retryLabel="Retry with changes"
          retryingLabel="Retrying..."
          saveLabel="Save selected script"
          savingLabel="Saving..."
          isRunning={state.isGenerating}
          isSaving={state.isSaving}
        />
      }
    >
      <fieldset
        disabled={state.isGenerating || state.isSaving || aiUnavailable}
        style={{ border: 0, margin: 0, minInlineSize: 0, padding: 0 }}
      >
        <div className="form-group">
          <label htmlFor="ai-pipeline-js-url" className="form-label">
            Target URL <span className="required">*</span>
          </label>
          <input
            id="ai-pipeline-js-url"
            type="url"
            className="form-input"
            value={state.url}
            onChange={(event) =>
              setState((prev) => ({ ...prev, url: event.target.value }))
            }
            placeholder="https://example.com/app"
            disabled={state.isGenerating}
          />
        </div>

        <div className="grid gap-4 md:grid-cols-2">
          <div className="form-group">
            <label htmlFor="ai-pipeline-js-name" className="form-label">
              Script Name
            </label>
            <input
              id="ai-pipeline-js-name"
              type="text"
              className="form-input"
              value={state.name}
              onChange={(event) =>
                setState((prev) => ({ ...prev, name: event.target.value }))
              }
              placeholder="example-app"
              disabled={state.isGenerating}
            />
          </div>
          <div className="form-group">
            <label
              htmlFor="ai-pipeline-js-host-patterns"
              className="form-label"
            >
              Host Patterns
            </label>
            <input
              id="ai-pipeline-js-host-patterns"
              type="text"
              className="form-input"
              value={state.hostPatterns}
              onChange={(event) =>
                setState((prev) => ({
                  ...prev,
                  hostPatterns: event.target.value,
                }))
              }
              placeholder="example.com, *.example.com"
              disabled={state.isGenerating}
            />
          </div>
        </div>

        <div className="form-group">
          <label htmlFor="ai-pipeline-js-instructions" className="form-label">
            Instructions
          </label>
          <textarea
            id="ai-pipeline-js-instructions"
            className="form-textarea"
            value={state.instructions}
            onChange={(event) =>
              setState((prev) => ({
                ...prev,
                instructions: event.target.value,
              }))
            }
            rows={4}
            placeholder="Optional. Describe what the script should wait for or automate. If left blank, Spartan will derive a goal from the page URL and fetch signals."
            disabled={state.isGenerating}
          />
        </div>

        {activeAttempt?.manualEdit.edited ? (
          <div className="rounded-md border border-emerald-500/30 bg-emerald-500/10 px-3 py-2 text-sm text-emerald-100">
            Retry will continue from Attempt {activeAttempt.ordinal}'s manually
            edited script instead of starting from scratch.
          </div>
        ) : null}

        <AIImageAttachments
          images={state.images}
          onChange={(images) => setState((prev) => ({ ...prev, images }))}
          disabled={state.isGenerating || state.isSaving || aiUnavailable}
          disabledReason={aiUnavailableMessage}
        />

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
            headlessLabel="Fetch headless"
            playwrightLabel="Use Playwright"
            helperText="Enable headless to unlock Playwright."
            showTimeout={false}
            disabled={state.isGenerating}
          />
          <label className="form-label flex items-center gap-2">
            <input
              type="checkbox"
              checked={state.visual}
              onChange={(event) =>
                setState((prev) => ({
                  ...prev,
                  ...updateAIAuthoringVisualState(prev, event.target.checked),
                }))
              }
              disabled={state.isGenerating}
            />
            Include screenshot context
          </label>
        </div>

        {state.error ? (
          <div className="error" role="alert">
            {state.error}
          </div>
        ) : null}

        <AIAuthoringAttemptComparison
          history={history}
          artifactKind="pipeline-js"
          emptyBaselineMessage="No pipeline JS artifact was returned for this attempt."
          emptySelectedMessage="No pipeline JS artifact was returned for this attempt."
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
