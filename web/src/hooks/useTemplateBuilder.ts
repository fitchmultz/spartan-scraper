/**
 * Purpose: Provide the use template builder hook for the web app.
 * Responsibilities: Encapsulate reusable state, derived values, and effect wiring for the owning feature.
 * Scope: Hook-local state orchestration only; rendering stays in calling components.
 * Usage: Call from React components that need this behavior.
 * Invariants/Assumptions: Hook inputs remain authoritative and returned state should stay predictable across renders.
 */

import { useState } from "react";

import {
  createTemplate,
  updateTemplate as persistTemplateUpdate,
  type CreateTemplateRequest,
  type SelectorRule,
  type Template,
} from "../api";
import { getApiErrorMessage } from "../lib/api-errors";

interface UseTemplateBuilderOptions {
  initialTemplate?: Template;
  onSave?: (template: Template) => void;
}

interface UseTemplateBuilderReturn {
  template: Template;
  updateTemplate: (updates: Partial<Template>) => void;
  addSelector: (selector: SelectorRule) => void;
  updateSelector: (index: number, updates: Partial<SelectorRule>) => void;
  removeSelector: (index: number) => void;
  saveTemplate: () => Promise<void>;
  isSaving: boolean;
  error: string | null;
  clearError: () => void;
}

export function useTemplateBuilder(
  options: UseTemplateBuilderOptions = {},
): UseTemplateBuilderReturn {
  const { initialTemplate, onSave } = options;

  const [template, setTemplate] = useState<Template>({
    name: initialTemplate?.name ?? "",
    selectors: initialTemplate?.selectors ?? [],
    jsonld: initialTemplate?.jsonld,
    regex: initialTemplate?.regex,
    normalize: initialTemplate?.normalize,
  });

  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const updateTemplate = (updates: Partial<Template>) => {
    setTemplate((prev) => ({ ...prev, ...updates }));
  };

  const addSelector = (selector: SelectorRule) => {
    setTemplate((prev) => ({
      ...prev,
      selectors: [...(prev.selectors ?? []), selector],
    }));
  };

  const updateSelector = (index: number, updates: Partial<SelectorRule>) => {
    setTemplate((prev) => ({
      ...prev,
      selectors: (prev.selectors ?? []).map((rule, i) =>
        i === index ? { ...rule, ...updates } : rule,
      ),
    }));
  };

  const removeSelector = (index: number) => {
    setTemplate((prev) => ({
      ...prev,
      selectors: (prev.selectors ?? []).filter((_, i) => i !== index),
    }));
  };

  const saveTemplate = async () => {
    const name = template.name?.trim();
    if (!name) {
      setError("Template name is required");
      return;
    }

    const selectors = template.selectors ?? [];
    if (selectors.length === 0) {
      setError("At least one selector is required");
      return;
    }

    setIsSaving(true);
    setError(null);

    try {
      const request: CreateTemplateRequest = {
        name,
        selectors,
        ...(template.jsonld ? { jsonld: template.jsonld } : {}),
        ...(template.regex ? { regex: template.regex } : {}),
        ...(template.normalize ? { normalize: template.normalize } : {}),
      };
      const isUpdate = initialTemplate?.name === template.name;
      if (isUpdate) {
        const response = await persistTemplateUpdate({
          path: { name },
          body: request,
        });
        if (response.error) {
          throw new Error(
            getApiErrorMessage(response.error, "Failed to save template"),
          );
        }
      } else {
        const response = await createTemplate({
          body: request,
        });
        if (response.error) {
          throw new Error(
            getApiErrorMessage(response.error, "Failed to save template"),
          );
        }
      }
      onSave?.({ ...template, name, selectors });
    } catch (err) {
      setError(getApiErrorMessage(err, "Failed to save template"));
    } finally {
      setIsSaving(false);
    }
  };

  const clearError = () => {
    setError(null);
  };

  return {
    template,
    updateTemplate,
    addSelector,
    updateSelector,
    removeSelector,
    saveTemplate,
    isSaving,
    error,
    clearError,
  };
}

export default useTemplateBuilder;
