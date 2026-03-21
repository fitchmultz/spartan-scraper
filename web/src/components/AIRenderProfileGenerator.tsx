/**
 * Purpose: Present the modal AI helper that generates saved render profiles from operator guidance and sample pages.
 * Responsibilities: Collect bounded AI inputs, call the render-profile generation endpoint, retain full session attempt history, hand selected attempts off to Settings, and save the operator-selected attempt.
 * Scope: Modal generation flow for Settings render profiles only.
 * Usage: Mount from `RenderProfileEditor` when operators opt into AI-assisted authoring.
 * Invariants/Assumptions: Generated profiles are never auto-saved, image attachments stay request-scoped, normal modal close resets the session, and save always targets the selected attempt.
 */

import { useState } from "react";
import {
  aiRenderProfileGenerate,
  postV1RenderProfiles,
  type AiRenderProfileGenerateResponse,
  type ComponentStatus,
  type RenderProfile,
} from "../api";
import {
  useAIAttemptHistory,
  type AIAttempt,
  type AIAttemptHistoryController,
} from "../hooks/useAIAttemptHistory";
import { toRenderProfileGenerateAttempt } from "../lib/ai-authoring-attempts";
import { buildManualEditContinuationGuidance } from "../lib/ai-authoring-roundtrip";
import { getApiBaseUrl } from "../lib/api-config";
import { getApiErrorMessage } from "../lib/api-errors";
import { toAIImagePayloads, type AttachedAIImage } from "../lib/ai-image-utils";
import { AIAttemptHistoryList } from "./AIAttemptHistoryList";
import { AIAuthoringAttemptPanel } from "./AIAuthoringAttemptPanel";
import { AICandidateDiffView } from "./AICandidateDiffView";
import { AIImageAttachments } from "./AIImageAttachments";
import { AIUnavailableNotice, describeAICapability } from "./ai-assistant";

interface AIRenderProfileGeneratorProps {
  isOpen: boolean;
  aiStatus?: ComponentStatus | null;
  onClose: () => void;
  onSaved: () => void;
  history?: AIAttemptHistoryController<RenderProfile>;
  onEditInSettings?: (attempt: AIAttempt<RenderProfile>) => void;
}

interface GeneratorState {
  url: string;
  name: string;
  hostPatterns: string;
  instructions: string;
  images: AttachedAIImage[];
  headless: boolean;
  playwright: boolean;
  visual: boolean;
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
  headless: false,
  playwright: false,
  visual: false,
  isGenerating: false,
  isSaving: false,
  error: null,
};

export function AIRenderProfileGenerator({
  isOpen,
  aiStatus = null,
  onClose,
  onSaved,
  history: providedHistory,
  onEditInSettings,
}: AIRenderProfileGeneratorProps) {
  const [state, setState] = useState<GeneratorState>(INITIAL_STATE);
  const ownedHistory = useAIAttemptHistory<RenderProfile>();
  const history = providedHistory ?? ownedHistory;

  const aiCapability = describeAICapability(
    aiStatus,
    "Create and edit render profiles manually in Settings.",
  );
  const aiUnavailable = aiCapability.unavailable;
  const aiUnavailableMessage = aiCapability.message;
  const activeAttempt = history.activeAttempt;
  const baselineAttempt = history.baselineAttempt;
  const latestAttempt = history.latestAttempt;

  const handleClose = () => {
    setState(INITIAL_STATE);
    history.reset();
    onClose();
  };

  const validateInputs = () => {
    if (!state.url.trim()) {
      return "URL is required";
    }
    try {
      new URL(state.url);
    } catch {
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
        artifactLabel: "render profile",
      });

      const { data, error } = await aiRenderProfileGenerate({
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
          getApiErrorMessage(error, "Failed to generate render profile"),
        );
      }

      const attempt = toRenderProfileGenerateAttempt(
        data as AiRenderProfileGenerateResponse,
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
            : "Failed to generate render profile",
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
      const { error } = await postV1RenderProfiles({
        baseUrl: getApiBaseUrl(),
        body: activeAttempt.artifact,
      });
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Failed to save render profile"),
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
            : "Failed to save render profile",
      }));
    }
  };

  if (!isOpen) {
    return null;
  }

  return (
    // biome-ignore lint/a11y/noStaticElementInteractions: modal overlay pattern
    // biome-ignore lint/a11y/useKeyWithClickEvents: handled via close controls
    <div className="modal-overlay" onClick={handleClose}>
      {/* biome-ignore lint/a11y/noStaticElementInteractions: modal content container */}
      {/* biome-ignore lint/a11y/useKeyWithClickEvents: modal content container */}
      <div
        className="modal-content modal-content--large"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="modal-header">
          <h2 className="modal-title">
            <span className="mr-2 text-purple-400">✨</span>
            Generate Render Profile with AI
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
            disabled={state.isGenerating || state.isSaving || aiUnavailable}
            style={{ border: 0, margin: 0, minInlineSize: 0, padding: 0 }}
          >
            <div className="form-group">
              <label htmlFor="ai-render-profile-url" className="form-label">
                Target URL <span className="required">*</span>
              </label>
              <input
                id="ai-render-profile-url"
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
                <label htmlFor="ai-render-profile-name" className="form-label">
                  Profile Name
                </label>
                <input
                  id="ai-render-profile-name"
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
                  htmlFor="ai-render-profile-host-patterns"
                  className="form-label"
                >
                  Host Patterns
                </label>
                <input
                  id="ai-render-profile-host-patterns"
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
              <label
                htmlFor="ai-render-profile-instructions"
                className="form-label"
              >
                Instructions
              </label>
              <textarea
                id="ai-render-profile-instructions"
                className="form-textarea"
                value={state.instructions}
                onChange={(event) =>
                  setState((prev) => ({
                    ...prev,
                    instructions: event.target.value,
                  }))
                }
                rows={4}
                placeholder="Optional. Describe the fetch behavior you want. If left blank, Spartan will derive a goal from the page URL and fetch signals."
                disabled={state.isGenerating}
              />
            </div>

            {activeAttempt?.manualEdit.edited ? (
              <div className="rounded-md border border-emerald-500/30 bg-emerald-500/10 px-3 py-2 text-sm text-emerald-100">
                Retry will continue from Attempt {activeAttempt.ordinal}'s
                manually edited render profile instead of starting from scratch.
              </div>
            ) : null}

            <AIImageAttachments
              images={state.images}
              onChange={(images) => setState((prev) => ({ ...prev, images }))}
              disabled={state.isGenerating || state.isSaving || aiUnavailable}
              disabledReason={aiUnavailableMessage}
            />

            <div className="grid gap-3 md:grid-cols-3">
              <label className="form-label flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={state.headless}
                  onChange={(event) =>
                    setState((prev) => ({
                      ...prev,
                      headless: event.target.checked,
                      ...(event.target.checked ? {} : { playwright: false }),
                    }))
                  }
                  disabled={state.isGenerating}
                />
                Fetch headless
              </label>
              <label className="form-label flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={state.playwright}
                  onChange={(event) =>
                    setState((prev) => ({
                      ...prev,
                      playwright: event.target.checked,
                      ...(event.target.checked ? { headless: true } : {}),
                    }))
                  }
                  disabled={state.isGenerating}
                />
                Use Playwright
              </label>
              <label className="form-label flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={state.visual}
                  onChange={(event) =>
                    setState((prev) => ({
                      ...prev,
                      visual: event.target.checked,
                      ...(event.target.checked ? { headless: true } : {}),
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
                resolvedGoal={baselineAttempt.resolvedGoal}
                explanation={baselineAttempt.explanation}
                rawResponse={baselineAttempt.rawResponse}
                manualEdit={baselineAttempt.manualEdit}
                muted
              >
                {baselineAttempt.artifact ? (
                  <AICandidateDiffView
                    artifactKind="render-profile"
                    selectedArtifact={baselineAttempt.artifact}
                    selectedLabel={`Attempt ${baselineAttempt.ordinal}`}
                  />
                ) : (
                  <div className="text-sm text-slate-400">
                    No render profile artifact was returned for this attempt.
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
                resolvedGoal={activeAttempt.resolvedGoal}
                explanation={activeAttempt.explanation}
                rawResponse={activeAttempt.rawResponse}
                manualEdit={activeAttempt.manualEdit}
              >
                {activeAttempt.artifact ? (
                  <AICandidateDiffView
                    artifactKind="render-profile"
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
                    No render profile artifact was returned for this attempt.
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
              Cancel
            </button>
            {history.attempts.length > 0 ? (
              <>
                <button
                  type="button"
                  className="button-secondary"
                  onClick={handleRetry}
                  disabled={
                    state.isGenerating || state.isSaving || aiUnavailable
                  }
                  title={aiUnavailableMessage ?? undefined}
                >
                  {state.isGenerating ? "Retrying..." : "Retry with changes"}
                </button>
                <button
                  type="button"
                  className="button-primary"
                  onClick={handleSave}
                  disabled={
                    state.isGenerating ||
                    state.isSaving ||
                    aiUnavailable ||
                    !activeAttempt?.artifact
                  }
                  title={aiUnavailableMessage ?? undefined}
                >
                  {state.isSaving ? "Saving..." : "Save selected profile"}
                </button>
              </>
            ) : (
              <button
                type="button"
                className="button-primary"
                onClick={() => void handleGenerate()}
                disabled={state.isGenerating || aiUnavailable}
                title={aiUnavailableMessage ?? undefined}
              >
                {state.isGenerating ? "Generating..." : "Generate Profile"}
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
