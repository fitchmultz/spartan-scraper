/**
 * useWatchForm Hook
 *
 * Manages form state and validation for watch creation/editing.
 * Encapsulates form field updates, validation logic, and submission handling.
 *
 * This hook does NOT handle:
 * - API calls (those are passed as callbacks)
 * - Modal visibility state
 * - Watch list management
 *
 * @module hooks/useWatchForm
 */

import { useState, useCallback } from "react";
import type { Watch, WatchInput } from "../api";
import type { WatchFormData } from "../types/watch";
import {
  defaultFormData,
  watchToFormData,
  formDataToWatchInput,
} from "../lib/watch-utils";

interface UseWatchFormReturn {
  formData: WatchFormData;
  formError: string | null;
  formSubmitting: boolean;
  editingId: string | null;
  setFormField: <K extends keyof WatchFormData>(
    field: K,
    value: WatchFormData[K],
  ) => void;
  setFormDataPartial: (data: Partial<WatchFormData>) => void;
  resetForm: () => void;
  initFormForEdit: (watch: Watch) => void;
  submitForm: (
    onCreate: (input: WatchInput) => Promise<void>,
    onUpdate: (id: string, input: WatchInput) => Promise<void>,
  ) => Promise<boolean>;
}

/**
 * Hook for managing watch form state and validation
 * @returns Form state and control functions
 */
export function useWatchForm(): UseWatchFormReturn {
  const [formData, setFormData] = useState<WatchFormData>(defaultFormData);
  const [formError, setFormError] = useState<string | null>(null);
  const [formSubmitting, setFormSubmitting] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);

  const setFormField = useCallback(
    <K extends keyof WatchFormData>(field: K, value: WatchFormData[K]) => {
      setFormData((prev) => ({ ...prev, [field]: value }));
    },
    [],
  );

  const setFormDataPartial = useCallback((data: Partial<WatchFormData>) => {
    setFormData((prev) => ({ ...prev, ...data }));
  }, []);

  const resetForm = useCallback(() => {
    setFormData(defaultFormData);
    setEditingId(null);
    setFormError(null);
  }, []);

  const initFormForEdit = useCallback((watch: Watch) => {
    setFormData(watchToFormData(watch));
    setEditingId(watch.id);
    setFormError(null);
  }, []);

  const validateForm = useCallback((): boolean => {
    if (!formData.url.trim()) {
      setFormError("URL is required");
      return false;
    }

    if (formData.intervalSeconds < 60) {
      setFormError("Interval must be at least 60 seconds");
      return false;
    }

    if (formData.jobTriggerKind && !formData.jobTriggerRequest.trim()) {
      setFormError(
        "Watch job trigger request JSON is required when a trigger kind is selected",
      );
      return false;
    }

    if (formData.jobTriggerRequest.trim()) {
      try {
        JSON.parse(formData.jobTriggerRequest);
      } catch (error) {
        setFormError(
          `Watch job trigger request must be valid JSON: ${error instanceof Error ? error.message : String(error)}`,
        );
        return false;
      }
    }

    return true;
  }, [
    formData.url,
    formData.intervalSeconds,
    formData.jobTriggerKind,
    formData.jobTriggerRequest,
  ]);

  const submitForm = useCallback(
    async (
      onCreate: (input: WatchInput) => Promise<void>,
      onUpdate: (id: string, input: WatchInput) => Promise<void>,
    ): Promise<boolean> => {
      if (!validateForm()) {
        return false;
      }

      setFormSubmitting(true);
      setFormError(null);

      try {
        const input = formDataToWatchInput(formData);
        if (editingId) {
          await onUpdate(editingId, input);
        } else {
          await onCreate(input);
        }
        resetForm();
        return true;
      } catch (err) {
        setFormError(
          err instanceof Error ? err.message : "Failed to save watch",
        );
        return false;
      } finally {
        setFormSubmitting(false);
      }
    },
    [formData, editingId, validateForm, resetForm],
  );

  return {
    formData,
    formError,
    formSubmitting,
    editingId,
    setFormField,
    setFormDataPartial,
    resetForm,
    initFormForEdit,
    submitForm,
  };
}
