/**
 * Purpose: Coordinate the quick-start preset rail and save-preset dialog for the new-job route.
 * Responsibilities: Bridge preset selection into route-owned navigation, open the save dialog with the current config snapshot, and keep the quick-start rail focused on discovery rather than direct form mutation.
 * Scope: Preset management for single-job creation only.
 * Usage: Render from `App.tsx` alongside `JobSubmissionContainer` on `/jobs/new`.
 * Invariants/Assumptions: The route owner applies presets exactly once, the active job type is controlled by the surrounding route, and scrolling to `#forms` remains the correct affordance after selecting a preset.
 */

import { useState, useCallback } from "react";
import { QuickStartPanel } from "../../components/QuickStartPanel";
import { SavePresetDialog } from "../../components/SavePresetDialog";
import type { JobPreset, JobType, PresetConfig } from "../../types/presets";

interface PresetContainerProps {
  presets: JobPreset[];
  activeTab: JobType;
  setActiveTab: (tab: JobType) => void;
  savePreset: (
    name: string,
    desc: string,
    type: JobType,
    config: PresetConfig,
  ) => void;
  getCurrentConfig: () => PresetConfig;
  getCurrentUrl: () => string;
  onSelectPreset: (preset: JobPreset) => void;
  onOpenAIPreview?: (url?: string) => void;
  onOpenTemplateGenerator?: () => void;
}

export function PresetContainer({
  presets,
  activeTab,
  setActiveTab,
  savePreset,
  getCurrentConfig,
  getCurrentUrl,
  onSelectPreset,
  onOpenAIPreview,
  onOpenTemplateGenerator,
}: PresetContainerProps) {
  const [isSaveDialogOpen, setIsSaveDialogOpen] = useState(false);

  const handleSelectPreset = useCallback(
    (preset: JobPreset) => {
      onSelectPreset(preset);

      const formsSection = document.getElementById("forms");
      formsSection?.scrollIntoView({ behavior: "smooth", block: "start" });
    },
    [onSelectPreset],
  );

  const handleSavePreset = useCallback(() => {
    setIsSaveDialogOpen(true);
  }, []);

  const handleConfirmSavePreset = useCallback(
    (name: string, description: string) => {
      const config = getCurrentConfig();
      savePreset(name, description, activeTab, config);
      setIsSaveDialogOpen(false);
    },
    [activeTab, getCurrentConfig, savePreset],
  );

  return (
    <>
      <div data-tour="quickstart">
        <QuickStartPanel
          presets={presets}
          activeJobType={activeTab}
          onJobTypeChange={setActiveTab}
          onSelectPreset={handleSelectPreset}
          onSavePreset={handleSavePreset}
          onOpenAIPreview={onOpenAIPreview}
          onOpenTemplateGenerator={onOpenTemplateGenerator}
          currentUrl={getCurrentUrl()}
        />
      </div>

      <SavePresetDialog
        isOpen={isSaveDialogOpen}
        onClose={() => setIsSaveDialogOpen(false)}
        jobType={activeTab}
        currentConfig={getCurrentConfig()}
        onSave={handleConfirmSavePreset}
      />
    </>
  );
}
