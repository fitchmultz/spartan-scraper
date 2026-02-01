/**
 * useTemplateBuilder Hook
 *
 * Custom hook for managing template builder state and API interactions.
 *
 * @module useTemplateBuilder
 */

import { useState, useCallback } from "react";

interface SelectorRule {
  name: string;
  selector: string;
  attr: string;
  all: boolean;
  join: string;
  trim: boolean;
  required: boolean;
}

interface Template {
  name: string;
  selectors: SelectorRule[];
  jsonld?: unknown[];
  regex?: unknown[];
  normalize?: {
    titleField?: string;
    descriptionField?: string;
    textField?: string;
    metaFields?: Record<string, string>;
  };
}

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

  const updateTemplate = useCallback((updates: Partial<Template>) => {
    setTemplate((prev) => ({ ...prev, ...updates }));
  }, []);

  const addSelector = useCallback((selector: SelectorRule) => {
    setTemplate((prev) => ({
      ...prev,
      selectors: [...prev.selectors, selector],
    }));
  }, []);

  const updateSelector = useCallback(
    (index: number, updates: Partial<SelectorRule>) => {
      setTemplate((prev) => ({
        ...prev,
        selectors: prev.selectors.map((rule, i) =>
          i === index ? { ...rule, ...updates } : rule,
        ),
      }));
    },
    [],
  );

  const removeSelector = useCallback((index: number) => {
    setTemplate((prev) => ({
      ...prev,
      selectors: prev.selectors.filter((_, i) => i !== index),
    }));
  }, []);

  const saveTemplate = useCallback(async () => {
    if (!template.name) {
      setError("Template name is required");
      return;
    }

    if (template.selectors.length === 0) {
      setError("At least one selector is required");
      return;
    }

    setIsSaving(true);
    setError(null);

    try {
      const isUpdate = initialTemplate?.name === template.name;
      const method = isUpdate ? "PUT" : "POST";
      const endpoint = isUpdate
        ? `/v1/templates/${encodeURIComponent(template.name)}`
        : "/v1/templates";

      const response = await fetch(endpoint, {
        method,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(template),
      });

      if (!response.ok) {
        const data = await response.json();
        throw new Error(data.error || "Failed to save template");
      }

      onSave?.(template);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      setIsSaving(false);
    }
  }, [template, initialTemplate, onSave]);

  const clearError = useCallback(() => {
    setError(null);
  }, []);

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
