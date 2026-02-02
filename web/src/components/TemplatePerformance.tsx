/**
 * Template Performance Component
 *
 * Displays template performance metrics including success rates,
 * field coverage, and extraction times.
 *
 * @module TemplatePerformance
 */

import { useState, useEffect } from "react";
import { VisualSelectorBuilder } from "./VisualSelectorBuilder";
import { getV1TemplateMetrics, getV1TemplateComparison } from "../api";
import type { TemplateMetrics, TemplateComparison } from "../api";
import { getApiBaseUrl } from "../lib/api-config";

interface TemplatePerformanceProps {
  templateName: string;
}

interface MetricCardProps {
  label: string;
  value: string | number;
  unit?: string;
  color?: "default" | "success" | "warning" | "error";
}

function MetricCard({
  label,
  value,
  unit,
  color = "default",
}: MetricCardProps) {
  const getColorClass = () => {
    switch (color) {
      case "success":
        return "metric-card--success";
      case "warning":
        return "metric-card--warning";
      case "error":
        return "metric-card--error";
      default:
        return "";
    }
  };

  return (
    <div className={`metric-card ${getColorClass()}`}>
      <div className="metric-card__label">{label}</div>
      <div className="metric-card__value">
        {value}
        {unit && <span className="metric-card__unit">{unit}</span>}
      </div>
    </div>
  );
}

export function TemplatePerformance({
  templateName,
}: TemplatePerformanceProps) {
  const [metrics, setMetrics] = useState<TemplateMetrics[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isEditing, setIsEditing] = useState(false);

  useEffect(() => {
    const fetchMetrics = async () => {
      try {
        setLoading(true);
        const to = new Date().toISOString();
        const from = new Date(
          Date.now() - 7 * 24 * 60 * 60 * 1000,
        ).toISOString();

        const { data, error: apiError } = await getV1TemplateMetrics({
          baseUrl: getApiBaseUrl(),
          query: {
            template: templateName,
            from,
            to,
          },
        });

        if (apiError) {
          throw new Error(String(apiError));
        }

        setMetrics(data?.metrics || []);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Unknown error");
      } finally {
        setLoading(false);
      }
    };

    fetchMetrics();
  }, [templateName]);

  if (loading) {
    return (
      <div className="template-performance loading">Loading metrics...</div>
    );
  }

  if (error) {
    return <div className="template-performance error">Error: {error}</div>;
  }

  if (metrics.length === 0) {
    return (
      <div className="template-performance empty">
        <p>No metrics available for template &quot;{templateName}&quot;</p>
        <p className="empty-hint">
          Run jobs with this template to collect performance data.
        </p>
      </div>
    );
  }

  // Aggregate metrics
  const totalExtractions = metrics.reduce(
    (sum, m) => sum + (m.extractions_total || 0),
    0,
  );
  const totalSuccess = metrics.reduce(
    (sum, m) => sum + (m.extractions_success || 0),
    0,
  );
  const avgSuccessRate =
    totalExtractions > 0 ? (totalSuccess / totalExtractions) * 100 : 0;
  const avgFieldCoverage =
    metrics.reduce((sum, m) => sum + (m.field_coverage_avg || 0), 0) /
    metrics.length;
  const avgExtractionTime =
    metrics.reduce((sum, m) => sum + (m.avg_extraction_time_ms || 0), 0) /
    metrics.length;

  const getSuccessRateColor = () => {
    if (avgSuccessRate >= 90) return "success";
    if (avgSuccessRate >= 70) return "warning";
    return "error";
  };

  return (
    <div className="template-performance">
      <div className="template-performance__header">
        <h3 className="template-performance__title">{templateName}</h3>
        <button
          type="button"
          className="btn btn--small btn--primary"
          onClick={() => setIsEditing(true)}
        >
          Edit Template
        </button>
      </div>

      {isEditing && (
        // biome-ignore lint/a11y/noStaticElementInteractions: modal overlay pattern
        // biome-ignore lint/a11y/useKeyWithClickEvents: handled by escape key in component
        <div className="modal-overlay" onClick={() => setIsEditing(false)}>
          {/* biome-ignore lint/a11y/useKeyWithClickEvents: handled by child component */}
          {/* biome-ignore lint/a11y/noStaticElementInteractions: modal content container */}
          <div className="modal-content" onClick={(e) => e.stopPropagation()}>
            <VisualSelectorBuilder
              onSave={() => {
                setIsEditing(false);
                window.location.reload();
              }}
              onCancel={() => setIsEditing(false)}
            />
          </div>
        </div>
      )}
      <div className="template-performance__metrics">
        <MetricCard
          label="Success Rate"
          value={avgSuccessRate.toFixed(1)}
          unit="%"
          color={getSuccessRateColor()}
        />
        <MetricCard
          label="Field Coverage"
          value={(avgFieldCoverage * 100).toFixed(1)}
          unit="%"
          color={avgFieldCoverage >= 0.8 ? "success" : "warning"}
        />
        <MetricCard
          label="Avg Extraction Time"
          value={avgExtractionTime.toFixed(0)}
          unit="ms"
        />
        <MetricCard
          label="Total Extractions"
          value={totalExtractions.toLocaleString()}
        />
      </div>
      <div className="template-performance__chart">
        {/* Sparkline chart could be added here */}
        <div className="sparkline-placeholder">
          {metrics.length} hours of data
        </div>
      </div>
    </div>
  );
}

interface TemplateComparisonViewProps {
  templateA: string;
  templateB: string;
}

export function TemplateComparisonView({
  templateA,
  templateB,
}: TemplateComparisonViewProps) {
  const [comparison, setComparison] = useState<TemplateComparison | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchComparison = async () => {
      try {
        setLoading(true);
        const to = new Date().toISOString();
        const from = new Date(
          Date.now() - 7 * 24 * 60 * 60 * 1000,
        ).toISOString();

        const { data, error: apiError } = await getV1TemplateComparison({
          baseUrl: getApiBaseUrl(),
          query: {
            template_a: templateA,
            template_b: templateB,
            from,
            to,
          },
        });

        if (apiError) {
          throw new Error(String(apiError));
        }

        setComparison(data || null);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Unknown error");
      } finally {
        setLoading(false);
      }
    };

    fetchComparison();
  }, [templateA, templateB]);

  if (loading) {
    return (
      <div className="template-comparison loading">Loading comparison...</div>
    );
  }

  if (error) {
    return <div className="template-comparison error">Error: {error}</div>;
  }

  if (!comparison) {
    return (
      <div className="template-comparison empty">
        <p>No comparison data available</p>
        <p className="empty-hint">
          Run jobs with both templates to generate comparison data.
        </p>
      </div>
    );
  }

  const getSignificanceBadge = () => {
    if (comparison.statistical_test?.is_significant) {
      return (
        <span className="badge badge--success">Statistically Significant</span>
      );
    }
    return <span className="badge badge--warning">Not Significant</span>;
  };

  const getWinnerBadge = () => {
    if (!comparison.winner) {
      return <span className="badge badge--neutral">No Winner</span>;
    }
    const winnerName = comparison.winner === "baseline" ? templateA : templateB;
    return <span className="badge badge--success">Winner: {winnerName}</span>;
  };

  return (
    <div className="template-comparison">
      <div className="template-comparison__header">
        <h3>Template Comparison</h3>
        <div className="template-comparison__badges">
          {getSignificanceBadge()}
          {getWinnerBadge()}
        </div>
      </div>

      <div className="template-comparison__grid">
        <div className="comparison-column">
          <h4>{templateA}</h4>
          <div className="comparison-metric">
            <span className="metric-label">Success Rate:</span>
            <span className="metric-value">
              {comparison.baseline_metrics?.success_rate?.toFixed(1) ?? "N/A"}%
            </span>
          </div>
          <div className="comparison-metric">
            <span className="metric-label">Field Coverage:</span>
            <span className="metric-value">
              {(
                (comparison.baseline_metrics?.field_coverage ?? 0) * 100
              ).toFixed(1)}
              %
            </span>
          </div>
          <div className="comparison-metric">
            <span className="metric-label">Avg Time:</span>
            <span className="metric-value">
              {comparison.baseline_metrics?.avg_extraction_time_ms ?? "N/A"}ms
            </span>
          </div>
          <div className="comparison-metric">
            <span className="metric-label">Samples:</span>
            <span className="metric-value">
              {comparison.baseline_metrics?.sample_size?.toLocaleString() ??
                "N/A"}
            </span>
          </div>
        </div>

        <div className="comparison-column">
          <h4>{templateB}</h4>
          <div className="comparison-metric">
            <span className="metric-label">Success Rate:</span>
            <span className="metric-value">
              {comparison.variant_metrics?.success_rate?.toFixed(1) ?? "N/A"}%
            </span>
          </div>
          <div className="comparison-metric">
            <span className="metric-label">Field Coverage:</span>
            <span className="metric-value">
              {(
                (comparison.variant_metrics?.field_coverage ?? 0) * 100
              ).toFixed(1)}
              %
            </span>
          </div>
          <div className="comparison-metric">
            <span className="metric-label">Avg Time:</span>
            <span className="metric-value">
              {comparison.variant_metrics?.avg_extraction_time_ms ?? "N/A"}ms
            </span>
          </div>
          <div className="comparison-metric">
            <span className="metric-label">Samples:</span>
            <span className="metric-value">
              {comparison.variant_metrics?.sample_size?.toLocaleString() ??
                "N/A"}
            </span>
          </div>
        </div>
      </div>

      <div className="template-comparison__stats">
        <h4>Statistical Analysis</h4>
        <div className="stat-row">
          <span className="stat-label">Test Type:</span>
          <span className="stat-value">
            {comparison.statistical_test?.test_type ?? "N/A"}
          </span>
        </div>
        <div className="stat-row">
          <span className="stat-label">P-Value:</span>
          <span className="stat-value">
            {comparison.statistical_test?.p_value?.toFixed(4) ?? "N/A"}
          </span>
        </div>
        <div className="stat-row">
          <span className="stat-label">Confidence Interval:</span>
          <span className="stat-value">
            [
            {comparison.statistical_test?.confidence_interval?.[0]?.toFixed(
              3,
            ) ?? "N/A"}
            ,{" "}
            {comparison.statistical_test?.confidence_interval?.[1]?.toFixed(
              3,
            ) ?? "N/A"}
            ]
          </span>
        </div>
        <div className="stat-row">
          <span className="stat-label">Effect Size:</span>
          <span className="stat-value">
            {comparison.statistical_test?.effect_size?.toFixed(3) ?? "N/A"}
          </span>
        </div>
      </div>

      <div className="template-comparison__recommendation">
        <h4>Recommendation</h4>
        <p>{comparison.recommendation}</p>
      </div>
    </div>
  );
}
