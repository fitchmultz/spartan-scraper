/**
 * AI Template Generator Component
 *
 * Modal component for generating extraction templates using AI.
 * Allows users to provide a URL and description to automatically
 * generate CSS selectors and extraction rules.
 *
 * @module AITemplateGenerator
 */

import { useState } from "react";
import {
  aiTemplateGenerate,
  createTemplate,
  type Template,
  type AiExtractTemplateGenerateResponse,
} from "../api";
import { getApiBaseUrl } from "../lib/api-config";

interface AITemplateGeneratorProps {
  isOpen: boolean;
  onClose: () => void;
  onTemplateSaved: () => void;
}

interface GeneratorState {
  url: string;
  description: string;
  sampleFields: string;
  headless: boolean;
  isGenerating: boolean;
  generatedTemplate: Template | null;
  explanation: string;
  error: string | null;
  templateName: string;
  isSaving: boolean;
}

export function AITemplateGenerator({
  isOpen,
  onClose,
  onTemplateSaved,
}: AITemplateGeneratorProps) {
  const [state, setState] = useState<GeneratorState>({
    url: "",
    description: "",
    sampleFields: "",
    headless: false,
    isGenerating: false,
    generatedTemplate: null,
    explanation: "",
    error: null,
    templateName: "",
    isSaving: false,
  });

  const resetState = () => {
    setState({
      url: "",
      description: "",
      sampleFields: "",
      headless: false,
      isGenerating: false,
      generatedTemplate: null,
      explanation: "",
      error: null,
      templateName: "",
      isSaving: false,
    });
  };

  const handleClose = () => {
    resetState();
    onClose();
  };

  const validateInputs = (): string | null => {
    if (!state.url.trim()) {
      return "URL is required";
    }
    try {
      new URL(state.url);
    } catch {
      return "Please enter a valid URL";
    }
    if (!state.description.trim()) {
      return "Description is required";
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
      error: null,
      generatedTemplate: null,
      explanation: "",
    }));

    try {
      const sampleFields = state.sampleFields
        .split(",")
        .map((f) => f.trim())
        .filter((f) => f.length > 0);

      const { data, error: apiError } = await aiTemplateGenerate({
        baseUrl: getApiBaseUrl(),
        body: {
          url: state.url,
          description: state.description,
          sample_fields: sampleFields.length > 0 ? sampleFields : undefined,
          headless: state.headless,
        },
      });

      if (apiError) {
        const errorMessage =
          typeof apiError === "object" && apiError !== null
            ? (apiError as { error?: string; message?: string }).error ||
              (apiError as { error?: string; message?: string }).message ||
              String(apiError)
            : String(apiError);
        if (
          errorMessage.includes("not configured") ||
          errorMessage.includes("AI extraction is not configured")
        ) {
          throw new Error(
            "AI extraction is not configured. Please set up AI provider credentials in your configuration.",
          );
        }
        throw new Error(errorMessage);
      }

      const response = data as AiExtractTemplateGenerateResponse;

      if (!response.template) {
        throw new Error("No template was generated. Please try again.");
      }

      setState((prev) => ({
        ...prev,
        generatedTemplate: response.template || null,
        explanation: response.explanation || "",
        templateName: response.template?.name || "",
        isGenerating: false,
      }));
    } catch (err) {
      setState((prev) => ({
        ...prev,
        isGenerating: false,
        error:
          err instanceof Error ? err.message : "Failed to generate template",
      }));
    }
  };

  const handleSave = async () => {
    if (!state.generatedTemplate || !state.templateName.trim()) {
      setState((prev) => ({
        ...prev,
        error: "Template name is required",
      }));
      return;
    }

    setState((prev) => ({ ...prev, isSaving: true, error: null }));

    try {
      const templateToSave = {
        ...state.generatedTemplate,
        name: state.templateName.trim(),
        selectors: state.generatedTemplate.selectors || [],
      };

      const { error: apiError } = await createTemplate({
        baseUrl: getApiBaseUrl(),
        body: templateToSave,
      });

      if (apiError) {
        throw new Error(String(apiError));
      }

      onTemplateSaved();
      handleClose();
    } catch (err) {
      setState((prev) => ({
        ...prev,
        isSaving: false,
        error: err instanceof Error ? err.message : "Failed to save template",
      }));
    }
  };

  if (!isOpen) {
    return null;
  }

  return (
    // biome-ignore lint/a11y/noStaticElementInteractions: modal overlay pattern
    // biome-ignore lint/a11y/useKeyWithClickEvents: handled by escape key
    <div className="modal-overlay" onClick={handleClose}>
      {/* biome-ignore lint/a11y/useKeyWithClickEvents: handled by child component */}
      {/* biome-ignore lint/a11y/noStaticElementInteractions: modal content container */}
      <div
        className="modal-content modal-content--large"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="modal-header">
          <h2 className="modal-title">
            <span className="text-purple-400 mr-2">✨</span>
            Generate Template with AI
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
          {/* Input Form */}
          <div className="form-section">
            <div className="form-group">
              <label htmlFor="ai-template-url" className="form-label">
                Target URL <span className="required">*</span>
              </label>
              <input
                id="ai-template-url"
                type="url"
                value={state.url}
                onChange={(e) =>
                  setState((prev) => ({ ...prev, url: e.target.value }))
                }
                placeholder="https://example.com/products"
                className="form-input"
                disabled={state.isGenerating}
              />
              <p className="form-help">
                The URL of the page to analyze for template generation
              </p>
            </div>

            <div className="form-group">
              <label htmlFor="ai-template-description" className="form-label">
                Description <span className="required">*</span>
              </label>
              <textarea
                id="ai-template-description"
                value={state.description}
                onChange={(e) =>
                  setState((prev) => ({ ...prev, description: e.target.value }))
                }
                placeholder="e.g., Extract product information including name, price, description, and rating from an e-commerce product page"
                rows={3}
                className="form-textarea"
                disabled={state.isGenerating}
              />
              <p className="form-help">
                Describe what data you want to extract in natural language
              </p>
            </div>

            <div className="form-group">
              <label htmlFor="ai-template-fields" className="form-label">
                Sample Fields (optional)
              </label>
              <input
                id="ai-template-fields"
                type="text"
                value={state.sampleFields}
                onChange={(e) =>
                  setState((prev) => ({
                    ...prev,
                    sampleFields: e.target.value,
                  }))
                }
                placeholder="e.g., title, price, description, rating"
                className="form-input"
                disabled={state.isGenerating}
              />
              <p className="form-help">
                Comma-separated field names to guide the AI extraction
              </p>
            </div>

            <div className="form-group">
              <label className="form-label">
                <input
                  type="checkbox"
                  checked={state.headless}
                  onChange={(e) =>
                    setState((prev) => ({
                      ...prev,
                      headless: e.target.checked,
                    }))
                  }
                  disabled={state.isGenerating}
                  className="form-checkbox"
                />
                Use headless browser (for JavaScript-rendered content)
              </label>
            </div>

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
                disabled={state.isGenerating || state.isSaving}
              >
                Cancel
              </button>
              <button
                type="button"
                className="btn btn--primary"
                onClick={handleGenerate}
                disabled={state.isGenerating || state.isSaving}
              >
                {state.isGenerating ? (
                  <>
                    <span className="spinner spinner--small mr-2" />
                    Generating...
                  </>
                ) : (
                  <>
                    <span className="mr-2">✨</span>
                    Generate Template
                  </>
                )}
              </button>
            </div>
          </div>

          {/* Results Panel */}
          {state.generatedTemplate && (
            <div className="results-section mt-6 border-t border-slate-700 pt-6">
              <h3 className="text-lg font-medium text-slate-200 mb-4">
                Generated Template
              </h3>

              {state.explanation && (
                <div className="explanation-panel mb-4 p-3 bg-slate-800 rounded-md">
                  <h4 className="text-sm font-medium text-slate-300 mb-1">
                    AI Explanation
                  </h4>
                  <p className="text-sm text-slate-400">{state.explanation}</p>
                </div>
              )}

              <div className="template-preview mb-4">
                <h4 className="text-sm font-medium text-slate-300 mb-2">
                  Selectors
                </h4>
                {state.generatedTemplate.selectors &&
                state.generatedTemplate.selectors.length > 0 ? (
                  <div className="selectors-list space-y-2">
                    {state.generatedTemplate.selectors.map((selector) => (
                      <div
                        key={`${selector.name}-${selector.selector}-${selector.attr || "text"}`}
                        className="selector-item p-2 bg-slate-800 rounded border border-slate-700"
                      >
                        <div className="flex justify-between items-center">
                          <span className="font-mono text-sm text-purple-300">
                            {selector.name}
                          </span>
                          <span className="text-xs text-slate-500">
                            {selector.attr || "text"}
                          </span>
                        </div>
                        <code className="block mt-1 text-xs text-slate-400 font-mono">
                          {selector.selector}
                        </code>
                      </div>
                    ))}
                  </div>
                ) : (
                  <p className="text-sm text-slate-500">
                    No selectors generated
                  </p>
                )}
              </div>

              <div className="form-group">
                <label htmlFor="template-name" className="form-label">
                  Template Name <span className="required">*</span>
                </label>
                <input
                  id="template-name"
                  type="text"
                  value={state.templateName}
                  onChange={(e) =>
                    setState((prev) => ({
                      ...prev,
                      templateName: e.target.value,
                    }))
                  }
                  placeholder="my-custom-template"
                  className="form-input"
                  disabled={state.isSaving}
                />
              </div>

              <div className="form-actions mt-4">
                <button
                  type="button"
                  className="btn btn--secondary"
                  onClick={() =>
                    setState((prev) => ({
                      ...prev,
                      generatedTemplate: null,
                      explanation: "",
                      error: null,
                    }))
                  }
                  disabled={state.isSaving}
                >
                  Back
                </button>
                <button
                  type="button"
                  className="btn btn--primary"
                  onClick={handleSave}
                  disabled={state.isSaving || !state.templateName.trim()}
                >
                  {state.isSaving ? (
                    <>
                      <span className="spinner spinner--small mr-2" />
                      Saving...
                    </>
                  ) : (
                    "Save Template"
                  )}
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
