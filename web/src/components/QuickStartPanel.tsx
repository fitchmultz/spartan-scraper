/**
 * Quick Start Panel Component
 *
 * Displays preset cards for common scraping patterns, allowing users to
 * quickly configure forms with pre-defined settings. Includes job type
 * filtering, URL-based suggestions, and visual resource indicators.
 *
 * @module components/QuickStartPanel
 */

import { useMemo, useState } from "react";
import { formatSecondsAsApproximateDuration } from "../lib/formatting";
import type { JobPreset, JobType } from "../types/presets";
import { detectPresetsForUrl } from "../lib/preset-matcher";

interface QuickStartPanelProps {
  /** All available presets */
  presets: JobPreset[];
  /** Currently selected job type tab */
  activeJobType: JobType;
  /** Called when a preset is selected */
  onSelectPreset: (preset: JobPreset) => void;
  /** Called when user wants to save current config as preset */
  onSavePreset?: () => void;
  /** Optional URL for showing matching presets */
  currentUrl?: string;
}

/**
 * Get resource level indicator.
 */
function ResourceIndicator({
  level,
  label,
}: {
  level: "low" | "medium" | "high";
  label: string;
}) {
  const dots = {
    low: 1,
    medium: 2,
    high: 3,
  };

  const color =
    level === "low"
      ? "var(--success, #22c55e)"
      : level === "medium"
        ? "var(--warning, #f59e0b)"
        : "var(--error, #ef4444)";

  return (
    <span
      title={`${label}: ${level}`}
      style={{
        display: "inline-flex",
        gap: "2px",
        alignItems: "center",
      }}
    >
      {Array.from({ length: dots[level] }).map((_, i) => (
        <span
          // biome-ignore lint/suspicious/noArrayIndexKey: static array
          key={i}
          style={{
            width: "6px",
            height: "6px",
            borderRadius: "50%",
            backgroundColor: color,
          }}
        />
      ))}
    </span>
  );
}

/**
 * Individual preset card component.
 */
function PresetCard({
  preset,
  onSelect,
  isRecommended,
}: {
  preset: JobPreset;
  onSelect: () => void;
  isRecommended?: boolean;
}) {
  const [isHovered, setIsHovered] = useState(false);

  return (
    <button
      type="button"
      onClick={onSelect}
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
      style={{
        display: "flex",
        flexDirection: "column",
        alignItems: "flex-start",
        padding: "16px",
        gap: "8px",
        background: isRecommended
          ? "rgba(255, 183, 0, 0.12)"
          : isHovered
            ? "rgba(255, 255, 255, 0.08)"
            : "rgba(12, 12, 16, 0.6)",
        border: `1px solid ${isRecommended ? "rgba(255, 183, 0, 0.4)" : isHovered ? "rgba(255, 183, 0, 0.3)" : "rgba(255, 255, 255, 0.08)"}`,
        borderRadius: "14px",
        cursor: "pointer",
        textAlign: "left",
        transition: "all 0.15s ease",
        width: "100%",
        position: "relative",
      }}
    >
      {isRecommended && (
        <span
          style={{
            position: "absolute",
            top: "8px",
            right: "8px",
            fontSize: "0.65rem",
            textTransform: "uppercase",
            letterSpacing: "0.05em",
            padding: "2px 6px",
            borderRadius: "4px",
            background: "rgba(255, 183, 0, 0.2)",
            color: "var(--accent)",
          }}
        >
          Recommended
        </span>
      )}
      <span style={{ fontSize: "1.5rem" }}>{preset.icon}</span>
      <span
        style={{
          fontWeight: 600,
          fontSize: "0.95rem",
          color: "var(--text)",
          marginTop: "4px",
        }}
      >
        {preset.name}
      </span>
      <span
        style={{
          fontSize: "0.8rem",
          color: "var(--muted)",
          lineHeight: 1.4,
        }}
      >
        {preset.description}
      </span>
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: "12px",
          marginTop: "8px",
          fontSize: "0.7rem",
          color: "var(--muted)",
        }}
      >
        <span style={{ display: "flex", alignItems: "center", gap: "4px" }}>
          ⏱️ {formatSecondsAsApproximateDuration(preset.resources.timeSeconds)}
        </span>
        <span style={{ display: "flex", alignItems: "center", gap: "4px" }}>
          <ResourceIndicator level={preset.resources.cpu} label="CPU" />
        </span>
        <span style={{ display: "flex", alignItems: "center", gap: "4px" }}>
          <ResourceIndicator level={preset.resources.memory} label="Memory" />
        </span>
      </div>
      <div
        style={{
          display: "flex",
          flexWrap: "wrap",
          gap: "4px",
          marginTop: "8px",
        }}
      >
        {preset.useCases.slice(0, 2).map((useCase) => (
          <span
            key={useCase}
            style={{
              fontSize: "0.65rem",
              padding: "2px 6px",
              borderRadius: "4px",
              background: "rgba(255, 255, 255, 0.06)",
              color: "var(--muted)",
            }}
          >
            {useCase}
          </span>
        ))}
      </div>
    </button>
  );
}

/**
 * Job type tab button.
 */
function JobTypeTab({
  label,
  isActive,
  count,
  onClick,
}: {
  label: string;
  isActive: boolean;
  count: number;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      style={{
        padding: "8px 16px",
        borderRadius: "999px",
        border: "none",
        background: isActive
          ? "linear-gradient(135deg, var(--accent), var(--accent-strong))"
          : "transparent",
        color: isActive ? "#1a1200" : "var(--text)",
        fontWeight: isActive ? 600 : 400,
        fontSize: "0.85rem",
        cursor: "pointer",
        display: "flex",
        alignItems: "center",
        gap: "6px",
      }}
    >
      {label}
      <span
        style={{
          fontSize: "0.7rem",
          opacity: 0.7,
          background: isActive
            ? "rgba(0, 0, 0, 0.2)"
            : "rgba(255, 255, 255, 0.1)",
          padding: "2px 6px",
          borderRadius: "999px",
        }}
      >
        {count}
      </span>
    </button>
  );
}

/**
 * Quick Start Panel Component
 *
 * Displays preset cards for quick job configuration.
 */
export function QuickStartPanel({
  presets,
  activeJobType,
  onSelectPreset,
  onSavePreset,
  currentUrl,
}: QuickStartPanelProps) {
  const [selectedType, setSelectedType] = useState<JobType | "all">("all");

  // Filter presets by selected type
  const filteredPresets = useMemo(() => {
    if (selectedType === "all") {
      return presets.filter((p) => p.jobType === activeJobType);
    }
    return presets.filter((p) => p.jobType === selectedType);
  }, [presets, selectedType, activeJobType]);

  // Get URL-matching presets
  const recommendedPresets = useMemo(() => {
    if (!currentUrl) return [];
    return detectPresetsForUrl(currentUrl, filteredPresets);
  }, [currentUrl, filteredPresets]);

  // Count presets by type
  const counts = useMemo(() => {
    return {
      scrape: presets.filter((p) => p.jobType === "scrape").length,
      crawl: presets.filter((p) => p.jobType === "crawl").length,
      research: presets.filter((p) => p.jobType === "research").length,
    };
  }, [presets]);

  // Split into recommended and other presets
  const recommendedIds = new Set(recommendedPresets.map((p) => p.id));
  const otherPresets = filteredPresets.filter((p) => !recommendedIds.has(p.id));

  return (
    <section
      style={{
        padding: "24px",
        background: "var(--panel)",
        border: "1px solid var(--stroke)",
        borderRadius: "18px",
        boxShadow: "var(--shadow)",
      }}
    >
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "flex-start",
          marginBottom: "20px",
          flexWrap: "wrap",
          gap: "16px",
        }}
      >
        <div>
          <h2
            style={{
              margin: "0 0 4px",
              fontSize: "1.25rem",
            }}
          >
            Quick Start
          </h2>
          <p
            style={{
              margin: 0,
              fontSize: "0.85rem",
              color: "var(--muted)",
            }}
          >
            Choose a preset to quickly configure your job
          </p>
        </div>
        {onSavePreset && (
          <button
            type="button"
            onClick={onSavePreset}
            className="secondary"
            style={{
              padding: "8px 16px",
              fontSize: "0.8rem",
            }}
          >
            + Save Current as Preset
          </button>
        )}
      </div>

      {/* Job Type Tabs */}
      <div
        style={{
          display: "flex",
          gap: "8px",
          marginBottom: "20px",
          flexWrap: "wrap",
        }}
      >
        <JobTypeTab
          label="Scrape"
          isActive={selectedType === "scrape"}
          count={counts.scrape}
          onClick={() => setSelectedType("scrape")}
        />
        <JobTypeTab
          label="Crawl"
          isActive={selectedType === "crawl"}
          count={counts.crawl}
          onClick={() => setSelectedType("crawl")}
        />
        <JobTypeTab
          label="Research"
          isActive={selectedType === "research"}
          count={counts.research}
          onClick={() => setSelectedType("research")}
        />
      </div>

      {/* Recommended Section */}
      {recommendedPresets.length > 0 && (
        <div style={{ marginBottom: "20px" }}>
          <h3
            style={{
              margin: "0 0 12px",
              fontSize: "0.8rem",
              textTransform: "uppercase",
              letterSpacing: "0.1em",
              color: "var(--accent)",
            }}
          >
            Recommended for this URL
          </h3>
          <div
            style={{
              display: "grid",
              gap: "12px",
              gridTemplateColumns: "repeat(auto-fill, minmax(200px, 1fr))",
            }}
          >
            {recommendedPresets.map((preset) => (
              <PresetCard
                key={preset.id}
                preset={preset}
                onSelect={() => onSelectPreset(preset)}
                isRecommended
              />
            ))}
          </div>
        </div>
      )}

      {/* All Presets Grid */}
      <div
        style={{
          display: "grid",
          gap: "12px",
          gridTemplateColumns: "repeat(auto-fill, minmax(200px, 1fr))",
        }}
      >
        {otherPresets.map((preset) => (
          <PresetCard
            key={preset.id}
            preset={preset}
            onSelect={() => onSelectPreset(preset)}
          />
        ))}
      </div>

      {filteredPresets.length === 0 && (
        <div
          style={{
            textAlign: "center",
            padding: "32px",
            color: "var(--muted)",
          }}
        >
          <p>No presets available for this job type.</p>
        </div>
      )}
    </section>
  );
}
