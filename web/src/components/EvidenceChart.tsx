/**
 * Purpose: Render the evidence chart UI surface for the web operator experience.
 * Responsibilities: Define the component, its local view helpers, and the presentation logic owned by this file.
 * Scope: File-local UI behavior only; routing, persistence, and broader feature orchestration stay outside this file.
 * Usage: Import from the surrounding feature or route components that own this surface.
 * Invariants/Assumptions: Props and callbacks come from the surrounding feature contracts and should remain the single source of truth.
 */

import { useMemo, useState } from "react";
import type { EvidenceItem, ClusterItem } from "../types";

interface EvidenceChartProps {
  /** Evidence items to visualize */
  evidence: EvidenceItem[];
  /** Cluster data for cross-referencing */
  clusters: ClusterItem[];
  /** Currently selected evidence URL */
  selectedEvidenceUrl: string | null;
  /** Callback when evidence is selected */
  onSelectEvidence: (item: EvidenceItem) => void;
}

/**
 * Get color for a confidence score (0-1).
 */
function getConfidenceColor(score: number): string {
  if (score >= 0.8) return "#00c982"; // High - green
  if (score >= 0.5) return "#f7b500"; // Medium - yellow/orange
  return "#ff4b4b"; // Low - red
}

/**
 * Simple bar chart component for confidence distribution.
 */
function ConfidenceDistribution({ evidence }: { evidence: EvidenceItem[] }) {
  const bins = useMemo(() => {
    const ranges = [
      { min: 0, max: 0.2, label: "0-20%" },
      { min: 0.2, max: 0.4, label: "20-40%" },
      { min: 0.4, max: 0.6, label: "40-60%" },
      { min: 0.6, max: 0.8, label: "60-80%" },
      { min: 0.8, max: 1.0, label: "80-100%" },
    ];

    return ranges.map((range) => ({
      ...range,
      count: evidence.filter((e) => {
        const conf = e.confidence ?? e.score;
        return conf >= range.min && conf < range.max;
      }).length,
    }));
  }, [evidence]);

  const maxCount = Math.max(...bins.map((b) => b.count), 1);

  return (
    <div className="evidence-chart-section">
      <h4>Confidence Distribution</h4>
      <div className="confidence-bars">
        {bins.map((bin) => (
          <div key={bin.label} className="confidence-bar">
            <div className="confidence-bar-label">{bin.label}</div>
            <div className="confidence-bar-track">
              <div
                className="confidence-bar-fill"
                style={{
                  width: `${(bin.count / maxCount) * 100}%`,
                  backgroundColor: getConfidenceColor((bin.min + bin.max) / 2),
                }}
              />
            </div>
            <div className="confidence-bar-count">{bin.count}</div>
          </div>
        ))}
      </div>
    </div>
  );
}

/**
 * Scatter plot component for evidence score vs confidence.
 */
function EvidenceScatterPlot({
  evidence,
  selectedEvidenceUrl,
  onSelectEvidence,
}: {
  evidence: EvidenceItem[];
  selectedEvidenceUrl: string | null;
  onSelectEvidence: (item: EvidenceItem) => void;
}) {
  const [hoveredUrl, setHoveredUrl] = useState<string | null>(null);

  // Chart dimensions
  const width = 400;
  const height = 200;
  const padding = { top: 10, right: 10, bottom: 30, left: 40 };
  const chartWidth = width - padding.left - padding.right;
  const chartHeight = height - padding.top - padding.bottom;

  const points = useMemo(() => {
    return evidence.map((item) => {
      const x = (item.score ?? 0) * chartWidth + padding.left;
      const confidence = item.confidence ?? item.score ?? 0;
      const y = chartHeight - confidence * chartHeight + padding.top;
      return { item, x, y, confidence };
    });
  }, [evidence, chartHeight, chartWidth, padding.left, padding.top]);

  return (
    <div className="evidence-chart-section">
      <h4>Score vs Confidence</h4>
      <div className="scatter-plot-container">
        <svg
          width={width}
          height={height}
          className="scatter-plot"
          viewBox={`0 0 ${width} ${height}`}
          role="img"
          aria-label="Scatter plot showing evidence score vs confidence"
        >
          {/* Grid lines */}
          {[0, 0.25, 0.5, 0.75, 1].map((tick) => (
            <g key={tick}>
              {/* Horizontal grid */}
              <line
                x1={padding.left}
                y1={chartHeight - tick * chartHeight + padding.top}
                x2={width - padding.right}
                y2={chartHeight - tick * chartHeight + padding.top}
                stroke="var(--stroke)"
                strokeWidth={1}
              />
              {/* Vertical grid */}
              <line
                x1={tick * chartWidth + padding.left}
                y1={padding.top}
                x2={tick * chartWidth + padding.left}
                y2={height - padding.bottom}
                stroke="var(--stroke)"
                strokeWidth={1}
              />
            </g>
          ))}

          {/* Axes */}
          <line
            x1={padding.left}
            y1={height - padding.bottom}
            x2={width - padding.right}
            y2={height - padding.bottom}
            stroke="var(--text)"
            strokeWidth={1}
          />
          <line
            x1={padding.left}
            y1={padding.top}
            x2={padding.left}
            y2={height - padding.bottom}
            stroke="var(--text)"
            strokeWidth={1}
          />

          {/* Axis labels */}
          <text
            x={width / 2}
            y={height - 5}
            textAnchor="middle"
            fill="var(--muted)"
            fontSize={10}
          >
            Score
          </text>
          <text
            x={15}
            y={height / 2}
            textAnchor="middle"
            transform={`rotate(-90, 15, ${height / 2})`}
            fill="var(--muted)"
            fontSize={10}
          >
            Confidence
          </text>

          {/* Data points */}
          {points.map(({ item, x, y, confidence }) => {
            const isSelected = selectedEvidenceUrl === item.url;
            const isHovered = hoveredUrl === item.url;
            const radius = isSelected ? 8 : isHovered ? 6 : 4;

            return (
              <g key={item.url}>
                {/* biome-ignore lint/a11y/useSemanticElements: SVG circle cannot be a button element */}
                <circle
                  cx={x}
                  cy={y}
                  r={radius}
                  fill={getConfidenceColor(confidence)}
                  stroke={isSelected ? "var(--accent)" : "none"}
                  strokeWidth={isSelected ? 2 : 0}
                  style={{ cursor: "pointer" }}
                  onMouseEnter={() => setHoveredUrl(item.url)}
                  onMouseLeave={() => setHoveredUrl(null)}
                  onClick={() => onSelectEvidence(item)}
                  role="button"
                  tabIndex={0}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" || e.key === " ") {
                      e.preventDefault();
                      onSelectEvidence(item);
                    }
                  }}
                />
              </g>
            );
          })}
        </svg>

        {/* Tooltip */}
        {hoveredUrl && (
          <div className="scatter-tooltip">
            {(() => {
              const item = evidence.find((e) => e.url === hoveredUrl);
              if (!item) return null;
              return (
                <>
                  <div className="tooltip-title">
                    {item.title || "Untitled"}
                  </div>
                  <div className="tooltip-score">
                    Score: {(item.score ?? 0).toFixed(2)}
                  </div>
                  <div className="tooltip-confidence">
                    Confidence:{" "}
                    {(item.confidence ?? item.score ?? 0).toFixed(2)}
                  </div>
                </>
              );
            })()}
          </div>
        )}
      </div>
    </div>
  );
}

/**
 * Cluster size visualization component.
 */
function ClusterSizes({ clusters }: { clusters: ClusterItem[] }) {
  const sortedClusters = useMemo(() => {
    return [...clusters].sort((a, b) => b.evidence.length - a.evidence.length);
  }, [clusters]);

  const maxSize = Math.max(...sortedClusters.map((c) => c.evidence.length), 1);

  return (
    <div className="evidence-chart-section">
      <h4>Cluster Sizes</h4>
      <div className="cluster-bars">
        {sortedClusters.map((cluster) => (
          <div key={cluster.id} className="cluster-bar">
            <div className="cluster-bar-label" title={cluster.label}>
              {cluster.label || cluster.id}
            </div>
            <div className="cluster-bar-track">
              <div
                className="cluster-bar-fill"
                style={{
                  width: `${(cluster.evidence.length / maxSize) * 100}%`,
                }}
              />
            </div>
            <div className="cluster-bar-count">{cluster.evidence.length}</div>
          </div>
        ))}
      </div>
    </div>
  );
}

/**
 * Evidence list component with selection.
 */
function EvidenceList({
  evidence,
  selectedEvidenceUrl,
  onSelectEvidence,
}: {
  evidence: EvidenceItem[];
  selectedEvidenceUrl: string | null;
  onSelectEvidence: (item: EvidenceItem) => void;
}) {
  const sortedEvidence = useMemo(() => {
    return [...evidence].sort((a, b) => (b.score ?? 0) - (a.score ?? 0));
  }, [evidence]);

  return (
    <div className="evidence-chart-section">
      <h4>Evidence Items ({evidence.length})</h4>
      <div className="evidence-list">
        {sortedEvidence.map((item) => {
          const isSelected = selectedEvidenceUrl === item.url;
          const confidence = item.confidence ?? item.score ?? 0;

          return (
            <button
              key={item.url}
              type="button"
              className={`evidence-list-item ${isSelected ? "selected" : ""}`}
              onClick={() => onSelectEvidence(item)}
            >
              <div className="evidence-list-header">
                <span
                  className="evidence-list-indicator"
                  style={{ backgroundColor: getConfidenceColor(confidence) }}
                />
                <span className="evidence-list-title">
                  {item.title || "Untitled"}
                </span>
                <span className="evidence-list-score">
                  {(item.score ?? 0).toFixed(2)}
                </span>
              </div>
              {item.snippet && (
                <div className="evidence-list-snippet">
                  {item.snippet.slice(0, 100)}
                  {item.snippet.length > 100 ? "..." : ""}
                </div>
              )}
            </button>
          );
        })}
      </div>
    </div>
  );
}

/**
 * Main EvidenceChart component.
 *
 * Provides multiple visualizations for research evidence data.
 */
export function EvidenceChart({
  evidence,
  clusters,
  selectedEvidenceUrl,
  onSelectEvidence,
}: EvidenceChartProps) {
  if (evidence.length === 0) {
    return (
      <div className="evidence-chart empty">
        <p>No evidence data available.</p>
      </div>
    );
  }

  return (
    <div className="evidence-chart">
      <div className="evidence-chart-grid">
        <ConfidenceDistribution evidence={evidence} />
        <EvidenceScatterPlot
          evidence={evidence}
          selectedEvidenceUrl={selectedEvidenceUrl}
          onSelectEvidence={onSelectEvidence}
        />
        {clusters.length > 0 && <ClusterSizes clusters={clusters} />}
        <EvidenceList
          evidence={evidence}
          selectedEvidenceUrl={selectedEvidenceUrl}
          onSelectEvidence={onSelectEvidence}
        />
      </div>
    </div>
  );
}

export default EvidenceChart;
