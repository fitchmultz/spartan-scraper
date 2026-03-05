/**
 * PresetContainer - Container component for preset management functionality
 *
 * This component encapsulates all preset-related UI state and operations:
 * - Managing save preset dialog state
 * - Handling preset selection and application
 * - Rendering QuickStartPanel and SavePresetDialog
 *
 * It does NOT handle:
 * - Job submission directly
 * - Form state management (uses applyPreset callback)
 * - Watch or chain management
 *
 * @module PresetContainer
 */

import { useState, useCallback } from "react";
import { QuickStartPanel } from "../../components/QuickStartPanel";
import { SavePresetDialog } from "../../components/SavePresetDialog";
import type { JobPreset, JobType, PresetConfig } from "../../types/presets";

interface PresetContainerProps {
  presets: JobPreset[];
  activeTab: JobType;
  setActiveTab: (tab: JobType) => void;
  applyPreset: (config: PresetConfig) => void;
  savePreset: (
    name: string,
    desc: string,
    type: JobType,
    config: PresetConfig,
  ) => void;
  getCurrentConfig: () => PresetConfig;
  getCurrentUrl: () => string;
  onSelectPreset: (preset: JobPreset) => void;
}

export function PresetContainer({
  presets,
  activeTab,
  setActiveTab,
  applyPreset,
  savePreset,
  getCurrentConfig,
  getCurrentUrl,
  onSelectPreset,
}: PresetContainerProps) {
  const [isSaveDialogOpen, setIsSaveDialogOpen] = useState(false);

  const handleSelectPreset = useCallback(
    (preset: JobPreset) => {
      // Switch to the appropriate tab
      setActiveTab(preset.jobType);

      // Apply preset config to form state
      applyPreset(preset.config);

      // Notify parent for form ref updates
      onSelectPreset(preset);

      // Scroll to forms section
      const formsSection = document.getElementById("forms");
      if (formsSection) {
        formsSection.scrollIntoView({ behavior: "smooth", block: "start" });
      }
    },
    [setActiveTab, applyPreset, onSelectPreset],
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
          onSelectPreset={handleSelectPreset}
          onSavePreset={handleSavePreset}
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
