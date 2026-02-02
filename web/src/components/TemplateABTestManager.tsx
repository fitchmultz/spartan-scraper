/**
 * Template A/B Test Manager Component
 *
 * Manages A/B tests for template comparison including creation,
 * monitoring, and winner selection.
 *
 * @module TemplateABTestManager
 */

import { useState, useEffect, useCallback } from "react";
import { VisualSelectorBuilder } from "./VisualSelectorBuilder";
import {
  getV1TemplateAbTests,
  postV1TemplateAbTests,
  deleteV1TemplateAbTestsById,
  postV1TemplateAbTestsByIdStart,
  postV1TemplateAbTestsByIdStop,
  getV1TemplateAbTestsByIdResults,
  postV1TemplateAbTestsByIdAutoSelect,
  listTemplates,
} from "../api";
import type { TemplateAbTest, TemplateComparison } from "../api";
import { getApiBaseUrl } from "../lib/api-config";

export function TemplateABTestManager() {
  const [tests, setTests] = useState<TemplateAbTest[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [editingTemplate, setEditingTemplate] = useState<string | null>(null);
  const [templates, setTemplates] = useState<string[]>([]);

  // Fetch available templates
  const fetchTemplates = useCallback(async () => {
    try {
      const { data, error } = await listTemplates({
        baseUrl: getApiBaseUrl(),
      });
      if (!error && data) {
        setTemplates(data.templates || []);
      }
    } catch {
      // Silently fail - templates are not critical
    }
  }, []);

  useEffect(() => {
    fetchTemplates();
  }, [fetchTemplates]);

  const fetchTests = useCallback(async () => {
    try {
      setLoading(true);
      const { data, error } = await getV1TemplateAbTests({
        baseUrl: getApiBaseUrl(),
      });

      if (error) {
        throw new Error(String(error));
      }

      setTests(data?.tests || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchTests();
  }, [fetchTests]);

  if (loading) {
    return <div className="ab-test-manager loading">Loading A/B tests...</div>;
  }

  if (error) {
    return <div className="ab-test-manager error">Error: {error}</div>;
  }

  return (
    <div className="ab-test-manager">
      <div className="ab-test-manager__header">
        <h2>A/B Tests</h2>
        <button
          type="button"
          className="btn btn--primary"
          onClick={() => setShowCreateForm(!showCreateForm)}
        >
          {showCreateForm ? "Cancel" : "Create Test"}
        </button>
      </div>

      {showCreateForm && (
        <CreateTestForm
          templates={templates}
          onSuccess={() => {
            setShowCreateForm(false);
            fetchTests();
          }}
          onCancel={() => setShowCreateForm(false)}
        />
      )}

      {editingTemplate && (
        // biome-ignore lint/a11y/noStaticElementInteractions: modal overlay pattern
        // biome-ignore lint/a11y/useKeyWithClickEvents: handled by escape key in component
        <div className="modal-overlay" onClick={() => setEditingTemplate(null)}>
          {/* biome-ignore lint/a11y/useKeyWithClickEvents: handled by child component */}
          {/* biome-ignore lint/a11y/noStaticElementInteractions: modal content container */}
          <div className="modal-content" onClick={(e) => e.stopPropagation()}>
            <VisualSelectorBuilder
              onSave={() => {
                setEditingTemplate(null);
                fetchTemplates();
              }}
              onCancel={() => setEditingTemplate(null)}
            />
          </div>
        </div>
      )}

      <div className="ab-test-manager__list">
        {tests.length === 0 ? (
          <div className="ab-test-manager__empty">
            <p>No A/B tests yet.</p>
            <p className="empty-hint">
              Create one to compare template performance and find the best
              extraction strategy.
            </p>
            <button
              type="button"
              className="btn btn--primary"
              onClick={() => setShowCreateForm(true)}
            >
              Create Your First Test
            </button>
          </div>
        ) : (
          tests.map((test) => (
            <ABTestCard
              key={test.id}
              test={test}
              onUpdate={fetchTests}
              onEditTemplate={setEditingTemplate}
            />
          ))
        )}
      </div>
    </div>
  );
}

interface CreateTestFormProps {
  templates: string[];
  onSuccess: () => void;
  onCancel: () => void;
}

function CreateTestForm({
  templates,
  onSuccess,
  onCancel,
}: CreateTestFormProps) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [baselineTemplate, setBaselineTemplate] = useState("");
  const [variantTemplate, setVariantTemplate] = useState("");
  const [baselineAllocation, setBaselineAllocation] = useState(50);
  const [minSampleSize, setMinSampleSize] = useState(100);
  const [confidenceLevel, setConfidenceLevel] = useState(0.95);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!name || !baselineTemplate || !variantTemplate) {
      setError("Name, baseline template, and variant template are required");
      return;
    }

    if (baselineTemplate === variantTemplate) {
      setError("Baseline and variant templates must be different");
      return;
    }

    try {
      setSubmitting(true);
      setError(null);

      const { error } = await postV1TemplateAbTests({
        baseUrl: getApiBaseUrl(),
        body: {
          name,
          description,
          baseline_template: baselineTemplate,
          variant_template: variantTemplate,
          allocation: {
            baseline: baselineAllocation,
            variant: 100 - baselineAllocation,
          },
          success_criteria: {
            metric: "success_rate",
            min_improvement: 0.05,
            required_fields: [],
            min_field_coverage: 0.8,
          },
          min_sample_size: minSampleSize,
          confidence_level: confidenceLevel,
        },
      });

      if (error) {
        throw new Error(String(error));
      }

      onSuccess();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form className="create-test-form" onSubmit={handleSubmit}>
      <h3>Create A/B Test</h3>

      {error && <div className="form-error">{error}</div>}

      <div className="form-group">
        <label htmlFor="test-name">Name *</label>
        <input
          id="test-name"
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="e.g., Article vs Product Template"
          required
        />
      </div>

      <div className="form-group">
        <label htmlFor="test-description">Description</label>
        <textarea
          id="test-description"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="Optional description of the test"
          rows={3}
        />
      </div>

      <div className="form-row">
        <div className="form-group">
          <label htmlFor="baseline-template">Baseline Template *</label>
          <select
            id="baseline-template"
            value={baselineTemplate}
            onChange={(e) => setBaselineTemplate(e.target.value)}
            required
          >
            <option value="">Select template...</option>
            {templates.map((t) => (
              <option key={t} value={t}>
                {t}
              </option>
            ))}
          </select>
        </div>

        <div className="form-group">
          <label htmlFor="variant-template">Variant Template *</label>
          <select
            id="variant-template"
            value={variantTemplate}
            onChange={(e) => setVariantTemplate(e.target.value)}
            required
          >
            <option value="">Select template...</option>
            {templates.map((t) => (
              <option key={t} value={t}>
                {t}
              </option>
            ))}
          </select>
        </div>
      </div>

      <div className="form-group">
        <label htmlFor="baseline-allocation">
          Baseline Allocation: {baselineAllocation}%
        </label>
        <input
          id="baseline-allocation"
          type="range"
          min={10}
          max={90}
          value={baselineAllocation}
          onChange={(e) => setBaselineAllocation(parseInt(e.target.value, 10))}
        />
        <div className="allocation-labels">
          <span>Baseline: {baselineAllocation}%</span>
          <span>Variant: {100 - baselineAllocation}%</span>
        </div>
      </div>

      <div className="form-row">
        <div className="form-group">
          <label htmlFor="min-sample-size">Min Sample Size</label>
          <input
            id="min-sample-size"
            type="number"
            min={10}
            max={10000}
            value={minSampleSize}
            onChange={(e) => setMinSampleSize(parseInt(e.target.value, 10))}
          />
        </div>

        <div className="form-group">
          <label htmlFor="confidence-level">Confidence Level</label>
          <select
            id="confidence-level"
            value={confidenceLevel}
            onChange={(e) => setConfidenceLevel(parseFloat(e.target.value))}
          >
            <option value={0.9}>90%</option>
            <option value={0.95}>95%</option>
            <option value={0.99}>99%</option>
          </select>
        </div>
      </div>

      <div className="form-actions">
        <button
          type="button"
          className="btn btn--secondary"
          onClick={onCancel}
          disabled={submitting}
        >
          Cancel
        </button>
        <button
          type="submit"
          className="btn btn--primary"
          disabled={submitting}
        >
          {submitting ? "Creating..." : "Create Test"}
        </button>
      </div>
    </form>
  );
}

interface ABTestCardProps {
  test: TemplateAbTest;
  onUpdate: () => void;
  onEditTemplate: (templateName: string) => void;
}

function ABTestCard({ test, onUpdate, onEditTemplate }: ABTestCardProps) {
  const [showResults, setShowResults] = useState(false);
  const [loading, setLoading] = useState(false);

  const getStatusBadge = () => {
    const statusClasses: Record<string, string> = {
      pending: "badge--neutral",
      running: "badge--success",
      paused: "badge--warning",
      completed: "badge--info",
    };

    const status = test.status ?? "pending";
    return (
      <span className={`badge ${statusClasses[status] || ""}`}>{status}</span>
    );
  };

  const handleStart = async () => {
    try {
      setLoading(true);
      const { error } = await postV1TemplateAbTestsByIdStart({
        baseUrl: getApiBaseUrl(),
        path: { id: test.id ?? "" },
      });

      if (error) {
        throw new Error(String(error));
      }

      onUpdate();
    } catch (err) {
      console.error("Failed to start test:", err);
    } finally {
      setLoading(false);
    }
  };

  const handleStop = async () => {
    try {
      setLoading(true);
      const { error } = await postV1TemplateAbTestsByIdStop({
        baseUrl: getApiBaseUrl(),
        path: { id: test.id || "" },
      });

      if (error) {
        throw new Error(String(error));
      }

      onUpdate();
    } catch (err) {
      console.error("Failed to stop test:", err);
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async () => {
    if (!confirm("Are you sure you want to delete this A/B test?")) {
      return;
    }

    try {
      setLoading(true);
      const { error } = await deleteV1TemplateAbTestsById({
        baseUrl: getApiBaseUrl(),
        path: { id: test.id || "" },
      });

      if (error) {
        throw new Error(String(error));
      }

      onUpdate();
    } catch (err) {
      console.error("Failed to delete test:", err);
    } finally {
      setLoading(false);
    }
  };

  const getProgress = () => {
    // This would ideally come from the test results
    // For now, show indeterminate progress for running tests
    if (test.status === "running") {
      return (
        <div className="test-progress">
          <div className="progress-bar progress-bar--indeterminate" />
          <span className="progress-label">Running...</span>
        </div>
      );
    }

    if (test.winner) {
      const winnerName =
        test.winner === "baseline"
          ? test.baseline_template
          : test.variant_template;
      return (
        <div className="test-progress">
          <span className="winner-badge">Winner: {winnerName}</span>
        </div>
      );
    }

    return null;
  };

  return (
    <div className={`ab-test-card ab-test-card--${test.status}`}>
      <div className="ab-test-card__header">
        <div className="ab-test-card__title">
          <h4>{test.name}</h4>
          {getStatusBadge()}
        </div>
        <div className="ab-test-card__actions">
          {test.status === "pending" && (
            <button
              type="button"
              className="btn btn--small btn--primary"
              onClick={handleStart}
              disabled={loading}
            >
              Start
            </button>
          )}
          {test.status === "running" && (
            <button
              type="button"
              className="btn btn--small btn--warning"
              onClick={handleStop}
              disabled={loading}
            >
              Stop
            </button>
          )}
          <button
            type="button"
            className="btn btn--small btn--secondary"
            onClick={() => setShowResults(!showResults)}
          >
            {showResults ? "Hide Results" : "View Results"}
          </button>
          <button
            type="button"
            className="btn btn--small btn--danger"
            onClick={handleDelete}
            disabled={loading}
          >
            Delete
          </button>
        </div>
      </div>

      {test.description && (
        <p className="ab-test-card__description">{test.description}</p>
      )}

      <div className="ab-test-card__details">
        <div className="detail-row">
          <span className="detail-label">Templates:</span>
          <span className="detail-value">
            {test.baseline_template}
            <button
              type="button"
              className="btn btn--link btn--small"
              onClick={() => onEditTemplate(test.baseline_template ?? "")}
              title="Edit baseline template"
            >
              Edit
            </button>
            {" vs "}
            {test.variant_template}
            <button
              type="button"
              className="btn btn--link btn--small"
              onClick={() => onEditTemplate(test.variant_template ?? "")}
              title="Edit variant template"
            >
              Edit
            </button>
          </span>
        </div>
        <div className="detail-row">
          <span className="detail-label">Allocation:</span>
          <span className="detail-value">
            {test.allocation?.baseline ?? 50}% /{" "}
            {test.allocation?.variant ?? 50}%
          </span>
        </div>
        <div className="detail-row">
          <span className="detail-label">Min Samples:</span>
          <span className="detail-value">{test.min_sample_size}</span>
        </div>
        <div className="detail-row">
          <span className="detail-label">Confidence:</span>
          <span className="detail-value">
            {((test.confidence_level ?? 0.95) * 100).toFixed(0)}%
          </span>
        </div>
      </div>

      {getProgress()}

      {showResults && test.id && <TestResults testId={test.id} />}
    </div>
  );
}

interface TestResultsProps {
  testId: string;
}

function TestResults({ testId }: TestResultsProps) {
  const [results, setResults] = useState<TemplateComparison | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

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
        <AutoSelectButton
          testId={testId}
          onUpdate={() => window.location.reload()}
        />
      )}
    </div>
  );
}

interface AutoSelectButtonProps {
  testId: string;
  onUpdate: () => void;
}

function AutoSelectButton({ testId, onUpdate }: AutoSelectButtonProps) {
  const [loading, setLoading] = useState(false);

  const handleClick = async () => {
    if (
      !confirm(
        "This will select the winning template and complete the test. Continue?",
      )
    ) {
      return;
    }

    try {
      setLoading(true);
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
      setLoading(false);
    }
  };

  return (
    <button
      type="button"
      className="btn btn--primary btn--small"
      onClick={handleClick}
      disabled={loading}
    >
      {loading ? "Selecting..." : "Auto-Select Winner"}
    </button>
  );
}
