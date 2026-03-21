/**
 * Purpose: Present the modal AI helper that debugs saved render profiles against representative pages.
 * Responsibilities: Collect bounded AI inputs, call the render-profile debug endpoint, surface diagnostics and suggestions, and require explicit save confirmation.
 * Scope: Modal debugging flow for Settings render profiles only.
 * Usage: Mount from `RenderProfileEditor` when operators opt into AI-assisted tuning.
 * Invariants/Assumptions: Suggested profiles are never auto-saved, image attachments stay request-scoped, and AI-unavailable states must remain self-explanatory.
 */

import { useState } from "react";

import {
  aiRenderProfileDebug,
  putV1RenderProfilesByName,
  type AiRenderProfileDebugResponse,
  type ComponentStatus,
  type RenderProfile,
} from "../api";
import { AIImageAttachments } from "./AIImageAttachments";
import { AIResolvedGoalCard } from "./AIResolvedGoalCard";
import { AIUnavailableNotice, describeAICapability } from "./ai-assistant";
import { getApiBaseUrl } from "../lib/api-config";
import { getApiErrorMessage } from "../lib/api-errors";
import { toAIImagePayloads, type AttachedAIImage } from "../lib/ai-image-utils";

interface AIRenderProfileDebuggerProps {
  isOpen: boolean;
  aiStatus?: ComponentStatus | null;
  profile: RenderProfile | null;
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
  result: AiRenderProfileDebugResponse | null;
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
  };
}

export function AIRenderProfileDebugger({
  isOpen,
  aiStatus = null,
  profile,
  onClose,
  onSaved,
}: AIRenderProfileDebuggerProps) {
  const [state, setState] = useState<DebugState>(createInitialState);
  const aiCapability = describeAICapability(
    aiStatus,
    "Tune render profiles manually in Settings.",
  );
  const aiUnavailable = aiCapability.unavailable;
  const aiUnavailableMessage = aiCapability.message;

  if (!isOpen || !profile) {
    return null;
  }

  const handleClose = () => {
    setState(createInitialState());
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
    try {
      new URL(state.url);
    } catch {
      setState((prev) => ({ ...prev, error: "Please enter a valid URL" }));
      return;
    }

    setState((prev) => ({
      ...prev,
      isLoading: true,
      error: null,
      result: null,
    }));

    try {
      const { data, error } = await aiRenderProfileDebug({
        baseUrl: getApiBaseUrl(),
        body: {
          url: state.url.trim(),
          profile,
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
          getApiErrorMessage(error, "Failed to debug render profile"),
        );
      }
      setState((prev) => ({
        ...prev,
        isLoading: false,
        result: (data as AiRenderProfileDebugResponse) ?? null,
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

  const handleSave = async () => {
    if (!state.result?.suggested_profile) {
      return;
    }

    setState((prev) => ({ ...prev, isSaving: true, error: null }));
    try {
      const { error } = await putV1RenderProfilesByName({
        baseUrl: getApiBaseUrl(),
        path: { name: profile.name },
        body: state.result.suggested_profile,
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
            Tune Render Profile with AI
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
                Current render profile
              </h3>
              <p className="text-sm text-slate-400">
                Tuning <code>{profile.name}</code> for hosts{" "}
                {profile.hostPatterns.join(", ")}
              </p>
            </div>

            <div className="form-group">
              <label
                htmlFor="ai-render-profile-debug-url"
                className="form-label"
              >
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

            {state.result ? (
              <div className="space-y-4 rounded-md border border-slate-700 bg-slate-900/60 p-4">
                <div className="space-y-2 text-sm text-slate-300">
                  {state.result.recheck_status ? (
                    <div>
                      <span className="font-medium text-slate-100">
                        Recheck:
                      </span>{" "}
                      HTTP {state.result.recheck_status}
                      {state.result.recheck_engine
                        ? ` via ${state.result.recheck_engine}`
                        : ""}
                    </div>
                  ) : null}
                  {state.result.recheck_error ? (
                    <div>{state.result.recheck_error}</div>
                  ) : null}
                  {state.result.route_id ? (
                    <div>Route: {state.result.route_id}</div>
                  ) : null}
                  {state.result.provider ? (
                    <div>Provider: {state.result.provider}</div>
                  ) : null}
                  {state.result.model ? (
                    <div>Model: {state.result.model}</div>
                  ) : null}
                  {state.result.visual_context_used ? (
                    <div>Used screenshot context</div>
                  ) : null}
                </div>

                {state.result.issues?.length ? (
                  <div>
                    <h3 className="mb-2 text-sm font-medium text-slate-100">
                      Detected issues
                    </h3>
                    <ul className="list-disc space-y-1 pl-5 text-sm text-slate-300">
                      {state.result.issues.map((issue) => (
                        <li key={issue}>{issue}</li>
                      ))}
                    </ul>
                  </div>
                ) : null}

                <AIResolvedGoalCard resolvedGoal={state.result.resolved_goal} />

                {state.result.explanation ? (
                  <p className="text-sm text-slate-200">
                    {state.result.explanation}
                  </p>
                ) : null}

                {state.result.suggested_profile ? (
                  <pre className="overflow-auto rounded bg-slate-950 p-3 text-xs text-slate-100">
                    {JSON.stringify(state.result.suggested_profile, null, 2)}
                  </pre>
                ) : null}
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
            {state.result?.suggested_profile ? (
              <button
                type="button"
                className="button-primary"
                onClick={handleSave}
                disabled={state.isSaving || aiUnavailable}
                title={aiUnavailableMessage ?? undefined}
              >
                {state.isSaving ? "Saving..." : "Save tuned profile"}
              </button>
            ) : (
              <button
                type="button"
                className="button-primary"
                onClick={handleDebug}
                disabled={state.isLoading || state.isSaving || aiUnavailable}
                title={aiUnavailableMessage ?? undefined}
              >
                {state.isLoading ? "Tuning..." : "Tune profile"}
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
