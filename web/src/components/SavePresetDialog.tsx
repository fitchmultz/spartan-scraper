/**
 * Purpose: Render the save preset dialog UI surface for the web operator experience.
 * Responsibilities: Define the component, its local view helpers, and the presentation logic owned by this file.
 * Scope: File-local UI behavior only; routing, persistence, and broader feature orchestration stay outside this file.
 * Usage: Import from the surrounding feature or route components that own this surface.
 * Invariants/Assumptions: Props and callbacks come from the surrounding feature contracts and should remain the single source of truth.
 */

import { useState, useCallback } from "react";
import { formatDisplayValue } from "../lib/formatting";
import type { JobType, PresetConfig } from "../types/presets";

interface SavePresetDialogProps {
  /** Whether the dialog is open */
  isOpen: boolean;
  /** Callback when dialog should close */
  onClose: () => void;
  /** Current job type */
  jobType: JobType;
  /** Current form configuration to save */
  currentConfig: PresetConfig;
  /** Callback when save is confirmed */
  onSave: (name: string, description: string) => void;
}

/**
 * Get a summary of the most important config values.
 */
function getConfigSummary(
  config: PresetConfig,
): { label: string; value: string }[] {
  const summary: { label: string; value: string }[] = [];

  if (config.headless !== undefined) {
    summary.push({
      label: "Headless",
      value: formatDisplayValue(config.headless, {
        emptyLabel: "Not set",
        trueLabel: "Yes",
        falseLabel: "No",
        maxLength: 30,
      }),
    });
  }
  if (config.usePlaywright !== undefined) {
    summary.push({
      label: "Playwright",
      value: formatDisplayValue(config.usePlaywright, {
        emptyLabel: "Not set",
        trueLabel: "Yes",
        falseLabel: "No",
        maxLength: 30,
      }),
    });
  }
  if (config.timeoutSeconds !== undefined) {
    summary.push({ label: "Timeout", value: `${config.timeoutSeconds}s` });
  }
  if (config.maxDepth !== undefined) {
    summary.push({ label: "Max Depth", value: String(config.maxDepth) });
  }
  if (config.maxPages !== undefined) {
    summary.push({ label: "Max Pages", value: String(config.maxPages) });
  }
  if (config.extractTemplate) {
    summary.push({ label: "Template", value: config.extractTemplate });
  }
  if (config.incremental !== undefined) {
    summary.push({
      label: "Incremental",
      value: formatDisplayValue(config.incremental, {
        emptyLabel: "Not set",
        trueLabel: "Yes",
        falseLabel: "No",
        maxLength: 30,
      }),
    });
  }

  return summary;
}

/**
 * Save Preset Dialog Component
 *
 * Modal for saving the current form configuration as a custom preset.
 */
export function SavePresetDialog({
  isOpen,
  onClose,
  jobType,
  currentConfig,
  onSave,
}: SavePresetDialogProps) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [error, setError] = useState<string | null>(null);

  const handleClose = useCallback(() => {
    setName("");
    setDescription("");
    setError(null);
    onClose();
  }, [onClose]);

  const handleSave = useCallback(() => {
    const trimmedName = name.trim();
    if (!trimmedName) {
      setError("Preset name is required");
      return;
    }
    if (trimmedName.length < 2) {
      setError("Name must be at least 2 characters");
      return;
    }
    if (trimmedName.length > 50) {
      setError("Name must be less than 50 characters");
      return;
    }

    onSave(trimmedName, description.trim());
    setName("");
    setDescription("");
    setError(null);
  }, [name, description, onSave]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Escape") {
        handleClose();
      } else if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
        handleSave();
      }
    },
    [handleClose, handleSave],
  );

  if (!isOpen) return null;

  const configSummary = getConfigSummary(currentConfig);
  const jobTypeLabel = jobType.charAt(0).toUpperCase() + jobType.slice(1);

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label="Save preset dialog"
      onKeyDown={handleKeyDown}
      style={{
        position: "fixed",
        inset: 0,
        background: "rgba(0, 0, 0, 0.7)",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        zIndex: 1000,
        padding: "20px",
      }}
      onClick={(e) => {
        if (e.target === e.currentTarget) {
          handleClose();
        }
      }}
    >
      <div
        style={{
          background: "var(--panel)",
          border: "1px solid var(--stroke)",
          borderRadius: "18px",
          padding: "24px",
          maxWidth: "480px",
          width: "100%",
          boxShadow: "var(--shadow)",
        }}
      >
        <h2 style={{ margin: "0 0 8px" }}>Save Custom Preset</h2>
        <p
          style={{
            margin: "0 0 20px",
            fontSize: "0.85rem",
            color: "var(--muted)",
          }}
        >
          Save the current {jobTypeLabel} configuration as a reusable preset.
        </p>

        {/* Config Summary */}
        <div
          style={{
            background: "rgba(0, 0, 0, 0.2)",
            borderRadius: "12px",
            padding: "12px",
            marginBottom: "20px",
          }}
        >
          <h3
            style={{
              margin: "0 0 8px",
              fontSize: "0.75rem",
              textTransform: "uppercase",
              letterSpacing: "0.1em",
              color: "var(--muted)",
            }}
          >
            Configuration Preview
          </h3>
          <div
            style={{
              display: "grid",
              gridTemplateColumns: "repeat(2, 1fr)",
              gap: "8px",
            }}
          >
            {configSummary.map(({ label, value }) => (
              <div key={label}>
                <div
                  style={{
                    fontSize: "0.7rem",
                    color: "var(--muted)",
                    textTransform: "uppercase",
                  }}
                >
                  {label}
                </div>
                <div
                  style={{
                    fontSize: "0.8rem",
                    color: "var(--text)",
                  }}
                >
                  {value}
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Name Input */}
        <div style={{ marginBottom: "16px" }}>
          <label htmlFor="preset-name" style={{ marginBottom: "6px" }}>
            Preset Name *
          </label>
          <input
            id="preset-name"
            type="text"
            value={name}
            onChange={(e) => {
              setName(e.target.value);
              setError(null);
            }}
            placeholder="e.g., My E-commerce Scraper"
            aria-invalid={error ? "true" : "false"}
            aria-describedby={error ? "preset-error" : undefined}
          />
        </div>

        {/* Description Input */}
        <div style={{ marginBottom: "16px" }}>
          <label htmlFor="preset-description" style={{ marginBottom: "6px" }}>
            Description (optional)
          </label>
          <textarea
            id="preset-description"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Brief description of what this preset is for..."
            rows={2}
            style={{ resize: "none" }}
          />
        </div>

        {/* Error Message */}
        {error && (
          <div
            id="preset-error"
            role="alert"
            style={{
              color: "var(--error, #ef4444)",
              fontSize: "0.8rem",
              marginBottom: "16px",
            }}
          >
            {error}
          </div>
        )}

        {/* Actions */}
        <div
          style={{
            display: "flex",
            gap: "12px",
            justifyContent: "flex-end",
          }}
        >
          <button type="button" className="secondary" onClick={handleClose}>
            Cancel
          </button>
          <button type="button" onClick={handleSave}>
            Save Preset
          </button>
        </div>

        {/* Keyboard hint */}
        <div
          style={{
            marginTop: "16px",
            textAlign: "center",
            fontSize: "0.7rem",
            color: "var(--muted)",
          }}
        >
          Press{" "}
          <kbd
            style={{
              background: "rgba(255, 255, 255, 0.1)",
              padding: "2px 6px",
              borderRadius: "4px",
            }}
          >
            Ctrl+Enter
          </kbd>{" "}
          to save,{" "}
          <kbd
            style={{
              background: "rgba(255, 255, 255, 0.1)",
              padding: "2px 6px",
              borderRadius: "4px",
            }}
          >
            Esc
          </kbd>{" "}
          to cancel
        </div>
      </div>
    </div>
  );
}
