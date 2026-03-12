/**
 * Quick Start Panel Component
 *
 * Presents a compact preset rail for the active workflow so the route stays
 * focused on one job type at a time.
 *
 * @module components/QuickStartPanel
 */

import { useMemo } from "react";
import { formatSecondsAsApproximateDuration } from "../lib/formatting";
import { detectPresetsForUrl } from "../lib/preset-matcher";
import type { JobPreset, JobType } from "../types/presets";

interface QuickStartPanelProps {
  presets: JobPreset[];
  activeJobType: JobType;
  onJobTypeChange: (jobType: JobType) => void;
  onSelectPreset: (preset: JobPreset) => void;
  onSavePreset?: () => void;
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

function JobTypeButton({
  jobType,
  activeJobType,
  count,
  onClick,
}: {
  jobType: JobType;
  activeJobType: JobType;
  count: number;
  onClick: () => void;
}) {
  const isActive = activeJobType === jobType;

  return (
    <button
      type="button"
      className={`job-quickstart__type ${isActive ? "is-active" : ""}`}
      onClick={onClick}
    >
      <span>{JOB_TYPE_COPY[jobType].label}</span>
      <strong>{count}</strong>
    </button>
  );
}

function PresetRailCard({
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
      className={`job-quickstart__preset ${tone === "recommended" ? "is-recommended" : ""}`}
      onClick={onSelect}
    >
      <div className="job-quickstart__preset-header">
        <div>
          <strong>{preset.name}</strong>
          <p>{preset.description}</p>
        </div>
        {tone === "recommended" ? (
          <span className="job-quickstart__badge">Suggested</span>
        ) : null}
      </div>
      <div className="job-quickstart__preset-meta">
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
  onJobTypeChange,
  onSelectPreset,
  onSavePreset,
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
  const fallbackPresets = activePresets
    .filter((preset) => !recommendedIds.has(preset.id))
    .slice(0, 4);

  const presetCounts = useMemo(
    () => ({
      scrape: presets.filter((preset) => preset.jobType === "scrape").length,
      crawl: presets.filter((preset) => preset.jobType === "crawl").length,
      research: presets.filter((preset) => preset.jobType === "research")
        .length,
    }),
    [presets],
  );

  const activeCopy = JOB_TYPE_COPY[activeJobType];

  return (
    <section className="panel job-quickstart">
      <div className="job-quickstart__header">
        <div>
          <div className="job-quickstart__eyebrow">Quick Start</div>
          <h2>Presets and workflow switching</h2>
          <p>{activeCopy.summary}</p>
        </div>
        {onSavePreset ? (
          <button type="button" className="secondary" onClick={onSavePreset}>
            Save Preset
          </button>
        ) : null}
      </div>

      <div
        className="job-quickstart__types"
        role="tablist"
        aria-label="Job types"
      >
        <JobTypeButton
          jobType="scrape"
          activeJobType={activeJobType}
          count={presetCounts.scrape}
          onClick={() => onJobTypeChange("scrape")}
        />
        <JobTypeButton
          jobType="crawl"
          activeJobType={activeJobType}
          count={presetCounts.crawl}
          onClick={() => onJobTypeChange("crawl")}
        />
        <JobTypeButton
          jobType="research"
          activeJobType={activeJobType}
          count={presetCounts.research}
          onClick={() => onJobTypeChange("research")}
        />
      </div>

      {recommendedPresets.length > 0 ? (
        <div className="job-quickstart__group">
          <div className="job-quickstart__group-label">Matches this target</div>
          <div className="job-quickstart__preset-list">
            {recommendedPresets.map((preset) => (
              <PresetRailCard
                key={preset.id}
                preset={preset}
                tone="recommended"
                onSelect={() => onSelectPreset(preset)}
              />
            ))}
          </div>
        </div>
      ) : null}

      <div className="job-quickstart__group">
        <div className="job-quickstart__group-label">
          {recommendedPresets.length > 0 ? "More presets" : "Available presets"}
        </div>
        {activePresets.length === 0 ? (
          <p className="job-quickstart__empty">{activeCopy.empty}</p>
        ) : (
          <div className="job-quickstart__preset-list">
            {(fallbackPresets.length > 0 ? fallbackPresets : activePresets).map(
              (preset) => (
                <PresetRailCard
                  key={preset.id}
                  preset={preset}
                  onSelect={() => onSelectPreset(preset)}
                />
              ),
            )}
          </div>
        )}
      </div>
    </section>
  );
}
