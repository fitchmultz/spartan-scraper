/**
 * Purpose: Present the modal AI helper that generates saved render profiles from operator guidance and sample pages.
 * Responsibilities: Collect bounded AI inputs, call the render-profile generation endpoint, surface generated profile details, and require explicit save confirmation.
 * Scope: Modal generation flow for Settings render profiles only.
 * Usage: Mount from `RenderProfileEditor` when operators opt into AI-assisted authoring.
 * Invariants/Assumptions: Generated profiles are never auto-saved, image attachments stay request-scoped, and AI-unavailable states must remain self-explanatory.
 */

import { useState } from "react";
import {
  aiRenderProfileGenerate,
  postV1RenderProfiles,
  type AiRenderProfileGenerateResponse,
  type ComponentStatus,
  type RenderProfile,
} from "../api";
import { AIUnavailableNotice, describeAICapability } from "./ai-assistant";
import { AIImageAttachments } from "./AIImageAttachments";
import { getApiBaseUrl } from "../lib/api-config";
import { getApiErrorMessage } from "../lib/api-errors";
import { toAIImagePayloads, type AttachedAIImage } from "../lib/ai-image-utils";

interface AIRenderProfileGeneratorProps {
  isOpen: boolean;
  aiStatus?: ComponentStatus | null;
  onClose: () => void;
  onSaved: () => void;
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
  generatedProfile: RenderProfile | null;
  explanation: string;
  routeId: string;
  provider: string;
  model: string;
  visualContextUsed: boolean;
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
  generatedProfile: null,
  explanation: "",
  routeId: "",
  provider: "",
  model: "",
  visualContextUsed: false,
  isSaving: false,
  error: null,
};

export function AIRenderProfileGenerator({
  isOpen,
  aiStatus = null,
  onClose,
  onSaved,
}: AIRenderProfileGeneratorProps) {
  const [state, setState] = useState<GeneratorState>(INITIAL_STATE);

  const aiCapability = describeAICapability(
    aiStatus,
    "Create and edit render profiles manually in Settings.",
  );
  const aiUnavailable = aiCapability.unavailable;
  const aiUnavailableMessage = aiCapability.message;
  const resetState = () => setState(INITIAL_STATE);

  const handleClose = () => {
    resetState();
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

    setState((prev) => ({
      ...prev,
      isGenerating: true,
      generatedProfile: null,
      explanation: "",
      routeId: "",
      provider: "",
      model: "",
      visualContextUsed: false,
      error: null,
    }));

    try {
      const hostPatterns = state.hostPatterns
        .split(",")
        .map((value) => value.trim())
        .filter(Boolean);

      const { data, error } = await aiRenderProfileGenerate({
        baseUrl: getApiBaseUrl(),
        body: {
          url: state.url,
          ...(state.name.trim() ? { name: state.name.trim() } : {}),
          ...(hostPatterns.length > 0 ? { host_patterns: hostPatterns } : {}),
          ...(state.instructions.trim()
            ? { instructions: state.instructions.trim() }
            : {}),
          ...(state.images.length > 0
            ? { images: toAIImagePayloads(state.images) }
            : {}),
          headless: state.headless,
          ...(state.headless ? { playwright: state.playwright } : {}),
          visual: state.visual,
        },
      });
      if (error) {
        throw new Error(
          getApiErrorMessage(error, "Failed to generate render profile"),
        );
      }
      const response = data as AiRenderProfileGenerateResponse;
      if (!response.profile) {
        throw new Error("No render profile was generated. Please try again.");
      }
      setState((prev) => ({
        ...prev,
        isGenerating: false,
        generatedProfile: response.profile || null,
        explanation: response.explanation || "",
        routeId: response.route_id || "",
        provider: response.provider || "",
        model: response.model || "",
        visualContextUsed: response.visual_context_used || false,
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

  const handleSave = async () => {
    if (!state.generatedProfile) {
      return;
    }

    setState((prev) => ({ ...prev, isSaving: true, error: null }));
    try {
      const { error } = await postV1RenderProfiles({
        baseUrl: getApiBaseUrl(),
        body: state.generatedProfile,
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
            <span className="text-purple-400 mr-2">✨</span>
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

            {state.generatedProfile ? (
              <div className="space-y-3 rounded-lg border border-slate-700 bg-slate-900/60 p-4">
                <div className="flex flex-wrap items-center gap-2 text-xs text-slate-300">
                  {state.routeId ? <span>Route: {state.routeId}</span> : null}
                  {state.provider ? (
                    <span>Provider: {state.provider}</span>
                  ) : null}
                  {state.model ? <span>Model: {state.model}</span> : null}
                  {state.visualContextUsed ? (
                    <span>Visual context used</span>
                  ) : null}
                </div>
                {state.explanation ? (
                  <p className="text-sm text-slate-200">{state.explanation}</p>
                ) : null}
                <pre className="overflow-auto rounded bg-slate-950 p-3 text-xs text-slate-100">
                  {JSON.stringify(state.generatedProfile, null, 2)}
                </pre>
              </div>
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
            {state.generatedProfile ? (
              <button
                type="button"
                className="button-primary"
                onClick={handleSave}
                disabled={state.isSaving || aiUnavailable}
                title={aiUnavailableMessage ?? undefined}
              >
                {state.isSaving ? "Saving..." : "Save Profile"}
              </button>
            ) : (
              <button
                type="button"
                className="button-primary"
                onClick={handleGenerate}
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
