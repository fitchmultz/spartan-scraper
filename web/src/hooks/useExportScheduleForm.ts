/**
 * Purpose: Manage export-schedule authoring state for create and edit flows.
 * Responsibilities: Own form field state, validate local/webhook destination rules, seed drafts for editing or promotion, and submit typed create/update payloads.
 * Scope: Export-schedule form state only; modal visibility, list refreshes, and network implementations stay in the calling manager/container.
 * Usage: Call from `ExportScheduleManager` and wire the returned state/actions into `ExportScheduleForm`.
 * Invariants/Assumptions: Local destinations must stay within `DATA_DIR/exports`, webhook URLs are syntax-validated before submit, and shape/transform remain mutually exclusive.
 */

import { useState, useCallback } from "react";
import type { ExportSchedule, ExportScheduleRequest } from "../api";
import type { ExportScheduleFormData } from "../types/export-schedule";
import {
  defaultFormData,
  scheduleToFormData,
  formDataToScheduleRequest,
  hasShapeFormData,
  hasTransformFormData,
  validateLocalExportDestinationPath,
} from "../lib/export-schedule-utils";

interface UseExportScheduleFormReturn {
  formData: ExportScheduleFormData;
  formError: string | null;
  formSubmitting: boolean;
  editingId: string | null;
  setFormField: <K extends keyof ExportScheduleFormData>(
    field: K,
    value: ExportScheduleFormData[K],
  ) => void;
  setFormDataPartial: (data: Partial<ExportScheduleFormData>) => void;
  resetForm: () => void;
  initFormForEdit: (schedule: ExportSchedule) => void;
  initFormFromDraft: (draft: ExportScheduleFormData) => void;
  submitForm: (
    onCreate: (request: ExportScheduleRequest) => Promise<void>,
    onUpdate: (id: string, request: ExportScheduleRequest) => Promise<void>,
  ) => Promise<boolean>;
}

/**
 * Hook for managing export schedule form state and validation
 * @returns Form state and control functions
 */
export function useExportScheduleForm(): UseExportScheduleFormReturn {
  const [formData, setFormData] =
    useState<ExportScheduleFormData>(defaultFormData);
  const [formError, setFormError] = useState<string | null>(null);
  const [formSubmitting, setFormSubmitting] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);

  const setFormField = useCallback(
    <K extends keyof ExportScheduleFormData>(
      field: K,
      value: ExportScheduleFormData[K],
    ) => {
      setFormData((prev) => ({ ...prev, [field]: value }));
    },
    [],
  );

  const setFormDataPartial = useCallback(
    (data: Partial<ExportScheduleFormData>) => {
      setFormData((prev) => ({ ...prev, ...data }));
    },
    [],
  );

  const resetForm = useCallback(() => {
    setFormData(defaultFormData);
    setEditingId(null);
    setFormError(null);
  }, []);

  const initFormForEdit = useCallback((schedule: ExportSchedule) => {
    setFormData(scheduleToFormData(schedule));
    setEditingId(schedule.id);
    setFormError(null);
  }, []);

  const initFormFromDraft = useCallback((draft: ExportScheduleFormData) => {
    setFormData(draft);
    setEditingId(null);
    setFormError(null);
  }, []);

  const validateForm = useCallback((): boolean => {
    // Name is required
    if (!formData.name.trim()) {
      setFormError("Name is required");
      return false;
    }

    // Destination-specific validation
    if (formData.destinationType === "local" && !formData.localPath.trim()) {
      setFormError("Local path is required for local destination");
      return false;
    }

    if (formData.destinationType === "local") {
      const localPathError = validateLocalExportDestinationPath(
        formData.localPath,
      );
      if (localPathError) {
        setFormError(localPathError);
        return false;
      }
      if (formData.pathTemplate.trim()) {
        const pathTemplateError = validateLocalExportDestinationPath(
          formData.pathTemplate,
        );
        if (pathTemplateError) {
          setFormError(pathTemplateError);
          return false;
        }
      }
    }

    if (formData.destinationType === "webhook" && !formData.webhookUrl.trim()) {
      setFormError("Webhook URL is required for webhook destination");
      return false;
    }

    if (formData.destinationType === "webhook" && formData.webhookUrl.trim()) {
      try {
        new URL(formData.webhookUrl);
      } catch {
        setFormError("Webhook URL must be a valid URL");
        return false;
      }
    }

    if (formData.maxRetries < 0) {
      setFormError("Max retries must be 0 or greater");
      return false;
    }

    if (formData.baseDelayMs < 0) {
      setFormError("Base delay must be 0 or greater");
      return false;
    }

    if (hasTransformFormData(formData) && hasShapeFormData(formData)) {
      setFormError(
        "Export transform and export shaping cannot be combined on the same schedule",
      );
      return false;
    }

    return true;
  }, [formData]);

  const submitForm = useCallback(
    async (
      onCreate: (request: ExportScheduleRequest) => Promise<void>,
      onUpdate: (id: string, request: ExportScheduleRequest) => Promise<void>,
    ): Promise<boolean> => {
      if (!validateForm()) {
        return false;
      }

      setFormSubmitting(true);
      setFormError(null);

      try {
        const request = formDataToScheduleRequest(formData);
        if (editingId) {
          await onUpdate(editingId, request);
        } else {
          await onCreate(request);
        }
        resetForm();
        return true;
      } catch (err) {
        setFormError(
          err instanceof Error ? err.message : "Failed to save export schedule",
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
    initFormFromDraft,
    submitForm,
  };
}
