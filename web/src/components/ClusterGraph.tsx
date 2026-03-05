/**
 * Cluster Graph Component
 *
 * Visualizes the relationship between evidence clusters and their evidence items
 * using a radial layout. Shows cluster confidence through color coding and
 * allows interactive exploration of cluster-evidence relationships.
 *
 * @module ClusterGraph
 */
import { useMemo, useState } from "react";
import type { EvidenceItem, ClusterItem } from "../types";

interface ClusterGraphProps {
  /** Cluster data */
  clusters: ClusterItem[];
  /** Evidence items */
  evidence: EvidenceItem[];
  /** Currently selected cluster ID */
  selectedClusterId: string | null;
  /** Callback when a cluster is selected */
  onSelectCluster: (cluster: ClusterItem) => void;
}

/**
 * Get color for confidence score.
 */
function getConfidenceColor(confidence: number): string {
  if (confidence >= 0.8) return "#00c982";
  if (confidence >= 0.5) return "#f7b500";
  return "#ff4b4b";
}

/**
 * Calculate node positions in a radial layout.
 */
function calculateRadialLayout(
  clusters: ClusterItem[],
  evidence: EvidenceItem[],
  width: number,
  height: number,
) {
  const centerX = width / 2;
  const centerY = height / 2;
  const clusterRadius = Math.min(width, height) * 0.25;
  const evidenceRadius = Math.min(width, height) * 0.4;

  // Position clusters in inner circle
  const clusterNodes = clusters.map((cluster, index) => {
    const angle = (index / clusters.length) * 2 * Math.PI - Math.PI / 2;
    return {
      ...cluster,
      x: centerX + Math.cos(angle) * clusterRadius,
      y: centerY + Math.sin(angle) * clusterRadius,
      type: "cluster" as const,
      radius: 20 + cluster.evidence.length * 2, // Size based on evidence count
    };
  });

  // Position evidence in outer circle, grouped by cluster
  const evidenceNodes: Array<{
    item: EvidenceItem;
    x: number;
    y: number;
    type: "evidence";
    radius: number;
    clusterId?: string;
  }> = [];

  const evidenceByCluster = new Map<string, EvidenceItem[]>();
  const unclusteredEvidence: EvidenceItem[] = [];

  // Group evidence by cluster
  for (const item of evidence) {
    if (item.clusterId) {
      const clusterItems = evidenceByCluster.get(item.clusterId) || [];
      clusterItems.push(item);
      evidenceByCluster.set(item.clusterId, clusterItems);
    } else {
      unclusteredEvidence.push(item);
    }
  }

  // Position clustered evidence
  for (const [clusterId, items] of evidenceByCluster) {
    const clusterNode = clusterNodes.find((c) => c.id === clusterId);
    if (!clusterNode) continue;

    const clusterAngle = Math.atan2(
      clusterNode.y - centerY,
      clusterNode.x - centerX,
    );
    const arcSize = Math.PI / 3; // 60 degree arc per cluster
    const startAngle = clusterAngle - arcSize / 2;

    for (let i = 0; i < items.length; i++) {
      const item = items[i];
      const angle = startAngle + (i / Math.max(items.length - 1, 1)) * arcSize;
      evidenceNodes.push({
        item,
        x: centerX + Math.cos(angle) * evidenceRadius,
        y: centerY + Math.sin(angle) * evidenceRadius,
        type: "evidence" as const,
        radius: 6,
        clusterId,
      });
    }
  }

  // Position unclustered evidence evenly around the circle
  for (let i = 0; i < unclusteredEvidence.length; i++) {
    const item = unclusteredEvidence[i];
    const angle =
      (i / Math.max(unclusteredEvidence.length, 1)) * 2 * Math.PI - Math.PI / 2;
    evidenceNodes.push({
      item,
      x: centerX + Math.cos(angle) * evidenceRadius,
      y: centerY + Math.sin(angle) * evidenceRadius,
      type: "evidence" as const,
      radius: 6,
    });
  }

  return { clusterNodes, evidenceNodes, centerX, centerY };
}

/**
 * Main ClusterGraph component.
 *
 * Displays clusters and evidence in a radial force-directed layout.
 */
export function ClusterGraph({
  clusters,
  evidence,
  selectedClusterId,
  onSelectCluster,
}: ClusterGraphProps) {
  const [hoveredId, setHoveredId] = useState<string | null>(null);
  const [dimensions] = useState({ width: 600, height: 400 });

  const { clusterNodes, evidenceNodes, centerX, centerY } = useMemo(
    () =>
      calculateRadialLayout(
        clusters,
        evidence,
        dimensions.width,
        dimensions.height,
      ),
    [clusters, evidence, dimensions],
  );

  // Create edges between clusters and their evidence
  const edges = useMemo(() => {
    const edgeList: Array<{
      from: { x: number; y: number };
      to: { x: number; y: number };
      clusterId: string;
    }> = [];

    for (const evidenceNode of evidenceNodes) {
      if (evidenceNode.clusterId) {
        const clusterNode = clusterNodes.find(
          (c) => c.id === evidenceNode.clusterId,
        );
        if (clusterNode) {
          edgeList.push({
            from: clusterNode,
            to: evidenceNode,
            clusterId: evidenceNode.clusterId,
          });
        }
      }
    }

    return edgeList;
  }, [clusterNodes, evidenceNodes]);

  if (clusters.length === 0) {
    return (
      <div className="cluster-graph empty">
        <p>No cluster data available.</p>
      </div>
    );
  }

  return (
    <div className="cluster-graph">
      <h4>Cluster Relationships</h4>
      <div className="cluster-graph-container">
        <svg
          width={dimensions.width}
          height={dimensions.height}
          viewBox={`0 0 ${dimensions.width} ${dimensions.height}`}
          className="cluster-graph-svg"
        >
          <title>
            Cluster relationship graph showing connections between clusters and
            evidence items
          </title>
          {/* Edges */}
          {edges.map((edge) => {
            const isHighlighted =
              hoveredId === edge.clusterId ||
              selectedClusterId === edge.clusterId;
            const isDimmed =
              hoveredId && hoveredId !== edge.clusterId && hoveredId !== ""; // Evidence doesn't have IDs in the same way
            // Create unique key from clusterId and evidence URL
            const edgeKey = `edge-${edge.clusterId}-${edge.to.x}-${edge.to.y}`;

            return (
              <line
                key={edgeKey}
                x1={edge.from.x}
                y1={edge.from.y}
                x2={edge.to.x}
                y2={edge.to.y}
                stroke={isHighlighted ? "var(--accent)" : "var(--stroke)"}
                strokeWidth={isHighlighted ? 2 : 1}
                opacity={isDimmed ? 0.2 : isHighlighted ? 1 : 0.4}
              />
            );
          })}

          {/* Evidence nodes (drawn first so they appear behind clusters) */}
          {evidenceNodes.map((node, index) => {
            const isHighlighted =
              hoveredId === node.item.url ||
              (node.clusterId &&
                (hoveredId === node.clusterId ||
                  selectedClusterId === node.clusterId));
            const isDimmed =
              hoveredId &&
              hoveredId !== node.item.url &&
              hoveredId !== node.clusterId;

            const uniqueKey = `evidence-${node.item.url}-${index}`;
            return (
              <g key={uniqueKey}>
                {/* biome-ignore lint/a11y/useSemanticElements: SVG circle cannot be a button element */}
                <circle
                  cx={node.x}
                  cy={node.y}
                  r={node.radius}
                  fill={getConfidenceColor(
                    node.item.confidence ?? node.item.score ?? 0,
                  )}
                  stroke={isHighlighted ? "var(--accent)" : "none"}
                  strokeWidth={isHighlighted ? 2 : 0}
                  opacity={isDimmed ? 0.3 : 1}
                  style={{ cursor: "pointer" }}
                  onMouseEnter={() => setHoveredId(node.item.url)}
                  onMouseLeave={() => setHoveredId(null)}
                  tabIndex={0}
                  role="button"
                  aria-label={`Evidence: ${node.item.title || "Untitled"}`}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" || e.key === " ") {
                      e.preventDefault();
                      setHoveredId(node.item.url);
                    }
                  }}
                />
              </g>
            );
          })}

          {/* Cluster nodes */}
          {clusterNodes.map((node) => {
            const isSelected = selectedClusterId === node.id;
            const isHighlighted = hoveredId === node.id || isSelected;
            const isDimmed = hoveredId && hoveredId !== node.id;

            return (
              <g key={`cluster-${node.id}`}>
                {/* Glow effect for selected/hovered */}
                {(isHighlighted || isSelected) && (
                  <circle
                    cx={node.x}
                    cy={node.y}
                    r={node.radius + 4}
                    fill="none"
                    stroke="var(--accent)"
                    strokeWidth={2}
                    opacity={0.5}
                  />
                )}
                {/* biome-ignore lint/a11y/useSemanticElements: SVG circle cannot be a button element */}
                <circle
                  cx={node.x}
                  cy={node.y}
                  r={node.radius}
                  fill={getConfidenceColor(node.confidence)}
                  stroke={isSelected ? "var(--accent-strong)" : "var(--stroke)"}
                  strokeWidth={isSelected ? 3 : 1}
                  opacity={isDimmed ? 0.4 : 0.9}
                  style={{ cursor: "pointer" }}
                  onMouseEnter={() => setHoveredId(node.id)}
                  onMouseLeave={() => setHoveredId(null)}
                  onClick={() => onSelectCluster(node)}
                  tabIndex={0}
                  role="button"
                  aria-label={`Cluster: ${node.label || node.id}`}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" || e.key === " ") {
                      e.preventDefault();
                      onSelectCluster(node);
                    }
                  }}
                />
                {/* Cluster label */}
                <text
                  x={node.x}
                  y={node.y + 4}
                  textAnchor="middle"
                  fill="#1a1200"
                  fontSize={10}
                  fontWeight="600"
                  style={{ pointerEvents: "none" }}
                >
                  {node.label.slice(0, 10)}
                  {node.label.length > 10 ? "..." : ""}
                </text>
                {/* Confidence label */}
                <text
                  x={node.x}
                  y={node.y + node.radius + 14}
                  textAnchor="middle"
                  fill="var(--muted)"
                  fontSize={9}
                >
                  {(node.confidence * 100).toFixed(0)}%
                </text>
              </g>
            );
          })}

          {/* Center label */}
          <text
            x={centerX}
            y={centerY}
            textAnchor="middle"
            fill="var(--muted)"
            fontSize={10}
          >
            {clusters.length} clusters
          </text>
          <text
            x={centerX}
            y={centerY + 12}
            textAnchor="middle"
            fill="var(--muted)"
            fontSize={10}
          >
            {evidence.length} evidence
          </text>
        </svg>

        {/* Legend */}
        <div className="cluster-graph-legend">
          <div className="legend-section">
            <div className="legend-title">Confidence</div>
            <div className="legend-item">
              <span
                className="legend-color"
                style={{ backgroundColor: "#00c982" }}
              />
              <span className="legend-label">High (80%+)</span>
            </div>
            <div className="legend-item">
              <span
                className="legend-color"
                style={{ backgroundColor: "#f7b500" }}
              />
              <span className="legend-label">Medium (50-80%)</span>
            </div>
            <div className="legend-item">
              <span
                className="legend-color"
                style={{ backgroundColor: "#ff4b4b" }}
              />
              <span className="legend-label">Low (&lt;50%)</span>
            </div>
          </div>
          <div className="legend-section">
            <div className="legend-title">Nodes</div>
            <div className="legend-item">
              <span className="legend-shape cluster" />
              <span className="legend-label">Cluster</span>
            </div>
            <div className="legend-item">
              <span className="legend-shape evidence" />
              <span className="legend-label">Evidence</span>
            </div>
          </div>
        </div>

        {/* Tooltip */}
        {hoveredId && (
          <div className="cluster-graph-tooltip">
            {(() => {
              const cluster = clusters.find((c) => c.id === hoveredId);
              if (cluster) {
                return (
                  <>
                    <div className="tooltip-title">
                      {cluster.label || cluster.id}
                    </div>
                    <div className="tooltip-confidence">
                      Confidence: {(cluster.confidence * 100).toFixed(1)}%
                    </div>
                    <div className="tooltip-count">
                      {cluster.evidence.length} evidence items
                    </div>
                  </>
                );
              }
              const evidenceItem = evidence.find((e) => e.url === hoveredId);
              if (evidenceItem) {
                return (
                  <>
                    <div className="tooltip-title">
                      {evidenceItem.title || "Untitled"}
                    </div>
                    <div className="tooltip-score">
                      Score: {(evidenceItem.score ?? 0).toFixed(2)}
                    </div>
                    {evidenceItem.clusterId && (
                      <div className="tooltip-cluster">
                        Cluster: {evidenceItem.clusterId}
                      </div>
                    )}
                  </>
                );
              }
              return null;
            })()}
          </div>
        )}
      </div>
    </div>
  );
}

export default ClusterGraph;
