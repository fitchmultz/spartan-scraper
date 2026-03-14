import { useMemo, useState } from "react";

import {
  aiTemplateDebug,
  updateTemplate,
  type AiExtractTemplateDebugResponse,
  type Template,
} from "../api";
import { AIImageAttachments } from "./AIImageAttachments";
import { getApiBaseUrl } from "../lib/api-config";
import { toAIImagePayloads, type AttachedAIImage } from "../lib/ai-image-utils";

interface AITemplateDebuggerProps {
  isOpen: boolean;
  template: Template | null;
  onClose: () => void;
  onTemplateSaved: () => void;
}

type DebugSource = "url" | "html";

interface DebugState {
  source: DebugSource;
  url: string;
  html: string;
  instructions: string;
  images: AttachedAIImage[];
  headless: boolean;
  playwright: boolean;
  visual: boolean;
  isLoading: boolean;
  isSaving: boolean;
  error: string | null;
  result: AiExtractTemplateDebugResponse | null;
}

function createInitialState(): DebugState {
  return {
    source: "url",
    url: "",
    html: "",
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

function getAPIErrorMessage(error: unknown) {
  if (typeof error === "string") {
    return error;
  }
  if (error && typeof error === "object") {
    const candidate = error as { error?: string; message?: string };
    return candidate.error || candidate.message || JSON.stringify(error);
  }
  return String(error);
}

export function AITemplateDebugger({
  isOpen,
  template,
  onClose,
  onTemplateSaved,
}: AITemplateDebuggerProps) {
  const [state, setState] = useState<DebugState>(createInitialState);

  const extractedFieldsJSON = useMemo(
    () =>
      state.result?.extracted_fields
        ? JSON.stringify(state.result.extracted_fields, null, 2)
        : "",
    [state.result],
  );
  const suggestedTemplateJSON = useMemo(
    () =>
      state.result?.suggested_template
        ? JSON.stringify(state.result.suggested_template, null, 2)
        : "",
    [state.result],
  );

  if (!isOpen || !template) {
    return null;
  }

  const handleClose = () => {
    setState(createInitialState());
    onClose();
  };

  const validate = () => {
    if (state.source === "url") {
      if (!state.url.trim()) {
        return "URL is required";
      }
      try {
        new URL(state.url);
      } catch {
        return "Please enter a valid URL";
      }
    }
    if (state.source === "html" && !state.html.trim()) {
      return "HTML is required when using pasted HTML mode";
    }
    return null;
  };

  const handleDebug = async () => {
    const validationError = validate();
    if (validationError) {
      setState((prev) => ({ ...prev, error: validationError }));
      return;
    }

    setState((prev) => ({
      ...prev,
      isLoading: true,
      error: null,
      result: null,
    }));

    try {
      const { data, error } = await aiTemplateDebug({
        baseUrl: getApiBaseUrl(),
        body: {
          ...(state.source === "url" ? { url: state.url.trim() } : {}),
          ...(state.source === "html"
            ? {
                ...(state.url.trim() ? { url: state.url.trim() } : {}),
                html: state.html,
              }
            : {}),
          template,
          ...(state.instructions.trim()
            ? { instructions: state.instructions.trim() }
            : {}),
          ...(state.images.length > 0
            ? { images: toAIImagePayloads(state.images) }
            : {}),
          ...(state.source === "url"
            ? {
                headless: state.headless,
                ...(state.headless ? { playwright: state.playwright } : {}),
                visual: state.visual,
              }
            : {}),
        },
      });

      if (error) {
        throw new Error(getAPIErrorMessage(error));
      }

      setState((prev) => ({
        ...prev,
        isLoading: false,
        result: (data as AiExtractTemplateDebugResponse) ?? null,
      }));
    } catch (error) {
      setState((prev) => ({
        ...prev,
        isLoading: false,
        error:
          error instanceof Error ? error.message : "Failed to debug template",
      }));
    }
  };

  const handleSave = async () => {
    if (!state.result?.suggested_template) {
      return;
    }

    setState((prev) => ({ ...prev, isSaving: true, error: null }));
    try {
      const suggestedTemplate = state.result.suggested_template;
      if (!suggestedTemplate?.name || !suggestedTemplate.selectors) {
        throw new Error(
          "Suggested template is incomplete and cannot be saved.",
        );
      }
      const { error } = await updateTemplate({
        baseUrl: getApiBaseUrl(),
        path: { name: template.name || "" },
        body: {
          name: suggestedTemplate.name,
          selectors: suggestedTemplate.selectors,
          ...(suggestedTemplate.jsonld
            ? { jsonld: suggestedTemplate.jsonld }
            : {}),
          ...(suggestedTemplate.regex
            ? { regex: suggestedTemplate.regex }
            : {}),
          ...(suggestedTemplate.normalize
            ? { normalize: suggestedTemplate.normalize }
            : {}),
        },
      });
      if (error) {
        throw new Error(getAPIErrorMessage(error));
      }
      onTemplateSaved();
      handleClose();
    } catch (error) {
      setState((prev) => ({
        ...prev,
        isSaving: false,
        error:
          error instanceof Error ? error.message : "Failed to save template",
      }));
    }
  };

  return (
    // biome-ignore lint/a11y/noStaticElementInteractions: modal overlay pattern
    // biome-ignore lint/a11y/useKeyWithClickEvents: handled by explicit controls
    <div className="modal-overlay" onClick={handleClose}>
      {/* biome-ignore lint/a11y/noStaticElementInteractions: modal content container */}
      {/* biome-ignore lint/a11y/useKeyWithClickEvents: handled by explicit controls */}
      <div
        className="modal-content modal-content--large"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="modal-header">
          <h2 className="modal-title">
            <span className="mr-2 text-purple-400">🛠️</span>
            Debug Template with AI
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

        <div className="modal-body">
          <div className="form-section space-y-4">
            <div className="rounded-md border border-slate-700 bg-slate-900/60 p-4">
              <h3 className="mb-2 text-sm font-medium text-slate-200">
                Current template
              </h3>
              <p className="text-sm text-slate-400">
                Debugging <code>{template.name}</code>
              </p>
            </div>

            <div className="form-group">
              <span className="form-label">Content Source</span>
              <div className="flex gap-2">
                <button
                  type="button"
                  className={`flex-1 rounded-md px-4 py-2 text-sm font-medium transition-colors ${
                    state.source === "url"
                      ? "bg-purple-600 text-white"
                      : "bg-slate-700 text-slate-300 hover:bg-slate-600"
                  }`}
                  onClick={() =>
                    setState((prev) => ({
                      ...prev,
                      source: "url",
                      error: null,
                    }))
                  }
                  disabled={state.isLoading || state.isSaving}
                >
                  Fetch URL
                </button>
                <button
                  type="button"
                  className={`flex-1 rounded-md px-4 py-2 text-sm font-medium transition-colors ${
                    state.source === "html"
                      ? "bg-purple-600 text-white"
                      : "bg-slate-700 text-slate-300 hover:bg-slate-600"
                  }`}
                  onClick={() =>
                    setState((prev) => ({
                      ...prev,
                      source: "html",
                      error: null,
                      headless: false,
                      playwright: false,
                      visual: false,
                    }))
                  }
                  disabled={state.isLoading || state.isSaving}
                >
                  Paste HTML
                </button>
              </div>
            </div>

            <div className="form-group">
              <label className="form-label" htmlFor="ai-template-debug-url">
                {state.source === "url" ? "Target URL" : "Page URL (optional)"}
                {state.source === "url" ? (
                  <span className="required">*</span>
                ) : null}
              </label>
              <input
                id="ai-template-debug-url"
                type="url"
                value={state.url}
                onChange={(event) =>
                  setState((prev) => ({ ...prev, url: event.target.value }))
                }
                placeholder="https://example.com/products"
                className="form-input"
                disabled={state.isLoading || state.isSaving}
              />
            </div>

            {state.source === "html" && (
              <div className="form-group">
                <label className="form-label" htmlFor="ai-template-debug-html">
                  HTML <span className="required">*</span>
                </label>
                <textarea
                  id="ai-template-debug-html"
                  value={state.html}
                  onChange={(event) =>
                    setState((prev) => ({ ...prev, html: event.target.value }))
                  }
                  rows={10}
                  className="form-textarea font-mono text-xs"
                  placeholder="<html>...</html>"
                  disabled={state.isLoading || state.isSaving}
                />
              </div>
            )}

            <AIImageAttachments
              images={state.images}
              onChange={(images) => setState((prev) => ({ ...prev, images }))}
              disabled={state.isLoading || state.isSaving}
            />

            <div className="form-group">
              <label
                className="form-label"
                htmlFor="ai-template-debug-instructions"
              >
                Repair instructions (optional)
              </label>
              <textarea
                id="ai-template-debug-instructions"
                value={state.instructions}
                onChange={(event) =>
                  setState((prev) => ({
                    ...prev,
                    instructions: event.target.value,
                  }))
                }
                rows={3}
                className="form-textarea"
                placeholder="Prefer visible headings, avoid brittle nth-child selectors, and preserve field names."
                disabled={state.isLoading || state.isSaving}
              />
            </div>

            {state.source === "url" && (
              <div className="rounded-md border border-slate-700 bg-slate-900/60 p-4">
                <h3 className="mb-3 text-sm font-medium text-slate-200">
                  Fetch Strategy
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
                      className="form-checkbox"
                    />
                    Use headless browser
                  </label>
                  <label
                    className={`form-label m-0 flex items-center gap-2 text-sm font-normal ${
                      state.headless ? "text-slate-300" : "text-slate-500"
                    }`}
                  >
                    <input
                      type="checkbox"
                      checked={state.playwright}
                      onChange={(event) =>
                        setState((prev) => ({
                          ...prev,
                          playwright: event.target.checked,
                        }))
                      }
                      disabled={
                        state.isLoading || state.isSaving || !state.headless
                      }
                      className="form-checkbox"
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
                      className="form-checkbox"
                    />
                    Include screenshot context
                  </label>
                </div>
              </div>
            )}

            {state.error && (
              <div className="form-error">
                <strong>Error:</strong> {state.error}
              </div>
            )}

            <div className="form-actions">
              <button
                type="button"
                className="btn btn--secondary"
                onClick={handleClose}
                disabled={state.isLoading || state.isSaving}
              >
                Cancel
              </button>
              <button
                type="button"
                className="btn btn--primary"
                onClick={handleDebug}
                disabled={state.isLoading || state.isSaving}
              >
                {state.isLoading ? "Debugging..." : "Debug Template"}
              </button>
            </div>
          </div>

          {state.result && (
            <div className="results-section mt-6 border-t border-slate-700 pt-6">
              <h3 className="mb-4 text-lg font-medium text-slate-200">
                Debug Results
              </h3>

              {state.result.issues && state.result.issues.length > 0 && (
                <div className="mb-4 rounded-md border border-amber-500/40 bg-amber-500/10 p-3 text-sm text-amber-100">
                  <h4 className="mb-2 font-medium">Detected issues</h4>
                  <ul className="list-disc space-y-1 pl-5">
                    {state.result.issues.map((issue) => (
                      <li key={issue}>{issue}</li>
                    ))}
                  </ul>
                </div>
              )}

              {state.result.explanation && (
                <div className="mb-4 rounded-md border border-slate-700 bg-slate-900/70 p-3">
                  <h4 className="mb-2 text-sm font-medium text-slate-300">
                    AI Explanation
                  </h4>
                  <p className="text-sm text-slate-400">
                    {state.result.explanation}
                  </p>
                </div>
              )}

              <div className="mb-4 rounded-md border border-slate-700 bg-slate-900/70 p-3">
                <h4 className="mb-2 text-sm font-medium text-slate-300">
                  AI Route
                </h4>
                <dl className="space-y-1 text-sm text-slate-400">
                  {state.result.route_id && (
                    <div className="flex flex-wrap gap-2">
                      <dt className="font-medium text-slate-300">Route</dt>
                      <dd className="font-mono text-emerald-300">
                        {state.result.route_id}
                      </dd>
                    </div>
                  )}
                  {state.result.provider && (
                    <div className="flex flex-wrap gap-2">
                      <dt className="font-medium text-slate-300">Provider</dt>
                      <dd>{state.result.provider}</dd>
                    </div>
                  )}
                  {state.result.model && (
                    <div className="flex flex-wrap gap-2">
                      <dt className="font-medium text-slate-300">Model</dt>
                      <dd>{state.result.model}</dd>
                    </div>
                  )}
                  <div className="flex flex-wrap gap-2">
                    <dt className="font-medium text-slate-300">
                      Visual context
                    </dt>
                    <dd>
                      {state.result.visual_context_used ? "Used" : "Not used"}
                    </dd>
                  </div>
                </dl>
              </div>

              {state.result.extracted_fields && (
                <div className="mb-4">
                  <h4 className="mb-2 text-sm font-medium text-slate-300">
                    Current extracted fields
                  </h4>
                  <pre className="max-h-64 overflow-auto rounded-md bg-slate-950 p-4 text-xs text-slate-300">
                    {extractedFieldsJSON}
                  </pre>
                </div>
              )}

              {state.result.suggested_template && (
                <div className="space-y-4">
                  <div>
                    <h4 className="mb-2 text-sm font-medium text-slate-300">
                      Suggested template
                    </h4>
                    <pre className="max-h-80 overflow-auto rounded-md bg-slate-950 p-4 text-xs text-slate-300">
                      {suggestedTemplateJSON}
                    </pre>
                  </div>
                  <div className="form-actions">
                    <button
                      type="button"
                      className="btn btn--primary"
                      onClick={handleSave}
                      disabled={state.isLoading || state.isSaving}
                    >
                      {state.isSaving ? "Saving..." : "Save suggested template"}
                    </button>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
