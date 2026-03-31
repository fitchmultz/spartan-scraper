/**
 * Purpose: Render the export schedule create/edit dialog for recurring operator exports.
 * Responsibilities: Derive transform/shape state, coordinate AI assistant visibility, and compose the extracted dialog sections around the controlled form contract.
 * Scope: Export schedule authoring only; API persistence, dialog visibility ownership, and draft state management stay in parent components/hooks.
 * Usage: Render from `ExportScheduleManager` with controlled `formData`, mutation callbacks, and optional AI/promotion context.
 * Invariants/Assumptions: Transform and shape authoring remain mutually exclusive, unsupported formats cannot save shape config, and AI assistants close automatically when AI capability is unavailable.
 */

import { useEffect, useState } from "react";

import {
  formDataToShapeConfig,
  formDataToTransformConfig,
  formatExportShapeSummary,
  formatExportTransformSummary,
  hasShapeFormData,
  hasTransformFormData,
  shapeConfigToFormData,
  supportsExportShapeFormat,
  transformConfigToFormData,
} from "../../lib/export-schedule-utils";
import type { ExportScheduleFormProps } from "../../types/export-schedule";
import { AIExportShapeAssistant } from "../AIExportShapeAssistant";
import { AIExportTransformAssistant } from "../AIExportTransformAssistant";
import { describeAICapability } from "../ai-assistant";
import {
  ExportScheduleBasicInfoSection,
  ExportScheduleConfigSection,
  ExportScheduleDialogShell,
  ExportScheduleFiltersSection,
  ExportScheduleFormActions,
  ExportScheduleRetrySection,
  ExportScheduleShapeSection,
  ExportScheduleTransformSection,
} from "./ExportScheduleFormSections";

export function ExportScheduleForm({
  formData,
  formError,
  formSubmitting,
  isEditing,
  onChange,
  onSubmit,
  onCancel,
  aiStatus = null,
  promotionSeed = null,
  onClearPromotionSeed,
  onOpenSourceJob,
}: ExportScheduleFormProps) {
  const [showShapeAssistant, setShowShapeAssistant] = useState(false);
  const [showTransformAssistant, setShowTransformAssistant] = useState(false);
  const aiCapability = describeAICapability(
    aiStatus,
    "Configure transforms and shapes manually in this form.",
  );
  const aiAssistantUnavailable = aiCapability.unavailable;
  const aiAssistantMessage = aiCapability.message;
  const shapeSupported = supportsExportShapeFormat(formData.format);
  const stagedShape = hasShapeFormData(formData);
  const stagedTransform = hasTransformFormData(formData);
  const currentTransform = formDataToTransformConfig(formData);
  const transformSummary = formatExportTransformSummary(currentTransform);
  const transformActive = stagedTransform;
  const shapeLockedByTransform = transformActive;
  const transformLockedByShape = stagedShape;
  const currentShape =
    shapeSupported && !shapeLockedByTransform
      ? formDataToShapeConfig(formData)
      : undefined;
  const shapeSummary = shapeSupported
    ? shapeLockedByTransform
      ? "Disabled by transform"
      : formatExportShapeSummary(currentShape)
    : stagedShape
      ? "Staged (unsupported format)"
      : "Default";

  useEffect(() => {
    if (!aiAssistantUnavailable) {
      return;
    }
    setShowShapeAssistant(false);
    setShowTransformAssistant(false);
  }, [aiAssistantUnavailable]);

  const toggleJobKind = (kind: string) => {
    const current = formData.filterJobKinds;
    if (current.includes(kind as (typeof current)[number])) {
      onChange({
        filterJobKinds: current.filter((entry) => entry !== kind),
      });
      return;
    }

    onChange({
      filterJobKinds: [...current, kind as (typeof current)[number]],
    });
  };

  const toggleJobStatus = (status: string) => {
    const current = formData.filterJobStatus;
    if (current.includes(status as (typeof current)[number])) {
      onChange({
        filterJobStatus: current.filter((entry) => entry !== status),
      });
      return;
    }

    onChange({
      filterJobStatus: [...current, status as (typeof current)[number]],
    });
  };

  return (
    <>
      <ExportScheduleDialogShell
        formError={formError}
        aiAssistantMessage={aiAssistantMessage}
        promotionSeed={promotionSeed}
        onClearPromotionSeed={onClearPromotionSeed}
        onOpenSourceJob={onOpenSourceJob}
        isEditing={isEditing}
        onCancel={onCancel}
        onSubmit={onSubmit}
      >
        <ExportScheduleBasicInfoSection
          formData={formData}
          onChange={onChange}
        />

        <ExportScheduleFiltersSection
          formData={formData}
          onChange={onChange}
          toggleJobKind={toggleJobKind}
          toggleJobStatus={toggleJobStatus}
        />

        <ExportScheduleConfigSection formData={formData} onChange={onChange} />

        <ExportScheduleTransformSection
          formData={formData}
          onChange={onChange}
          transformSummary={transformSummary}
          stagedTransform={stagedTransform}
          transformLockedByShape={transformLockedByShape}
          aiAssistantUnavailable={aiAssistantUnavailable}
          aiAssistantMessage={aiAssistantMessage}
          onOpenTransformAssistant={() => setShowTransformAssistant(true)}
        />

        <ExportScheduleShapeSection
          formData={formData}
          onChange={onChange}
          shapeSummary={shapeSummary}
          stagedShape={stagedShape}
          shapeSupported={shapeSupported}
          shapeLockedByTransform={shapeLockedByTransform}
          aiAssistantUnavailable={aiAssistantUnavailable}
          aiAssistantMessage={aiAssistantMessage}
          onOpenShapeAssistant={() => setShowShapeAssistant(true)}
        />

        <ExportScheduleRetrySection formData={formData} onChange={onChange} />

        <ExportScheduleFormActions
          formSubmitting={formSubmitting}
          isEditing={isEditing}
          onCancel={onCancel}
        />
      </ExportScheduleDialogShell>

      {showTransformAssistant ? (
        <AIExportTransformAssistant
          isOpen={showTransformAssistant}
          onClose={() => setShowTransformAssistant(false)}
          aiStatus={aiStatus}
          currentTransform={currentTransform}
          onApplyTransform={(transform) =>
            onChange(transformConfigToFormData(transform))
          }
        />
      ) : null}

      {shapeSupported && showShapeAssistant ? (
        <AIExportShapeAssistant
          isOpen={showShapeAssistant}
          onClose={() => setShowShapeAssistant(false)}
          aiStatus={aiStatus}
          format={formData.format as "md" | "csv" | "xlsx"}
          currentShape={currentShape}
          onApplyShape={(shape) => onChange(shapeConfigToFormData(shape))}
        />
      ) : null}
    </>
  );
}
