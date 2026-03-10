/**
 * Template A/B test creation form.
 *
 * Purpose:
 * - Render and submit the create-test flow for template A/B tests.
 *
 * Responsibilities:
 * - Manage local form state and validation.
 * - Submit new A/B tests through the API and report success to the parent.
 *
 * Scope:
 * - Template A/B test creation UI only.
 *
 * Usage:
 * - Render inside TemplateABTestManager when the create form is visible.
 *
 * Invariants/Assumptions:
 * - Baseline and variant templates must both be selected and must differ.
 */

import { useState } from "react";

import { postV1TemplateAbTests } from "../../../api";
import { getApiBaseUrl } from "../../../lib/api-config";

interface CreateTestFormProps {
  templates: string[];
  onSuccess: () => void;
  onCancel: () => void;
}

export function CreateTestForm({
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
            {templates.map((templateName) => (
              <option key={templateName} value={templateName}>
                {templateName}
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
            {templates.map((templateName) => (
              <option key={templateName} value={templateName}>
                {templateName}
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
