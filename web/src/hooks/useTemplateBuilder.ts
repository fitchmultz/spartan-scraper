/**
 * useTemplateBuilder Hook
 *
 * Purpose:
 * - Manage template editing state and persistence for selector-driven builders.
 *
 * Responsibilities:
 * - Hold editable template state.
 * - Apply selector mutations.
 * - Save templates through the generated API client.
 *
 * Scope:
 * - Local UI state and save behavior for template-building flows.
 *
 * Usage:
 * - Used by visual selector and AI-assisted template editing surfaces.
 *
 * Invariants/Assumptions:
 * - Template names are required before saving.
 * - At least one selector is required for persistence.
 * - Save errors stay local to the hook until cleared or retried.
 */

import { useState } from "react";

import {
  createTemplate,
  updateTemplate as persistTemplateUpdate,
  type CreateTemplateRequest,
  type SelectorRule,
  type Template,
} from "../api";

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
          throw new Error(String(response.error) || "Failed to save template");
        }
      } else {
        const response = await createTemplate({
          body: request,
        });
        if (response.error) {
          throw new Error(String(response.error) || "Failed to save template");
        }
      }
      onSave?.({ ...template, name, selectors });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
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
