/**
 * Template A/B test results panel.
 *
 * Purpose:
 * - Fetch and render comparison results for a single template A/B test.
 *
 * Responsibilities:
 * - Load comparison results for the selected test.
 * - Render metrics, significance, and winner auto-selection actions.
 *
 * Scope:
 * - Result display for a single template A/B test only.
 *
 * Usage:
 * - Render inside an A/B test card when results are expanded.
 *
 * Invariants/Assumptions:
 * - Successful auto-selection should refresh the parent list without a full
 *   page reload.
 */

import { useEffect, useState } from "react";

import {
  getV1TemplateAbTestsByIdResults,
  postV1TemplateAbTestsByIdAutoSelect,
} from "../../../api";
import type { TemplateComparison } from "../../../api";
import { getApiBaseUrl } from "../../../lib/api-config";

interface TestResultsProps {
  testId: string;
  onUpdate: () => void;
}

export function TestResults({ testId, onUpdate }: TestResultsProps) {
  const [results, setResults] = useState<TemplateComparison | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [autoSelecting, setAutoSelecting] = useState(false);

  useEffect(() => {
    const fetchResults = async () => {
      try {
        setLoading(true);
        const { data, error } = await getV1TemplateAbTestsByIdResults({
          baseUrl: getApiBaseUrl(),
          path: { id: testId },
        });

        if (error) {
          throw new Error(String(error));
        }

        setResults(data || null);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Unknown error");
      } finally {
        setLoading(false);
      }
    };

    fetchResults();
  }, [testId]);

  if (loading) {
    return <div className="test-results loading">Loading results...</div>;
  }

  if (error) {
    return <div className="test-results error">Error: {error}</div>;
  }

  if (!results) {
    return <div className="test-results empty">No results available</div>;
  }

  const handleAutoSelect = async () => {
    if (
      !confirm(
        "This will select the winning template and complete the test. Continue?",
      )
    ) {
      return;
    }

    try {
      setAutoSelecting(true);
      const { error } = await postV1TemplateAbTestsByIdAutoSelect({
        baseUrl: getApiBaseUrl(),
        path: { id: testId },
      });

      if (error) {
        throw new Error(String(error));
      }

      onUpdate();
    } catch (err) {
      console.error("Failed to auto-select winner:", err);
    } finally {
      setAutoSelecting(false);
    }
  };

  return (
    <div className="test-results">
      <h5>Test Results</h5>

      <div className="test-results__comparison">
        <div className="comparison-column">
          <h6>{results.baseline_template}</h6>
          <div className="metric">
            <span className="metric-label">Success Rate:</span>
            <span className="metric-value">
              {results.baseline_metrics?.success_rate?.toFixed(1) ?? "N/A"}%
            </span>
          </div>
          <div className="metric">
            <span className="metric-label">Field Coverage:</span>
            <span className="metric-value">
              {((results.baseline_metrics?.field_coverage ?? 0) * 100).toFixed(
                1,
              )}
              %
            </span>
          </div>
          <div className="metric">
            <span className="metric-label">Samples:</span>
            <span className="metric-value">
              {results.baseline_metrics?.sample_size?.toLocaleString() ?? "N/A"}
            </span>
          </div>
        </div>

        <div className="comparison-column">
          <h6>{results.variant_template}</h6>
          <div className="metric">
            <span className="metric-label">Success Rate:</span>
            <span className="metric-value">
              {results.variant_metrics?.success_rate?.toFixed(1) ?? "N/A"}%
            </span>
          </div>
          <div className="metric">
            <span className="metric-label">Field Coverage:</span>
            <span className="metric-value">
              {((results.variant_metrics?.field_coverage ?? 0) * 100).toFixed(
                1,
              )}
              %
            </span>
          </div>
          <div className="metric">
            <span className="metric-label">Samples:</span>
            <span className="metric-value">
              {results.variant_metrics?.sample_size?.toLocaleString() ?? "N/A"}
            </span>
          </div>
        </div>
      </div>

      <div className="test-results__stats">
        <div className="stat-row">
          <span className="stat-label">P-Value:</span>
          <span className="stat-value">
            {results.statistical_test?.p_value?.toFixed(4) ?? "N/A"}
          </span>
        </div>
        <div className="stat-row">
          <span className="stat-label">Significant:</span>
          <span
            className={`stat-value ${
              results.statistical_test?.is_significant ? "success" : "neutral"
            }`}
          >
            {results.statistical_test?.is_significant ? "Yes" : "No"}
          </span>
        </div>
      </div>

      <div className="test-results__recommendation">
        <strong>Recommendation:</strong>{" "}
        {results.recommendation ?? "No recommendation available"}
      </div>

      {!results.winner && results.statistical_test?.is_significant && (
        <button
          type="button"
          className="btn btn--primary btn--small"
          onClick={handleAutoSelect}
          disabled={autoSelecting}
        >
          {autoSelecting ? "Selecting..." : "Auto-Select Winner"}
        </button>
      )}
    </div>
  );
}
