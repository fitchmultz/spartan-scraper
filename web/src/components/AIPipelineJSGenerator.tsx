import { useState } from "react";
import {
  aiPipelineJsGenerate,
  postV1PipelineJs,
  type AiPipelineJsGenerateResponse,
  type JsTargetScript,
} from "../api";
import { AIImageAttachments } from "./AIImageAttachments";
import { getApiBaseUrl } from "../lib/api-config";
import { getApiErrorMessage } from "../lib/api-errors";
import { toAIImagePayloads, type AttachedAIImage } from "../lib/ai-image-utils";

interface AIPipelineJSGeneratorProps {
  isOpen: boolean;
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
  generatedScript: JsTargetScript | null;
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
  generatedScript: null,
  explanation: "",
  routeId: "",
  provider: "",
  model: "",
  visualContextUsed: false,
  isSaving: false,
  error: null,
};

export function AIPipelineJSGenerator({
  isOpen,
  onClose,
  onSaved,
}: AIPipelineJSGeneratorProps) {
  const [state, setState] = useState<GeneratorState>(INITIAL_STATE);

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
    if (!state.instructions.trim()) {
      return "Instructions are required";
    }
    return null;
  };

  const handleGenerate = async () => {
    const validationError = validateInputs();
    if (validationError) {
      setState((prev) => ({ ...prev, error: validationError }));
      return;
    }

    setState((prev) => ({
      ...prev,
      isGenerating: true,
      generatedScript: null,
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

      const { data, error } = await aiPipelineJsGenerate({
        baseUrl: getApiBaseUrl(),
        body: {
          url: state.url,
          ...(state.name.trim() ? { name: state.name.trim() } : {}),
          ...(hostPatterns.length > 0 ? { host_patterns: hostPatterns } : {}),
          instructions: state.instructions.trim(),
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
          getApiErrorMessage(error, "Failed to generate pipeline JS script"),
        );
      }
      const response = data as AiPipelineJsGenerateResponse;
      if (!response.script) {
        throw new Error(
          "No pipeline JS script was generated. Please try again.",
        );
      }
      setState((prev) => ({
        ...prev,
        isGenerating: false,
        generatedScript: response.script || null,
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
            : "Failed to generate pipeline JS script",
      }));
    }
  };

  const handleSave = async () => {
    if (!state.generatedScript) {
      return;
    }

    setState((prev) => ({ ...prev, isSaving: true, error: null }));
    try {
      const { error } = await postV1PipelineJs({
        baseUrl: getApiBaseUrl(),
        body: state.generatedScript,
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
            Generate Pipeline JS with AI
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
              Instructions <span className="required">*</span>
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
              placeholder="Describe what the script should wait for or automate, such as waiting for the main app shell, dismissing a cookie banner, or scrolling the page before extraction."
              disabled={state.isGenerating}
            />
          </div>

          <AIImageAttachments
            images={state.images}
            onChange={(images) => setState((prev) => ({ ...prev, images }))}
            disabled={state.isGenerating || state.isSaving}
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

          {state.generatedScript ? (
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
                {JSON.stringify(state.generatedScript, null, 2)}
              </pre>
            </div>
          ) : null}
        </div>

        <div className="modal-footer gap-3">
          <button
            type="button"
            className="button-secondary"
            onClick={handleClose}
          >
            Cancel
          </button>
          {state.generatedScript ? (
            <button
              type="button"
              className="button-primary"
              onClick={handleSave}
              disabled={state.isSaving}
            >
              {state.isSaving ? "Saving..." : "Save Script"}
            </button>
          ) : (
            <button
              type="button"
              className="button-primary"
              onClick={handleGenerate}
              disabled={state.isGenerating}
            >
              {state.isGenerating ? "Generating..." : "Generate Script"}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
