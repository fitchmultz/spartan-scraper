/**
 * Template Performance Component
 *
 * Displays template performance metrics including success rates,
 * field coverage, and extraction times.
 *
 * @module TemplatePerformance
 */

import { useState, useEffect } from "react";

interface TemplateMetrics {
  hour: string;
  template_name: string;
  extractions_total: number;
  extractions_success: number;
  field_coverage_avg: number;
  avg_extraction_time_ms: number;
}

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

  useEffect(() => {
    const fetchMetrics = async () => {
      try {
        setLoading(true);
        const to = new Date().toISOString();
        const from = new Date(
          Date.now() - 7 * 24 * 60 * 60 * 1000,
        ).toISOString();

        const response = await fetch(
          `/v1/template-metrics?template=${encodeURIComponent(templateName)}&from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`,
        );

        if (!response.ok) {
          throw new Error("Failed to fetch metrics");
        }

        const data = await response.json();
        setMetrics(data.metrics || []);
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
        No metrics available for template "{templateName}"
      </div>
    );
  }

  // Aggregate metrics
  const totalExtractions = metrics.reduce(
    (sum, m) => sum + m.extractions_total,
    0,
  );
  const totalSuccess = metrics.reduce(
    (sum, m) => sum + m.extractions_success,
    0,
  );
  const avgSuccessRate =
    totalExtractions > 0 ? (totalSuccess / totalExtractions) * 100 : 0;
  const avgFieldCoverage =
    metrics.reduce((sum, m) => sum + m.field_coverage_avg, 0) / metrics.length;
  const avgExtractionTime =
    metrics.reduce((sum, m) => sum + m.avg_extraction_time_ms, 0) /
    metrics.length;

  const getSuccessRateColor = () => {
    if (avgSuccessRate >= 90) return "success";
    if (avgSuccessRate >= 70) return "warning";
    return "error";
  };

  return (
    <div className="template-performance">
      <h3 className="template-performance__title">{templateName}</h3>
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

interface ComparisonData {
  template_a: string;
  template_b: string;
  template_a_metrics: {
    sample_size: number;
    success_rate: number;
    field_coverage: number;
    avg_extraction_time_ms: number;
  };
  template_b_metrics: {
    sample_size: number;
    success_rate: number;
    field_coverage: number;
    avg_extraction_time_ms: number;
  };
  statistical_test: {
    test_type: string;
    p_value: number;
    is_significant: boolean;
    confidence_interval: [number, number];
    effect_size: number;
  };
  winner: string | null;
  recommendation: string;
}

export function TemplateComparisonView({
  templateA,
  templateB,
}: TemplateComparisonViewProps) {
  const [comparison, setComparison] = useState<ComparisonData | null>(null);
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

        const response = await fetch(
          `/v1/template-comparison?template_a=${encodeURIComponent(templateA)}&template_b=${encodeURIComponent(templateB)}&from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`,
        );

        if (!response.ok) {
          throw new Error("Failed to fetch comparison");
        }

        const data = await response.json();
        setComparison(data);
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
        No comparison data available
      </div>
    );
  }

  const getSignificanceBadge = () => {
    if (comparison.statistical_test.is_significant) {
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
    const winnerName =
      comparison.winner === "template_a" ? templateA : templateB;
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
              {comparison.template_a_metrics.success_rate.toFixed(1)}%
            </span>
          </div>
          <div className="comparison-metric">
            <span className="metric-label">Field Coverage:</span>
            <span className="metric-value">
              {(comparison.template_a_metrics.field_coverage * 100).toFixed(1)}%
            </span>
          </div>
          <div className="comparison-metric">
            <span className="metric-label">Avg Time:</span>
            <span className="metric-value">
              {comparison.template_a_metrics.avg_extraction_time_ms}ms
            </span>
          </div>
          <div className="comparison-metric">
            <span className="metric-label">Samples:</span>
            <span className="metric-value">
              {comparison.template_a_metrics.sample_size.toLocaleString()}
            </span>
          </div>
        </div>

        <div className="comparison-column">
          <h4>{templateB}</h4>
          <div className="comparison-metric">
            <span className="metric-label">Success Rate:</span>
            <span className="metric-value">
              {comparison.template_b_metrics.success_rate.toFixed(1)}%
            </span>
          </div>
          <div className="comparison-metric">
            <span className="metric-label">Field Coverage:</span>
            <span className="metric-value">
              {(comparison.template_b_metrics.field_coverage * 100).toFixed(1)}%
            </span>
          </div>
          <div className="comparison-metric">
            <span className="metric-label">Avg Time:</span>
            <span className="metric-value">
              {comparison.template_b_metrics.avg_extraction_time_ms}ms
            </span>
          </div>
          <div className="comparison-metric">
            <span className="metric-label">Samples:</span>
            <span className="metric-value">
              {comparison.template_b_metrics.sample_size.toLocaleString()}
            </span>
          </div>
        </div>
      </div>

      <div className="template-comparison__stats">
        <h4>Statistical Analysis</h4>
        <div className="stat-row">
          <span className="stat-label">Test Type:</span>
          <span className="stat-value">
            {comparison.statistical_test.test_type}
          </span>
        </div>
        <div className="stat-row">
          <span className="stat-label">P-Value:</span>
          <span className="stat-value">
            {comparison.statistical_test.p_value.toFixed(4)}
          </span>
        </div>
        <div className="stat-row">
          <span className="stat-label">Confidence Interval:</span>
          <span className="stat-value">
            [{comparison.statistical_test.confidence_interval[0].toFixed(3)},{" "}
            {comparison.statistical_test.confidence_interval[1].toFixed(3)}]
          </span>
        </div>
        <div className="stat-row">
          <span className="stat-label">Effect Size:</span>
          <span className="stat-value">
            {comparison.statistical_test.effect_size.toFixed(3)}
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
