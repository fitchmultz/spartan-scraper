import { useEffect, useMemo, useState } from "react";

import {
  aiExtractPreview,
  type AiExtractPreviewResponse,
  type FieldValue,
} from "../api";
import { getApiBaseUrl } from "../lib/api-config";

type PreviewSource = "url" | "html";
type PreviewMode = "natural_language" | "schema_guided";

interface AIExtractPreviewProps {
  isOpen: boolean;
  onClose: () => void;
  initialUrl?: string;
}

interface PreviewState {
  source: PreviewSource;
  url: string;
  html: string;
  mode: PreviewMode;
  prompt: string;
  schemaText: string;
  fields: string;
  headless: boolean;
  playwright: boolean;
  isLoading: boolean;
  result: AiExtractPreviewResponse | null;
  error: string | null;
}

const NATURAL_LANGUAGE_PLACEHOLDER =
  "Extract the product title, price, shipping details, and any in-stock indicators from this page.";
const SCHEMA_GUIDED_PLACEHOLDER = `{
  "title": "Example product",
  "price": "$19.99",
  "in_stock": true,
  "rating": 4.5
}`;

function createInitialState(initialUrl?: string): PreviewState {
  return {
    source: "url",
    url: initialUrl ?? "",
    html: "",
    mode: "natural_language",
    prompt: "",
    schemaText: SCHEMA_GUIDED_PLACEHOLDER,
    fields: "",
    headless: false,
    playwright: false,
    isLoading: false,
    result: null,
    error: null,
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

function formatFieldValues(field: FieldValue) {
  if (field.values && field.values.length > 0) {
    return field.values;
  }
  return [];
}

function PreviewModeButton({
  isActive,
  label,
  onClick,
}: {
  isActive: boolean;
  label: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      className={`flex-1 rounded-md px-4 py-2 text-sm font-medium transition-colors ${
        isActive
          ? "bg-purple-600 text-white"
          : "bg-slate-700 text-slate-300 hover:bg-slate-600"
      }`}
      onClick={onClick}
    >
      {label}
    </button>
  );
}

function ResultMetaCard({ result }: { result: AiExtractPreviewResponse }) {
  return (
    <div className="mb-4 rounded-md border border-slate-700 bg-slate-900/70 p-3">
      <h4 className="mb-2 text-sm font-medium text-slate-300">AI Route</h4>
      <dl className="space-y-1 text-sm text-slate-400">
        {result.route_id && (
          <div className="flex flex-wrap gap-2">
            <dt className="font-medium text-slate-300">Route</dt>
            <dd className="font-mono text-emerald-300">{result.route_id}</dd>
          </div>
        )}
        {result.provider && (
          <div className="flex flex-wrap gap-2">
            <dt className="font-medium text-slate-300">Provider</dt>
            <dd>{result.provider}</dd>
          </div>
        )}
        {result.model && (
          <div className="flex flex-wrap gap-2">
            <dt className="font-medium text-slate-300">Model</dt>
            <dd>{result.model}</dd>
          </div>
        )}
      </dl>
    </div>
  );
}

export function AIExtractPreview({
  isOpen,
  onClose,
  initialUrl,
}: AIExtractPreviewProps) {
  const [state, setState] = useState<PreviewState>(() =>
    createInitialState(initialUrl),
  );

  useEffect(() => {
    if (isOpen) {
      setState(createInitialState(initialUrl));
    }
  }, [initialUrl, isOpen]);

  const resultJSON = useMemo(
    () => (state.result ? JSON.stringify(state.result, null, 2) : ""),
    [state.result],
  );

  if (!isOpen) {
    return null;
  }

  const handleClose = () => {
    setState(createInitialState(initialUrl));
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

    if (state.mode === "natural_language" && !state.prompt.trim()) {
      return "Extraction instructions are required";
    }

    if (state.mode === "schema_guided") {
      if (!state.schemaText.trim()) {
        return "Schema example JSON is required";
      }
      try {
        const parsed = JSON.parse(state.schemaText);
        if (
          parsed === null ||
          Array.isArray(parsed) ||
          typeof parsed !== "object"
        ) {
          return "Schema example must be a JSON object";
        }
      } catch {
        return "Schema example must be valid JSON";
      }
    }

    return null;
  };

  const handlePreview = async () => {
    const validationError = validate();
    if (validationError) {
      setState((prev) => ({
        ...prev,
        error: validationError,
      }));
      return;
    }

    setState((prev) => ({
      ...prev,
      isLoading: true,
      error: null,
      result: null,
    }));

    try {
      const trimmedFields = state.fields
        .split(",")
        .map((field) => field.trim())
        .filter((field) => field.length > 0);

      const body = {
        url: state.url.trim(),
        ...(state.source === "html" ? { html: state.html } : {}),
        mode: state.mode,
        ...(state.mode === "natural_language"
          ? { prompt: state.prompt.trim() }
          : {
              schema: JSON.parse(state.schemaText) as Record<string, unknown>,
            }),
        ...(trimmedFields.length > 0 ? { fields: trimmedFields } : {}),
        ...(state.source === "url"
          ? {
              headless: state.headless,
              playwright: state.headless ? state.playwright : false,
            }
          : {}),
      };

      const { data, error } = await aiExtractPreview({
        baseUrl: getApiBaseUrl(),
        body,
      });

      if (error) {
        const message = getAPIErrorMessage(error);
        if (
          message.includes("not configured") ||
          message.includes("AI extraction is not configured")
        ) {
          throw new Error(
            "AI extraction is not configured. Enable the pi bridge and ensure your pi credentials are available.",
          );
        }
        throw new Error(message);
      }

      setState((prev) => ({
        ...prev,
        isLoading: false,
        result: (data as AiExtractPreviewResponse) ?? null,
      }));
    } catch (error) {
      setState((prev) => ({
        ...prev,
        isLoading: false,
        error:
          error instanceof Error
            ? error.message
            : "Failed to preview extraction",
      }));
    }
  };

  const resultFields = Object.entries(state.result?.fields ?? {});

  return (
    // biome-ignore lint/a11y/noStaticElementInteractions: modal overlay pattern
    // biome-ignore lint/a11y/useKeyWithClickEvents: handled by close button and overlay click
    <div className="modal-overlay" onClick={handleClose}>
      {/* biome-ignore lint/a11y/noStaticElementInteractions: modal content container */}
      {/* biome-ignore lint/a11y/useKeyWithClickEvents: handled by parent modal semantics */}
      <div
        className="modal-content modal-content--large"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="modal-header">
          <h2 className="modal-title">
            <span className="mr-2 text-purple-400">✨</span>
            Preview Extraction with AI
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
          <div className="form-section">
            <div className="form-group">
              <span className="form-label">Content Source</span>
              <div className="flex gap-2">
                <PreviewModeButton
                  isActive={state.source === "url"}
                  label="Fetch URL"
                  onClick={() =>
                    setState((prev) => ({
                      ...prev,
                      source: "url",
                      error: null,
                    }))
                  }
                />
                <PreviewModeButton
                  isActive={state.source === "html"}
                  label="Paste HTML"
                  onClick={() =>
                    setState((prev) => ({
                      ...prev,
                      source: "html",
                      error: null,
                      headless: false,
                      playwright: false,
                    }))
                  }
                />
              </div>
            </div>

            <div className="form-group">
              <label htmlFor="ai-preview-url" className="form-label">
                {state.source === "url" ? "Target URL" : "Page URL (optional)"}
                {state.source === "url" ? (
                  <span className="required">*</span>
                ) : null}
              </label>
              <input
                id="ai-preview-url"
                type="url"
                value={state.url}
                onChange={(event) =>
                  setState((prev) => ({
                    ...prev,
                    url: event.target.value,
                  }))
                }
                placeholder="https://example.com/products"
                className="form-input"
                disabled={state.isLoading}
              />
              <p className="form-help">
                {state.source === "url"
                  ? "Spartan will fetch this page and run AI extraction against the captured content."
                  : "Optional page context for prompts, logs, and downstream debugging."}
              </p>
            </div>

            {state.source === "html" && (
              <div className="form-group">
                <label htmlFor="ai-preview-html" className="form-label">
                  HTML <span className="required">*</span>
                </label>
                <textarea
                  id="ai-preview-html"
                  value={state.html}
                  onChange={(event) =>
                    setState((prev) => ({
                      ...prev,
                      html: event.target.value,
                    }))
                  }
                  rows={10}
                  className="form-textarea font-mono text-xs"
                  placeholder="<html>...</html>"
                  disabled={state.isLoading}
                />
                <p className="form-help">
                  Paste captured HTML when you want deterministic previewing
                  without refetching the page.
                </p>
              </div>
            )}

            <div className="form-group">
              <span className="form-label">Extraction Mode</span>
              <div className="flex gap-2">
                <PreviewModeButton
                  isActive={state.mode === "natural_language"}
                  label="Natural Language"
                  onClick={() =>
                    setState((prev) => ({
                      ...prev,
                      mode: "natural_language",
                      error: null,
                    }))
                  }
                />
                <PreviewModeButton
                  isActive={state.mode === "schema_guided"}
                  label="Schema Guided"
                  onClick={() =>
                    setState((prev) => ({
                      ...prev,
                      mode: "schema_guided",
                      error: null,
                    }))
                  }
                />
              </div>
            </div>

            <div className="form-group">
              <label htmlFor="ai-preview-fields" className="form-label">
                Specific Fields (optional)
              </label>
              <input
                id="ai-preview-fields"
                type="text"
                value={state.fields}
                onChange={(event) =>
                  setState((prev) => ({
                    ...prev,
                    fields: event.target.value,
                  }))
                }
                placeholder="title, price, description, rating"
                className="form-input"
                disabled={state.isLoading}
              />
              <p className="form-help">
                Comma-separated field names to focus the preview response and
                downstream template work.
              </p>
            </div>

            {state.mode === "natural_language" ? (
              <div className="form-group">
                <label htmlFor="ai-preview-prompt" className="form-label">
                  Extraction Instructions <span className="required">*</span>
                </label>
                <textarea
                  id="ai-preview-prompt"
                  value={state.prompt}
                  onChange={(event) =>
                    setState((prev) => ({
                      ...prev,
                      prompt: event.target.value,
                    }))
                  }
                  rows={4}
                  className="form-textarea"
                  placeholder={NATURAL_LANGUAGE_PLACEHOLDER}
                  disabled={state.isLoading}
                />
                <p className="form-help">
                  Describe exactly what the AI should extract and any heuristics
                  or constraints it should respect.
                </p>
              </div>
            ) : (
              <div className="form-group">
                <label htmlFor="ai-preview-schema" className="form-label">
                  Schema Example JSON <span className="required">*</span>
                </label>
                <textarea
                  id="ai-preview-schema"
                  value={state.schemaText}
                  onChange={(event) =>
                    setState((prev) => ({
                      ...prev,
                      schemaText: event.target.value,
                    }))
                  }
                  rows={8}
                  className="form-textarea font-mono text-sm"
                  placeholder={SCHEMA_GUIDED_PLACEHOLDER}
                  disabled={state.isLoading}
                />
                <p className="form-help">
                  Provide a representative JSON object so the model can align
                  field names and output structure.
                </p>
              </div>
            )}

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
                        }))
                      }
                      disabled={state.isLoading}
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
                      disabled={state.isLoading || !state.headless}
                      className="form-checkbox"
                    />
                    Use Playwright
                  </label>
                </div>
                {!state.headless && (
                  <p className="mt-2 text-sm text-slate-500">
                    Enable headless mode to unlock Playwright and browser-only
                    rendering diagnostics.
                  </p>
                )}
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
                disabled={state.isLoading}
              >
                Cancel
              </button>
              <button
                type="button"
                className="btn btn--primary"
                onClick={handlePreview}
                disabled={state.isLoading}
              >
                {state.isLoading ? (
                  <>
                    <span className="spinner spinner--small mr-2" />
                    Running Preview...
                  </>
                ) : (
                  <>
                    <span className="mr-2">✨</span>
                    Run AI Preview
                  </>
                )}
              </button>
            </div>
          </div>

          {state.result && (
            <div className="results-section mt-6 border-t border-slate-700 pt-6">
              <h3 className="mb-4 text-lg font-medium text-slate-200">
                Preview Results
              </h3>

              <div className="mb-4 flex flex-wrap gap-3 rounded-md border border-slate-700 bg-slate-900/70 p-3 text-sm text-slate-300">
                <span>
                  Confidence{" "}
                  <strong className="text-emerald-300">
                    {Math.round((state.result.confidence ?? 0) * 100)}%
                  </strong>
                </span>
                <span>
                  Tokens{" "}
                  <strong className="text-purple-300">
                    {(state.result.tokens_used ?? 0).toLocaleString()}
                  </strong>
                </span>
                <span>
                  Cache{" "}
                  <strong
                    className={
                      state.result.cached ? "text-sky-300" : "text-slate-300"
                    }
                  >
                    {state.result.cached ? "Hit" : "Miss"}
                  </strong>
                </span>
                <span>
                  Fields <strong>{resultFields.length}</strong>
                </span>
              </div>

              {state.result.explanation && (
                <div className="explanation-panel mb-4 rounded-md bg-slate-800 p-3">
                  <h4 className="mb-1 text-sm font-medium text-slate-300">
                    AI Explanation
                  </h4>
                  <p className="text-sm text-slate-400">
                    {state.result.explanation}
                  </p>
                </div>
              )}

              {(state.result.route_id ||
                state.result.provider ||
                state.result.model) && <ResultMetaCard result={state.result} />}

              <div className="mb-4 rounded-md border border-slate-700 overflow-hidden">
                <div className="border-b border-slate-700 bg-slate-900 px-4 py-3">
                  <h4 className="text-sm font-medium text-slate-200">
                    Extracted Fields
                  </h4>
                </div>
                {resultFields.length > 0 ? (
                  <div className="divide-y divide-slate-700 bg-slate-900/40">
                    {resultFields.map(([name, field]) => {
                      const values = formatFieldValues(field);
                      return (
                        <div key={name} className="px-4 py-3">
                          <div className="mb-2 flex flex-wrap items-center gap-2">
                            <span className="font-mono text-sm text-purple-300">
                              {name}
                            </span>
                            {field.source && (
                              <span className="rounded bg-slate-800 px-1.5 py-0.5 text-xs text-slate-400">
                                {field.source}
                              </span>
                            )}
                          </div>
                          {values.length > 0 ? (
                            <div className="space-y-2">
                              {values.map((value) => (
                                <div
                                  key={`${name}-${value}`}
                                  className="rounded border border-slate-800 bg-slate-950/80 px-3 py-2 text-sm text-slate-300"
                                >
                                  {value}
                                </div>
                              ))}
                            </div>
                          ) : (
                            <p className="text-sm text-slate-500">
                              No string values returned for this field.
                            </p>
                          )}
                          {field.rawObject && (
                            <div className="mt-3">
                              <div className="mb-1 text-xs font-medium uppercase tracking-wide text-slate-500">
                                Structured value
                              </div>
                              <pre className="overflow-x-auto rounded border border-slate-800 bg-slate-950/80 p-3 text-xs text-slate-300">
                                {field.rawObject}
                              </pre>
                            </div>
                          )}
                        </div>
                      );
                    })}
                  </div>
                ) : (
                  <div className="px-4 py-6 text-sm text-slate-500">
                    The preview completed, but no fields were returned.
                  </div>
                )}
              </div>

              <details className="rounded-md border border-slate-700 bg-slate-900/60 p-3">
                <summary className="cursor-pointer text-sm font-medium text-slate-300">
                  Raw response JSON
                </summary>
                <pre className="mt-3 overflow-x-auto rounded border border-slate-800 bg-slate-950/80 p-3 text-xs text-slate-300">
                  {resultJSON}
                </pre>
              </details>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
