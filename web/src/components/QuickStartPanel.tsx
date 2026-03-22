/**
 * Purpose: Present a compact preset rail for the active job workflow on the new-job route.
 * Responsibilities: Highlight recommended presets for the current target and expose quick launch actions for presets, AI help, and saving.
 * Scope: `/jobs/new` quick-start surface only.
 * Usage: Render from the job-submission workspace beside the guided and expert flows.
 * Invariants/Assumptions: Preset recommendations stay advisory, the active workflow is controlled by the main wizard, and quick actions never submit work automatically.
 */

import { useMemo } from "react";
import { formatSecondsAsApproximateDuration } from "../lib/formatting";
import { detectPresetsForUrl } from "../lib/preset-matcher";
import type { JobPreset, JobType } from "../types/presets";

interface QuickStartPanelProps {
  presets: JobPreset[];
  activeJobType: JobType;
  onSelectPreset: (preset: JobPreset) => void;
  onSavePreset?: () => void;
  onOpenAssistant?: () => void;
  onOpenTemplateAssistant?: () => void;
  currentUrl?: string;
}

const JOB_TYPE_COPY: Record<
  JobType,
  { label: string; summary: string; empty: string }
> = {
  scrape: {
    label: "Scrape",
    summary: "Single-page extraction presets for fast one-off jobs.",
    empty: "No scrape presets yet. Save this workflow once you tune it.",
  },
  crawl: {
    label: "Crawl",
    summary: "Site-wide patterns for bounded, repeatable collection runs.",
    empty: "No crawl presets yet. Save a crawl recipe to reuse it here.",
  },
  research: {
    label: "Research",
    summary: "Research prompts and source bundles for synthesis-heavy runs.",
    empty:
      "No research presets yet. Save one after shaping a strong source set.",
  },
};

function PresetChip({
  preset,
  onSelect,
  tone = "default",
}: {
  preset: JobPreset;
  onSelect: () => void;
  tone?: "default" | "recommended";
}) {
  return (
    <button
      type="button"
      className={`job-quickstart__preset-chip ${tone === "recommended" ? "is-recommended" : ""}`}
      onClick={onSelect}
    >
      <div className="job-quickstart__preset-chip-copy">
        <strong>{preset.name}</strong>
        <span>{preset.description}</span>
      </div>
      <div className="job-quickstart__preset-chip-meta">
        {tone === "recommended" ? (
          <span className="job-quickstart__badge">Suggested</span>
        ) : null}
        <span>
          {formatSecondsAsApproximateDuration(preset.resources.timeSeconds)}
        </span>
        <span>{preset.isBuiltIn ? "Built-in" : "Saved"}</span>
      </div>
    </button>
  );
}

export function QuickStartPanel({
  presets,
  activeJobType,
  onSelectPreset,
  onSavePreset,
  onOpenAssistant,
  onOpenTemplateAssistant,
  currentUrl,
}: QuickStartPanelProps) {
  const activePresets = useMemo(
    () => presets.filter((preset) => preset.jobType === activeJobType),
    [activeJobType, presets],
  );

  const recommendedPresets = useMemo(() => {
    if (!currentUrl) {
      return [];
    }
    return detectPresetsForUrl(currentUrl, activePresets).slice(0, 2);
  }, [activePresets, currentUrl]);

  const recommendedIds = new Set(recommendedPresets.map((preset) => preset.id));

  const activeCopy = JOB_TYPE_COPY[activeJobType];
  const featuredPresets =
    recommendedPresets.length > 0
      ? recommendedPresets
      : activePresets.slice(0, 2);
  const remainingPresets = activePresets.filter(
    (preset) => !featuredPresets.some((featured) => featured.id === preset.id),
  );

  return (
    <section className="panel job-quickstart">
      <div className="job-quickstart__header">
        <div>
          <div className="job-quickstart__eyebrow">Quick Start</div>
          <h2>{activeCopy.label} presets</h2>
          <p>{activeCopy.summary}</p>
        </div>
        <div className="job-quickstart__header-actions">
          {onOpenAssistant ? (
            <button
              type="button"
              className="secondary"
              onClick={onOpenAssistant}
            >
              Open AI assistant
            </button>
          ) : null}
          {onOpenTemplateAssistant ? (
            <button
              type="button"
              className="secondary"
              onClick={onOpenTemplateAssistant}
            >
              Open template assistant
            </button>
          ) : null}
          {onSavePreset ? (
            <button type="button" className="secondary" onClick={onSavePreset}>
              Save Preset
            </button>
          ) : null}
        </div>
      </div>

      <div className="job-quickstart__group">
        <div className="job-quickstart__group-label">
          {recommendedPresets.length > 0
            ? "Matches this target"
            : "Featured presets"}
        </div>
        {activePresets.length === 0 ? (
          <p className="job-quickstart__empty">{activeCopy.empty}</p>
        ) : (
          <div className="job-quickstart__preset-grid">
            {featuredPresets.map((preset) => (
              <PresetChip
                key={preset.id}
                preset={preset}
                tone={recommendedIds.has(preset.id) ? "recommended" : "default"}
                onSelect={() => onSelectPreset(preset)}
              />
            ))}
          </div>
        )}
      </div>

      {remainingPresets.length > 0 ? (
        <details className="job-quickstart__more">
          <summary>
            <span>More {activeCopy.label.toLowerCase()} presets</span>
            <small>
              {remainingPresets.length} additional workflow
              {remainingPresets.length === 1 ? "" : "s"}
            </small>
          </summary>
          <div className="job-quickstart__preset-grid job-quickstart__preset-grid--more">
            {remainingPresets.map((preset) => (
              <PresetChip
                key={preset.id}
                preset={preset}
                onSelect={() => onSelectPreset(preset)}
              />
            ))}
          </div>
        </details>
      ) : null}
    </section>
  );
}
